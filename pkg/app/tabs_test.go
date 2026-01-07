package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTabID(t *testing.T) {
	assert.Equal(t, TabID(0), TabVMs)
	assert.Equal(t, TabID(1), TabCreate)
	assert.Equal(t, TabID(2), TabDoctor)
	assert.Equal(t, TabID(3), TabConfig)
}

func TestNewBaseTab(t *testing.T) {
	tab := NewBaseTab(TabCreate, "Create", "1")

	assert.Equal(t, TabCreate, tab.ID())
	assert.Equal(t, "Create", tab.Name())
	assert.Equal(t, "1", tab.ShortKey())
	assert.Equal(t, 0, tab.Width())
	assert.Equal(t, 0, tab.Height())
	assert.False(t, tab.IsFocused())
}

func TestBaseTab_SetSize(t *testing.T) {
	tab := NewBaseTab(TabCreate, "Create", "1")

	tab.SetSize(100, 50)

	assert.Equal(t, 100, tab.Width())
	assert.Equal(t, 50, tab.Height())
}

func TestBaseTab_Focus(t *testing.T) {
	tab := NewBaseTab(TabCreate, "Create", "1")
	assert.False(t, tab.IsFocused())

	cmd := tab.Focus()

	assert.True(t, tab.IsFocused())
	assert.Nil(t, cmd)
}

func TestBaseTab_Blur(t *testing.T) {
	tab := NewBaseTab(TabCreate, "Create", "1")
	tab.Focus()
	assert.True(t, tab.IsFocused())

	tab.Blur()

	assert.False(t, tab.IsFocused())
}

func TestBaseTab_HasFocusedInput(t *testing.T) {
	tab := NewBaseTab(TabCreate, "Create", "1")

	// BaseTab should return false by default
	assert.False(t, tab.HasFocusedInput())
}
