package create

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// Ensure app.Tab is used (for interface compliance)
var _ app.Tab = (*Model)(nil)

// Multipass-specific field indices
const (
	multipassFieldVMName = iota
	multipassFieldImage
	multipassFieldCPU
	multipassFieldMemory
	multipassFieldDisk
	multipassFieldKeepOnFailure
	multipassFieldCount
)

// Multipass image options
var multipassImages = []struct {
	label string
	value string
}{
	{"24.04 LTS (Noble Numbat)", "24.04"},
	{"22.04 LTS (Jammy Jellyfish)", "22.04"},
	{"25.04 (Plucky Puffin)", "25.04"},
	{"daily:26.04 (Resolute)", "daily:26.04"},
}

// CPU options
var cpuOptions = []struct {
	label string
	value int
}{
	{"1 CPU", 1},
	{"2 CPUs (recommended)", 2},
	{"4 CPUs", 4},
}

// Memory options
var memoryOptions = []struct {
	label string
	value int
}{
	{"2 GB", 2048},
	{"4 GB (recommended)", 4096},
	{"8 GB", 8192},
}

// Disk options
var diskOptions = []struct {
	label string
	value int
}{
	{"10 GB", 10},
	{"20 GB (recommended)", 20},
	{"40 GB", 40},
}

// Helper functions to extract labels for renderSelectField
func getImageLabels() []string {
	labels := make([]string, len(multipassImages))
	for i, opt := range multipassImages {
		labels[i] = opt.label
	}
	return labels
}

func getCPULabels() []string {
	labels := make([]string, len(cpuOptions))
	for i, opt := range cpuOptions {
		labels[i] = opt.label
	}
	return labels
}

func getMemoryLabels() []string {
	labels := make([]string, len(memoryOptions))
	for i, opt := range memoryOptions {
		labels[i] = opt.label
	}
	return labels
}

func getDiskLabels() []string {
	labels := make([]string, len(diskOptions))
	for i, opt := range diskOptions {
		labels[i] = opt.label
	}
	return labels
}

// initMultipassPhase initializes the Multipass options phase
func (m *Model) initMultipassPhase() {
	// VM Name input
	vmName := textinput.New()
	vmName.Placeholder = "cloud-init-" + time.Now().Format("0102-1504")
	vmName.SetValue(vmName.Placeholder)
	vmName.CharLimit = 64
	vmName.Focus()
	m.wizard.TextInputs["vm_name"] = vmName

	// Set default selections
	m.wizard.SelectIdxs["image"] = 0   // 24.04 LTS
	m.wizard.SelectIdxs["cpu"] = 1     // 2 CPUs
	m.wizard.SelectIdxs["memory"] = 1  // 4 GB
	m.wizard.SelectIdxs["disk"] = 1    // 20 GB
	m.wizard.CheckStates["keep_on_failure"] = false
}

// handleMultipassPhase handles input for the Multipass options phase
func (m *Model) handleMultipassPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
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
		if m.wizard.FocusedField < multipassFieldCount-1 {
			m.wizard.FocusedField++
		}
		m.focusCurrentInput()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		// Forward to text input if on VM name field
		if m.wizard.FocusedField == multipassFieldVMName {
			return m.updateActiveTextInput(msg)
		}
		m.cycleMultipassOption(-1)
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
		// Forward to text input if on VM name field
		if m.wizard.FocusedField == multipassFieldVMName {
			return m.updateActiveTextInput(msg)
		}
		m.cycleMultipassOption(1)
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
		// Toggle checkbox
		if m.wizard.FocusedField == multipassFieldKeepOnFailure {
			m.wizard.CheckStates["keep_on_failure"] = !m.wizard.CheckStates["keep_on_failure"]
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Validate and advance
		m.saveMultipassOptions()
		m.wizard.Advance()
		m.initPhase(m.wizard.Phase)
		return m, nil
	}

	// Forward to text input if on VM name field
	if m.wizard.FocusedField == multipassFieldVMName {
		return m.updateActiveTextInput(msg)
	}

	return m, nil
}

