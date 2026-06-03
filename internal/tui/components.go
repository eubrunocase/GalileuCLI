package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHeader renders the top bar shared by all screens.
func renderHeader(title, rightSection string, width int) string {
	left := headerStyle.Render(" " + title + " ")
	right := rightSection

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

// renderFooter renders the bottom key-hint bar with an optional status line.
func renderFooter(hints, status string, isErr bool, width int) string {
	bar := dimStyle.Render(hints)
	if status == "" {
		return bar
	}
	if isErr {
		return bar + "\n" + errorStyle.Render("  "+status)
	}
	return bar + "\n" + successStyle.Render("  "+status)
}

// renderDivider returns a full-width horizontal rule.
func renderDivider(width int) string {
	return dividerStyle.Render(strings.Repeat("─", width))
}

// checkboxStr renders a checkbox glyph.
func checkboxStr(enabled bool) string {
	if enabled {
		return "☑"
	}
	return "☐"
}

// renderConfirmDialog renders a centered yes/no dialog.
func (m model) renderConfirmDialog(width int) string {
	prompt := m.confirm.prompt
	line := "  " + warnStyle.Render(prompt) + "  " + dimStyle.Render("[Enter/y confirma  Qualquer tecla cancela]")
	pad := (width - lipgloss.Width(line)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat("\n", m.height/2) + strings.Repeat(" ", pad) + line
}

// truncate truncates s to max runes, appending "…" if truncated.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// columns lays out two strings side by side filling width.
func columns(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// padRight pads s to exactly n visible characters.
func padRight(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

// labelValue renders a "label: value" pair using shared styles.
func labelValue(label, value string) string {
	return labelStyle.Render(label+": ") + valueStyle.Render(value)
}

// stat formats a numeric counter.
func stat(label string, n int) string {
	return labelValue(label, fmt.Sprintf("%d", n))
}
