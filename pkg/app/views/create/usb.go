package create

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// USB-specific field indices
const (
	usbFieldSourceISO = iota
	usbFieldOutputPath
	usbFieldStorage
	usbFieldTimezone
	usbFieldCount
)

// Storage layout options
var storageOptions = []struct {
	label string
	value string
}{
	{"LVM (recommended)", "lvm"},
	{"Direct", "direct"},
	{"ZFS", "zfs"},
}

// getStorageLabels extracts labels for renderSelectField
func getStorageLabels() []string {
	labels := make([]string, len(storageOptions))
	for i, opt := range storageOptions {
		labels[i] = opt.label
	}
	return labels
}

// initUSBPhase initializes the USB/ISO options phase
func (m *Model) initUSBPhase() {
	// Source ISO input
	sourceISO := textinput.New()
	sourceISO.Placeholder = "ubuntu-24.04-live-server-amd64.iso"
	sourceISO.CharLimit = 256
	sourceISO.Focus()
	m.wizard.TextInputs["source_iso"] = sourceISO

	// Output path input
	outputPath := textinput.New()
	outputPath.Placeholder = "output/ubuntu-autoinstall.iso"
	outputPath.SetValue("output/ubuntu-autoinstall.iso")
	outputPath.CharLimit = 256
	m.wizard.TextInputs["output_path"] = outputPath

	// Timezone input
	timezone := textinput.New()
	timezone.Placeholder = "UTC"
	timezone.SetValue("UTC")
	timezone.CharLimit = 64
	m.wizard.TextInputs["timezone"] = timezone

	// Default storage layout
	m.wizard.SelectIdxs["storage"] = 0 // LVM
}

// handleUSBPhase handles input for the USB options phase
func (m *Model) handleUSBPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		m.blurCurrentInput()
		if m.wizard.FocusedField > 0 {
			m.wizard.FocusedField--
		}
		m.focusCurrentInput()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j", "tab"))):
		m.blurCurrentInput()
		if m.wizard.FocusedField < usbFieldCount-1 {
			m.wizard.FocusedField++
		}
		m.focusCurrentInput()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		m.cycleUSBOption(-1)
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
		m.cycleUSBOption(1)
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Validate and advance
		m.saveUSBOptions()
		m.wizard.Advance()
		m.initPhase(m.wizard.Phase)
		return m, nil
	}

	// Forward to text input for text fields
	switch m.wizard.FocusedField {
	case usbFieldSourceISO, usbFieldOutputPath, usbFieldTimezone:
		return m.updateActiveTextInput(msg)
	}

	return m, nil
}

// cycleUSBOption cycles through options for select fields
func (m *Model) cycleUSBOption(delta int) {
	if m.wizard.FocusedField == usbFieldStorage {
		idx := m.wizard.SelectIdxs["storage"] + delta
		if idx < 0 {
			idx = len(storageOptions) - 1
		} else if idx >= len(storageOptions) {
			idx = 0
		}
		m.wizard.SelectIdxs["storage"] = idx
	}
}

// saveUSBOptions saves the USB options to wizard data
func (m *Model) saveUSBOptions() {
	sourceISO := m.wizard.GetTextInput("source_iso")
	outputPath := m.wizard.GetTextInput("output_path")
	if outputPath == "" {
		outputPath = "output/ubuntu-autoinstall.iso"
	}
	timezone := m.wizard.GetTextInput("timezone")
	if timezone == "" {
		timezone = "UTC"
	}

	m.wizard.Data.USBOpts = USBOptions{
		SourceISO:     sourceISO,
		OutputPath:    outputPath,
		StorageLayout: storageOptions[m.wizard.SelectIdxs["storage"]].value,
		Timezone:      timezone,
	}
}

// viewUSBPhase renders the USB options phase
func (m *Model) viewUSBPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Bootable USB/ISO Options"))
	b.WriteString("\n\n")

	// Source ISO
	b.WriteString(m.renderUSBTextField("Source ISO", "source_iso", usbFieldSourceISO))

	// Output path
	b.WriteString(m.renderUSBTextField("Output Path", "output_path", usbFieldOutputPath))

	// Storage layout
	b.WriteString(m.renderSelectField("Storage Layout", "storage", usbFieldStorage, getStorageLabels()))

	// Timezone
	b.WriteString(m.renderUSBTextField("Timezone", "timezone", usbFieldTimezone))

	return b.String()
}

// renderUSBTextField renders a text input field for USB
func (m *Model) renderUSBTextField(label, name string, fieldIdx int) string {
	var b strings.Builder

	focused := m.wizard.FocusedField == fieldIdx
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

	if ti, ok := m.wizard.TextInputs[name]; ok {
		b.WriteString(ti.View())
	}
	b.WriteString("\n\n")

	return b.String()
}
