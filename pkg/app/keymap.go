package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the application.
type KeyMap struct {
	// Navigation
	Quit    key.Binding
	Help    key.Binding
	NextTab key.Binding
	PrevTab key.Binding

	// Tab shortcuts
	Tab1 key.Binding
	Tab2 key.Binding
	Tab3 key.Binding
	Tab4 key.Binding
	Tab5 key.Binding

	// Common actions
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Back   key.Binding
	Escape key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("Shift+Tab", "prev tab"),
		),
		Tab1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "VMs"),
		),
		Tab2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "Create"),
		),
		Tab3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "ISO"),
		),
		Tab4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "Doctor"),
		),
		Tab5: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "Config"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace", "esc"),
			key.WithHelp("esc", "back"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// keys is the global key map instance.
var keys = DefaultKeyMap()

// Keys returns the global key map.
func Keys() KeyMap {
	return keys
}

// VMListKeyMap defines key bindings specific to the VM list view.
type VMListKeyMap struct {
	Start   key.Binding
	Stop    key.Binding
	Delete  key.Binding
	Console key.Binding
	SSH     key.Binding
	Refresh key.Binding
	Details key.Binding
}

// DefaultVMListKeyMap returns default VM list key bindings.
func DefaultVMListKeyMap() VMListKeyMap {
	return VMListKeyMap{
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start"),
		),
		Stop: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "stop"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Console: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "console"),
		),
		SSH: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "ssh"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Details: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "details"),
		),
	}
}

// CreateKeyMap defines key bindings specific to the create view.
type CreateKeyMap struct {
	Next key.Binding
	Prev key.Binding
}

// DefaultCreateKeyMap returns default create view key bindings.
func DefaultCreateKeyMap() CreateKeyMap {
	return CreateKeyMap{
		Next: key.NewBinding(
			key.WithKeys("enter", "tab"),
			key.WithHelp("enter", "next"),
		),
		Prev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "back"),
		),
	}
}
