package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// mockTab is a simple Tab implementation for testing.
type mockTab struct {
	BaseTab
	content string
}

func newMockTab(id TabID, name, shortKey, content string) *mockTab {
	return &mockTab{
		BaseTab: NewBaseTab(id, name, shortKey),
		content: content,
	}
}

func (t *mockTab) Init() tea.Cmd                        { return nil }
func (t *mockTab) Update(msg tea.Msg) (Tab, tea.Cmd)    { return t, nil }
func (t *mockTab) View() string                         { return t.content }
func (t *mockTab) Focus() tea.Cmd                       { t.BaseTab.Focus(); return nil }
func (t *mockTab) Blur()                                { t.BaseTab.Blur() }
func (t *mockTab) SetSize(width, height int)            { t.BaseTab.SetSize(width, height) }
func (t *mockTab) KeyBindings() []string                { return nil }
func (t *mockTab) HasFocusedInput() bool                { return false }

func TestNew(t *testing.T) {
	m := New("/test/project")

	assert.Equal(t, "/test/project", m.projectDir)
	assert.Equal(t, 0, m.activeTab)
	assert.Empty(t, m.tabs)
	assert.False(t, m.quitting)
	assert.NoError(t, m.err)
}

func TestModel_WithTabs(t *testing.T) {
	tab1 := newMockTab(TabVMs, "VMs", "1", "VMs content")
	tab2 := newMockTab(TabCreate, "Create", "2", "Create content")

	m := New("/test").WithTabs(tab1, tab2)

	assert.Len(t, m.tabs, 2)
	assert.Equal(t, "VMs", m.tabs[0].Name())
	assert.Equal(t, "Create", m.tabs[1].Name())
}

func TestModel_Init(t *testing.T) {
	tab1 := newMockTab(TabCreate, "Create", "1", "content")
	m := New("/test").WithTabs(tab1)

	cmd := m.Init()

	// Mock tabs return nil commands
	assert.Nil(t, cmd)
}

func TestModel_Update_WindowSizeMsg(t *testing.T) {
	tab := newMockTab(TabCreate, "Create", "1", "content")
	m := New("/test").WithTabs(tab)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, 100, model.width)
	assert.Equal(t, 50, model.height)
	assert.Nil(t, cmd)
}

func TestModel_Update_QuitKey(t *testing.T) {
	m := New("/test")

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.True(t, model.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_Update_TabSwitching(t *testing.T) {
	tab1 := newMockTab(TabVMs, "VMs", "1", "VMs content")
	tab2 := newMockTab(TabCreate, "Create", "2", "Create content")
	tab3 := newMockTab(TabDoctor, "Doctor", "3", "Doctor content")

	m := New("/test").WithTabs(tab1, tab2, tab3)
	assert.Equal(t, 0, m.activeTab)

	// Switch to tab 2
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	assert.Equal(t, 1, model.activeTab)

	// Switch to tab 3
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
	updated, _ = model.Update(msg)
	model = updated.(Model)
	assert.Equal(t, 2, model.activeTab)

	// Switch back to tab 1
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
	updated, _ = model.Update(msg)
	model = updated.(Model)
	assert.Equal(t, 0, model.activeTab)
}

func TestModel_Update_TabKey(t *testing.T) {
	tab1 := newMockTab(TabVMs, "VMs", "1", "content")
	tab2 := newMockTab(TabCreate, "Create", "2", "content")

	m := New("/test").WithTabs(tab1, tab2)
	assert.Equal(t, 0, m.activeTab)

	// Tab key should cycle to next tab
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	assert.Equal(t, 1, model.activeTab)

	// Tab again should wrap to first
	updated, _ = model.Update(msg)
	model = updated.(Model)
	assert.Equal(t, 0, model.activeTab)
}

func TestModel_View_Loading(t *testing.T) {
	m := New("/test")
	m.width = 0 // Not sized yet

	view := m.View()
	assert.Equal(t, "Loading...", view)
}

func TestModel_View_Quitting(t *testing.T) {
	m := New("/test")
	m.quitting = true

	view := m.View()
	assert.Equal(t, "", view)
}

func TestModel_ActiveTab(t *testing.T) {
	tab1 := newMockTab(TabVMs, "VMs", "1", "content")
	tab2 := newMockTab(TabCreate, "Create", "2", "content")

	m := New("/test").WithTabs(tab1, tab2)

	assert.Equal(t, 0, m.ActiveTab())

	m.SetActiveTab(1)
	assert.Equal(t, 1, m.ActiveTab())

	// Out of bounds should not change
	m.SetActiveTab(10)
	assert.Equal(t, 1, m.ActiveTab())

	m.SetActiveTab(-1)
	assert.Equal(t, 1, m.ActiveTab())
}

func TestModel_ProjectDir(t *testing.T) {
	m := New("/my/project")
	assert.Equal(t, "/my/project", m.ProjectDir())
}

func TestModel_Error(t *testing.T) {
	m := New("/test")
	assert.NoError(t, m.Error())

	// Simulate error message
	updated, _ := m.Update(assert.AnError)
	model := updated.(Model)
	assert.Error(t, model.Error())
}

func TestModel_SwitchTab_BoundsCheck(t *testing.T) {
	tab1 := newMockTab(TabCreate, "Create", "1", "content")
	m := New("/test").WithTabs(tab1)

	// Try to switch to non-existent tab
	updated, cmd := m.switchTab(5)
	model := updated.(Model)

	assert.Equal(t, 0, model.activeTab) // Should stay at 0
	assert.Nil(t, cmd)

	// Negative index
	updated, cmd = m.switchTab(-1)
	model = updated.(Model)
	assert.Equal(t, 0, model.activeTab)
	assert.Nil(t, cmd)
}

func TestModel_NoTabs(t *testing.T) {
	m := New("/test")
	m.width = 100
	m.height = 50

	// Should not panic with no tabs
	view := m.View()
	assert.Contains(t, view, "ucli")

	// Update should not panic
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	_, cmd := m.Update(msg)
	assert.Nil(t, cmd)
}
