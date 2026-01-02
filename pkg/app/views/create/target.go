package create

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// TargetConfigOnly is an alias to deploy.TargetConfigOnly for convenience
const TargetConfigOnly = deploy.TargetConfigOnly

// targetItem represents a deployment target option
type targetItem struct {
	target      deploy.DeploymentTarget
	name        string
	description string
	icon        string
}

// targets is the list of available deployment targets
var targets = []targetItem{
	{
		target:      deploy.TargetTerraform,
		name:        "Terraform/libvirt",
		description: "Create VM using Terraform with libvirt provider",
		icon:        "🖥️ ",
	},
	{
		target:      deploy.TargetMultipass,
		name:        "Multipass",
		description: "Create VM using Canonical Multipass",
		icon:        "☁️ ",
	},
	{
		target:      deploy.TargetUSB,
		name:        "Bootable USB",
		description: "Create bootable USB installer with autoinstall",
		icon:        "💾",
	},
	{
		target:      TargetConfigOnly,
		name:        "Generate Config",
		description: "Generate config files only (no deployment)",
		icon:        "📄",
	},
}

// handleTargetPhase handles the target selection phase
func (m *Model) handleTargetPhase(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.wizard.TargetSelected > 0 {
			m.wizard.TargetSelected--
		}
		return nil, false

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.wizard.TargetSelected < len(targets)-1 {
			m.wizard.TargetSelected++
		}
		return nil, false

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Save selected target
		m.wizard.Data.Target = targets[m.wizard.TargetSelected].target
		return nil, true // Advance to next phase
	}

	return nil, false
}

// viewTargetPhase renders the target selection phase
func (m *Model) viewTargetPhase() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Select Deployment Target"))
	b.WriteString("\n\n")

	// Description
	b.WriteString(dimStyle.Render("Choose how to deploy your configuration:"))
	b.WriteString("\n\n")

	// Target list
	for i, target := range targets {
		cursor := "  "
		style := unselectedStyle

		if i == m.wizard.TargetSelected {
			cursor = "▸ "
			style = selectedStyle
		}

		// Target line
		b.WriteString(cursor)
		b.WriteString(target.icon)
		b.WriteString(" ")
		b.WriteString(style.Render(target.name))
		b.WriteString("\n")

		// Description
		descStyle := dimStyle.Copy().MarginLeft(5)
		b.WriteString(descStyle.Render(target.description))
		b.WriteString("\n\n")
	}

	return b.String()
}

// targetKeyBindings returns key bindings for the target selection phase
func (m *Model) targetKeyBindings() []string {
	return []string{
		"[↑/↓] navigate",
		"[Enter] select",
	}
}
