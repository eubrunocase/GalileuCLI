package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"Galileu/internal/guardian"
)

// screen identifies which screen is currently active.
type screen int

const (
	screenDashboard screen = iota
	screenPatterns
	screenProxy
)

// eventMsg wraps an incoming LogRequest so bubbletea can process it.
type eventMsg guardian.LogRequest

// tickMsg is sent every second to refresh the dashboard.
type tickMsg time.Time

// clearStatusMsg clears the footer status line after a delay.
type clearStatusMsg struct{}

// confirmMsg carries a pending confirmation action.
type confirmMsg struct {
	action func(model) (model, tea.Cmd)
	prompt string
}

// Stats aggregates running counters for the dashboard.
type Stats struct {
	Total     int
	Redacted  int
	Errors    int
	Providers map[string]int
}

const maxEvents = 50

// model is the root bubbletea application state.
type model struct {
	currentScreen screen
	width         int
	height        int

	// shared state
	port    int
	dryRun  bool
	startAt time.Time

	// dashboard
	stats  Stats
	events []guardian.LogRequest

	// patterns screen
	pat patternScreenState

	// proxy config screen
	prx proxyScreenState

	// footer status (used across screens)
	statusMsg    string
	statusIsErr  bool
	configPath   string

	// pending confirmation dialog
	confirm *confirmMsg
}

// New returns the initial model.
func New(port int, dryRun bool, configPath string) model {
	return model{
		currentScreen: screenDashboard,
		port:          port,
		dryRun:        dryRun,
		startAt:       time.Now(),
		stats:         Stats{Providers: make(map[string]int)},
		events:        make([]guardian.LogRequest, 0, maxEvents),
		configPath:    configPath,
	}
}

// Init starts the first tick.
func (m model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update dispatches all messages to the active screen or handles global keys.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tickCmd()

	case clearStatusMsg:
		m.statusMsg = ""
		m.statusIsErr = false
		return m, nil

	case eventMsg:
		m = m.applyEvent(guardian.LogRequest(msg))
		return m, nil

	case tea.KeyMsg:
		// Confirmation dialog intercepts all keys.
		if m.confirm != nil {
			return m.handleConfirmKey(msg)
		}
		return m.routeKey(msg)
	}

	return m, nil
}

// routeKey dispatches key messages to the active screen.
func (m model) routeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case screenPatterns:
		return m.updatePatterns(msg)
	case screenProxy:
		return m.updateProxy(msg)
	default:
		return m.updateDashboard(msg)
	}
}

// handleConfirmKey processes keys when the confirmation dialog is shown.
func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		action := m.confirm.action
		m.confirm = nil
		return action(m)
	default:
		m.confirm = nil
		return m, nil
	}
}

// navigateTo switches screens and loads config when needed.
func (m model) navigateTo(s screen) (model, tea.Cmd) {
	m.currentScreen = s
	switch s {
	case screenPatterns:
		if !m.pat.loaded {
			m = m.loadConfigForPatterns()
		}
	case screenProxy:
		if !m.prx.loaded {
			m = m.loadConfigForProxy()
		}
	}
	return m, nil
}

func (m model) loadConfigForPatterns() model {
	cfg, err := guardian.LoadRawConfig(m.configPath)
	if err != nil {
		m.statusMsg = "Erro ao carregar config: " + err.Error()
		m.statusIsErr = true
		return m
	}
	m.pat = newPatternScreenState(cfg)
	return m
}

func (m model) loadConfigForProxy() model {
	cfg, err := guardian.LoadRawConfig(m.configPath)
	if err != nil {
		m.statusMsg = "Erro ao carregar config: " + err.Error()
		m.statusIsErr = true
		return m
	}
	m.prx = newProxyScreenState(cfg)
	return m
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

// scheduleStatusClear returns a command that clears the status after 2 seconds.
func scheduleStatusClear() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// View delegates rendering to the active screen.
func (m model) View() string {
	w := m.width
	if w < 60 {
		w = 80
	}

	if m.confirm != nil {
		return m.renderConfirmDialog(w)
	}

	switch m.currentScreen {
	case screenPatterns:
		return m.viewPatterns(w)
	case screenProxy:
		return m.viewProxy(w)
	default:
		return m.viewDashboard(w)
	}
}
