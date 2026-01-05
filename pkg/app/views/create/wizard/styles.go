package wizard

import "github.com/charmbracelet/lipgloss"

// Exported styles for use by phases package
var (
	// TitleStyle is used for phase titles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)

	// DimStyle is used for hints and secondary text
	DimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	// SuccessStyle is used for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("40")).
			Bold(true)

	// ErrorStyle is used for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	// WarningStyle is used for warning messages
	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	// SelectedStyle is used for selected items in lists
	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Bold(true)

	// UnselectedStyle is used for unselected items in lists
	UnselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// LabelStyle is used for field labels
	LabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// ValueStyle is used for displaying values
	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// FocusedInputStyle is used for focused input fields
	FocusedInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Bold(true)
)
