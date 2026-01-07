package create

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/utils"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// Review-specific field indices
const (
	reviewFieldConfirm = iota
	reviewFieldSaveConfig
	reviewFieldCancel
	reviewFieldCount
)

// Review phase state keys for wizard.CheckStates
const (
	reviewStateShowingSaveDialog = "review_showing_save_dialog"
	reviewStateConfigNameInput   = "config_name"
)

// initReviewPhase initializes the Review phase
func (m *Model) initReviewPhase() {
	// Reset focus to confirm button
	m.wizard.FocusedField = reviewFieldConfirm

	// Initialize save config input if needed
	if _, ok := m.wizard.TextInputs[reviewStateConfigNameInput]; !ok {
		ti := textinput.New()
		ti.Placeholder = "my-config"
		ti.CharLimit = utils.MaxConfigNameLength
		m.wizard.TextInputs[reviewStateConfigNameInput] = ti
	}

	// Reset the save dialog state
	m.wizard.CheckStates[reviewStateShowingSaveDialog] = false
}

// handleReviewPhase handles input for the Review phase
func (m *Model) handleReviewPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	// Handle save dialog if showing
	if m.wizard.CheckStates[reviewStateShowingSaveDialog] {
		return m.handleSaveConfigDialog(msg)
	}

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
		switch m.wizard.FocusedField {
		case reviewFieldConfirm:
			// Start deployment
			m.wizard.Advance()
			m.initPhase(m.wizard.Phase)
			return m, m.startDeploy()

		case reviewFieldSaveConfig:
			// Show save config dialog
			m.wizard.CheckStates[reviewStateShowingSaveDialog] = true
			ti := m.wizard.TextInputs[reviewStateConfigNameInput]
			ti.SetValue("")
			ti.Focus()
			m.wizard.TextInputs[reviewStateConfigNameInput] = ti
			return m, nil

		case reviewFieldCancel:
			// Go back to target selection
			m.wizard.Phase = wizard.PhaseTarget
			m.initPhase(m.wizard.Phase)
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

// handleSaveConfigDialog handles input in the save config dialog
func (m *Model) handleSaveConfigDialog(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		// Cancel dialog - reset input state
		m.wizard.CheckStates[reviewStateShowingSaveDialog] = false
		ti := m.wizard.TextInputs[reviewStateConfigNameInput]
		ti.SetValue("") // Clear the input so it doesn't persist
		ti.Blur()
		m.wizard.TextInputs[reviewStateConfigNameInput] = ti
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Save config
		ti := m.wizard.TextInputs[reviewStateConfigNameInput]
		name := utils.SanitizeConfigName(ti.Value())

		// Validate config name
		if err := utils.ValidateConfigName(name); err != nil {
			m.message = err.Error()
			return m, nil
		}

		// Create and save the config using atomic LoadAndSave
		if m.store != nil {
			cfg := wizard.ToVMConfig(&m.wizard.Data, name, "")
			err := m.store.LoadAndSave(func(s *settings.Settings) error {
				s.AddVMConfig(cfg)
				return nil
			})
			if err != nil {
				m.message = fmt.Sprintf("Failed to save config: %v", err)
			} else {
				m.message = fmt.Sprintf("Saved config: %s", name)
			}
		} else {
			m.message = "Settings store not available"
		}

		// Close dialog
		m.wizard.CheckStates[reviewStateShowingSaveDialog] = false
		ti.Blur()
		m.wizard.TextInputs[reviewStateConfigNameInput] = ti
		return m, nil

	default:
		// Forward to text input
		ti := m.wizard.TextInputs[reviewStateConfigNameInput]
		var cmd tea.Cmd
		ti, cmd = ti.Update(msg)
		m.wizard.TextInputs[reviewStateConfigNameInput] = ti
		return m, cmd
	}
}

// viewReviewPhase renders the Review phase
func (m *Model) viewReviewPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Review Configuration"))
	b.WriteString("\n\n")

	// Show save dialog if active
	if m.wizard.CheckStates[reviewStateShowingSaveDialog] {
		return m.viewSaveConfigDialog()
	}

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
	saveFocused := m.wizard.FocusedField == reviewFieldSaveConfig
	cancelFocused := m.wizard.FocusedField == reviewFieldCancel

	confirmCursor := "  "
	saveCursor := "  "
	cancelCursor := "  "

	if confirmFocused {
		confirmCursor = "▸ "
	}
	if saveFocused {
		saveCursor = "▸ "
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

	b.WriteString(saveCursor)
	if saveFocused {
		b.WriteString(focusedInputStyle.Render("[Save as Config]"))
	} else {
		b.WriteString(labelStyle.Render("[Save as Config]"))
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

// viewSaveConfigDialog renders the save config dialog
func (m *Model) viewSaveConfigDialog() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Save Configuration"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Enter a name for this configuration:"))
	b.WriteString("\n\n")

	ti := m.wizard.TextInputs[reviewStateConfigNameInput]
	b.WriteString(labelStyle.Render("Name: "))
	b.WriteString(ti.View())
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("[Enter] save  [Esc] cancel"))
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
