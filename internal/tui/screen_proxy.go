package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"Galileu/internal/guardian"
)

// proxyFocusArea identifies which interactive zone has keyboard focus.
type proxyFocusArea int

const (
	focusPort proxyFocusArea = iota
	focusMode
	focusAllowed
	focusSkip
	focusSave
)

// proxyScreenState holds all state for screen 3.
type proxyScreenState struct {
	loaded  bool
	cfg     guardian.GalileuConfig
	dirty   bool
	focus   proxyFocusArea
	portStr string // raw string while editing

	// list cursors
	allowedCursor int
	skipCursor    int

	// inline input
	addingAllowed bool
	addingSkip    bool
	inputBuf      string
}

func newProxyScreenState(cfg guardian.GalileuConfig) proxyScreenState {
	return proxyScreenState{
		loaded:  true,
		cfg:     cfg,
		portStr: strconv.Itoa(cfg.Port),
	}
}

// updateProxy handles all key input on the proxy config screen.
func (m model) updateProxy(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Inline add-host input is active.
	if m.prx.addingAllowed || m.prx.addingSkip {
		return m.updateProxyInput(key)
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "b", "B", "esc":
		if m.prx.dirty {
			m.confirm = &confirmMsg{
				prompt: "Há alterações não guardadas. Descartar?",
				action: func(mm model) (model, tea.Cmd) {
					mm.currentScreen = screenDashboard
					mm.prx.dirty = false
					return mm, nil
				},
			}
			return m, nil
		}
		m.currentScreen = screenDashboard
		return m, nil
	case "tab", "down":
		if m.prx.focus < focusSave {
			m.prx.focus++
		}
	case "shift+tab", "up":
		if m.prx.focus > focusPort {
			m.prx.focus--
		}
	case "enter", " ":
		return m.proxyActivateFocus()
	case "d", "D":
		return m.proxyDeleteFocused()
	case "left":
		if m.prx.focus == focusMode {
			m.prx.cfg.Proxy.Mode = "whitelist"
			m.prx.dirty = true
		}
	case "right":
		if m.prx.focus == focusMode {
			m.prx.cfg.Proxy.Mode = "passive"
			m.prx.dirty = true
		}
	case "backspace":
		if m.prx.focus == focusPort && len(m.prx.portStr) > 0 {
			m.prx.portStr = m.prx.portStr[:len(m.prx.portStr)-1]
			m.prx.dirty = true
		}
	default:
		if m.prx.focus == focusPort && len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
			m.prx.portStr += key
			m.prx.dirty = true
		}
	}

	return m, nil
}

func (m model) proxyActivateFocus() (model, tea.Cmd) {
	switch m.prx.focus {
	case focusMode:
		if m.prx.cfg.Proxy.Mode == "whitelist" {
			m.prx.cfg.Proxy.Mode = "passive"
		} else {
			m.prx.cfg.Proxy.Mode = "whitelist"
		}
		m.prx.dirty = true
	case focusAllowed:
		m.prx.addingAllowed = true
		m.prx.inputBuf = ""
	case focusSkip:
		m.prx.addingSkip = true
		m.prx.inputBuf = ""
	case focusSave:
		return m.saveProxyConfig()
	}
	return m, nil
}

func (m model) proxyDeleteFocused() (model, tea.Cmd) {
	switch m.prx.focus {
	case focusAllowed:
		idx := m.prx.allowedCursor
		hosts := m.prx.cfg.Proxy.AllowedHosts
		if idx < len(hosts) {
			name := hosts[idx]
			m.confirm = &confirmMsg{
				prompt: fmt.Sprintf("Remover host '%s'?", name),
				action: func(mm model) (model, tea.Cmd) {
					mm.prx.cfg.Proxy.AllowedHosts = append(hosts[:idx], hosts[idx+1:]...)
					if mm.prx.allowedCursor >= len(mm.prx.cfg.Proxy.AllowedHosts) && mm.prx.allowedCursor > 0 {
						mm.prx.allowedCursor--
					}
					mm.prx.dirty = true
					return mm, nil
				},
			}
		}
	case focusSkip:
		idx := m.prx.skipCursor
		hosts := m.prx.cfg.Proxy.SkipHosts
		if idx < len(hosts) {
			name := hosts[idx]
			m.confirm = &confirmMsg{
				prompt: fmt.Sprintf("Remover host ignorado '%s'?", name),
				action: func(mm model) (model, tea.Cmd) {
					mm.prx.cfg.Proxy.SkipHosts = append(hosts[:idx], hosts[idx+1:]...)
					if mm.prx.skipCursor >= len(mm.prx.cfg.Proxy.SkipHosts) && mm.prx.skipCursor > 0 {
						mm.prx.skipCursor--
					}
					mm.prx.dirty = true
					return mm, nil
				},
			}
		}
	}
	return m, nil
}

