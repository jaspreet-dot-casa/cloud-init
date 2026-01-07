package iso

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	isobuilder "github.com/jaspreet-dot-casa/cloud-init/pkg/iso"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	m := New("/test/project")

	assert.Equal(t, app.TabISO, m.ID())
	assert.Equal(t, "ISO", m.Name())
	assert.Equal(t, "2", m.ShortKey())
	assert.Equal(t, "/test/project", m.projectDir)
	assert.False(t, m.building)
	assert.Equal(t, fieldSourceISO, m.focusedField)
}

func TestModel_Init(t *testing.T) {
	m := New("/test/project")

	cmd := m.Init()

	assert.NotNil(t, cmd) // Returns textinput.Blink
}

func TestModel_KeyBindings(t *testing.T) {
	m := New("/test/project")

	bindings := m.KeyBindings()

	assert.NotEmpty(t, bindings)
	assert.Contains(t, bindings, "[Tab] next")
	assert.Contains(t, bindings, "[←/→] select")
	assert.Contains(t, bindings, "[Enter] build")
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
	m.focusedField = fieldStorage

	cmd := m.Focus()

	assert.True(t, m.IsFocused())
	assert.Empty(t, m.message)
	assert.Equal(t, fieldSourceISO, m.focusedField)
	assert.NotNil(t, cmd) // Returns blink command
}

func TestModel_Blur(t *testing.T) {
	m := New("/test/project")
	m.Focus()

	m.Blur()

	assert.False(t, m.IsFocused())
}

func TestModel_Update_Tab_NextField(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldSourceISO

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, fieldOutputPath, model.focusedField)
}

func TestModel_Update_ShiftTab_PrevField(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldOutputPath

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, fieldSourceISO, model.focusedField)
}

func TestModel_Update_Down_NextField(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldSourceISO

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, fieldOutputPath, model.focusedField)
}

func TestModel_Update_Up_PrevField(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldOutputPath

	msg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, fieldSourceISO, model.focusedField)
}

func TestModel_Update_Left_VersionSelect(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldVersion
	m.versionIdx = 1

	msg := tea.KeyMsg{Type: tea.KeyLeft}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, 0, model.versionIdx)
}

func TestModel_Update_Right_VersionSelect(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldVersion
	m.versionIdx = 0

	msg := tea.KeyMsg{Type: tea.KeyRight}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, 1, model.versionIdx)
}

func TestModel_Update_Left_StorageSelect(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldStorage
	m.storageIdx = 1

	msg := tea.KeyMsg{Type: tea.KeyLeft}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, 0, model.storageIdx)
}

func TestModel_Update_Right_StorageSelect(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldStorage
	m.storageIdx = 0

	msg := tea.KeyMsg{Type: tea.KeyRight}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, 1, model.storageIdx)
}

func TestModel_Update_Enter_OnSubmit_ValidationError(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldSubmit
	// Source ISO is empty, should fail validation

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.Update(msg)
	model := updated.(*Model)

	assert.Contains(t, model.message, "Validation error")
	assert.Nil(t, cmd) // Validation failed, no command
	assert.False(t, model.building)
}

func TestModel_Update_Enter_NotOnSubmit(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldSourceISO

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	// Should move to next field
	assert.Equal(t, fieldOutputPath, model.focusedField)
}

func TestModel_Update_SkipsWhenBuilding(t *testing.T) {
	m := New("/test/project")
	m.building = true

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, cmd := m.Update(msg)

	assert.Equal(t, m, updated)
	assert.Nil(t, cmd)
}

func TestModel_Update_BuildCompleteMsg_Success(t *testing.T) {
	m := New("/test/project")
	m.building = true

	msg := BuildCompleteMsg{Success: true, OutputPath: "/path/to/output.iso"}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.False(t, model.building)
	assert.Contains(t, model.message, "/path/to/output.iso")
}

func TestModel_Update_BuildCompleteMsg_Error(t *testing.T) {
	m := New("/test/project")
	m.building = true

	msg := BuildCompleteMsg{Success: false, Error: assert.AnError}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.False(t, model.building)
	assert.Contains(t, model.message, "Error")
}