// cycleMultipassOption cycles through options for select fields
func (m *Model) cycleMultipassOption(delta int) {
	switch m.wizard.FocusedField {
	case multipassFieldImage:
		idx := m.wizard.SelectIdxs["image"] + delta
		if idx < 0 {
			idx = len(multipassImages) - 1
		} else if idx >= len(multipassImages) {
			idx = 0
		}
		m.wizard.SelectIdxs["image"] = idx

	case multipassFieldCPU:
		idx := m.wizard.SelectIdxs["cpu"] + delta
		if idx < 0 {
			idx = len(cpuOptions) - 1
		} else if idx >= len(cpuOptions) {
			idx = 0
		}
		m.wizard.SelectIdxs["cpu"] = idx

	case multipassFieldMemory:
		idx := m.wizard.SelectIdxs["memory"] + delta
		if idx < 0 {
			idx = len(memoryOptions) - 1
		} else if idx >= len(memoryOptions) {
			idx = 0
		}
		m.wizard.SelectIdxs["memory"] = idx

	case multipassFieldDisk:
		idx := m.wizard.SelectIdxs["disk"] + delta
		if idx < 0 {
			idx = len(diskOptions) - 1
		} else if idx >= len(diskOptions) {
			idx = 0
		}
		m.wizard.SelectIdxs["disk"] = idx
	}
}

// saveMultipassOptions saves the Multipass options to wizard data
func (m *Model) saveMultipassOptions() {
	vmName := m.wizard.GetTextInput("vm_name")
	if vmName == "" {
		vmName = "cloud-init-" + time.Now().Format("0102-1504")
	}

	m.wizard.Data.MultipassOpts = deploy.MultipassOptions{
		VMName:        vmName,
		UbuntuVersion: multipassImages[m.wizard.SelectIdxs["image"]].value,
		CPUs:          cpuOptions[m.wizard.SelectIdxs["cpu"]].value,
		MemoryMB:      memoryOptions[m.wizard.SelectIdxs["memory"]].value,
		DiskGB:        diskOptions[m.wizard.SelectIdxs["disk"]].value,
		KeepOnFailure: m.wizard.CheckStates["keep_on_failure"],
	}
}

// viewMultipassPhase renders the Multipass options phase
func (m *Model) viewMultipassPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Multipass VM Options"))
	b.WriteString("\n\n")

	// VM Name
	b.WriteString(m.renderTextField("VM Name", "vm_name", multipassFieldVMName))

	// Image selection
	b.WriteString(m.renderSelectField("Ubuntu Image", "image", multipassFieldImage, getImageLabels()))

	// CPU selection
	b.WriteString(m.renderSelectField("CPUs", "cpu", multipassFieldCPU, getCPULabels()))

	// Memory selection
	b.WriteString(m.renderSelectField("Memory", "memory", multipassFieldMemory, getMemoryLabels()))

	// Disk selection
	b.WriteString(m.renderSelectField("Disk Size", "disk", multipassFieldDisk, getDiskLabels()))

	// Keep on failure checkbox
	b.WriteString(m.renderCheckbox("Keep VM on failure", "keep_on_failure", multipassFieldKeepOnFailure))

	return b.String()
}

// renderTextField renders a text input field
func (m *Model) renderTextField(label, name string, fieldIdx int) string {
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

// renderSelectField renders a select field with options
func (m *Model) renderSelectField(label, name string, fieldIdx int, options []string) string {
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

	idx := m.wizard.SelectIdxs[name]
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

// renderCheckbox renders a checkbox field
func (m *Model) renderCheckbox(label, name string, fieldIdx int) string {
	var b strings.Builder

	focused := m.wizard.FocusedField == fieldIdx
	cursor := "  "
	if focused {
		cursor = "▸ "
	}

	checked := m.wizard.CheckStates[name]
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
