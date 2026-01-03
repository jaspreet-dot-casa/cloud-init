package create

import (
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// Terraform-specific field indices
const (
	terraformFieldVMName = iota
	terraformFieldCPU
	terraformFieldMemory
	terraformFieldDisk
	terraformFieldImagePath
	terraformFieldLibvirtURI
	terraformFieldCount
)

// Default paths
const (
	defaultLibvirtURI  = "qemu:///system"
	defaultStoragePool = "default"
	defaultNetwork     = "default"
)

// initTerraformPhase initializes the Terraform options phase
func (m *Model) initTerraformPhase() {
	// VM Name input
	vmName := textinput.New()
	vmName.Placeholder = "cloud-init-" + time.Now().Format("0102-1504")
	vmName.SetValue(vmName.Placeholder)
	vmName.CharLimit = 64
	vmName.Focus()
	m.wizard.TextInputs["vm_name"] = vmName

	// Image path input - use first available cloud image or default
	imagePath := textinput.New()
	imagePath.CharLimit = 256
	if len(m.cloudImages) > 0 {
		// Use the first cloud image from settings
		imagePath.Placeholder = m.cloudImages[0].Path
		imagePath.SetValue(m.cloudImages[0].Path)
	} else {
		imagePath.Placeholder = "/var/lib/libvirt/images/noble-server-cloudimg-amd64.img"
	}
	m.wizard.TextInputs["image_path"] = imagePath

	// Libvirt URI input
	libvirtURI := textinput.New()
	libvirtURI.Placeholder = defaultLibvirtURI
	libvirtURI.SetValue(defaultLibvirtURI)
	libvirtURI.CharLimit = 128
	m.wizard.TextInputs["libvirt_uri"] = libvirtURI

	// Set default selections (same as Multipass)
	m.wizard.SelectIdxs["cpu"] = 1    // 2 CPUs
	m.wizard.SelectIdxs["memory"] = 1 // 4 GB
	m.wizard.SelectIdxs["disk"] = 1   // 20 GB

	// Initialize image selection index (for cycling through available images)
	m.wizard.SelectIdxs["image"] = 0
}

// handleTerraformPhase handles input for the Terraform options phase
func (m *Model) handleTerraformPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
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
		if m.wizard.FocusedField < terraformFieldCount-1 {
			m.wizard.FocusedField++
		}
		m.focusCurrentInput()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		m.cycleTerraformOption(-1)
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
		m.cycleTerraformOption(1)
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Validate and advance
		m.saveTerraformOptions()
		m.wizard.Advance()
		m.initPhase(m.wizard.Phase)
		return m, nil
	}

	// Forward to text input for text fields
	switch m.wizard.FocusedField {
	case terraformFieldVMName, terraformFieldImagePath, terraformFieldLibvirtURI:
		return m.updateActiveTextInput(msg)
	}

	return m, nil
}

// cycleTerraformOption cycles through options for select fields
func (m *Model) cycleTerraformOption(delta int) {
	switch m.wizard.FocusedField {
	case terraformFieldCPU:
		idx := m.wizard.SelectIdxs["cpu"] + delta
		if idx < 0 {
			idx = len(cpuOptions) - 1
		} else if idx >= len(cpuOptions) {
			idx = 0
		}
		m.wizard.SelectIdxs["cpu"] = idx

	case terraformFieldMemory:
		idx := m.wizard.SelectIdxs["memory"] + delta
		if idx < 0 {
			idx = len(memoryOptions) - 1
		} else if idx >= len(memoryOptions) {
			idx = 0
		}
		m.wizard.SelectIdxs["memory"] = idx

	case terraformFieldDisk:
		idx := m.wizard.SelectIdxs["disk"] + delta
		if idx < 0 {
			idx = len(diskOptions) - 1
		} else if idx >= len(diskOptions) {
			idx = 0
		}
		m.wizard.SelectIdxs["disk"] = idx
	}
}

// saveTerraformOptions saves the Terraform options to wizard data
func (m *Model) saveTerraformOptions() {
	vmName := m.wizard.GetTextInput("vm_name")
	if vmName == "" {
		vmName = "cloud-init-" + time.Now().Format("0102-1504")
	}

	imagePath := m.wizard.GetTextInput("image_path")
	if imagePath == "" {
		// Try to use cloud image from settings
		if len(m.cloudImages) > 0 {
			imagePath = m.cloudImages[0].Path
		} else {
			imagePath = "/var/lib/libvirt/images/noble-server-cloudimg-amd64.img"
		}
	}

	libvirtURI := m.wizard.GetTextInput("libvirt_uri")
	if libvirtURI == "" {
		libvirtURI = defaultLibvirtURI
	}

	m.wizard.Data.TerraformOpts = deploy.TerraformOptions{
		VMName:       vmName,
		CPUs:         cpuOptions[m.wizard.SelectIdxs["cpu"]].value,
		MemoryMB:     memoryOptions[m.wizard.SelectIdxs["memory"]].value,
		DiskGB:       diskOptions[m.wizard.SelectIdxs["disk"]].value,
		UbuntuImage:  imagePath,
		LibvirtURI:   libvirtURI,
		StoragePool:  defaultStoragePool,
		NetworkName:  defaultNetwork,
	}
}

// viewTerraformPhase renders the Terraform options phase
func (m *Model) viewTerraformPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Terraform/libvirt VM Options"))
	b.WriteString("\n\n")

	// Platform warning for non-Linux
	if runtime.GOOS != "linux" {
		b.WriteString(warningStyle.Render("⚠ Warning: "))
		b.WriteString(dimStyle.Render("libvirt/KVM is not available on " + runtime.GOOS + "."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  You can still configure options for a remote Linux server."))
		b.WriteString("\n\n")
	}

	// VM Name
	b.WriteString(m.renderTerraformTextField("VM Name", "vm_name", terraformFieldVMName))

	// CPU selection
	b.WriteString(m.renderSelectField("CPUs", "cpu", terraformFieldCPU, getCPULabels()))

	// Memory selection
	b.WriteString(m.renderSelectField("Memory", "memory", terraformFieldMemory, getMemoryLabels()))

	// Disk selection
	b.WriteString(m.renderSelectField("Disk Size", "disk", terraformFieldDisk, getDiskLabels()))

	// Image path
	b.WriteString(m.renderTerraformTextField("Ubuntu Image", "image_path", terraformFieldImagePath))

	// Libvirt URI
	b.WriteString(m.renderTerraformTextField("Libvirt URI", "libvirt_uri", terraformFieldLibvirtURI))

	return b.String()
}

// renderTerraformTextField renders a text input field for Terraform
func (m *Model) renderTerraformTextField(label, name string, fieldIdx int) string {
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
