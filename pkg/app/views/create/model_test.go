package create

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/phases"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	m := New("/test/project", nil)

	assert.Equal(t, app.TabCreate, m.ID())
	assert.Equal(t, "Create", m.Name())
	assert.Equal(t, "2", m.ShortKey())
	assert.Equal(t, "/test/project", m.ProjectDir())
	assert.Equal(t, wizard.PhaseTarget, m.wizard.Phase)
}

func TestModel_Init(t *testing.T) {
	m := New("/test/project", nil)

	cmd := m.Init()

	// Init returns nil - packages are loaded lazily in Focus()
	assert.Nil(t, cmd)
}

func TestModel_KeyBindings(t *testing.T) {
	m := New("/test/project", nil)

	bindings := m.KeyBindings()

	assert.NotEmpty(t, bindings)
	assert.Contains(t, bindings, "[↑/↓] navigate")
}

func TestModel_SetSize(t *testing.T) {
	m := New("/test/project", nil)

	m.SetSize(100, 40)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 40, m.Height())
}

func TestModel_Focus(t *testing.T) {
	m := New("/test/project", nil)
	m.message = "old message"

	cmd := m.Focus()

	assert.True(t, m.IsFocused())
	assert.Empty(t, m.message)
	// First focus triggers package loading
	assert.NotNil(t, cmd)
	assert.True(t, m.loadingPackages)

	// Second focus should not re-load
	m.packagesLoaded = true
	m.loadingPackages = false
	cmd = m.Focus()
	assert.Nil(t, cmd)
}

func TestModel_Blur(t *testing.T) {
	m := New("/test/project", nil)
	m.Focus()

	m.Blur()

	assert.False(t, m.IsFocused())
}

func TestModel_Update_NavigateDown_TargetPhase(t *testing.T) {
	m := New("/test/project", nil)
	m.wizard.Phase = wizard.PhaseTarget
	m.wizard.TargetSelected = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, 1, model.wizard.TargetSelected)
}

func TestModel_Update_NavigateUp_TargetPhase(t *testing.T) {
	m := New("/test/project", nil)
	m.wizard.Phase = wizard.PhaseTarget
	m.wizard.TargetSelected = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, 0, model.wizard.TargetSelected)
}

func TestModel_Update_NavigateDown_AtEnd(t *testing.T) {
	m := New("/test/project", nil)
	m.wizard.Phase = wizard.PhaseTarget
	m.wizard.TargetSelected = len(phases.Targets) - 1

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	// Should not go past last item
	assert.Equal(t, len(phases.Targets)-1, model.wizard.TargetSelected)
}

func TestModel_Update_NavigateUp_AtStart(t *testing.T) {
	m := New("/test/project", nil)
	m.wizard.Phase = wizard.PhaseTarget
	m.wizard.TargetSelected = 0

	msg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	// Should not go before first item
	assert.Equal(t, 0, model.wizard.TargetSelected)
}

func TestModel_Update_VimKeys(t *testing.T) {
	m := New("/test/project", nil)
	m.wizard.Phase = wizard.PhaseTarget
	m.wizard.TargetSelected = 0

	// j for down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	updated, _ := m.Update(msg)
	model := updated.(*Model)
	assert.Equal(t, 1, model.wizard.TargetSelected)

	// k for up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	updated, _ = model.Update(msg)
	model = updated.(*Model)
	assert.Equal(t, 0, model.wizard.TargetSelected)
}

func TestModel_Update_Enter_AdvancesPhase(t *testing.T) {
	m := New("/test/project", nil)
	m.wizard.Phase = wizard.PhaseTarget
	m.wizard.TargetSelected = 0 // Terraform

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	// Should advance to target options phase
	assert.Equal(t, wizard.PhaseTargetOptions, model.wizard.Phase)
	assert.Equal(t, deploy.TargetTerraform, model.wizard.Data.Target)
}

func TestModel_View_ZeroWidth(t *testing.T) {
	m := New("/test/project", nil)

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

func TestModel_View_TargetPhase(t *testing.T) {
	m := New("/test/project", nil)
	m.SetSize(100, 40)
	m.wizard.Phase = wizard.PhaseTarget

	view := m.View()

	assert.Contains(t, view, "Select Deployment Target")
	assert.Contains(t, view, "Terraform")
	assert.Contains(t, view, "Multipass")
	assert.Contains(t, view, "Bootable USB")
	assert.Contains(t, view, "Generate Config")
}

func TestModel_View_WithMessage(t *testing.T) {
	m := New("/test/project", nil)
	m.SetSize(100, 40)
	m.message = "Test message"

	view := m.View()

	assert.Contains(t, view, "Test message")
}

func TestTargets(t *testing.T) {
	// Verify all targets are set up correctly
	assert.Equal(t, deploy.TargetTerraform, phases.Targets[0].Target)
	assert.Equal(t, "Terraform/libvirt", phases.Targets[0].Name)

	assert.Equal(t, deploy.TargetMultipass, phases.Targets[1].Target)
	assert.Equal(t, "Multipass", phases.Targets[1].Name)

	assert.Equal(t, deploy.TargetUSB, phases.Targets[2].Target)
	assert.Equal(t, "Bootable USB", phases.Targets[2].Name)

	assert.Equal(t, deploy.TargetConfigOnly, phases.Targets[3].Target)
	assert.Equal(t, "Generate Config", phases.Targets[3].Name)
}

func TestWizardState_NextPhase(t *testing.T) {
	w := wizard.NewState()

	assert.Equal(t, wizard.PhaseTarget, w.Phase)

	// NextPhase returns the next phase without modifying state
	assert.Equal(t, wizard.PhaseTargetOptions, w.NextPhase())
	assert.Equal(t, wizard.PhaseTarget, w.Phase) // State unchanged

	// Advance actually moves to next phase
	w.Advance()
	assert.Equal(t, wizard.PhaseTargetOptions, w.Phase)

	w.Advance()
	assert.Equal(t, wizard.PhaseSSH, w.Phase)
}

func TestWizardState_PrevPhase(t *testing.T) {
	w := wizard.NewState()
	w.Phase = wizard.PhaseSSH

	// PrevPhase returns the prev phase without modifying state
	assert.Equal(t, wizard.PhaseTargetOptions, w.PrevPhase())
	assert.Equal(t, wizard.PhaseSSH, w.Phase) // State unchanged

	// GoBack actually moves to previous phase
	w.GoBack()
	assert.Equal(t, wizard.PhaseTargetOptions, w.Phase)

	w.GoBack()
	assert.Equal(t, wizard.PhaseTarget, w.Phase)

	// Should not go before target (CanGoBack returns false)
	assert.False(t, w.CanGoBack())
}

func TestPhase_String(t *testing.T) {
	tests := []struct {
		phase    wizard.Phase
		expected string
	}{
		{wizard.PhaseTarget, "Select Target"},
		{wizard.PhaseTargetOptions, "Target Options"},
		{wizard.PhaseSSH, "SSH Keys"},
		{wizard.PhaseGit, "Git Config"},
		{wizard.PhaseHost, "Host Details"},
		{wizard.PhasePackages, "Packages"},
		{wizard.PhaseOptional, "Optional Services"},
		{wizard.PhaseReview, "Review"},
		{wizard.PhaseDeploy, "Deploying"},
		{wizard.PhaseComplete, "Complete"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.phase.String())
		})
	}
}
