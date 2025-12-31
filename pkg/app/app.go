// Package app provides the full-screen TUI application for VM management.
// It follows the Bubble Tea architecture with tabs for different features.
package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the main application model.
type Model struct {
	tabs       []Tab
	activeTab  int
	width      int
	height     int
	quitting   bool
	err        error
	projectDir string
}

// New creates a new application model.
func New(projectDir string) Model {
	return Model{
		tabs:       []Tab{},
		activeTab:  0,
		projectDir: projectDir,
	}
}

// WithTabs sets the tabs for the application.
func (m Model) WithTabs(tabs ...Tab) Model {
	m.tabs = tabs
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Initialize all tabs
	for i := range m.tabs {
		if cmd := m.tabs[i].Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update all tabs with new size
		contentHeight := m.height - headerHeight - footerHeight
		for i := range m.tabs {
			m.tabs[i].SetSize(m.width, contentHeight)
		}

		return m, nil

	case error:
		m.err = msg
		return m, nil
	}

	// Forward to active tab
	if len(m.tabs) > 0 && m.activeTab < len(m.tabs) {
		var cmd tea.Cmd
		m.tabs[m.activeTab], cmd = m.tabs[m.activeTab].Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKeyMsg processes key events.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check if active tab has a focused text input
	hasFocusedInput := false
	if len(m.tabs) > 0 && m.activeTab < len(m.tabs) {
		hasFocusedInput = m.tabs[m.activeTab].HasFocusedInput()
	}

	// Always allow Ctrl+C to quit
	if msg.Type == tea.KeyCtrlC {
		m.quitting = true
		return m, tea.Quit
	}

	// When text input is focused, skip most global keybindings
	// to allow typing alphanumeric characters
	if !hasFocusedInput {
		// Global keybindings (only when no input is focused)
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Help):
			// Toggle help in active tab if it supports it
			if len(m.tabs) > 0 && m.activeTab < len(m.tabs) {
				if h, ok := m.tabs[m.activeTab].(interface{ ToggleHelp() }); ok {
					h.ToggleHelp()
				}
			}
			return m, nil

		case key.Matches(msg, keys.Tab1):
			return m.switchTab(0)
		case key.Matches(msg, keys.Tab2):
			return m.switchTab(1)
		case key.Matches(msg, keys.Tab3):
			return m.switchTab(2)
		case key.Matches(msg, keys.Tab4):
			return m.switchTab(3)

		case key.Matches(msg, keys.NextTab):
			return m.switchTab((m.activeTab + 1) % len(m.tabs))
		case key.Matches(msg, keys.PrevTab):
			idx := m.activeTab - 1
			if idx < 0 {
				idx = len(m.tabs) - 1
			}
			return m.switchTab(idx)
		}
	}

	// Forward to active tab
	if len(m.tabs) > 0 && m.activeTab < len(m.tabs) {
		var cmd tea.Cmd
		m.tabs[m.activeTab], cmd = m.tabs[m.activeTab].Update(msg)
		return m, cmd
	}

	return m, nil
}

// switchTab changes the active tab.
func (m Model) switchTab(idx int) (tea.Model, tea.Cmd) {
	if idx >= 0 && idx < len(m.tabs) {
		m.activeTab = idx
		// Focus the new tab
		if cmd := m.tabs[m.activeTab].Focus(); cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 {
		return "Loading..."
	}

	header := m.renderHeader()
	content := m.renderContent()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// renderHeader renders the application header with tabs.
func (m Model) renderHeader() string {
	return renderHeader(m.tabs, m.activeTab, m.width)
}

// renderContent renders the active tab's content.
func (m Model) renderContent() string {
	if len(m.tabs) == 0 || m.activeTab >= len(m.tabs) {
		return ""
	}

	contentHeight := m.height - headerHeight - footerHeight
	content := m.tabs[m.activeTab].View()

	return lipgloss.NewStyle().
		Height(contentHeight).
		Width(m.width).
		Render(content)
}

// renderFooter renders the footer with keybindings.
func (m Model) renderFooter() string {
	var tabBindings []string
	if len(m.tabs) > 0 && m.activeTab < len(m.tabs) {
		tabBindings = m.tabs[m.activeTab].KeyBindings()
	}
	return renderFooter(tabBindings, m.width)
}

// ActiveTab returns the currently active tab index.
func (m Model) ActiveTab() int {
	return m.activeTab
}

// SetActiveTab sets the active tab by index.
func (m *Model) SetActiveTab(idx int) {
	if idx >= 0 && idx < len(m.tabs) {
		m.activeTab = idx
	}
}

// Error returns the last error.
func (m Model) Error() error {
	return m.err
}

// ProjectDir returns the project directory.
func (m Model) ProjectDir() string {
	return m.projectDir
}
