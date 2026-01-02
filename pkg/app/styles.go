package app

import "github.com/charmbracelet/lipgloss"

// Common styles used across the application.
var (
	// Status colors
	StatusRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("40")).
				Bold(true)

	StatusStoppedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	StatusUnknownStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	// Text styles
	BoldStyle = lipgloss.NewStyle().Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	AccentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("40"))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	// Layout styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Bold(true)

	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(lipgloss.Color("240"))

	// Spinner style
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
)

// StatusColor returns the appropriate color for a VM status.
func StatusColor(status string) lipgloss.Style {
	switch status {
	case "running":
		return StatusRunningStyle
	case "stopped", "shutoff", "shut off":
		return StatusStoppedStyle
	case "crashed", "error":
		return StatusErrorStyle
	default:
		return StatusUnknownStyle
	}
}

// RenderStatus renders a status string with appropriate coloring.
func RenderStatus(status string) string {
	return StatusColor(status).Render(status)
}
