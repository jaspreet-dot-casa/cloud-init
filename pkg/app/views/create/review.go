package create

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// Review-specific field indices
const (
	reviewFieldConfirm = iota
	reviewFieldCancel
	reviewFieldCount
)

// initReviewPhase initializes the Review phase
func (m *Model) initReviewPhase() {
	// Reset focus to confirm button
	m.wizard.FocusedField = reviewFieldConfirm
}

// handleReviewPhase handles input for the Review phase
func (m *Model) handleReviewPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.wizard.FocusedField > 0 {
			m.wizard.FocusedField--
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j", "tab"))):
		if m.wizard.FocusedField < reviewFieldCount-1 {
			m.wizard.FocusedField++
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if m.wizard.FocusedField == reviewFieldConfirm {
			// Start deployment
			m.wizard.Advance()
			m.initPhase(m.wizard.Phase)
			return m, m.startDeploy()
		} else if m.wizard.FocusedField == reviewFieldCancel {
			// Go back to target selection
			m.wizard.Phase = wizard.PhaseTarget
			m.initPhase(m.wizard.Phase)
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

// viewReviewPhase renders the Review phase
func (m *Model) viewReviewPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Review Configuration"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Review your settings before proceeding."))
	b.WriteString("\n\n")

	// Target
	b.WriteString(labelStyle.Render("Target: "))
	b.WriteString(valueStyle.Render(m.getTargetName()))
	b.WriteString("\n\n")

	// Target-specific options
	b.WriteString(m.viewTargetSpecificReview())

	// SSH
	b.WriteString(labelStyle.Render("GitHub User: "))
	if m.wizard.Data.GitHubUser != "" {
		b.WriteString(valueStyle.Render(m.wizard.Data.GitHubUser))
	} else {
		b.WriteString(dimStyle.Render("(none)"))
	}
	b.WriteString("\n")

	// Git
	b.WriteString(labelStyle.Render("Git Name: "))
	if m.wizard.Data.GitName != "" {
		b.WriteString(valueStyle.Render(m.wizard.Data.GitName))
	} else {
		b.WriteString(dimStyle.Render("(none)"))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Git Email: "))
	if m.wizard.Data.GitEmail != "" {
		b.WriteString(valueStyle.Render(m.wizard.Data.GitEmail))
	} else {
		b.WriteString(dimStyle.Render("(none)"))
	}
	b.WriteString("\n\n")

	// Host
	b.WriteString(labelStyle.Render("Username: "))
	b.WriteString(valueStyle.Render(m.wizard.Data.Username))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Hostname: "))
	b.WriteString(valueStyle.Render(m.wizard.Data.Hostname))
	b.WriteString("\n\n")

	// Packages
	b.WriteString(labelStyle.Render("Packages: "))
	if len(m.wizard.Data.Packages) > 0 {
		b.WriteString(valueStyle.Render(fmt.Sprintf("%d selected", len(m.wizard.Data.Packages))))
	} else {
		b.WriteString(dimStyle.Render("(none)"))
	}
	b.WriteString("\n\n")

	// Optional services
	b.WriteString(labelStyle.Render("Tailscale: "))
	if m.wizard.Data.TailscaleKey != "" {
		b.WriteString(valueStyle.Render("configured"))
	} else {
		b.WriteString(dimStyle.Render("(none)"))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("GitHub PAT: "))
	if m.wizard.Data.GitHubPAT != "" {
		b.WriteString(valueStyle.Render("configured"))
	} else {
		b.WriteString(dimStyle.Render("(none)"))
	}
	b.WriteString("\n\n")

	// Separator
	b.WriteString(dimStyle.Render(strings.Repeat("-", 40)))
	b.WriteString("\n\n")

	// Action buttons
	confirmFocused := m.wizard.FocusedField == reviewFieldConfirm
	cancelFocused := m.wizard.FocusedField == reviewFieldCancel

	confirmCursor := "  "
	cancelCursor := "  "
	if confirmFocused {
		confirmCursor = "▸ "
	}
	if cancelFocused {
		cancelCursor = "▸ "
	}

	b.WriteString(confirmCursor)
	if confirmFocused {
		b.WriteString(focusedInputStyle.Render("[Generate Config]"))
	} else {
		b.WriteString(labelStyle.Render("[Generate Config]"))
	}
	b.WriteString("\n")

	b.WriteString(cancelCursor)
	if cancelFocused {
		b.WriteString(focusedInputStyle.Render("[Cancel]"))
	} else {
		b.WriteString(labelStyle.Render("[Cancel]"))
	}
	b.WriteString("\n")

	return b.String()
}

// getTargetName returns a human-readable name for the selected target
func (m *Model) getTargetName() string {
	switch m.wizard.Data.Target {
	case deploy.TargetTerragrunt:
		return "Terragrunt/libvirt"
	case deploy.TargetMultipass:
		return "Multipass"
	case deploy.TargetUSB:
		return "Bootable USB"
	case deploy.TargetConfigOnly:
		return "Generate Config Only"
	default:
		return "Unknown"
	}
}

// viewTargetSpecificReview renders target-specific options for review
func (m *Model) viewTargetSpecificReview() string {
	var b strings.Builder

	switch m.wizard.Data.Target {
	case deploy.TargetMultipass:
		opts := m.wizard.Data.MultipassOpts
		b.WriteString(labelStyle.Render("VM Name: "))
		b.WriteString(valueStyle.Render(opts.VMName))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Ubuntu Image: "))
		b.WriteString(valueStyle.Render(opts.UbuntuVersion))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("CPUs: "))
		b.WriteString(valueStyle.Render(fmt.Sprintf("%d", opts.CPUs)))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Memory: "))
		b.WriteString(valueStyle.Render(fmt.Sprintf("%d MB", opts.MemoryMB)))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Disk: "))
		b.WriteString(valueStyle.Render(fmt.Sprintf("%d GB", opts.DiskGB)))
		b.WriteString("\n\n")

	case deploy.TargetTerragrunt:
		opts := m.wizard.Data.TerragruntOpts
		b.WriteString(labelStyle.Render("VM Name: "))
		b.WriteString(valueStyle.Render(opts.VMName))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("CPUs: "))
		b.WriteString(valueStyle.Render(fmt.Sprintf("%d", opts.CPUs)))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Memory: "))
		b.WriteString(valueStyle.Render(fmt.Sprintf("%d MB", opts.MemoryMB)))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Disk: "))
		b.WriteString(valueStyle.Render(fmt.Sprintf("%d GB", opts.DiskGB)))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Image Path: "))
		b.WriteString(valueStyle.Render(opts.UbuntuImage))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Libvirt URI: "))
		b.WriteString(valueStyle.Render(opts.LibvirtURI))
		b.WriteString("\n\n")

	case deploy.TargetUSB:
		opts := m.wizard.Data.USBOpts
		b.WriteString(labelStyle.Render("Source ISO: "))
		b.WriteString(valueStyle.Render(opts.SourceISO))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Output Path: "))
		b.WriteString(valueStyle.Render(opts.OutputPath))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Storage Layout: "))
		b.WriteString(valueStyle.Render(opts.StorageLayout))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Timezone: "))
		b.WriteString(valueStyle.Render(opts.Timezone))
		b.WriteString("\n\n")

	case deploy.TargetConfigOnly:
		opts := m.wizard.Data.GenerateOpts
		b.WriteString(labelStyle.Render("Output Directory: "))
		b.WriteString(valueStyle.Render(opts.OutputDir))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Generate cloud-init: "))
		if opts.GenerateCloudInit {
			b.WriteString(valueStyle.Render("yes"))
		} else {
			b.WriteString(valueStyle.Render("no"))
		}
		b.WriteString("\n\n")
	}

	return b.String()
}
