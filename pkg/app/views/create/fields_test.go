package create

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/stretchr/testify/assert"
)

func TestRenderTextField_Focused(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 0

	// Create a text input
	ti := textinput.New()
	ti.SetValue("test-value")
	wizard.TextInputs["test_field"] = ti

	result := RenderTextField(wizard, "Test Label", "test_field", 0)

	// Check cursor is present when focused
	assert.Contains(t, result, "▸ ")
	assert.Contains(t, result, "Test Label:")
}

func TestRenderTextField_Unfocused(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 1 // Focus on different field

	// Create a text input
	ti := textinput.New()
	ti.SetValue("test-value")
	wizard.TextInputs["test_field"] = ti

	result := RenderTextField(wizard, "Test Label", "test_field", 0)

	// Check no cursor when unfocused
	assert.True(t, strings.HasPrefix(result, "  "))
	assert.Contains(t, result, "Test Label:")
}

func TestRenderSelectField_Focused(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 0
	wizard.SelectIdxs["test_select"] = 1

	options := []string{"Option A", "Option B", "Option C"}
	result := RenderSelectField(wizard, "Test Select", "test_select", 0, options)

	// Check focused state shows arrows
	assert.Contains(t, result, "▸ ")
	assert.Contains(t, result, "Test Select:")
	assert.Contains(t, result, "◀")
	assert.Contains(t, result, "▶")
	assert.Contains(t, result, "Option B")
}

func TestRenderSelectField_Unfocused(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 1 // Focus on different field
	wizard.SelectIdxs["test_select"] = 0

	options := []string{"Option A", "Option B", "Option C"}
	result := RenderSelectField(wizard, "Test Select", "test_select", 0, options)

	// Check unfocused state doesn't show arrows
	assert.True(t, strings.HasPrefix(result, "  "))
	assert.Contains(t, result, "Test Select:")
	assert.NotContains(t, result, "◀")
	assert.NotContains(t, result, "▶")
	assert.Contains(t, result, "Option A")
}

func TestRenderSelectField_InvalidIndex(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 0
	wizard.SelectIdxs["test_select"] = 99 // Invalid index

	options := []string{"Option A", "Option B", "Option C"}
	result := RenderSelectField(wizard, "Test Select", "test_select", 0, options)

	// Should not crash, just not show the value
	assert.Contains(t, result, "Test Select:")
}

func TestRenderCheckbox_Checked(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 0
	wizard.CheckStates["test_check"] = true

	result := RenderCheckbox(wizard, "Test Checkbox", "test_check", 0)

	assert.Contains(t, result, "[✓]")
	assert.Contains(t, result, "Test Checkbox")
	assert.Contains(t, result, "▸ ")
}

func TestRenderCheckbox_Unchecked(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 1 // Focus on different field
	wizard.CheckStates["test_check"] = false

	result := RenderCheckbox(wizard, "Test Checkbox", "test_check", 0)

	assert.Contains(t, result, "[ ]")
	assert.Contains(t, result, "Test Checkbox")
	assert.True(t, strings.HasPrefix(result, "  "))
}

func TestRenderCheckbox_DefaultState(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 0
	// Not setting CheckStates["test_check"], so it defaults to false

	result := RenderCheckbox(wizard, "Test Checkbox", "test_check", 0)

	assert.Contains(t, result, "[ ]")
}
