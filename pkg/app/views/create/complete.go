package create

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// initCompletePhase initializes the Complete phase
func (m *Model) initCompletePhase() {
	// Nothing special to initialize
}

// handleCompletePhase handles input for the Complete phase
func (m *Model) handleCompletePhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "r"))):
		// Reset and start over
		m.wizard.Reset()
		m.initPhase(m.wizard.Phase)
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("q"))):
		// Quit (handled by parent)
		return m, nil
	}

	return m, nil
}

// viewCompletePhase renders the Complete phase
func (m *Model) viewCompletePhase() string {
	var b strings.Builder

	state := m.getDeployState()

	// Check if we have deployment results
	if state == nil || state.result == nil {
		b.WriteString(titleStyle.Render("Complete"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("No deployment results available."))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Press [Enter] to start over or [q] to quit"))
		b.WriteString("\n")
		return b.String()
	}

	result := state.result

	if result.Success {
		b.WriteString(successStyle.Render("  Deployment Successful"))
		b.WriteString("\n\n")

		// Show outputs if any
		if len(result.Outputs) > 0 {
			b.WriteString(titleStyle.Render("Outputs"))
			b.WriteString("\n\n")

			for k, value := range result.Outputs {
				b.WriteString(labelStyle.Render("  " + k + ": "))
				b.WriteString(valueStyle.Render(value))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}

		// Show next steps based on target
		b.WriteString(m.viewNextSteps())

		// Show duration
		if result.Duration > 0 {
			b.WriteString(dimStyle.Render("Duration: "))
			b.WriteString(dimStyle.Render(result.Duration.String()))
			b.WriteString("\n\n")
		}

	} else {
		b.WriteString(errorStyle.Render("  Deployment Failed"))
		b.WriteString("\n\n")

		if result.Error != nil {
			b.WriteString(errorStyle.Render("Error: "))
			b.WriteString(valueStyle.Render(result.Error.Error()))
			b.WriteString("\n\n")
		}

		// Show logs if any
		if len(result.Logs) > 0 {
			b.WriteString(dimStyle.Render("Logs:"))
			b.WriteString("\n")
			for _, log := range result.Logs {
				b.WriteString(dimStyle.Render("  " + log))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	// Separator
	b.WriteString(dimStyle.Render(strings.Repeat("-", 40)))
	b.WriteString("\n\n")

	// Actions
	b.WriteString(dimStyle.Render("Press [Enter] or [r] to start over"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Press [Tab] to switch tabs"))
	b.WriteString("\n")

	return b.String()
}

// viewNextSteps renders helpful commands based on the deployment target
func (m *Model) viewNextSteps() string {
	var b strings.Builder

	cmdStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	b.WriteString(titleStyle.Render("Next Steps"))
	b.WriteString("\n\n")

	switch m.wizard.Data.Target {
	case deploy.TargetConfigOnly:
		// Config-only: show cloud-init and manual install commands
		if m.wizard.Data.GenerateOpts.GenerateCloudInit {
			b.WriteString(labelStyle.Render("  Apply cloud-init on existing Ubuntu desktop/server:"))
			b.WriteString("\n\n")

			b.WriteString(dimStyle.Render("  # Step 1: Copy cloud-init.yaml to nocloud seed directory"))
			b.WriteString("\n")
			b.WriteString("  ")
			b.WriteString(cmdStyle.Render("sudo mkdir -p /var/lib/cloud/seed/nocloud"))
			b.WriteString("\n")
			b.WriteString("  ")
			b.WriteString(cmdStyle.Render("sudo cp cloud-init.yaml /var/lib/cloud/seed/nocloud/user-data"))
			b.WriteString("\n")
			b.WriteString("  ")
			b.WriteString(cmdStyle.Render("echo 'instance-id: manual-$(date +%s)' | sudo tee /var/lib/cloud/seed/nocloud/meta-data"))
			b.WriteString("\n\n")

			b.WriteString(dimStyle.Render("  # Step 2: Run cloud-init"))
			b.WriteString("\n")
			b.WriteString("  ")
			b.WriteString(cmdStyle.Render("sudo cloud-init clean --logs"))
			b.WriteString("\n")
			b.WriteString("  ")
			b.WriteString(cmdStyle.Render("sudo cloud-init init --local && sudo cloud-init init"))
			b.WriteString("\n")
			b.WriteString("  ")
			b.WriteString(cmdStyle.Render("sudo cloud-init modules --mode=config"))
			b.WriteString("\n")
			b.WriteString("  ")
			b.WriteString(cmdStyle.Render("sudo cloud-init modules --mode=final"))
			b.WriteString("\n\n")

			b.WriteString(dimStyle.Render("  Note: Reboot after completion for all changes to take effect"))
			b.WriteString("\n\n")
		}

		b.WriteString(labelStyle.Render("  Or run install scripts directly (simpler):"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render("bash scripts/cloud-init/install-all.sh"))
		b.WriteString("\n\n")

		b.WriteString(labelStyle.Render("  Run individual package installers:"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render("./scripts/packages/<package>.sh install"))
		b.WriteString("\n\n")

	case deploy.TargetMultipass:
		// Multipass: show SSH command
		vmName := m.wizard.Data.MultipassOpts.VMName
		if vmName == "" {
			vmName = "<vm-name>"
		}
		b.WriteString(labelStyle.Render("  SSH into the VM:"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render(fmt.Sprintf("multipass shell %s", vmName)))
		b.WriteString("\n\n")

		b.WriteString(labelStyle.Render("  List VMs:"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render("multipass list"))
		b.WriteString("\n\n")

	case deploy.TargetTerragrunt:
		// Terragrunt: show next steps for generated config
		vmName := m.wizard.Data.TerragruntOpts.VMName
		if vmName == "" {
			vmName = "<vm-name>"
		}
		b.WriteString(labelStyle.Render("  Navigate to generated config:"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render(fmt.Sprintf("cd tf/%s", vmName)))
		b.WriteString("\n\n")

		b.WriteString(labelStyle.Render("  Initialize and apply with Terragrunt:"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render("terragrunt init"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render("terragrunt apply"))
		b.WriteString("\n\n")

		b.WriteString(labelStyle.Render("  SSH into the VM (after apply):"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render("ssh ubuntu@<vm-ip>"))
		b.WriteString("\n\n")

		b.WriteString(labelStyle.Render("  Destroy resources:"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render("terragrunt destroy"))
		b.WriteString("\n\n")

	case deploy.TargetUSB:
		// USB/ISO: show how to write to USB
		b.WriteString(labelStyle.Render("  Write ISO to USB (Linux):"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render("sudo dd if=<iso-path> of=/dev/sdX bs=4M status=progress && sync"))
		b.WriteString("\n\n")

		b.WriteString(labelStyle.Render("  Write ISO to USB (macOS):"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(cmdStyle.Render("sudo dd if=<iso-path> of=/dev/rdiskN bs=4m && sync"))
		b.WriteString("\n\n")

		b.WriteString(dimStyle.Render("  Note: Replace /dev/sdX or /dev/rdiskN with your USB device"))
		b.WriteString("\n\n")

	default:
		b.WriteString(dimStyle.Render("  See documentation for next steps."))
		b.WriteString("\n\n")
	}

	return b.String()
}
