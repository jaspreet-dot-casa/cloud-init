package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTabID(t *testing.T) {
	assert.Equal(t, TabID(0), TabVMs)
	assert.Equal(t, TabID(1), TabCreate)
	assert.Equal(t, TabID(2), TabISO)
	assert.Equal(t, TabID(3), TabConfig)
}

func TestNewBaseTab(t *testing.T) {
	tab := NewBaseTab(TabVMs, "VMs", "1")

	assert.Equal(t, TabVMs, tab.ID())
	assert.Equal(t, "VMs", tab.Name())
	assert.Equal(t, "1", tab.ShortKey())
	assert.Equal(t, 0, tab.Width())
	assert.Equal(t, 0, tab.Height())
	assert.False(t, tab.IsFocused())
}

func TestBaseTab_SetSize(t *testing.T) {
	tab := NewBaseTab(TabVMs, "VMs", "1")

	tab.SetSize(100, 50)

	assert.Equal(t, 100, tab.Width())
	assert.Equal(t, 50, tab.Height())
}

func TestBaseTab_Focus(t *testing.T) {
	tab := NewBaseTab(TabVMs, "VMs", "1")
	assert.False(t, tab.IsFocused())

	cmd := tab.Focus()

	assert.True(t, tab.IsFocused())
	assert.Nil(t, cmd)
}

func TestBaseTab_Blur(t *testing.T) {
	tab := NewBaseTab(TabVMs, "VMs", "1")
	tab.Focus()
	assert.True(t, tab.IsFocused())

	tab.Blur()

	assert.False(t, tab.IsFocused())
}

func TestNewPlaceholderTab(t *testing.T) {
	tab := NewPlaceholderTab(TabCreate, "Create", "2", "Create a new VM")

	assert.Equal(t, TabCreate, tab.ID())
	assert.Equal(t, "Create", tab.Name())
	assert.Equal(t, "2", tab.ShortKey())
	assert.Equal(t, "Create a new VM", tab.View())
}

func TestPlaceholderTab_Init(t *testing.T) {
	tab := NewPlaceholderTab(TabVMs, "VMs", "1", "content")

	cmd := tab.Init()

	assert.Nil(t, cmd)
}

func TestPlaceholderTab_Update(t *testing.T) {
	tab := NewPlaceholderTab(TabVMs, "VMs", "1", "content")

	updated, cmd := tab.Update(nil)

	assert.Equal(t, tab, updated)
	assert.Nil(t, cmd)
}

func TestPlaceholderTab_Focus(t *testing.T) {
	tab := NewPlaceholderTab(TabVMs, "VMs", "1", "content")

	cmd := tab.Focus()

	assert.True(t, tab.IsFocused())
	assert.Nil(t, cmd)
}

func TestPlaceholderTab_Blur(t *testing.T) {
	tab := NewPlaceholderTab(TabVMs, "VMs", "1", "content")
	tab.Focus()

	tab.Blur()

	assert.False(t, tab.IsFocused())
}

func TestPlaceholderTab_SetSize(t *testing.T) {
	tab := NewPlaceholderTab(TabVMs, "VMs", "1", "content")

	tab.SetSize(80, 24)

	assert.Equal(t, 80, tab.Width())
	assert.Equal(t, 24, tab.Height())
}

func TestPlaceholderTab_KeyBindings(t *testing.T) {
	tab := NewPlaceholderTab(TabVMs, "VMs", "1", "content")

	bindings := tab.KeyBindings()

	assert.Nil(t, bindings)
}
