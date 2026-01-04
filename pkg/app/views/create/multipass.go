package create

import (
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

// MultipassImageOptions defines available Ubuntu images for Multipass.
var MultipassImageOptions = []SelectOption[string]{
	{Label: "24.04 LTS (Noble Numbat)", Value: "24.04"},
	{Label: "22.04 LTS (Jammy Jellyfish)", Value: "22.04"},
	{Label: "25.04 (Plucky Puffin)", Value: "25.04"},
	{Label: "daily:26.04 (Resolute)", Value: "daily:26.04"},
}

// GetMultipassImageLabels returns labels for Multipass image options.
func GetMultipassImageLabels() []string {
	labels := make([]string, len(MultipassImageOptions))
	for i, opt := range MultipassImageOptions {
		labels[i] = opt.Label
	}
	return labels
}

// GetMultipassImageValue returns the image value at the given index.
func GetMultipassImageValue(idx int) string {
	if idx < 0 || idx >= len(MultipassImageOptions) {
		return MultipassImageOptions[0].Value // Default to first
	}
	return MultipassImageOptions[idx].Value
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
		m.wizard.CycleSelect("image", len(MultipassImageOptions), delta)
	case multipassFieldCPU:
		m.wizard.CycleSelect("cpu", len(CPUOptions), delta)
	case multipassFieldMemory:
		m.wizard.CycleSelect("memory", len(MemoryOptions), delta)
	case multipassFieldDisk:
		m.wizard.CycleSelect("disk", len(DiskOptions), delta)
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
		UbuntuVersion: GetMultipassImageValue(m.wizard.SelectIdxs["image"]),
		CPUs:          GetCPUValue(m.wizard.SelectIdxs["cpu"]),
		MemoryMB:      GetMemoryValue(m.wizard.SelectIdxs["memory"]),
		DiskGB:        GetDiskValue(m.wizard.SelectIdxs["disk"]),
		KeepOnFailure: m.wizard.CheckStates["keep_on_failure"],
	}
}

// viewMultipassPhase renders the Multipass options phase
func (m *Model) viewMultipassPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Multipass VM Options"))
	b.WriteString("\n\n")

	// VM Name
	b.WriteString(RenderTextField(m.wizard, "VM Name", "vm_name", multipassFieldVMName))

	// Image selection
	b.WriteString(RenderSelectField(m.wizard, "Ubuntu Image", "image", multipassFieldImage, GetMultipassImageLabels()))

	// CPU selection
	b.WriteString(RenderSelectField(m.wizard, "CPUs", "cpu", multipassFieldCPU, GetCPULabels()))

	// Memory selection
	b.WriteString(RenderSelectField(m.wizard, "Memory", "memory", multipassFieldMemory, GetMemoryLabels()))

	// Disk selection
	b.WriteString(RenderSelectField(m.wizard, "Disk Size", "disk", multipassFieldDisk, GetDiskLabels()))

	// Keep on failure checkbox
	b.WriteString(RenderCheckbox(m.wizard, "Keep VM on failure", "keep_on_failure", multipassFieldKeepOnFailure))

	return b.String()
}

