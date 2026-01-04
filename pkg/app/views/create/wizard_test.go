package create

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/stretchr/testify/assert"
)

func TestNewWizardState(t *testing.T) {
	wizard := NewWizardState()

	assert.Equal(t, PhaseTarget, wizard.Phase)
	assert.Equal(t, 0, wizard.FocusedField)
	assert.NotNil(t, wizard.TextInputs)
	assert.NotNil(t, wizard.SelectIdxs)
	assert.NotNil(t, wizard.CheckStates)
	assert.NotNil(t, wizard.PackageSelected)
	assert.NotNil(t, wizard.SSHKeySelected)
}

func TestWizardState_NavigateField_Down(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 0

	wizard.NavigateField(1, 5)
	assert.Equal(t, 1, wizard.FocusedField)

	wizard.NavigateField(1, 5)
	assert.Equal(t, 2, wizard.FocusedField)
}

func TestWizardState_NavigateField_Up(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 3

	wizard.NavigateField(-1, 5)
	assert.Equal(t, 2, wizard.FocusedField)

	wizard.NavigateField(-1, 5)
	assert.Equal(t, 1, wizard.FocusedField)
}

func TestWizardState_NavigateField_BoundsMin(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 0

	// Should not go below 0
	wizard.NavigateField(-1, 5)
	assert.Equal(t, 0, wizard.FocusedField)

	wizard.NavigateField(-5, 5)
	assert.Equal(t, 0, wizard.FocusedField)
}

func TestWizardState_NavigateField_BoundsMax(t *testing.T) {
	wizard := NewWizardState()
	wizard.FocusedField = 5

	// Should not go above maxField
	wizard.NavigateField(1, 5)
	assert.Equal(t, 5, wizard.FocusedField)

	wizard.NavigateField(10, 5)
	assert.Equal(t, 5, wizard.FocusedField)
}

func TestWizardState_CycleSelect_Forward(t *testing.T) {
	wizard := NewWizardState()
	wizard.SelectIdxs["test"] = 0

	wizard.CycleSelect("test", 3, 1)
	assert.Equal(t, 1, wizard.SelectIdxs["test"])

	wizard.CycleSelect("test", 3, 1)
	assert.Equal(t, 2, wizard.SelectIdxs["test"])
}

func TestWizardState_CycleSelect_Backward(t *testing.T) {
	wizard := NewWizardState()
	wizard.SelectIdxs["test"] = 2

	wizard.CycleSelect("test", 3, -1)
	assert.Equal(t, 1, wizard.SelectIdxs["test"])

	wizard.CycleSelect("test", 3, -1)
	assert.Equal(t, 0, wizard.SelectIdxs["test"])
}

func TestWizardState_CycleSelect_WrapForward(t *testing.T) {
	wizard := NewWizardState()
	wizard.SelectIdxs["test"] = 2 // Last index

	wizard.CycleSelect("test", 3, 1)
	assert.Equal(t, 0, wizard.SelectIdxs["test"]) // Should wrap to first
}

func TestWizardState_CycleSelect_WrapBackward(t *testing.T) {
	wizard := NewWizardState()
	wizard.SelectIdxs["test"] = 0 // First index

	wizard.CycleSelect("test", 3, -1)
	assert.Equal(t, 2, wizard.SelectIdxs["test"]) // Should wrap to last
}

func TestWizardState_CycleSelect_EmptyOptions(t *testing.T) {
	wizard := NewWizardState()
	wizard.SelectIdxs["test"] = 0

	// Should do nothing with 0 options
	wizard.CycleSelect("test", 0, 1)
	assert.Equal(t, 0, wizard.SelectIdxs["test"])
}

func TestWizardState_Advance(t *testing.T) {
	wizard := NewWizardState()
	wizard.Phase = PhaseTarget
	wizard.FocusedField = 3

	wizard.Advance()

	assert.Equal(t, PhaseTargetOptions, wizard.Phase)
	assert.Equal(t, 0, wizard.FocusedField) // Should reset focus
}

func TestWizardState_GoBack(t *testing.T) {
	wizard := NewWizardState()
	wizard.Phase = PhaseSSH
	wizard.FocusedField = 3

	wizard.GoBack()

	assert.Equal(t, PhaseTargetOptions, wizard.Phase)
	assert.Equal(t, 0, wizard.FocusedField) // Should reset focus
}

func TestWizardState_GoBack_FromTarget(t *testing.T) {
	wizard := NewWizardState()
	wizard.Phase = PhaseTarget

	// Should not go back from target phase
	wizard.GoBack()
	assert.Equal(t, PhaseTarget, wizard.Phase)
}

func TestWizardState_CanGoBack(t *testing.T) {
	wizard := NewWizardState()

	// Cannot go back from Target
	wizard.Phase = PhaseTarget
	assert.False(t, wizard.CanGoBack())

	// Cannot go back from Deploy
	wizard.Phase = PhaseDeploy
	assert.False(t, wizard.CanGoBack())

	// Cannot go back from Complete
	wizard.Phase = PhaseComplete
	assert.False(t, wizard.CanGoBack())

	// Can go back from other phases
	wizard.Phase = PhaseSSH
	assert.True(t, wizard.CanGoBack())

	wizard.Phase = PhaseReview
	assert.True(t, wizard.CanGoBack())
}

func TestWizardState_Reset(t *testing.T) {
	wizard := NewWizardState()
	wizard.Phase = PhaseSSH
	wizard.FocusedField = 5
	wizard.TextInputs["test"] = textinput.Model{}
	wizard.SelectIdxs["test"] = 2
	wizard.CheckStates["test"] = true

	wizard.Reset()

	assert.Equal(t, PhaseTarget, wizard.Phase)
	assert.Equal(t, 0, wizard.FocusedField)
	assert.Empty(t, wizard.TextInputs)
	assert.Empty(t, wizard.SelectIdxs)
	assert.Empty(t, wizard.CheckStates)
}

func TestWizardState_GetSetTextInput(t *testing.T) {
	wizard := NewWizardState()
	wizard.InitTextInput("name", "placeholder", 64)

	// Default value should be empty
	assert.Equal(t, "", wizard.GetTextInput("name"))

	// Set and get
	wizard.SetTextInput("name", "test-value")
	assert.Equal(t, "test-value", wizard.GetTextInput("name"))

	// Non-existent field returns empty
	assert.Equal(t, "", wizard.GetTextInput("nonexistent"))
}

func TestWizardState_GetSetSelectIdx(t *testing.T) {
	wizard := NewWizardState()

	// Default is 0
	assert.Equal(t, 0, wizard.GetSelectIdx("test"))

	wizard.SetSelectIdx("test", 2)
	assert.Equal(t, 2, wizard.GetSelectIdx("test"))
}

func TestWizardState_GetSetCheckState(t *testing.T) {
	wizard := NewWizardState()

	// Default is false
	assert.False(t, wizard.GetCheckState("test"))

	wizard.SetCheckState("test", true)
	assert.True(t, wizard.GetCheckState("test"))

	wizard.SetCheckState("test", false)
	assert.False(t, wizard.GetCheckState("test"))
}

func TestWizardState_PhaseProgress(t *testing.T) {
	wizard := NewWizardState()

	wizard.Phase = PhaseTarget
	assert.Equal(t, 0.0, wizard.PhaseProgress())

	wizard.Phase = PhaseComplete
	assert.Equal(t, 1.0, wizard.PhaseProgress())
}
