package phases

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// Ensure TargetPhase implements PhaseHandler
var _ wizard.PhaseHandler = (*TargetPhase)(nil)

// TargetPhase handles the deployment target selection step.
type TargetPhase struct {
	wizard.BasePhase
}

// NewTargetPhase creates a new TargetPhase.
func NewTargetPhase() *TargetPhase {
	return &TargetPhase{
		BasePhase: wizard.NewBasePhase("Select Target", len(Targets)),
	}
}

// TargetItem represents a deployment target option.
type TargetItem struct {
	Target      deploy.DeploymentTarget
	Name        string
	Description string
	Icon        string
}

// Targets is the list of available deployment targets.
var Targets = []TargetItem{
	{
		Target:      deploy.TargetTerragrunt,
		Name:        "Terragrunt/libvirt",
		Description: "Generate Terragrunt config for libvirt VM (run manually)",
		Icon:        "🖥️ ",
	},
	{
		Target:      deploy.TargetMultipass,
		Name:        "Multipass",
		Description: "Create VM using Canonical Multipass",
		Icon:        "☁️ ",
	},
	{
		Target:      deploy.TargetUSB,
		Name:        "Bootable USB",
		Description: "Create bootable USB installer with autoinstall",
		Icon:        "💾",
	},
	{
		Target:      deploy.TargetConfigOnly,
		Name:        "Generate Config",
		Description: "Generate config files only (no deployment)",
		Icon:        "📄",
	},
}

// Init initializes the target phase state.
func (p *TargetPhase) Init(ctx *wizard.PhaseContext) {
	// Target selection starts at 0 (Terraform)
	ctx.Wizard.TargetSelected = 0
}

// Update handles keyboard input for the target phase.
func (p *TargetPhase) Update(ctx *wizard.PhaseContext, msg tea.KeyMsg) (advance bool, cmd tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if ctx.Wizard.TargetSelected > 0 {
			ctx.Wizard.TargetSelected--
		}
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if ctx.Wizard.TargetSelected < len(Targets)-1 {
			ctx.Wizard.TargetSelected++
		}
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return true, nil
	}

	return false, nil
}

// View renders the target phase.
func (p *TargetPhase) View(ctx *wizard.PhaseContext) string {
	var b strings.Builder

	b.WriteString(wizard.TitleStyle.Render("Select Deployment Target"))
	b.WriteString("\n\n")

	b.WriteString(wizard.DimStyle.Render("Choose how to deploy your configuration:"))
	b.WriteString("\n\n")

	for i, target := range Targets {
		cursor := "  "
		style := wizard.UnselectedStyle

		if i == ctx.Wizard.TargetSelected {
			cursor = "▸ "
			style = wizard.SelectedStyle
		}

		b.WriteString(cursor)
		b.WriteString(target.Icon)
		b.WriteString(" ")
		b.WriteString(style.Render(target.Name))
		b.WriteString("\n")

		descStyle := wizard.DimStyle.MarginLeft(5)
		b.WriteString(descStyle.Render(target.Description))
		b.WriteString("\n\n")
	}

	return b.String()
}

// Save persists the target selection to wizard data.
func (p *TargetPhase) Save(ctx *wizard.PhaseContext) {
	if ctx.Wizard.TargetSelected >= 0 && ctx.Wizard.TargetSelected < len(Targets) {
		ctx.Wizard.Data.Target = Targets[ctx.Wizard.TargetSelected].Target
	}
}
