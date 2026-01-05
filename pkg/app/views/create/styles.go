package create

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
)

// Re-export styles from wizard package as the canonical source.
// This ensures consistency across the create package and its subpackages.
var (
	TitleStyle        = wizard.TitleStyle
	DimStyle          = wizard.DimStyle
	SuccessStyle      = wizard.SuccessStyle
	ErrorStyle        = wizard.ErrorStyle
	WarningStyle      = wizard.WarningStyle
	SelectedStyle     = wizard.SelectedStyle
	UnselectedStyle   = wizard.UnselectedStyle
	LabelStyle        = wizard.LabelStyle
	ValueStyle        = wizard.ValueStyle
	FocusedInputStyle = wizard.FocusedInputStyle
)

// Internal styles (unexported, specific to create package views)
var (
	titleStyle    = wizard.TitleStyle
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	successStyle = wizard.SuccessStyle
	errorStyle   = wizard.ErrorStyle
	warningStyle = wizard.WarningStyle
	dimStyle     = wizard.DimStyle

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

	selectedStyle     = wizard.SelectedStyle
	unselectedStyle   = wizard.UnselectedStyle
	labelStyle        = wizard.LabelStyle
	valueStyle        = wizard.ValueStyle
	focusedInputStyle = wizard.FocusedInputStyle

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
