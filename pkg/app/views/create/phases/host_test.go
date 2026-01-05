package phases

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/stretchr/testify/assert"
)

func newTestContext() *wizard.PhaseContext {
	return &wizard.PhaseContext{
		Wizard: wizard.NewState(),
	}
}

func TestHostPhase_Name(t *testing.T) {
	phase := NewHostPhase()
	assert.Equal(t, "Host Details", phase.Name())
}

func TestHostPhase_FieldCount(t *testing.T) {
	phase := NewHostPhase()
	assert.Equal(t, 3, phase.FieldCount())
}

func TestHostPhase_Init(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()

	phase.Init(ctx)

	// Check text inputs were created
	assert.Contains(t, ctx.Wizard.TextInputs, "display_name")
	assert.Contains(t, ctx.Wizard.TextInputs, "username")
	assert.Contains(t, ctx.Wizard.TextInputs, "hostname")

	// Check defaults
	assert.NotEmpty(t, ctx.Wizard.TextInputs["username"].Value())
	assert.Equal(t, "ubuntu-server", ctx.Wizard.TextInputs["hostname"].Value())

	// Check focus is on first field
	assert.Equal(t, 0, ctx.Wizard.FocusedField)
}

func TestHostPhase_Init_WithGitName(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	ctx.Wizard.Data.GitName = "Test User"

	phase.Init(ctx)

	// Display name should be pre-filled from git name
	assert.Equal(t, "Test User", ctx.Wizard.TextInputs["display_name"].Value())
}

func TestHostPhase_Update_NavigateDown(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	phase.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyDown}
	advance, _ := phase.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)
}

func TestHostPhase_Update_NavigateUp(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	phase.Init(ctx)
	ctx.Wizard.FocusedField = 2

	msg := tea.KeyMsg{Type: tea.KeyUp}
	advance, _ := phase.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)
}

func TestHostPhase_Update_VimKeys(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	phase.Init(ctx)

	// Test 'j' for down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	advance, _ := phase.Update(ctx, msg)
	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)

	// Test 'k' for up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	advance, _ = phase.Update(ctx, msg)
	assert.False(t, advance)
	assert.Equal(t, 0, ctx.Wizard.FocusedField)
}

func TestHostPhase_Update_Tab(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	phase.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyTab}
	advance, _ := phase.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)
}

func TestHostPhase_Update_EnterOnMiddleField(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	phase.Init(ctx)
	ctx.Wizard.FocusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	advance, _ := phase.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 2, ctx.Wizard.FocusedField) // Should move to next field
}

func TestHostPhase_Update_EnterOnLastField(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	phase.Init(ctx)
	ctx.Wizard.FocusedField = 2 // Last field

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	advance, _ := phase.Update(ctx, msg)

	assert.True(t, advance) // Should advance to next phase
}

func TestHostPhase_View(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	phase.Init(ctx)

	view := phase.View(ctx)

	assert.Contains(t, view, "Host Details")
	assert.Contains(t, view, "Display Name")
	assert.Contains(t, view, "Username")
	assert.Contains(t, view, "Hostname")
}

func TestHostPhase_View_ShowsCursor(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	phase.Init(ctx)
	ctx.Wizard.FocusedField = 0

	view := phase.View(ctx)

	// Should show cursor on first field
	assert.True(t, strings.Contains(view, "▸"))
}

func TestHostPhase_Save(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	phase.Init(ctx)

	// Set values using wizard helper
	ctx.Wizard.SetTextInput("display_name", "Test User")
	ctx.Wizard.SetTextInput("username", "testuser")
	ctx.Wizard.SetTextInput("hostname", "test-host")

	phase.Save(ctx)

	assert.Equal(t, "Test User", ctx.Wizard.Data.DisplayName)
	assert.Equal(t, "testuser", ctx.Wizard.Data.Username)
	assert.Equal(t, "test-host", ctx.Wizard.Data.Hostname)
}

func TestHostPhase_Save_AppliesDefaults(t *testing.T) {
	phase := NewHostPhase()
	ctx := newTestContext()
	phase.Init(ctx)

	// Clear values to test defaults
	ctx.Wizard.SetTextInput("username", "")
	ctx.Wizard.SetTextInput("hostname", "")

	phase.Save(ctx)

	assert.Equal(t, "ubuntu", ctx.Wizard.Data.Username)
	assert.Equal(t, "ubuntu-server", ctx.Wizard.Data.Hostname)
}

func TestHostPhase_ImplementsPhaseHandler(t *testing.T) {
	// This test verifies that HostPhase implements the PhaseHandler interface
	var _ wizard.PhaseHandler = (*HostPhase)(nil)
}

// Helper to create a key message
func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(s),
	}
}

func specialKeyMsg(k tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: k}
}
