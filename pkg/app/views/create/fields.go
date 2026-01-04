// Package create provides shared field rendering functions for the wizard.
package create

import (
	"fmt"
	"strings"
)

// RenderTextField renders a text input field with cursor and label styling.
func RenderTextField(wizard *WizardState, label, inputName string, fieldIdx int) string {
	var b strings.Builder

	focused := wizard.FocusedField == fieldIdx
	cursor := "  "
	if focused {
		cursor = "▸ "
	}

	b.WriteString(cursor)
	if focused {
		b.WriteString(focusedInputStyle.Render(label + ": "))
	} else {
		b.WriteString(labelStyle.Render(label + ": "))
	}

	if ti, ok := wizard.TextInputs[inputName]; ok {
		b.WriteString(ti.View())
	}
	b.WriteString("\n\n")

	return b.String()
}

// RenderSelectField renders a select field with left/right navigation arrows.
func RenderSelectField(wizard *WizardState, label, selectName string, fieldIdx int, options []string) string {
	var b strings.Builder

	focused := wizard.FocusedField == fieldIdx
	cursor := "  "
	if focused {
		cursor = "▸ "
	}

	b.WriteString(cursor)
	if focused {
		b.WriteString(focusedInputStyle.Render(label + ": "))
	} else {
		b.WriteString(labelStyle.Render(label + ": "))
	}

	idx := wizard.SelectIdxs[selectName]
	if idx >= 0 && idx < len(options) {
		if focused {
			b.WriteString(fmt.Sprintf("◀ %s ▶", selectedStyle.Render(options[idx])))
		} else {
			b.WriteString(valueStyle.Render(options[idx]))
		}
	}
	b.WriteString("\n\n")

	return b.String()
}

// RenderCheckbox renders a checkbox field with checked/unchecked state.
func RenderCheckbox(wizard *WizardState, label, checkName string, fieldIdx int) string {
	var b strings.Builder

	focused := wizard.FocusedField == fieldIdx
	cursor := "  "
	if focused {
		cursor = "▸ "
	}

	checked := wizard.CheckStates[checkName]
	checkbox := "[ ]"
	if checked {
		checkbox = "[✓]"
	}

	b.WriteString(cursor)
	if focused {
		b.WriteString(focusedInputStyle.Render(checkbox + " " + label))
	} else {
		b.WriteString(labelStyle.Render(checkbox + " " + label))
	}
	b.WriteString("\n\n")

	return b.String()
}
