package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// TabID identifies a tab type.
type TabID int

const (
	TabCreate TabID = iota
	TabISO
	TabDoctor
	TabConfig
)

// Tab is the interface that all tabs must implement.
type Tab interface {
	// ID returns the tab's unique identifier.
	ID() TabID

	// Name returns the display name of the tab.
	Name() string

	// ShortKey returns the keyboard shortcut (e.g., "1", "2").
	ShortKey() string

	// Init initializes the tab and returns an optional command.
	Init() tea.Cmd

	// Update handles messages and returns the updated tab and command.
	Update(msg tea.Msg) (Tab, tea.Cmd)

	// View returns the tab's view string.
	View() string

	// Focus is called when the tab becomes active.
	Focus() tea.Cmd

	// Blur is called when the tab becomes inactive.
	Blur()

	// SetSize sets the available width and height for the tab.
	SetSize(width, height int)

	// KeyBindings returns the context-sensitive key bindings for the footer.
	KeyBindings() []string

	// HasFocusedInput returns true if this tab has a focused text input.
	// When true, the app should not intercept alphanumeric keys for tab switching.
	HasFocusedInput() bool
}

// BaseTab provides common functionality for tabs.
type BaseTab struct {
	id       TabID
	name     string
	shortKey string
	width    int
	height   int
	focused  bool
}

// NewBaseTab creates a new base tab.
func NewBaseTab(id TabID, name, shortKey string) BaseTab {
	return BaseTab{
		id:       id,
		name:     name,
		shortKey: shortKey,
	}
}

// ID returns the tab's ID.
func (t BaseTab) ID() TabID {
	return t.id
}

// Name returns the tab's name.
func (t BaseTab) Name() string {
	return t.name
}

// ShortKey returns the tab's keyboard shortcut.
func (t BaseTab) ShortKey() string {
	return t.shortKey
}

// Width returns the available width.
func (t BaseTab) Width() int {
	return t.width
}

// Height returns the available height.
func (t BaseTab) Height() int {
	return t.height
}

// IsFocused returns true if the tab is focused.
func (t BaseTab) IsFocused() bool {
	return t.focused
}

// SetSize sets the available dimensions.
func (t *BaseTab) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// Focus marks the tab as focused.
func (t *BaseTab) Focus() tea.Cmd {
	t.focused = true
	return nil
}

// Blur marks the tab as not focused.
func (t *BaseTab) Blur() {
	t.focused = false
}

// HasFocusedInput returns false by default (no text input focused).
func (t BaseTab) HasFocusedInput() bool {
	return false
}
