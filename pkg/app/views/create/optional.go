package create

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// Optional-specific field indices
const (
	optionalFieldTailscale = iota
	optionalFieldGitHubPAT
	optionalFieldCount
)

// initOptionalPhase initializes the Optional services phase
func (m *Model) initOptionalPhase() {
	// Tailscale auth key input
	tailscaleKey := textinput.New()
	tailscaleKey.Placeholder = "tskey-auth-..."
	tailscaleKey.CharLimit = 256
	tailscaleKey.EchoMode = textinput.EchoPassword
	tailscaleKey.Focus()
	m.wizard.TextInputs["tailscale_key"] = tailscaleKey

	// GitHub PAT input
	githubPAT := textinput.New()
	githubPAT.Placeholder = "ghp_..."
	githubPAT.CharLimit = 256
	githubPAT.EchoMode = textinput.EchoPassword
	m.wizard.TextInputs["github_pat"] = githubPAT
}

// handleOptionalPhase handles input for the Optional services phase
func (m *Model) handleOptionalPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
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
		if m.wizard.FocusedField < optionalFieldCount-1 {
			m.wizard.FocusedField++
		}
		m.focusCurrentInput()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// If on last field, advance to next phase
		if m.wizard.FocusedField == optionalFieldCount-1 {
			m.saveOptionalOptions()
			m.wizard.Advance()
			m.initPhase(m.wizard.Phase)
			return m, nil
		}
		// Otherwise, move to next field
		m.blurCurrentInput()
		m.wizard.FocusedField++
		m.focusCurrentInput()
		return m, nil
	}

	// Forward to text input
	return m.updateActiveTextInput(msg)
}

// saveOptionalOptions saves the Optional options to wizard data
func (m *Model) saveOptionalOptions() {
	m.wizard.Data.TailscaleKey = m.wizard.GetTextInput("tailscale_key")
	m.wizard.Data.GitHubPAT = m.wizard.GetTextInput("github_pat")
}

// viewOptionalPhase renders the Optional services phase
func (m *Model) viewOptionalPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Optional Services"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Configure optional services. Leave empty to skip."))
	b.WriteString("\n\n")

	// Tailscale auth key
	b.WriteString(RenderTextField(m.wizard, "Tailscale Auth Key", "tailscale_key", optionalFieldTailscale))
	b.WriteString(dimStyle.Render("  Used for automatic Tailscale authentication"))
	b.WriteString("\n\n")

	// GitHub PAT
	b.WriteString(RenderTextField(m.wizard, "GitHub PAT", "github_pat", optionalFieldGitHubPAT))
	b.WriteString(dimStyle.Render("  Personal Access Token for private repos"))
	b.WriteString("\n")

	return b.String()
}
