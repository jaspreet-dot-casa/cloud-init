package phases

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create"
)

// Git-specific field indices
const (
	gitFieldName = iota
	gitFieldEmail
	gitFieldCount
)

// Ensure GitPhase implements PhaseHandler
var _ create.PhaseHandler = (*GitPhase)(nil)

// GitPhase handles the git configuration step of the wizard.
type GitPhase struct {
	create.BasePhase
}

// NewGitPhase creates a new GitPhase.
func NewGitPhase() *GitPhase {
	return &GitPhase{
		BasePhase: create.NewBasePhase("Git Config", gitFieldCount),
	}
}

// Init initializes the git phase state.
func (p *GitPhase) Init(ctx *create.PhaseContext) {
	// Git name input - pre-fill from wizard data if available
	gitName := textinput.New()
	gitName.Placeholder = "Your Name"
	gitName.CharLimit = 128
	if ctx.Wizard.Data.GitName != "" {
		gitName.SetValue(ctx.Wizard.Data.GitName)
	}
	gitName.Focus()
	ctx.Wizard.TextInputs["git_name"] = gitName

	// Git email input - pre-fill from wizard data if available
	gitEmail := textinput.New()
	gitEmail.Placeholder = "you@example.com"
	gitEmail.CharLimit = 128
	if ctx.Wizard.Data.GitEmail != "" {
		gitEmail.SetValue(ctx.Wizard.Data.GitEmail)
	}
	ctx.Wizard.TextInputs["git_email"] = gitEmail

	ctx.Wizard.FocusedField = 0
}

// Update handles keyboard input for the git phase.
func (p *GitPhase) Update(ctx *create.PhaseContext, msg tea.KeyMsg) (advance bool, cmd tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		p.blurCurrentInput(ctx)
		ctx.Wizard.NavigateField(-1, gitFieldCount-1)
		p.focusCurrentInput(ctx)
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j", "tab"))):
		p.blurCurrentInput(ctx)
		ctx.Wizard.NavigateField(1, gitFieldCount-1)
		p.focusCurrentInput(ctx)
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if ctx.Wizard.FocusedField == gitFieldCount-1 {
			return true, nil
		}
		p.blurCurrentInput(ctx)
		ctx.Wizard.FocusedField++
		p.focusCurrentInput(ctx)
		return false, nil
	}

	return false, p.updateActiveTextInput(ctx, msg)
}

// View renders the git phase.
func (p *GitPhase) View(ctx *create.PhaseContext) string {
	var b strings.Builder

	b.WriteString(create.TitleStyle.Render("Git Configuration"))
	b.WriteString("\n\n")

	b.WriteString(create.DimStyle.Render("Configure git user settings for commits."))
	b.WriteString("\n\n")

	b.WriteString(create.RenderTextField(ctx.Wizard, "Name", "git_name", gitFieldName))
	b.WriteString(create.RenderTextField(ctx.Wizard, "Email", "git_email", gitFieldEmail))

	return b.String()
}

// Save persists the git options to wizard data.
func (p *GitPhase) Save(ctx *create.PhaseContext) {
	ctx.Wizard.Data.GitName = ctx.Wizard.GetTextInput("git_name")
	ctx.Wizard.Data.GitEmail = ctx.Wizard.GetTextInput("git_email")
}

// Helper methods

func (p *GitPhase) blurCurrentInput(ctx *create.PhaseContext) {
	name := p.getInputName(ctx.Wizard.FocusedField)
	create.BlurInput(ctx, name)
}

func (p *GitPhase) focusCurrentInput(ctx *create.PhaseContext) {
	name := p.getInputName(ctx.Wizard.FocusedField)
	create.FocusInput(ctx, name)
}

func (p *GitPhase) updateActiveTextInput(ctx *create.PhaseContext, msg tea.KeyMsg) tea.Cmd {
	name := p.getInputName(ctx.Wizard.FocusedField)
	return create.HandleTextInput(ctx, name, msg)
}

func (p *GitPhase) getInputName(field int) string {
	switch field {
	case gitFieldName:
		return "git_name"
	case gitFieldEmail:
		return "git_email"
	default:
		return ""
	}
}
