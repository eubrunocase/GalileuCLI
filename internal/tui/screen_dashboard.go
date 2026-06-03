package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"Galileu/internal/guardian"
)

// auditLine holds a parsed line from galileu_audit.log for display.
type auditLine struct {
	timestamp string
	host      string
	redacted  bool
}

// updateDashboard handles key input on the dashboard screen.
func (m model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "Q":
		return m, tea.Quit
	case "p", "P":
		nm, cmd := m.navigateTo(screenPatterns)
		return nm, cmd
	case "c", "C":
		nm, cmd := m.navigateTo(screenProxy)
		return nm, cmd
	}
	return m, nil
}

// viewDashboard renders the full dashboard.
func (m model) viewDashboard(width int) string {
	var b strings.Builder

	// Header
	right := statusRunningStyle.Render("● A CORRER")
	if m.dryRun {
		right = " " + dryRunBadge.Render("DRY-RUN") + "  " + right
	}
	b.WriteString(renderHeader("GALILEU  v2.0", right, width))
	b.WriteString("\n")
	b.WriteString(renderDivider(width))
	b.WriteString("\n")

	// Two-column body
	half := width / 2
	maxLines := m.height - 6
	if maxLines < 4 {
		maxLines = 4
	}
	leftPanel := m.renderDashboardLeft(half - 1)
	rightPanel := m.renderDashboardRight(width-half-1, maxLines)

	leftLines := strings.Split(leftPanel, "\n")
	rightLines := strings.Split(rightPanel, "\n")

	// Pad to same number of lines
	bodyHeight := m.height - 6
	if bodyHeight < 4 {
		bodyHeight = 4
	}
	for len(leftLines) < bodyHeight {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < bodyHeight {
		rightLines = append(rightLines, "")
	}

	sep := dividerStyle.Render("│")
	for i := 0; i < bodyHeight; i++ {
		lLine := leftLines[i]
		rLine := ""
		if i < len(rightLines) {
			rLine = rightLines[i]
		}
		// Pad left column to half width
		lPadded := padRight(lLine, half)
		b.WriteString(lPadded + sep + rLine + "\n")
	}

	b.WriteString(renderDivider(width))
	b.WriteString("\n")
	b.WriteString(renderFooter(
		"  [P] Padrões    [C] Config Proxy    [Q] Sair",
		m.statusMsg, m.statusIsErr, width,
	))

	return b.String()
}

func (m model) renderDashboardLeft(width int) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("  PROXY") + "\n")
	b.WriteString(labelValue("  Porta", fmt.Sprintf(":%d", m.port)) + "\n")

	mode := "whitelist"
	if m.prx.loaded {
		mode = m.prx.cfg.Proxy.Mode
	}
	b.WriteString(labelValue("  Modo", mode) + "\n")
	b.WriteString("\n")

	allowedCount := 0
	if m.prx.loaded {
		allowedCount = len(m.prx.cfg.Proxy.AllowedHosts)
	}
	patCount := 0
	if m.pat.loaded {
		patCount = len(m.pat.cfg.Analyzer.CustomPatterns)
	}

	b.WriteString(stat("  Hosts permitidos", allowedCount) + "\n")
	b.WriteString(stat("  Padrões activos", patCount) + "\n")
	b.WriteString("\n")
	b.WriteString(labelValue("  Total", fmt.Sprintf("%d", m.stats.Total)) + "\n")
	b.WriteString(labelValue("  Redatados", fmt.Sprintf("%d", m.stats.Redacted)) + "\n")
	b.WriteString(labelValue("  Erros", fmt.Sprintf("%d", m.stats.Errors)) + "\n")

	uptime := time.Since(m.startAt).Round(time.Second)
	b.WriteString("\n")
	b.WriteString(labelValue("  Uptime", uptime.String()) + "\n")

	return b.String()
}

func (m model) renderDashboardRight(width, maxLines int) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("  AUDIT LOG (tempo real)") + "\n")

	lines := readAuditLines(maxLines - 2)
	// Also show in-memory events if audit file is empty
	if len(lines) == 0 && len(m.events) > 0 {
		for i := len(m.events) - 1; i >= 0 && len(lines) < maxLines-2; i-- {
			req := m.events[i]
			ts := ""
			redacted := req.Redacted
			lines = append([]auditLine{{timestamp: ts, host: req.Host, redacted: redacted}}, lines...)
		}
	}

	if len(lines) == 0 {
		b.WriteString(dimStyle.Render("  Aguardando requisições...") + "\n")
		return b.String()
	}

	for _, al := range lines {
		redactStr := "não"
		if al.redacted {
			redactStr = "sim"
		}
		ts := al.timestamp
		if len(ts) >= 5 {
			ts = ts[11:16] // HH:MM from RFC3339
		}
		line := fmt.Sprintf("  %s %-24s ✂ %s", ts, truncate(al.host, 24), redactStr)
		if al.redacted {
			b.WriteString(warnStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

// readAuditLines reads the last n JSON lines from galileu_audit.log.
func readAuditLines(n int) []auditLine {
	if n <= 0 {
		return nil
	}
	f, err := os.Open("galileu_audit.log")
	if err != nil {
		return nil
	}
	defer f.Close()

	var all []auditLine
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		raw := scanner.Bytes()
		var e guardian.AuditEntry
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
		if e.Provider == "skipped" {
			continue
		}
		all = append(all, auditLine{
			timestamp: e.Timestamp,
			host:      e.Host,
			redacted:  e.Redacted,
		})
	}

	if len(all) <= n {
		return all
	}
	return all[len(all)-n:]
}
