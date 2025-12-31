package create

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	m := New("/test/project")

	assert.Equal(t, app.TabCreate, m.ID())
	assert.Equal(t, "Create", m.Name())
	assert.Equal(t, "2", m.ShortKey())
	assert.Equal(t, "/test/project", m.ProjectDir())
	assert.False(t, m.launching)
	assert.Equal(t, 0, m.selected)
	assert.Len(t, m.targets, 3) // Terraform, Multipass, USB
}

func TestModel_Init(t *testing.T) {
	m := New("/test/project")

	cmd := m.Init()

	assert.Nil(t, cmd)
}

func TestModel_KeyBindings(t *testing.T) {
	m := New("/test/project")

	bindings := m.KeyBindings()

	assert.NotEmpty(t, bindings)
	assert.Contains(t, bindings, "[↑/↓] navigate")
	assert.Contains(t, bindings, "[Enter] create")
}

func TestModel_SetSize(t *testing.T) {
	m := New("/test/project")

	m.SetSize(100, 40)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 40, m.Height())
}

func TestModel_Focus(t *testing.T) {
	m := New("/test/project")
	m.message = "old message"

	cmd := m.Focus()

	assert.True(t, m.IsFocused())
	assert.Empty(t, m.message)
	assert.Nil(t, cmd)
}

func TestModel_Blur(t *testing.T) {
	m := New("/test/project")
	m.Focus()

	m.Blur()

	assert.False(t, m.IsFocused())
}

func TestModel_Update_NavigateDown(t *testing.T) {
	m := New("/test/project")
	m.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, 1, model.selected)
}

func TestModel_Update_NavigateUp(t *testing.T) {
	m := New("/test/project")
	m.selected = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, 0, model.selected)
}

func TestModel_Update_NavigateDown_AtEnd(t *testing.T) {
	m := New("/test/project")
	m.selected = len(m.targets) - 1

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	// Should not go past last item
	assert.Equal(t, len(m.targets)-1, model.selected)
}

func TestModel_Update_NavigateUp_AtStart(t *testing.T) {
	m := New("/test/project")
	m.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	// Should not go before first item
	assert.Equal(t, 0, model.selected)
}

func TestModel_Update_VimKeys(t *testing.T) {
	m := New("/test/project")
	m.selected = 0

	// j for down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	updated, _ := m.Update(msg)
	model := updated.(*Model)
	assert.Equal(t, 1, model.selected)

	// k for up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	updated, _ = model.Update(msg)
	model = updated.(*Model)
	assert.Equal(t, 0, model.selected)
}

func TestModel_Update_Enter_LaunchesCreate(t *testing.T) {
	m := New("/test/project")
	m.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.Update(msg)
	model := updated.(*Model)

	assert.True(t, model.launching)
	assert.Empty(t, model.message)
	assert.NotNil(t, cmd)
}

func TestModel_Update_SkipsWhenLaunching(t *testing.T) {
	m := New("/test/project")
	m.launching = true

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, cmd := m.Update(msg)

	assert.Equal(t, m, updated)
	assert.Nil(t, cmd)
}

func TestModel_Update_CreateCompleteMsg_Success(t *testing.T) {
	m := New("/test/project")
	m.launching = true

	msg := CreateCompleteMsg{Success: true}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.False(t, model.launching)
	assert.Contains(t, model.message, "successfully")
}

func TestModel_Update_CreateCompleteMsg_Error(t *testing.T) {
	m := New("/test/project")
	m.launching = true

	msg := CreateCompleteMsg{Success: false, Error: assert.AnError}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.False(t, model.launching)
	assert.Contains(t, model.message, "Error")
}

func TestModel_SelectedTarget(t *testing.T) {
	m := New("/test/project")

	// Default selection (Terraform)
	m.selected = 0
	assert.Equal(t, deploy.TargetTerraform, m.SelectedTarget())

	// Multipass
	m.selected = 1
	assert.Equal(t, deploy.TargetMultipass, m.SelectedTarget())

	// USB
	m.selected = 2
	assert.Equal(t, deploy.TargetUSB, m.SelectedTarget())
}

func TestModel_SelectedTarget_OutOfBounds(t *testing.T) {
	m := New("/test/project")
	m.selected = -1

	// Should return default (Terraform)
	assert.Equal(t, deploy.TargetTerraform, m.SelectedTarget())
}

func TestModel_View(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Create New VM")
	assert.Contains(t, view, "Terraform")
	assert.Contains(t, view, "Multipass")
	assert.Contains(t, view, "Bootable USB")
}

func TestModel_View_ZeroWidth(t *testing.T) {
	m := New("/test/project")

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

func TestModel_View_WithMessage(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 40)
	m.message = "Test message"

	view := m.View()

	assert.Contains(t, view, "Test message")
}

func TestModel_View_Launching(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 40)
	m.launching = true

	view := m.View()

	assert.Contains(t, view, "Launching")
}

func TestModel_handleKeyMsg(t *testing.T) {
	m := New("/test/project")

	// Test that handleKeyMsg is called correctly
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.Equal(t, 1, model.selected)
}

func TestRunCreateMsg(t *testing.T) {
	msg := app.RunCreateMsg{Target: deploy.TargetTerraform, ProjectDir: "/test"}
	assert.Equal(t, deploy.TargetTerraform, msg.Target)
	assert.Equal(t, "/test", msg.ProjectDir)
}

func TestCreateCompleteMsg(t *testing.T) {
	msg := CreateCompleteMsg{Success: true, Error: nil}
	assert.True(t, msg.Success)
	assert.Nil(t, msg.Error)
}

func TestModel_launchCreate(t *testing.T) {
	m := New("/test/project")
	m.selected = 1 // Multipass

	cmd := m.launchCreate()

	assert.NotNil(t, cmd)

	// Execute the command
	msg := cmd()
	launchMsg, ok := msg.(app.RunCreateMsg)
	assert.True(t, ok)
	assert.Equal(t, deploy.TargetMultipass, launchMsg.Target)
	assert.Equal(t, "/test/project", launchMsg.ProjectDir)
}

func TestModel_targets(t *testing.T) {
	m := New("/test/project")

	// Verify all targets are set up correctly
	assert.Equal(t, deploy.TargetTerraform, m.targets[0].target)
	assert.Equal(t, "Terraform/libvirt", m.targets[0].name)

	assert.Equal(t, deploy.TargetMultipass, m.targets[1].target)
	assert.Equal(t, "Multipass", m.targets[1].name)

	assert.Equal(t, deploy.TargetUSB, m.targets[2].target)
	assert.Equal(t, "Bootable USB", m.targets[2].name)
}

func TestModel_handleKeyMsg_KeyBinding(t *testing.T) {
	m := New("/test/project")

	tests := []struct {
		name     string
		key      key.Binding
		startSel int
		endSel   int
	}{
		{"up from 1", key.NewBinding(key.WithKeys("up")), 1, 0},
		{"down from 0", key.NewBinding(key.WithKeys("down")), 0, 1},
		{"k from 1", key.NewBinding(key.WithKeys("k")), 1, 0},
		{"j from 0", key.NewBinding(key.WithKeys("j")), 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.selected = tt.startSel
			keys := tt.key.Keys()
			var msg tea.KeyMsg
			if keys[0] == "up" {
				msg = tea.KeyMsg{Type: tea.KeyUp}
			} else if keys[0] == "down" {
				msg = tea.KeyMsg{Type: tea.KeyDown}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(keys[0])}
			}
			updated, _ := m.handleKeyMsg(msg)
			model := updated.(*Model)
			assert.Equal(t, tt.endSel, model.selected)
		})
	}
}
