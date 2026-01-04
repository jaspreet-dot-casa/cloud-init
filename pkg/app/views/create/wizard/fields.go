package wizard

import (
	"fmt"
	"strings"
)

// RenderTextField renders a text input field with cursor and label styling.
func RenderTextField(wizard *State, label, inputName string, fieldIdx int) string {
	var b strings.Builder

	focused := wizard.FocusedField == fieldIdx
	cursor := "  "
	if focused {
		cursor = "▸ "
	}

	b.WriteString(cursor)
	if focused {
		b.WriteString(FocusedInputStyle.Render(label + ": "))
	} else {
		b.WriteString(LabelStyle.Render(label + ": "))
	}

	if ti, ok := wizard.TextInputs[inputName]; ok {
		b.WriteString(ti.View())
	}
	b.WriteString("\n\n")

	return b.String()
}

// RenderSelectField renders a select field with left/right navigation arrows.
func RenderSelectField(wizard *State, label, selectName string, fieldIdx int, options []string) string {
	var b strings.Builder

	focused := wizard.FocusedField == fieldIdx
	cursor := "  "
	if focused {
		cursor = "▸ "
	}

	b.WriteString(cursor)
	if focused {
		b.WriteString(FocusedInputStyle.Render(label + ": "))
	} else {
		b.WriteString(LabelStyle.Render(label + ": "))
	}

	idx := wizard.SelectIdxs[selectName]
	if idx >= 0 && idx < len(options) {
		if focused {
			b.WriteString(fmt.Sprintf("◀ %s ▶", SelectedStyle.Render(options[idx])))
		} else {
			b.WriteString(ValueStyle.Render(options[idx]))
		}
	}
	b.WriteString("\n\n")

	return b.String()
}

// RenderCheckbox renders a checkbox field with checked/unchecked state.
func RenderCheckbox(wizard *State, label, checkName string, fieldIdx int) string {
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
		b.WriteString(FocusedInputStyle.Render(checkbox + " " + label))
	} else {
		b.WriteString(LabelStyle.Render(checkbox + " " + label))
	}
	b.WriteString("\n\n")

	return b.String()
}