func TestModel_View(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "ISO Builder")
	assert.Contains(t, view, "Source ISO")
	assert.Contains(t, view, "Output path")
	assert.Contains(t, view, "Ubuntu version")
	assert.Contains(t, view, "Storage layout")
	assert.Contains(t, view, "Build ISO")
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

func TestModel_View_Building(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 40)
	m.building = true

	view := m.View()

	// Help text should not appear when building
	assert.NotContains(t, view, "Tab: next field")
}

func TestModel_GetOptions(t *testing.T) {
	m := New("/test/project")
	m.sourceInput.SetValue("/path/to/source.iso")
	m.outputInput.SetValue("/path/to/output.iso")
	m.versionIdx = 0 // 24.04
	m.storageIdx = 1 // direct
	m.timezoneInput.SetValue("America/New_York")

	opts := m.GetOptions()

	assert.Equal(t, "/path/to/source.iso", opts.SourceISO)
	assert.Equal(t, "/path/to/output.iso", opts.OutputPath)
	assert.Equal(t, "24.04", opts.UbuntuVersion)
	assert.Equal(t, isobuilder.StorageDirect, opts.StorageLayout)
	assert.Equal(t, "America/New_York", opts.Timezone)
}

func TestModel_GetOptions_DefaultTimezone(t *testing.T) {
	m := New("/test/project")
	// Leave timezone empty

	opts := m.GetOptions()

	assert.Equal(t, "UTC", opts.Timezone)
}

func TestModel_nextField(t *testing.T) {
	m := New("/test/project")

	fields := []field{
		fieldSourceISO,
		fieldOutputPath,
		fieldVersion,
		fieldStorage,
		fieldTimezone,
		fieldSubmit,
	}

	for i, expected := range fields[1:] {
		m.focusedField = fields[i]
		m.nextField()
		assert.Equal(t, expected, m.focusedField)
	}

	// Wrap around
	m.focusedField = fieldSubmit
	m.nextField()
	assert.Equal(t, fieldSourceISO, m.focusedField)
}

func TestModel_prevField(t *testing.T) {
	m := New("/test/project")

	// From first to last (wrap around)
	m.focusedField = fieldSourceISO
	m.prevField()
	assert.Equal(t, fieldSubmit, m.focusedField)

	// From second to first
	m.focusedField = fieldOutputPath
	m.prevField()
	assert.Equal(t, fieldSourceISO, m.focusedField)
}

func TestModel_Update_VimKeys(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldVersion
	m.versionIdx = 0

	// l for right
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
	updated, _ := m.Update(msg)
	model := updated.(*Model)
	assert.Equal(t, 1, model.versionIdx)

	// h for left
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}
	updated, _ = model.Update(msg)
	model = updated.(*Model)
	assert.Equal(t, 0, model.versionIdx)
}

func TestModel_Update_VimNavigation(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldSourceISO

	// j for down/next
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	updated, _ := m.Update(msg)
	model := updated.(*Model)
	assert.Equal(t, fieldOutputPath, model.focusedField)

	// k for up/prev
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	updated, _ = model.Update(msg)
	model = updated.(*Model)
	assert.Equal(t, fieldSourceISO, model.focusedField)
}

func TestBuildISOMsg(t *testing.T) {
	opts := &isobuilder.ISOOptions{SourceISO: "/test.iso"}
	msg := BuildISOMsg{Options: opts}
	assert.Equal(t, "/test.iso", msg.Options.SourceISO)
}

func TestBuildCompleteMsg(t *testing.T) {
	msg := BuildCompleteMsg{Success: true, OutputPath: "/output.iso", Error: nil}
	assert.True(t, msg.Success)
	assert.Equal(t, "/output.iso", msg.OutputPath)
	assert.Nil(t, msg.Error)
}

func TestModel_versions(t *testing.T) {
	assert.Equal(t, []string{"24.04", "22.04"}, versions)
}

func TestModel_storages(t *testing.T) {
	assert.Equal(t, []string{"lvm", "direct", "zfs"}, storages)
}

func TestModel_Left_AtStart_DoesNothing(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldVersion
	m.versionIdx = 0

	msg := tea.KeyMsg{Type: tea.KeyLeft}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, 0, model.versionIdx)
}

