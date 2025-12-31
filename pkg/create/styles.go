package create

import "github.com/charmbracelet/lipgloss"

// Styles for the create TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("40")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 2)

	progressBarStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				PaddingRight(2)

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
)
