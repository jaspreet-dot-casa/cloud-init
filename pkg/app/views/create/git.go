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

// Git-specific field indices
const (
	gitFieldName = iota
	gitFieldEmail
	gitFieldCount
)

// initGitPhase initializes the Git configuration phase
func (m *Model) initGitPhase() {
	// Git name input - pre-fill from GitHub if available
	gitName := textinput.New()
	gitName.Placeholder = "Your Name"
	gitName.CharLimit = 128
	if m.wizard.Data.GitName != "" {
		gitName.SetValue(m.wizard.Data.GitName)
	}
	gitName.Focus()
	m.wizard.TextInputs["git_name"] = gitName

	// Git email input - pre-fill from GitHub if available
	gitEmail := textinput.New()
	gitEmail.Placeholder = "you@example.com"
	gitEmail.CharLimit = 128
	if m.wizard.Data.GitEmail != "" {
		gitEmail.SetValue(m.wizard.Data.GitEmail)
	}
	m.wizard.TextInputs["git_email"] = gitEmail
}

// handleGitPhase handles input for the Git configuration phase
func (m *Model) handleGitPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
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
		if m.wizard.FocusedField < gitFieldCount-1 {
			m.wizard.FocusedField++
		}
		m.focusCurrentInput()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// If on last field, advance to next phase
		if m.wizard.FocusedField == gitFieldCount-1 {
			m.saveGitOptions()
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

// saveGitOptions saves the Git options to wizard data
func (m *Model) saveGitOptions() {
	m.wizard.Data.GitName = m.wizard.GetTextInput("git_name")
	m.wizard.Data.GitEmail = m.wizard.GetTextInput("git_email")
}

// viewGitPhase renders the Git configuration phase
func (m *Model) viewGitPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Git Configuration"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Configure git user settings for commits."))
	b.WriteString("\n\n")

	// Git name
	b.WriteString(RenderTextField(m.wizard, "Name", "git_name", gitFieldName))

	// Git email
	b.WriteString(RenderTextField(m.wizard, "Email", "git_email", gitFieldEmail))

	return b.String()
}
