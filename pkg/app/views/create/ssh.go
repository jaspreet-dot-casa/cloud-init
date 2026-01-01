package create

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// SSH-specific field indices
const (
	sshFieldGitHubUser = iota
	sshFieldLocalKeys  // Start of local key checkboxes
)

// githubProfile holds fetched GitHub profile data
type githubProfile struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Login string `json:"login"`
}

// noReplyEmail returns the GitHub noreply email address
func (p *githubProfile) noReplyEmail() string {
	if p.ID == 0 || p.Login == "" {
		return ""
	}
	return fmt.Sprintf("%d+%s@users.noreply.github.com", p.ID, p.Login)
}

// bestEmail returns the public email if available, otherwise noreply
func (p *githubProfile) bestEmail() string {
	if p.Email != "" {
		return p.Email
	}
	return p.noReplyEmail()
}

// initSSHPhase initializes the SSH configuration phase
func (m *Model) initSSHPhase() {
	// GitHub username input
	githubUser := textinput.New()
	githubUser.Placeholder = "your-github-username"
	githubUser.CharLimit = 64
	githubUser.Focus()
	m.wizard.TextInputs["github_user"] = githubUser
}

// handleSSHPhase handles input for the SSH configuration phase
func (m *Model) handleSSHPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	// Don't process keys while fetching
	if m.fetchingGitHub {
		return m, nil
	}

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
		m.wizard.FocusedField++
		m.focusCurrentInput()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Save GitHub username
		m.wizard.Data.GitHubUser = m.wizard.GetTextInput("github_user")

		// If GitHub username provided, fetch profile and keys
		if m.wizard.Data.GitHubUser != "" {
			m.fetchingGitHub = true
			m.message = "Fetching data from GitHub..."
			return m, m.fetchGitHubData(m.wizard.Data.GitHubUser)
		}

		// No GitHub username, advance directly
		m.saveSSHOptions()
		m.wizard.Advance()
		m.initPhase(m.wizard.Phase)
		return m, nil
	}

	// Forward to text input
	if m.wizard.FocusedField == sshFieldGitHubUser {
		return m.updateActiveTextInput(msg)
	}

	return m, nil
}

// fetchGitHubData fetches SSH keys and profile from GitHub
func (m *Model) fetchGitHubData(username string) tea.Cmd {
	return func() tea.Msg {
		var keys []string
		var profile *githubProfile
		var keysErr, profileErr error

		// Fetch SSH keys
		keys, keysErr = fetchGitHubSSHKeys(username)

		// Fetch profile
		profile, profileErr = fetchGitHubProfile(username)

		return githubDataMsg{
			keys:       keys,
			profile:    profile,
			keysErr:    keysErr,
			profileErr: profileErr,
		}
	}
}

// githubDataMsg contains fetched GitHub data
type githubDataMsg struct {
	keys       []string
	profile    *githubProfile
	keysErr    error
	profileErr error
}

// fetchGitHubSSHKeys fetches public SSH keys from GitHub
func fetchGitHubSSHKeys(username string) ([]string, error) {
	url := fmt.Sprintf("https://github.com/%s.keys", username)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("GitHub user '%s' not found", username)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub returned status %d", resp.StatusCode)
	}

	// Limit to 1MB to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	keys := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			keys = append(keys, line)
		}
	}

	return keys, nil
}

// fetchGitHubProfile fetches public profile from GitHub API
func fetchGitHubProfile(username string) (*githubProfile, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", username)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("GitHub user '%s' not found", username)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Limit to 1MB to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var profile githubProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	return &profile, nil
}

// saveSSHOptions saves the SSH options to wizard data
func (m *Model) saveSSHOptions() {
	m.wizard.Data.GitHubUser = m.wizard.GetTextInput("github_user")

	// Collect selected SSH keys
	var keys []string
	for key, selected := range m.wizard.SSHKeySelected {
		if selected {
			keys = append(keys, key)
		}
	}
	m.wizard.Data.SSHKeys = keys
}

// viewSSHPhase renders the SSH configuration phase
func (m *Model) viewSSHPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("SSH Key Configuration"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Configure SSH keys for server access."))
	b.WriteString("\n\n")

	// GitHub username
	b.WriteString(m.renderSSHTextField("GitHub Username", "github_user", sshFieldGitHubUser))
	b.WriteString(dimStyle.Render("  Leave empty to skip GitHub SSH key import"))
	b.WriteString("\n\n")

	// TODO: Show local SSH keys if discovered
	// For now, just show a message
	b.WriteString(dimStyle.Render("SSH keys will be fetched from GitHub if username is provided."))
	b.WriteString("\n")

	return b.String()
}

// renderSSHTextField renders a text input field for SSH
func (m *Model) renderSSHTextField(label, name string, fieldIdx int) string {
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
	b.WriteString("\n")

	return b.String()
}