func TestModel_Right_AtEnd_DoesNothing(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldVersion
	m.versionIdx = len(versions) - 1

	msg := tea.KeyMsg{Type: tea.KeyRight}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, len(versions)-1, model.versionIdx)
}

func TestModel_SetSize_AdjustsInputWidth(t *testing.T) {
	m := New("/test/project")

	// Small width
	m.SetSize(40, 30)
	assert.Equal(t, 30, m.sourceInput.Width) // Minimum 30

	// Large width
	m.SetSize(200, 30)
	assert.Equal(t, 60, m.sourceInput.Width) // Maximum 60

	// Normal width
	m.SetSize(80, 30)
	assert.Equal(t, 55, m.sourceInput.Width) // 80 - 25 = 55
}

func TestModel_blurCurrent(t *testing.T) {
	m := New("/test/project")

	// Test blurring each text input field
	m.focusedField = fieldSourceISO
	m.sourceInput.Focus()
	m.blurCurrent()
	// Input should be blurred (no direct way to check, just verify no panic)

	m.focusedField = fieldOutputPath
	m.outputInput.Focus()
	m.blurCurrent()

	m.focusedField = fieldTimezone
	m.timezoneInput.Focus()
	m.blurCurrent()
}

func TestModel_focusCurrent(t *testing.T) {
	m := New("/test/project")

	// Test focusing each text input field
	m.focusedField = fieldSourceISO
	m.focusCurrent()
	// Just verify no panic

	m.focusedField = fieldOutputPath
	m.focusCurrent()

	m.focusedField = fieldTimezone
	m.focusCurrent()
}

func TestModel_Update_TextInput(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldSourceISO

	// Type a character
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Contains(t, model.sourceInput.Value(), "a")
}

func TestModel_Update_TextInput_OutputPath(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldOutputPath
	m.outputInput.Focus()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Contains(t, model.outputInput.Value(), "b")
}

func TestModel_Update_TextInput_Timezone(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldTimezone
	m.timezoneInput.Focus()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Contains(t, model.timezoneInput.Value(), "c")
}

func TestModel_handleKeyMsg_TextInput_OutputPath(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldOutputPath
	m.outputInput.Focus()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.Contains(t, model.outputInput.Value(), "x")
	assert.NotNil(t, cmd)
}

func TestModel_handleKeyMsg_TextInput_Timezone(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldTimezone
	m.timezoneInput.Focus()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.Contains(t, model.timezoneInput.Value(), "y")
	assert.NotNil(t, cmd)
}

func TestModel_handleKeyMsg_Left_OnNonSelect(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldSourceISO

	msg := tea.KeyMsg{Type: tea.KeyLeft}
	updated, _ := m.handleKeyMsg(msg)

	// Should not crash, just pass through
	assert.NotNil(t, updated)
}

func TestModel_handleKeyMsg_Right_OnNonSelect(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldSourceISO

	msg := tea.KeyMsg{Type: tea.KeyRight}
	updated, _ := m.handleKeyMsg(msg)

	// Should not crash, just pass through
	assert.NotNil(t, updated)
}

func TestModel_View_FocusedFields(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 40)

	// Test each focused field renders differently
	fields := []field{
		fieldSourceISO,
		fieldOutputPath,
		fieldVersion,
		fieldStorage,
		fieldTimezone,
		fieldSubmit,
	}

	for _, f := range fields {
		m.focusedField = f
		view := m.View()
		assert.NotEmpty(t, view)
	}
}

func TestModel_Storage_AtStart_DoesNothing(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldStorage
	m.storageIdx = 0

	msg := tea.KeyMsg{Type: tea.KeyLeft}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, 0, model.storageIdx)
}

func TestModel_Storage_AtEnd_DoesNothing(t *testing.T) {
	m := New("/test/project")
	m.focusedField = fieldStorage
	m.storageIdx = len(storages) - 1

	msg := tea.KeyMsg{Type: tea.KeyRight}
	updated, _ := m.Update(msg)
	model := updated.(*Model)

	assert.Equal(t, len(storages)-1, model.storageIdx)
}