// updateProxyInput handles keystrokes while the inline add-host input is open.
func (m model) updateProxyInput(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.prx.addingAllowed = false
		m.prx.addingSkip = false
		m.prx.inputBuf = ""
	case "enter":
		host := strings.TrimSpace(m.prx.inputBuf)
		if host != "" {
			if m.prx.addingAllowed {
				m.prx.cfg.Proxy.AllowedHosts = append(m.prx.cfg.Proxy.AllowedHosts, host)
			} else {
				m.prx.cfg.Proxy.SkipHosts = append(m.prx.cfg.Proxy.SkipHosts, host)
			}
			m.prx.dirty = true
		}
		m.prx.addingAllowed = false
		m.prx.addingSkip = false
		m.prx.inputBuf = ""
	case "backspace":
		if len(m.prx.inputBuf) > 0 {
			m.prx.inputBuf = m.prx.inputBuf[:len(m.prx.inputBuf)-1]
		}
	default:
		if len(key) == 1 {
			m.prx.inputBuf += key
		}
	}
	return m, nil
}

func (m model) saveProxyConfig() (model, tea.Cmd) {
	// Validate port
	if p, err := strconv.Atoi(m.prx.portStr); err == nil && p > 0 && p <= 65535 {
		m.prx.cfg.Port = p
	} else {
		m.statusMsg = "Porta inválida (1-65535)"
		m.statusIsErr = true
		return m, scheduleStatusClear()
	}

	if err := guardian.SaveConfig(m.configPath, m.prx.cfg); err != nil {
		m.statusMsg = "Erro ao guardar: " + err.Error()
		m.statusIsErr = true
		return m, scheduleStatusClear()
	}

	m.prx.dirty = false
	m.statusMsg = "✓ Guardado — reinicia o proxy para aplicar"
	m.statusIsErr = false
	return m, scheduleStatusClear()
}

// viewProxy renders the proxy config screen.
func (m model) viewProxy(width int) string {
	var b strings.Builder

	b.WriteString(renderHeader("CONFIGURAÇÃO DO PROXY", "", width))
	b.WriteString("\n")
	b.WriteString(renderDivider(width))
	b.WriteString("\n\n")

	// Port and mode row
	portLine := m.proxyFieldRow(focusPort, "  Porta", m.prx.portStr)
	modeLine := m.proxyFieldRow(focusMode, "  Modo", "[ "+m.prx.cfg.Proxy.Mode+" ▼]")
	b.WriteString(portLine + "        " + modeLine + "\n\n")

	// Two host lists
	half := (width - 4) / 2
	allowedPanel := m.renderHostList("HOSTS PERMITIDOS", m.prx.cfg.Proxy.AllowedHosts,
		m.prx.allowedCursor, m.prx.focus == focusAllowed, m.prx.addingAllowed, half)
	skipPanel := m.renderHostList("HOSTS IGNORADOS", m.prx.cfg.Proxy.SkipHosts,
		m.prx.skipCursor, m.prx.focus == focusSkip, m.prx.addingSkip, half)

	aLines := strings.Split(allowedPanel, "\n")
	sLines := strings.Split(skipPanel, "\n")
	maxLen := len(aLines)
	if len(sLines) > maxLen {
		maxLen = len(sLines)
	}
	for len(aLines) < maxLen {
		aLines = append(aLines, "")
	}
	for len(sLines) < maxLen {
		sLines = append(sLines, "")
	}
	for i := 0; i < maxLen; i++ {
		b.WriteString(padRight(aLines[i], half+2) + "  " + sLines[i] + "\n")
	}

	b.WriteString("\n")
	saveLine := "  [Guardar alterações]"
	if m.prx.focus == focusSave {
		b.WriteString(focusedStyle.Render(saveLine) + "\n")
	} else {
		b.WriteString(saveLine + "\n")
	}

	b.WriteString("\n")
	hints := "  Tab/↑↓ navegar  D apagar  Enter seleccionar  B voltar"
	b.WriteString(renderFooter(hints, m.statusMsg, m.statusIsErr, width))

	return b.String()
}

func (m model) proxyFieldRow(area proxyFocusArea, label, value string) string {
	line := label + ": " + value
	if m.prx.focus == area {
		return focusedStyle.Render(line)
	}
	return line
}

func (m model) renderHostList(title string, hosts []string, cursor int, focused, adding bool, width int) string {
	var b strings.Builder
	titleStyle := sectionStyle
	if focused {
		titleStyle = focusedStyle
	}
	b.WriteString(titleStyle.Render("  "+title) + "\n")
	b.WriteString(dividerStyle.Render("  "+strings.Repeat("─", width-2)) + "\n")

	for i, h := range hosts {
		line := fmt.Sprintf("  %s", truncate(h, width-4))
		if i == cursor && focused {
			b.WriteString(focusedStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	if adding {
		b.WriteString(focusedStyle.Render("  > "+m.prx.inputBuf+"_") + "\n")
	} else {
		addLabel := "  [+ Adicionar]"
		if focused {
			b.WriteString(dimStyle.Render(addLabel) + "\n")
		} else {
			b.WriteString(dimStyle.Render(addLabel) + "\n")
		}
	}

	b.WriteString(dividerStyle.Render("  "+strings.Repeat("─", width-2)) + "\n")
	return b.String()
}
