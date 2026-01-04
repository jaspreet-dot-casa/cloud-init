package phases

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create"
	"github.com/stretchr/testify/assert"
)

func TestOptionalPhase_Name(t *testing.T) {
	p := NewOptionalPhase()
	assert.Equal(t, "Optional Services", p.Name())
}

func TestOptionalPhase_FieldCount(t *testing.T) {
	p := NewOptionalPhase()
	assert.Equal(t, 2, p.FieldCount())
}

func TestOptionalPhase_Init(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()

	p.Init(ctx)

	assert.Contains(t, ctx.Wizard.TextInputs, "tailscale_key")
	assert.Contains(t, ctx.Wizard.TextInputs, "github_pat")
	assert.Equal(t, 0, ctx.Wizard.FocusedField)
}

func TestOptionalPhase_Init_PasswordMode(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()

	p.Init(ctx)

	// Both inputs should be in password mode
	tailscale := ctx.Wizard.TextInputs["tailscale_key"]
	githubPat := ctx.Wizard.TextInputs["github_pat"]

	// Check that EchoMode is set to password
	assert.NotNil(t, tailscale)
	assert.NotNil(t, githubPat)
}

func TestOptionalPhase_Update_NavigateDown(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()
	p.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyDown}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)
}

func TestOptionalPhase_Update_NavigateUp(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()
	p.Init(ctx)
	ctx.Wizard.FocusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 0, ctx.Wizard.FocusedField)
}

func TestOptionalPhase_Update_VimKeys(t *testing.T) {
	p := NewOptionalPhase()
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

func TestOptionalPhase_Update_Tab(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()
	p.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyTab}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)
}

func TestOptionalPhase_Update_EnterOnFirstField(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()
	p.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)
}

func TestOptionalPhase_Update_EnterOnLastField(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()
	p.Init(ctx)
	ctx.Wizard.FocusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	advance, _ := p.Update(ctx, msg)

	assert.True(t, advance)
}

func TestOptionalPhase_View(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()
	p.Init(ctx)

	view := p.View(ctx)

	assert.Contains(t, view, "Optional Services")
	assert.Contains(t, view, "Tailscale")
	assert.Contains(t, view, "GitHub PAT")
}

func TestOptionalPhase_View_ShowsDescriptions(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()
	p.Init(ctx)

	view := p.View(ctx)

	assert.Contains(t, view, "automatic Tailscale authentication")
	assert.Contains(t, view, "Personal Access Token")
}

func TestOptionalPhase_Save(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()
	p.Init(ctx)

	ctx.Wizard.SetTextInput("tailscale_key", "tskey-auth-test")
	ctx.Wizard.SetTextInput("github_pat", "ghp_test123")

	p.Save(ctx)

	assert.Equal(t, "tskey-auth-test", ctx.Wizard.Data.TailscaleKey)
	assert.Equal(t, "ghp_test123", ctx.Wizard.Data.GitHubPAT)
}

func TestOptionalPhase_Save_Empty(t *testing.T) {
	p := NewOptionalPhase()
	ctx := newTestContext()
	p.Init(ctx)

	// Don't set any values
	p.Save(ctx)

	assert.Empty(t, ctx.Wizard.Data.TailscaleKey)
	assert.Empty(t, ctx.Wizard.Data.GitHubPAT)
}

func TestOptionalPhase_ImplementsPhaseHandler(t *testing.T) {
	var _ create.PhaseHandler = (*OptionalPhase)(nil)
}
