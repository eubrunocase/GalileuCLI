package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"Galileu/internal/guardian"
)

// Start launches the Bubble Tea program with the proxy event channel.
// It blocks until the user quits.
func Start(port int, dryRun bool, events <-chan guardian.LogRequest) error {
	m := New(port, dryRun)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Forward proxy events into bubbletea.
	go func() {
		for req := range events {
			p.Send(eventMsg(req))
		}
	}()

	_, err := p.Run()
	return err
}
