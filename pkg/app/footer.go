package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	footerHeight = 2
)

var (
	// Footer styles
	footerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	keyBindingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	bindingSeparator = keyBindingStyle.Render("  ")
)

// GlobalBindings returns the global key bindings shown in all tabs.
func GlobalBindings() []string {
	return []string{
		"[?] help",
		"[Tab] next",
		"[q] quit",
	}
}

// renderFooter renders the footer with key bindings.
func renderFooter(tabBindings []string, width int) string {
	// Combine tab-specific bindings with global bindings
	allBindings := make([]string, 0, len(tabBindings)+len(GlobalBindings()))

	// Tab-specific bindings first
	for _, b := range tabBindings {
		allBindings = append(allBindings, formatBinding(b))
	}

	// Add global bindings
	for _, b := range GlobalBindings() {
		allBindings = append(allBindings, formatBinding(b))
	}

	bindingsStr := strings.Join(allBindings, bindingSeparator)

	return footerStyle.Width(width).Render(bindingsStr)
}

// formatBinding formats a key binding string like "[k] action" with proper styling.
func formatBinding(binding string) string {
	// Parse "[key] action" format
	if len(binding) < 3 || binding[0] != '[' {
		return keyBindingStyle.Render(binding)
	}

	closeIdx := strings.Index(binding, "]")
	if closeIdx == -1 {
		return keyBindingStyle.Render(binding)
	}

	key := binding[0 : closeIdx+1]
	action := binding[closeIdx+1:]

	return keyStyle.Render(key) + keyBindingStyle.Render(action)
}

// RenderKeyBindings renders a list of key bindings.
func RenderKeyBindings(bindings []string) string {
	formatted := make([]string, len(bindings))
	for i, b := range bindings {
		formatted[i] = formatBinding(b)
	}
	return strings.Join(formatted, bindingSeparator)
}

// BindingHelp represents a single key binding for help display.
type BindingHelp struct {
	Key         string
	Description string
}

// FormatBindingHelp formats a binding for the footer.
func FormatBindingHelp(b BindingHelp) string {
	return formatBinding("[" + b.Key + "] " + b.Description)
}
