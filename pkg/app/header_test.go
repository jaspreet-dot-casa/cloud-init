package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// mockTabForHeader is a simple Tab implementation for header tests.
type mockTabForHeader struct {
	BaseTab
}

func newMockTabForHeader(id TabID, name, shortKey string) *mockTabForHeader {
	return &mockTabForHeader{BaseTab: NewBaseTab(id, name, shortKey)}
}

func (t *mockTabForHeader) Init() tea.Cmd                     { return nil }
func (t *mockTabForHeader) Update(msg tea.Msg) (Tab, tea.Cmd) { return t, nil }
func (t *mockTabForHeader) View() string                      { return "" }
func (t *mockTabForHeader) Focus() tea.Cmd                    { t.BaseTab.Focus(); return nil }
func (t *mockTabForHeader) Blur()                             { t.BaseTab.Blur() }
func (t *mockTabForHeader) SetSize(width, height int)         { t.BaseTab.SetSize(width, height) }
func (t *mockTabForHeader) KeyBindings() []string             { return nil }
func (t *mockTabForHeader) HasFocusedInput() bool             { return false }

func TestRenderHeader(t *testing.T) {
	tabs := []Tab{
		newMockTabForHeader(TabCreate, "Create", "1"),
		newMockTabForHeader(TabISO, "ISO", "2"),
	}

	header := renderHeader(tabs, 0, 100)

	// Should contain title
	assert.Contains(t, header, "ucli")

	// Should contain tab names
	assert.Contains(t, header, "Create")
	assert.Contains(t, header, "ISO")

	// Should contain quit hint (could be "[q]uit" or "quit")
	assert.True(t, strings.Contains(header, "quit") || strings.Contains(header, "[q]uit"))
}

func TestRenderHeader_ActiveTab(t *testing.T) {
	tabs := []Tab{
		newMockTabForHeader(TabCreate, "Create", "1"),
		newMockTabForHeader(TabISO, "ISO", "2"),
	}

	// Active tab 0
	header := renderHeader(tabs, 0, 100)
	assert.Contains(t, header, "Create")

	// Active tab 1
	header = renderHeader(tabs, 1, 100)
	assert.Contains(t, header, "ISO")
}

func TestRenderHeader_NoTabs(t *testing.T) {
	header := renderHeader(nil, 0, 100)

	// Should still contain title
	assert.Contains(t, header, "ucli")
}

func TestRenderTabBar(t *testing.T) {
	tabs := []TabInfo{
		{Name: "VMs", ShortKey: "1", Active: true},
		{Name: "Create", ShortKey: "2", Active: false},
	}

	tabBar := RenderTabBar(tabs)

	assert.Contains(t, tabBar, "VMs")
	assert.Contains(t, tabBar, "Create")
	assert.Contains(t, tabBar, "[1]")
	assert.Contains(t, tabBar, "[2]")
}

func TestRenderTabBar_Empty(t *testing.T) {
	tabBar := RenderTabBar(nil)
	assert.Equal(t, "", tabBar)
}

func TestRenderTabBar_SingleTab(t *testing.T) {
	tabs := []TabInfo{
		{Name: "VMs", ShortKey: "1", Active: true},
	}

	tabBar := RenderTabBar(tabs)

	assert.Contains(t, tabBar, "VMs")
	assert.Contains(t, tabBar, "[1]")
}
