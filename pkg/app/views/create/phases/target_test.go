package phases

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/stretchr/testify/assert"
)

func TestTargetPhase_Name(t *testing.T) {
	p := NewTargetPhase()
	assert.Equal(t, "Select Target", p.Name())
}

func TestTargetPhase_FieldCount(t *testing.T) {
	p := NewTargetPhase()
	assert.Equal(t, len(Targets), p.FieldCount())
}

func TestTargetPhase_Init(t *testing.T) {
	p := NewTargetPhase()
	ctx := newTestContext()

	// Set an initial value to prove Init resets it
	ctx.Wizard.TargetSelected = 2

	p.Init(ctx)

	assert.Equal(t, 0, ctx.Wizard.TargetSelected)
}

func TestTargetPhase_Update_NavigateDown(t *testing.T) {
	p := NewTargetPhase()
	ctx := newTestContext()
	p.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyDown}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.TargetSelected)
}

func TestTargetPhase_Update_NavigateUp(t *testing.T) {
	p := NewTargetPhase()
	ctx := newTestContext()
	p.Init(ctx)
	ctx.Wizard.TargetSelected = 2

	msg := tea.KeyMsg{Type: tea.KeyUp}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.TargetSelected)
}

func TestTargetPhase_Update_NavigateUp_AtStart(t *testing.T) {
	p := NewTargetPhase()
	ctx := newTestContext()
	p.Init(ctx)

	// At the start, navigate up should stay at 0
	msg := tea.KeyMsg{Type: tea.KeyUp}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 0, ctx.Wizard.TargetSelected)
}

func TestTargetPhase_Update_NavigateDown_AtEnd(t *testing.T) {
	p := NewTargetPhase()
	ctx := newTestContext()
	p.Init(ctx)
	ctx.Wizard.TargetSelected = len(Targets) - 1

	// At the end, navigate down should stay at last index
	msg := tea.KeyMsg{Type: tea.KeyDown}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, len(Targets)-1, ctx.Wizard.TargetSelected)
}

func TestTargetPhase_Update_VimKeys(t *testing.T) {
	p := NewTargetPhase()
	ctx := newTestContext()
	p.Init(ctx)

	// Test j for down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	advance, _ := p.Update(ctx, msg)
	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.TargetSelected)

	// Test k for up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	advance, _ = p.Update(ctx, msg)
	assert.False(t, advance)
	assert.Equal(t, 0, ctx.Wizard.TargetSelected)
}

func TestTargetPhase_Update_Enter(t *testing.T) {
	p := NewTargetPhase()
	ctx := newTestContext()
	p.Init(ctx)
	ctx.Wizard.TargetSelected = 1 // Select Multipass

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	advance, _ := p.Update(ctx, msg)

	assert.True(t, advance)
}

func TestTargetPhase_View(t *testing.T) {
	p := NewTargetPhase()
	ctx := newTestContext()
	p.Init(ctx)

	view := p.View(ctx)

	// Check that the view contains expected elements
	assert.Contains(t, view, "Select Deployment Target")
	assert.Contains(t, view, "Terragrunt/libvirt")
	assert.Contains(t, view, "Multipass")
	assert.Contains(t, view, "Bootable USB")
	assert.Contains(t, view, "Generate Config")
}

func TestTargetPhase_View_SelectedItem(t *testing.T) {
	p := NewTargetPhase()
	ctx := newTestContext()
	p.Init(ctx)
	ctx.Wizard.TargetSelected = 1 // Select Multipass

	view := p.View(ctx)

	// The view should show the cursor on Multipass
	assert.Contains(t, view, "▸")
	assert.Contains(t, view, "Multipass")
}

func TestTargetPhase_Save(t *testing.T) {
	p := NewTargetPhase()
	ctx := newTestContext()
	p.Init(ctx)
	ctx.Wizard.TargetSelected = 0 // Terragrunt

	p.Save(ctx)

	assert.Equal(t, deploy.TargetTerragrunt, ctx.Wizard.Data.Target)
}

func TestTargetPhase_Save_AllTargets(t *testing.T) {
	tests := []struct {
		index    int
		expected deploy.DeploymentTarget
	}{
		{0, deploy.TargetTerragrunt},
		{1, deploy.TargetMultipass},
		{2, deploy.TargetUSB},
		{3, deploy.TargetConfigOnly},
	}

	for _, tt := range tests {
		t.Run(Targets[tt.index].Name, func(t *testing.T) {
			p := NewTargetPhase()
			ctx := newTestContext()
			ctx.Wizard.TargetSelected = tt.index

			p.Save(ctx)

			assert.Equal(t, tt.expected, ctx.Wizard.Data.Target)
		})
	}
}

func TestTargetPhase_ImplementsPhaseHandler(t *testing.T) {
	var _ wizard.PhaseHandler = (*TargetPhase)(nil)
}

func TestTargets_HasExpectedCount(t *testing.T) {
	assert.Equal(t, 4, len(Targets))
}

func TestTargets_HasCorrectIcons(t *testing.T) {
	assert.NotEmpty(t, Targets[0].Icon) // Terragrunt
	assert.NotEmpty(t, Targets[1].Icon) // Multipass
	assert.NotEmpty(t, Targets[2].Icon) // USB
	assert.NotEmpty(t, Targets[3].Icon) // Config only
}
