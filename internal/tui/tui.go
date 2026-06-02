package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"Galileu/internal/guardian"
)

// eventMsg wraps an incoming LogRequest so bubbletea can process it.
type eventMsg guardian.LogRequest

// tickMsg is sent on each clock tick to refresh the elapsed-time counter.
type tickMsg time.Time

// Stats aggregates running counters displayed in the header panel.
type Stats struct {
	Total     int
	Redacted  int
	Errors    int
	Providers map[string]int
}

// model is the bubbletea application state.
type model struct {
	port    int
	dryRun  bool
	startAt time.Time

	stats  Stats
	events []guardian.LogRequest // ring-buffer — last maxEvents entries
	width  int
	height int
}

const maxEvents = 50

// styles — all colour/layout decisions live here.
var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	danger    = lipgloss.AdaptiveColor{Light: "#CC3333", Dark: "#FF5F57"}
	warn      = lipgloss.AdaptiveColor{Light: "#CC8800", Dark: "#FFCC00"}

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#888", Dark: "#777"})

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#222", Dark: "#EEE"}).
			Bold(true)

	redactedBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#CC3333")).
			Padding(0, 1)

	cleanBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#43BF6D")).
			Padding(0, 1)

	errorBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#CC8800")).
			Padding(0, 1)

	dimStyle = lipgloss.NewStyle().
			Foreground(subtle)

	dividerStyle = lipgloss.NewStyle().
			Foreground(subtle)

	dryRunBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#CC8800")).
			Bold(true).
			Padding(0, 1)
)

// New returns the initial model ready to render.
func New(port int, dryRun bool) model {
	return model{
		port:    port,
		dryRun:  dryRun,
		startAt: time.Now(),
		stats:   Stats{Providers: make(map[string]int)},
		events:  make([]guardian.LogRequest, 0, maxEvents),
	}
}

// Init returns the first command: start ticking every second.
func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles all messages.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tick()

	case eventMsg:
		req := guardian.LogRequest(msg)
		m = m.applyEvent(req)
	}

	return m, nil
}

// applyEvent integrates a new request into stats and the event ring.
func (m model) applyEvent(req guardian.LogRequest) model {
	if req.Provider == "skipped" {
		return m
	}

	m.stats.Total++
	if req.Redacted {
		m.stats.Redacted++
	}
	if req.ProxyError {
		m.stats.Errors++
	}
	if req.Provider != "" && req.Provider != "unknown" {
		m.stats.Providers[req.Provider]++
	}

	m.events = append(m.events, req)
	if len(m.events) > maxEvents {
		m.events = m.events[len(m.events)-maxEvents:]
	}

	return m
}

// View renders the full TUI frame.
func (m model) View() string {
	width := m.width
	if width < 60 {
		width = 80
	}

	var b strings.Builder

	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(m.renderStats(width))
	b.WriteString("\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")
	b.WriteString(m.renderEventList(width))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  q / ctrl+c para encerrar"))

	return b.String()
}

func (m model) renderHeader(width int) string {
	elapsed := time.Since(m.startAt).Round(time.Second)

	title := headerStyle.Render(" GALILEU — Proxy de Segurança ")

	mode := ""
	if m.dryRun {
		mode = " " + dryRunBadge.Render("DRY-RUN")
	}

	portStr := labelStyle.Render("  porta ") + valueStyle.Render(fmt.Sprintf(":%d", m.port))
	uptime := labelStyle.Render("  uptime ") + valueStyle.Render(elapsed.String())

	right := portStr + "   " + uptime + mode

	// Pad to fill the width
	gap := width - lipgloss.Width(title) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return title + strings.Repeat(" ", gap) + right
}

func (m model) renderStats(width int) string {
	totalStr := fmt.Sprintf("%d", m.stats.Total)
	redactedStr := fmt.Sprintf("%d", m.stats.Redacted)
	errorsStr := fmt.Sprintf("%d", m.stats.Errors)

	col := func(label, val string) string {
		return "  " + labelStyle.Render(label+": ") + valueStyle.Render(val)
	}

	providers := ""
	if len(m.stats.Providers) > 0 {
		parts := make([]string, 0, len(m.stats.Providers))
		for p, n := range m.stats.Providers {
			parts = append(parts, fmt.Sprintf("%s(%d)", p, n))
		}
		providers = col("providers", strings.Join(parts, " "))
	}

	return col("total", totalStr) +
		col("redatados", redactedStr) +
		col("erros", errorsStr) +
		providers
}

func (m model) renderEventList(width int) string {
	if len(m.events) == 0 {
		return dimStyle.Render("  Aguardando requisições...\n")
	}

	// Show the most recent events that fit, newest at top.
	availableLines := m.height - 7 // header(1) + stats(1) + divider(1) + footer(1) + margins
	if availableLines < 4 {
		availableLines = 4
	}

	start := len(m.events) - availableLines
	if start < 0 {
		start = 0
	}
	visible := m.events[start:]

	var b strings.Builder
	// Reverse: newest first
	for i := len(visible) - 1; i >= 0; i-- {
		b.WriteString(m.renderEvent(visible[i], width))
	}
	return b.String()
}

func (m model) renderEvent(req guardian.LogRequest, width int) string {
	badge := ""
	switch {
	case req.ProxyError:
		badge = errorBadge.Render("ERRO")
	case req.Redacted:
		badge = redactedBadge.Render(fmt.Sprintf("REDATADO x%d", req.PatternCount))
	default:
		badge = cleanBadge.Render("OK")
	}

	provider := req.Provider
	if provider == "" {
		provider = "unknown"
	}

	model := req.Model
	if model == "" {
		model = "—"
	}

	latency := fmt.Sprintf("%dms", req.ProxyLatencyMs)

	main := fmt.Sprintf("  %-18s %-12s %-30s %s",
		provider, model, req.Host, latency)

	// Trim if wider than terminal
	mainWidth := width - lipgloss.Width(badge) - 3
	if mainWidth > 0 && len(main) > mainWidth {
		main = main[:mainWidth]
	}

	line := badge + " " + main

	if req.Redacted && len(req.DetectedPatterns) > 0 {
		detail := dimStyle.Render(fmt.Sprintf("       padroes: %s", strings.Join(req.DetectedPatterns, ", ")))
		return line + "\n" + detail + "\n"
	}

	return line + "\n"
}

// Run starts the bubbletea program, consuming events from the channel.
// It blocks until the user quits or the channel is closed.
func Run(events <-chan guardian.LogRequest, port int, dryRun bool) error {
	m := New(port, dryRun)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Forward incoming proxy events to bubbletea via Send.
	go func() {
		for req := range events {
			p.Send(eventMsg(req))
		}
	}()

	_, err := p.Run()
	return err
}
