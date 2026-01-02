package create

import (
	"os/user"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// Host-specific field indices
const (
	hostFieldDisplayName = iota
	hostFieldUsername
	hostFieldHostname
	hostFieldCount
)

// initHostPhase initializes the Host configuration phase
func (m *Model) initHostPhase() {
	// Get current username as default
	currentUser := "ubuntu"
	if u, err := user.Current(); err == nil {
		currentUser = u.Username
	}

	// Display name input (defaults to git name if set)
	displayName := textinput.New()
	displayName.Placeholder = "Your Display Name"
	if m.wizard.Data.GitName != "" {
		displayName.SetValue(m.wizard.Data.GitName)
	}
	displayName.CharLimit = 128
	displayName.Focus()
	m.wizard.TextInputs["display_name"] = displayName

	// Username input
	username := textinput.New()
	username.Placeholder = currentUser
	username.SetValue(currentUser)
	username.CharLimit = 32
	m.wizard.TextInputs["username"] = username

	// Hostname input
	hostname := textinput.New()
	hostname.Placeholder = "ubuntu-server"
	hostname.SetValue("ubuntu-server")
	hostname.CharLimit = 64
	m.wizard.TextInputs["hostname"] = hostname
}

// handleHostPhase handles input for the Host configuration phase
func (m *Model) handleHostPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
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
		if m.wizard.FocusedField < hostFieldCount-1 {
			m.wizard.FocusedField++
		}
		m.focusCurrentInput()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// If on last field, advance to next phase
		if m.wizard.FocusedField == hostFieldCount-1 {
			m.saveHostOptions()
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

// saveHostOptions saves the Host options to wizard data
func (m *Model) saveHostOptions() {
	m.wizard.Data.DisplayName = m.wizard.GetTextInput("display_name")
	m.wizard.Data.Username = m.wizard.GetTextInput("username")
	m.wizard.Data.Hostname = m.wizard.GetTextInput("hostname")

	// Apply defaults if empty
	if m.wizard.Data.Username == "" {
		m.wizard.Data.Username = "ubuntu"
	}
	if m.wizard.Data.Hostname == "" {
		m.wizard.Data.Hostname = "ubuntu-server"
	}
}

// viewHostPhase renders the Host configuration phase
func (m *Model) viewHostPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Host Details"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Configure the target system."))
	b.WriteString("\n\n")

	// Display name
	b.WriteString(m.renderHostTextField("Display Name", "display_name", hostFieldDisplayName))

	// Username
	b.WriteString(m.renderHostTextField("Username", "username", hostFieldUsername))

	// Hostname
	b.WriteString(m.renderHostTextField("Hostname", "hostname", hostFieldHostname))

	return b.String()
}

// renderHostTextField renders a text input field for Host
func (m *Model) renderHostTextField(label, name string, fieldIdx int) string {
	var b strings.Builder

	focused := m.wizard.FocusedField == fieldIdx
	cursor := "  "
	if focused {
		cursor = "▸ "
	}

	b.WriteString(cursor)
	if focused {
		b.WriteString(focusedInputStyle.Render(label + ": "))
	} else {
		b.WriteString(labelStyle.Render(label + ": "))
	}

	if ti, ok := m.wizard.TextInputs[name]; ok {
		b.WriteString(ti.View())
	}
	b.WriteString("\n\n")

	return b.String()
}
