package phases

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
)

// Optional-specific field indices
const (
	optionalFieldTailscale = iota
	optionalFieldGitHubPAT
	optionalFieldCount
)

// Ensure OptionalPhase implements PhaseHandler
var _ wizard.PhaseHandler = (*OptionalPhase)(nil)

// OptionalPhase handles the optional services configuration step.
type OptionalPhase struct {
	wizard.BasePhase
}

// NewOptionalPhase creates a new OptionalPhase.
func NewOptionalPhase() *OptionalPhase {
	return &OptionalPhase{
		BasePhase: wizard.NewBasePhase("Optional Services", optionalFieldCount),
	}
}

// Init initializes the optional phase state.
func (p *OptionalPhase) Init(ctx *wizard.PhaseContext) {
	// Tailscale auth key input
	tailscaleKey := textinput.New()
	tailscaleKey.Placeholder = "tskey-auth-..."
	tailscaleKey.CharLimit = 256
	tailscaleKey.EchoMode = textinput.EchoPassword
	tailscaleKey.Focus()
	ctx.Wizard.TextInputs["tailscale_key"] = tailscaleKey

	// GitHub PAT input
	githubPAT := textinput.New()
	githubPAT.Placeholder = "ghp_..."
	githubPAT.CharLimit = 256
	githubPAT.EchoMode = textinput.EchoPassword
	ctx.Wizard.TextInputs["github_pat"] = githubPAT

	ctx.Wizard.FocusedField = 0
}

// Update handles keyboard input for the optional phase.
func (p *OptionalPhase) Update(ctx *wizard.PhaseContext, msg tea.KeyMsg) (advance bool, cmd tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		p.blurCurrentInput(ctx)
		ctx.Wizard.NavigateField(-1, optionalFieldCount-1)
		p.focusCurrentInput(ctx)
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j", "tab"))):
		p.blurCurrentInput(ctx)
		ctx.Wizard.NavigateField(1, optionalFieldCount-1)
		p.focusCurrentInput(ctx)
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if ctx.Wizard.FocusedField == optionalFieldCount-1 {
			return true, nil
		}
		p.blurCurrentInput(ctx)
		ctx.Wizard.FocusedField++
		p.focusCurrentInput(ctx)
		return false, nil
	}

	return false, p.updateActiveTextInput(ctx, msg)
}

// View renders the optional phase.
func (p *OptionalPhase) View(ctx *wizard.PhaseContext) string {
	var b strings.Builder

	b.WriteString(wizard.TitleStyle.Render("Optional Services"))
	b.WriteString("\n\n")

	b.WriteString(wizard.DimStyle.Render("Configure optional services. Leave empty to skip."))
	b.WriteString("\n\n")

	// Tailscale auth key
	b.WriteString(wizard.RenderTextField(ctx.Wizard, "Tailscale Auth Key", "tailscale_key", optionalFieldTailscale))
	b.WriteString(wizard.DimStyle.Render("  Used for automatic Tailscale authentication"))
	b.WriteString("\n\n")

	// GitHub PAT
	b.WriteString(wizard.RenderTextField(ctx.Wizard, "GitHub PAT", "github_pat", optionalFieldGitHubPAT))
	b.WriteString(wizard.DimStyle.Render("  Personal Access Token for private repos"))
	b.WriteString("\n")

	return b.String()
}

// Save persists the optional options to wizard data.
func (p *OptionalPhase) Save(ctx *wizard.PhaseContext) {
	ctx.Wizard.Data.TailscaleKey = ctx.Wizard.GetTextInput("tailscale_key")
	ctx.Wizard.Data.GitHubPAT = ctx.Wizard.GetTextInput("github_pat")
}

// Helper methods

func (p *OptionalPhase) blurCurrentInput(ctx *wizard.PhaseContext) {
	name := p.getInputName(ctx.Wizard.FocusedField)
	wizard.BlurInput(ctx, name)
}

func (p *OptionalPhase) focusCurrentInput(ctx *wizard.PhaseContext) {
	name := p.getInputName(ctx.Wizard.FocusedField)
	wizard.FocusInput(ctx, name)
}

func (p *OptionalPhase) updateActiveTextInput(ctx *wizard.PhaseContext, msg tea.KeyMsg) tea.Cmd {
	name := p.getInputName(ctx.Wizard.FocusedField)
	return wizard.HandleTextInput(ctx, name, msg)
}

func (p *OptionalPhase) getInputName(field int) string {
	switch field {
	case optionalFieldTailscale:
		return "tailscale_key"
	case optionalFieldGitHubPAT:
		return "github_pat"
	default:
		return ""
	}
}
