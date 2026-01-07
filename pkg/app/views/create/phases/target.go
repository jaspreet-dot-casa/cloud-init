package phases

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/utils"
)

// Ensure TargetPhase implements PhaseHandler
var _ wizard.PhaseHandler = (*TargetPhase)(nil)

// TargetPhase handles the deployment target selection step.
type TargetPhase struct {
	wizard.BasePhase

	// State for loading saved configs
	showingConfigs  bool
	savedConfigs    []settings.VMConfig
	configSelected  int
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
		Target:      deploy.TargetConfigOnly,
		Name:        "Generate Config",
		Description: "Generate config files only (no deployment)",
		Icon:        "📄",
	},
}

// Init initializes the target phase state.
func (p *TargetPhase) Init(ctx *wizard.PhaseContext) {
	// Target selection starts at 0 (Terragrunt)
	ctx.Wizard.TargetSelected = 0
	p.showingConfigs = false
	p.configSelected = 0

	// Load saved configs from store
	p.refreshConfigs(ctx)
}

// refreshConfigs reloads configs from the store.
func (p *TargetPhase) refreshConfigs(ctx *wizard.PhaseContext) {
	if ctx.Store == nil {
		return
	}

	s, err := ctx.Store.Load()
	if err != nil {
		if ctx.Message != nil {
			*ctx.Message = fmt.Sprintf("Warning: could not load saved configs: %v", err)
		}
		p.savedConfigs = nil
		return
	}
	p.savedConfigs = s.VMConfigs
}

// Update handles keyboard input for the target phase.
func (p *TargetPhase) Update(ctx *wizard.PhaseContext, msg tea.KeyMsg) (advance bool, cmd tea.Cmd) {
	// Handle config picker mode
	if p.showingConfigs {
		return p.updateConfigPicker(ctx, msg)
	}

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

	case key.Matches(msg, key.NewBinding(key.WithKeys("l"))):
		// Refresh and show config picker if there are saved configs
		p.refreshConfigs(ctx)
		if len(p.savedConfigs) > 0 {
			p.showingConfigs = true
			p.configSelected = 0
		} else if ctx.Message != nil {
			*ctx.Message = "No saved configs available"
		}
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return true, nil
	}

	return false, nil
}

// updateConfigPicker handles input when showing the config picker.
func (p *TargetPhase) updateConfigPicker(ctx *wizard.PhaseContext, msg tea.KeyMsg) (advance bool, cmd tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if p.configSelected > 0 {
			p.configSelected--
		}
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if p.configSelected < len(p.savedConfigs)-1 {
			p.configSelected++
		}
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		p.showingConfigs = false
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Load the selected config
		if p.configSelected >= 0 && p.configSelected < len(p.savedConfigs) {
			cfg := &p.savedConfigs[p.configSelected]

			if err := wizard.LoadFromConfig(cfg, ctx.Wizard); err != nil {
				if ctx.Message != nil {
					*ctx.Message = fmt.Sprintf("Failed to load config: %v", err)
				}
				p.showingConfigs = false
				return false, nil
			}

			// Update last used timestamp using atomic LoadAndSave
			if ctx.Store != nil {
				cfgID := cfg.ID
				if err := ctx.Store.LoadAndSave(func(s *settings.Settings) error {
					s.UpdateVMConfigLastUsed(cfgID)
					return nil
				}); err != nil && ctx.Message != nil {
					*ctx.Message = fmt.Sprintf("Warning: failed to update last used time: %v", err)
				}
			}

			p.showingConfigs = false

			// Skip to review phase
			ctx.Wizard.Phase = wizard.PhaseReview
			if ctx.Message != nil {
				*ctx.Message = fmt.Sprintf("Loaded config: %s", cfg.Name)
			}
		}
		return false, nil
	}

	return false, nil
}

// View renders the target phase.
func (p *TargetPhase) View(ctx *wizard.PhaseContext) string {
	if p.showingConfigs {
		return p.viewConfigPicker(ctx)
	}

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

	// Show hint about loading configs if available
	if len(p.savedConfigs) > 0 {
		b.WriteString(wizard.DimStyle.Render(strings.Repeat("-", 40)))
		b.WriteString("\n")
		b.WriteString(wizard.DimStyle.Render(fmt.Sprintf("[l] load saved config (%d available)", len(p.savedConfigs))))
		b.WriteString("\n")
	}

	return b.String()
}

// viewConfigPicker renders the saved config picker.
func (p *TargetPhase) viewConfigPicker(ctx *wizard.PhaseContext) string {
	var b strings.Builder

	b.WriteString(wizard.TitleStyle.Render("Load Saved Config"))
	b.WriteString("\n\n")

	b.WriteString(wizard.DimStyle.Render("Select a saved configuration to load:"))
	b.WriteString("\n\n")

	for i, cfg := range p.savedConfigs {
		cursor := "  "
		style := wizard.UnselectedStyle

		if i == p.configSelected {
			cursor = "▸ "
			style = wizard.SelectedStyle
		}

		b.WriteString(cursor)
		b.WriteString(style.Render(cfg.Name))
		b.WriteString(" ")
		b.WriteString(wizard.DimStyle.Render(fmt.Sprintf("(%s)", cfg.Target)))
		b.WriteString("\n")

		// Show description and last used
		if cfg.Description != "" {
			b.WriteString("     ")
			b.WriteString(wizard.DimStyle.Render(cfg.Description))
			b.WriteString("\n")
		}

		if !cfg.LastUsedAt.IsZero() {
			b.WriteString("     ")
			b.WriteString(wizard.DimStyle.Render("Last used: " + utils.FormatTimeAgo(cfg.LastUsedAt)))
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	b.WriteString(wizard.DimStyle.Render("[Enter] load  [Esc] back"))
	b.WriteString("\n")

	return b.String()
}

// Save persists the target selection to wizard data.
func (p *TargetPhase) Save(ctx *wizard.PhaseContext) {
	if ctx.Wizard.TargetSelected >= 0 && ctx.Wizard.TargetSelected < len(Targets) {
		ctx.Wizard.Data.Target = Targets[ctx.Wizard.TargetSelected].Target
	}
}
