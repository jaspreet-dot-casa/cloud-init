// Package tui provides the terminal user interface for ucli.
package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Theme returns the custom theme for the TUI forms.
func Theme() *huh.Theme {
	t := huh.ThemeBase()

	// Customize colors
	t.Focused.Title = t.Focused.Title.Foreground(lipgloss.Color("39"))           // Cyan
	t.Focused.Description = t.Focused.Description.Foreground(lipgloss.Color("8")) // Gray
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(lipgloss.Color("40")).Bold(true)

	return t
}

// Styles for various TUI components
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("40")).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
)
