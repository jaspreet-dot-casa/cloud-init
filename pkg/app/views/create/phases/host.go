// Package phases provides concrete Phase implementations for the wizard.
package phases

import (
	"os/user"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create"
)

// Host-specific field indices
const (
	hostFieldDisplayName = iota
	hostFieldUsername
	hostFieldHostname
	hostFieldCount
)

// Ensure HostPhase implements PhaseHandler
var _ create.PhaseHandler = (*HostPhase)(nil)

// HostPhase handles the host configuration step of the wizard.
type HostPhase struct {
	create.BasePhase
}

// NewHostPhase creates a new HostPhase.
func NewHostPhase() *HostPhase {
	return &HostPhase{
		BasePhase: create.NewBasePhase("Host Details", hostFieldCount),
	}
}

// Init initializes the host phase state.
func (p *HostPhase) Init(ctx *create.PhaseContext) {
	// Get current username as default
	currentUser := "ubuntu"
	if u, err := user.Current(); err == nil {
		currentUser = u.Username
	}

	// Display name input (defaults to git name if set)
	displayName := textinput.New()
	displayName.Placeholder = "Your Display Name"
	if ctx.Wizard.Data.GitName != "" {
		displayName.SetValue(ctx.Wizard.Data.GitName)
	}
	displayName.CharLimit = 128
	displayName.Focus()
	ctx.Wizard.TextInputs["display_name"] = displayName

	// Username input
	username := textinput.New()
	username.Placeholder = currentUser
	username.SetValue(currentUser)
	username.CharLimit = 32
	ctx.Wizard.TextInputs["username"] = username

	// Hostname input
	hostname := textinput.New()
	hostname.Placeholder = "ubuntu-server"
	hostname.SetValue("ubuntu-server")
	hostname.CharLimit = 64
	ctx.Wizard.TextInputs["hostname"] = hostname

	ctx.Wizard.FocusedField = 0
}

// Update handles keyboard input for the host phase.
func (p *HostPhase) Update(ctx *create.PhaseContext, msg tea.KeyMsg) (advance bool, cmd tea.Cmd) {
	// Handle navigation
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		p.blurCurrentInput(ctx)
		ctx.Wizard.NavigateField(-1, hostFieldCount-1)
		p.focusCurrentInput(ctx)
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j", "tab"))):
		p.blurCurrentInput(ctx)
		ctx.Wizard.NavigateField(1, hostFieldCount-1)
		p.focusCurrentInput(ctx)
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// If on last field, advance to next phase
		if ctx.Wizard.FocusedField == hostFieldCount-1 {
			return true, nil
		}
		// Otherwise, move to next field
		p.blurCurrentInput(ctx)
		ctx.Wizard.FocusedField++
		p.focusCurrentInput(ctx)
		return false, nil
	}

	// Forward to text input
	return false, p.updateActiveTextInput(ctx, msg)
}

// View renders the host phase.
func (p *HostPhase) View(ctx *create.PhaseContext) string {
	var b strings.Builder

	b.WriteString(create.TitleStyle.Render("Host Details"))
	b.WriteString("\n\n")

	b.WriteString(create.DimStyle.Render("Configure the target system."))
	b.WriteString("\n\n")

	// Display name
	b.WriteString(create.RenderTextField(ctx.Wizard, "Display Name", "display_name", hostFieldDisplayName))

	// Username
	b.WriteString(create.RenderTextField(ctx.Wizard, "Username", "username", hostFieldUsername))

	// Hostname
	b.WriteString(create.RenderTextField(ctx.Wizard, "Hostname", "hostname", hostFieldHostname))

	return b.String()
}

// Save persists the host options to wizard data.
func (p *HostPhase) Save(ctx *create.PhaseContext) {
	ctx.Wizard.Data.DisplayName = ctx.Wizard.GetTextInput("display_name")
	ctx.Wizard.Data.Username = ctx.Wizard.GetTextInput("username")
	ctx.Wizard.Data.Hostname = ctx.Wizard.GetTextInput("hostname")

	// Apply defaults if empty
	if ctx.Wizard.Data.Username == "" {
		ctx.Wizard.Data.Username = "ubuntu"
	}
	if ctx.Wizard.Data.Hostname == "" {
		ctx.Wizard.Data.Hostname = "ubuntu-server"
	}
}

// Helper methods

func (p *HostPhase) blurCurrentInput(ctx *create.PhaseContext) {
	name := p.getInputName(ctx.Wizard.FocusedField)
	create.BlurInput(ctx, name)
}

func (p *HostPhase) focusCurrentInput(ctx *create.PhaseContext) {
	name := p.getInputName(ctx.Wizard.FocusedField)
	create.FocusInput(ctx, name)
}

func (p *HostPhase) updateActiveTextInput(ctx *create.PhaseContext, msg tea.KeyMsg) tea.Cmd {
	name := p.getInputName(ctx.Wizard.FocusedField)
	return create.HandleTextInput(ctx, name, msg)
}

func (p *HostPhase) getInputName(field int) string {
	switch field {
	case hostFieldDisplayName:
		return "display_name"
	case hostFieldUsername:
		return "username"
	case hostFieldHostname:
		return "hostname"
	default:
		return ""
	}
}
