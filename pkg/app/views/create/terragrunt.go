package create

import (
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// Terragrunt-specific field indices
const (
	terragruntFieldVMName = iota
	terragruntFieldCPU
	terragruntFieldMemory
	terragruntFieldDisk
	terragruntFieldImagePath
	terragruntFieldLibvirtURI
	terragruntFieldCount
)

// Default paths
const (
	defaultLibvirtURI  = "qemu:///system"
	defaultStoragePool = "default"
	defaultNetwork     = "default"
)

// initTerragruntPhase initializes the Terragrunt options phase
func (m *Model) initTerragruntPhase() {
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
}

// handleTerragruntPhase handles input for the Terragrunt options phase
func (m *Model) handleTerragruntPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
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
		if m.wizard.FocusedField < terragruntFieldCount-1 {
			m.wizard.FocusedField++
		}
		m.focusCurrentInput()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		m.cycleTerragruntOption(-1)
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
		m.cycleTerragruntOption(1)
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Validate and advance
		m.saveTerragruntOptions()
		m.wizard.Advance()
		m.initPhase(m.wizard.Phase)
		return m, nil
	}

	// Forward to text input for text fields
	switch m.wizard.FocusedField {
	case terragruntFieldVMName, terragruntFieldImagePath, terragruntFieldLibvirtURI:
		return m.updateActiveTextInput(msg)
	}

	return m, nil
}

// cycleTerragruntOption cycles through options for select fields
func (m *Model) cycleTerragruntOption(delta int) {
	switch m.wizard.FocusedField {
	case terragruntFieldCPU:
		m.wizard.CycleSelect("cpu", len(CPUOptions), delta)
	case terragruntFieldMemory:
		m.wizard.CycleSelect("memory", len(MemoryOptions), delta)
	case terragruntFieldDisk:
		m.wizard.CycleSelect("disk", len(DiskOptions), delta)
	}
}

// saveTerragruntOptions saves the Terragrunt options to wizard data
func (m *Model) saveTerragruntOptions() {
	vmName := m.wizard.GetTextInput("vm_name")
	if vmName == "" {
		vmName = "vm-" + time.Now().Format("0102-1504")
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

	m.wizard.Data.TerragruntOpts = deploy.TerragruntOptions{
		VMName:      vmName,
		CPUs:        GetCPUValue(m.wizard.SelectIdxs["cpu"]),
		MemoryMB:    GetMemoryValue(m.wizard.SelectIdxs["memory"]),
		DiskGB:      GetDiskValue(m.wizard.SelectIdxs["disk"]),
		UbuntuImage: imagePath,
		LibvirtURI:  libvirtURI,
		StoragePool: defaultStoragePool,
		NetworkName: defaultNetwork,
	}
}

// viewTerragruntPhase renders the Terragrunt options phase
func (m *Model) viewTerragruntPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Terragrunt/libvirt VM Options"))
	b.WriteString("\n\n")

	// Platform warning for non-Linux
	if runtime.GOOS != "linux" {
		b.WriteString(warningStyle.Render("Note: "))
		b.WriteString(dimStyle.Render("libvirt/KVM is only available on Linux."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  Configure options here, then apply on a Linux machine."))
		b.WriteString("\n\n")
	}

	// VM Name
	b.WriteString(wizard.RenderTextField(m.wizard, "VM Name", "vm_name", terragruntFieldVMName))

	// CPU selection
	b.WriteString(wizard.RenderSelectField(m.wizard, "CPUs", "cpu", terragruntFieldCPU, GetCPULabels()))

	// Memory selection
	b.WriteString(wizard.RenderSelectField(m.wizard, "Memory", "memory", terragruntFieldMemory, GetMemoryLabels()))

	// Disk selection
	b.WriteString(wizard.RenderSelectField(m.wizard, "Disk Size", "disk", terragruntFieldDisk, GetDiskLabels()))

	// Image path
	b.WriteString(wizard.RenderTextField(m.wizard, "Ubuntu Image", "image_path", terragruntFieldImagePath))

	// Libvirt URI
	b.WriteString(wizard.RenderTextField(m.wizard, "Libvirt URI", "libvirt_uri", terragruntFieldLibvirtURI))

	return b.String()
}

