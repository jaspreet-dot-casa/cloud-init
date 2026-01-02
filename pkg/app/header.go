package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	headerHeight = 3
)

var (
	// Header styles
	headerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Padding(0, 1)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Background(lipgloss.Color("236")).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Padding(0, 2)

	tabSeparator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(" | ")

	shortKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Faint(true)
)

// renderHeader renders the application header with title and tabs.
func renderHeader(tabs []Tab, activeIdx, width int) string {
	// Title
	title := titleStyle.Render("ucli - Cloud-Init VM Manager")

	// Build tab bar
	var tabParts []string
	for i, tab := range tabs {
		shortKey := shortKeyStyle.Render("[" + tab.ShortKey() + "] ")
		name := tab.Name()

		if i == activeIdx {
			tabParts = append(tabParts, activeTabStyle.Render(shortKey+name))
		} else {
			tabParts = append(tabParts, inactiveTabStyle.Render(shortKey+name))
		}
	}

	tabBar := strings.Join(tabParts, tabSeparator)

	// Right-align quit hint
	quitHint := shortKeyStyle.Render("[q]uit")

	// Calculate spacing
	titleWidth := lipgloss.Width(title)
	tabBarWidth := lipgloss.Width(tabBar)
	quitWidth := lipgloss.Width(quitHint)
	spacing := width - titleWidth - tabBarWidth - quitWidth - 4 // padding

	if spacing < 1 {
		spacing = 1
	}

	// Build header line
	headerLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		title,
		strings.Repeat(" ", 2),
		tabBar,
		strings.Repeat(" ", spacing),
		quitHint,
	)

	return headerStyle.Width(width).Render(headerLine)
}

// TabInfo contains information about a tab for display.
type TabInfo struct {
	Name     string
	ShortKey string
	Active   bool
}

// RenderTabBar renders just the tab bar portion.
func RenderTabBar(tabs []TabInfo) string {
	var parts []string
	for _, tab := range tabs {
		shortKey := shortKeyStyle.Render("[" + tab.ShortKey + "] ")
		if tab.Active {
			parts = append(parts, activeTabStyle.Render(shortKey+tab.Name))
		} else {
			parts = append(parts, inactiveTabStyle.Render(shortKey+tab.Name))
		}
	}
	return strings.Join(parts, tabSeparator)
}
