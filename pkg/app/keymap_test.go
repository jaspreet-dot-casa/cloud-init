package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// Helper to create a tea.KeyMsg for testing
func keyMsg(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func ctrlKeyMsg(k string) tea.KeyMsg {
	switch k {
	case "c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
}

func specialKeyMsg(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	// Test that all bindings are defined
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
	assert.NotEmpty(t, km.NextTab.Keys())
	assert.NotEmpty(t, km.PrevTab.Keys())
	assert.NotEmpty(t, km.Tab1.Keys())
	assert.NotEmpty(t, km.Tab2.Keys())
	assert.NotEmpty(t, km.Tab3.Keys())
	assert.NotEmpty(t, km.Tab4.Keys())
	assert.NotEmpty(t, km.Up.Keys())
	assert.NotEmpty(t, km.Down.Keys())
	assert.NotEmpty(t, km.Enter.Keys())
	assert.NotEmpty(t, km.Back.Keys())
	assert.NotEmpty(t, km.Escape.Keys())
}

func TestKeys(t *testing.T) {
	km := Keys()

	// Should return the global key map
	assert.Equal(t, keys, km)
}

func TestKeyMap_QuitKeys(t *testing.T) {
	km := DefaultKeyMap()

	// 'q' should match Quit
	assert.True(t, key.Matches(keyMsg("q"), km.Quit))

	// 'ctrl+c' should match Quit
	assert.True(t, key.Matches(ctrlKeyMsg("c"), km.Quit))
}

func TestKeyMap_TabKeys(t *testing.T) {
	km := DefaultKeyMap()

	// Number keys should match tabs
	assert.True(t, key.Matches(keyMsg("1"), km.Tab1))
	assert.True(t, key.Matches(keyMsg("2"), km.Tab2))
	assert.True(t, key.Matches(keyMsg("3"), km.Tab3))
	assert.True(t, key.Matches(keyMsg("4"), km.Tab4))
}

func TestKeyMap_NavigationKeys(t *testing.T) {
	km := DefaultKeyMap()

	// Arrow keys
	assert.True(t, key.Matches(specialKeyMsg(tea.KeyUp), km.Up))
	assert.True(t, key.Matches(specialKeyMsg(tea.KeyDown), km.Down))

	// Vim keys
	assert.True(t, key.Matches(keyMsg("k"), km.Up))
	assert.True(t, key.Matches(keyMsg("j"), km.Down))
}

func TestDefaultVMListKeyMap(t *testing.T) {
	km := DefaultVMListKeyMap()

	assert.NotEmpty(t, km.Start.Keys())
	assert.NotEmpty(t, km.Stop.Keys())
	assert.NotEmpty(t, km.Delete.Keys())
	assert.NotEmpty(t, km.Console.Keys())
	assert.NotEmpty(t, km.SSH.Keys())
	assert.NotEmpty(t, km.Refresh.Keys())
	assert.NotEmpty(t, km.Details.Keys())
}

func TestVMListKeyMap_Keys(t *testing.T) {
	km := DefaultVMListKeyMap()

	assert.True(t, key.Matches(keyMsg("s"), km.Start))
	assert.True(t, key.Matches(keyMsg("S"), km.Stop))
	assert.True(t, key.Matches(keyMsg("d"), km.Delete))
	assert.True(t, key.Matches(keyMsg("c"), km.Console))
	assert.True(t, key.Matches(keyMsg("r"), km.Refresh))
}

func TestDefaultCreateKeyMap(t *testing.T) {
	km := DefaultCreateKeyMap()

	assert.NotEmpty(t, km.Next.Keys())
	assert.NotEmpty(t, km.Prev.Keys())
}

func TestKeyMap_HelpStrings(t *testing.T) {
	km := DefaultKeyMap()

	// Help strings should be set
	assert.Equal(t, "quit", km.Quit.Help().Desc)
	assert.Equal(t, "help", km.Help.Help().Desc)
}
