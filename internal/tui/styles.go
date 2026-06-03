package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPurple  = lipgloss.Color("#7D56F4")
	colorGreen   = lipgloss.Color("#43BF6D")
	colorRed     = lipgloss.Color("#CC3333")
	colorOrange  = lipgloss.Color("#CC8800")
	colorYellow  = lipgloss.Color("#FFCC00")
	colorWhite   = lipgloss.Color("#FFFFFF")
	colorGrayDim = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	colorGrayMid = lipgloss.AdaptiveColor{Light: "#888", Dark: "#777"}
	colorFg      = lipgloss.AdaptiveColor{Light: "#222", Dark: "#EEE"}
	colorSection = lipgloss.AdaptiveColor{Light: "#555", Dark: "#AAA"}
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorPurple).
			Padding(0, 1)

	statusRunningStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorGrayMid)

	valueStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorGrayDim)

	dividerStyle = lipgloss.NewStyle().
			Foreground(colorGrayDim)

	focusedStyle = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSection)

	successStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	warnStyle = lipgloss.NewStyle().
			Foreground(colorOrange).
			Bold(true)

	redactedBadge = lipgloss.NewStyle().
			Foreground(colorWhite).
			Background(colorRed).
			Padding(0, 1)

	cleanBadge = lipgloss.NewStyle().
			Foreground(colorWhite).
			Background(colorGreen).
			Padding(0, 1)

	errorBadge = lipgloss.NewStyle().
			Foreground(colorWhite).
			Background(colorOrange).
			Padding(0, 1)

	dryRunBadge = lipgloss.NewStyle().
			Foreground(colorWhite).
			Background(colorOrange).
			Bold(true).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGrayDim).
			Padding(0, 1)

	boxFocusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurple).
			Padding(0, 1)

	inputValidStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Padding(0, 1)

	inputInvalidStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorRed).
				Padding(0, 1)

	inputPartialStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorYellow).
				Padding(0, 1)
)
