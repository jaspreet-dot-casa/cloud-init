package create

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
			Foreground(lipgloss.Color("8"))

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

// Internal styles (not exported, for backward compatibility)
var (
	titleStyle    = TitleStyle
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	successStyle = SuccessStyle
	errorStyle   = ErrorStyle
	warningStyle = WarningStyle
	dimStyle     = DimStyle

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

	selectedStyle     = SelectedStyle
	unselectedStyle   = UnselectedStyle
	labelStyle        = LabelStyle
	valueStyle        = ValueStyle
	focusedInputStyle = FocusedInputStyle

	blurredInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	phaseIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)
