package phases

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/stretchr/testify/assert"
)

func TestGitPhase_Name(t *testing.T) {
	p := NewGitPhase()
	assert.Equal(t, "Git Config", p.Name())
}

func TestGitPhase_FieldCount(t *testing.T) {
	p := NewGitPhase()
	assert.Equal(t, 2, p.FieldCount())
}

func TestGitPhase_Init(t *testing.T) {
	p := NewGitPhase()
	ctx := newTestContext()

	p.Init(ctx)

	assert.Contains(t, ctx.Wizard.TextInputs, "git_name")
	assert.Contains(t, ctx.Wizard.TextInputs, "git_email")
	assert.Equal(t, 0, ctx.Wizard.FocusedField)
}

func TestGitPhase_Init_WithPreFilledData(t *testing.T) {
	p := NewGitPhase()
	ctx := newTestContext()
	ctx.Wizard.Data.GitName = "Existing Name"
	ctx.Wizard.Data.GitEmail = "existing@example.com"

	p.Init(ctx)

	assert.Equal(t, "Existing Name", ctx.Wizard.TextInputs["git_name"].Value())
	assert.Equal(t, "existing@example.com", ctx.Wizard.TextInputs["git_email"].Value())
}

func TestGitPhase_Update_NavigateDown(t *testing.T) {
	p := NewGitPhase()
	ctx := newTestContext()
	p.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyDown}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)
}

func TestGitPhase_Update_NavigateUp(t *testing.T) {
	p := NewGitPhase()
	ctx := newTestContext()
	p.Init(ctx)
	ctx.Wizard.FocusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 0, ctx.Wizard.FocusedField)
}

func TestGitPhase_Update_VimKeys(t *testing.T) {
	p := NewGitPhase()
	ctx := newTestContext()
	p.Init(ctx)

	// Test j for down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	advance, _ := p.Update(ctx, msg)
	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)

	// Test k for up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	advance, _ = p.Update(ctx, msg)
	assert.False(t, advance)
	assert.Equal(t, 0, ctx.Wizard.FocusedField)
}

func TestGitPhase_Update_Tab(t *testing.T) {
	p := NewGitPhase()
	ctx := newTestContext()
	p.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyTab}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)
}

func TestGitPhase_Update_EnterOnFirstField(t *testing.T) {
	p := NewGitPhase()
	ctx := newTestContext()
	p.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)
}

func TestGitPhase_Update_EnterOnLastField(t *testing.T) {
	p := NewGitPhase()
	ctx := newTestContext()
	p.Init(ctx)
	ctx.Wizard.FocusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	advance, _ := p.Update(ctx, msg)

	assert.True(t, advance)
}

func TestGitPhase_View(t *testing.T) {
	p := NewGitPhase()
	ctx := newTestContext()
	p.Init(ctx)

	view := p.View(ctx)

	assert.Contains(t, view, "Git Configuration")
	assert.Contains(t, view, "Name")
	assert.Contains(t, view, "Email")
}

func TestGitPhase_Save(t *testing.T) {
	p := NewGitPhase()
	ctx := newTestContext()
	p.Init(ctx)

	ctx.Wizard.SetTextInput("git_name", "Test User")
	ctx.Wizard.SetTextInput("git_email", "test@example.com")

	p.Save(ctx)

	assert.Equal(t, "Test User", ctx.Wizard.Data.GitName)
	assert.Equal(t, "test@example.com", ctx.Wizard.Data.GitEmail)
}

func TestGitPhase_ImplementsPhaseHandler(t *testing.T) {
	var _ wizard.PhaseHandler = (*GitPhase)(nil)
}
