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

// Generate-specific field indices
const (
	generateFieldOutputDir = iota
	generateFieldCloudInit
	generateFieldCount
)

// initGeneratePhase initializes the Generate Config options phase
func (m *Model) initGeneratePhase() {
	// Output directory input
	outputDir := textinput.New()
	outputDir.Placeholder = "."
	outputDir.SetValue(".")
	outputDir.CharLimit = 256
	outputDir.Focus()
	m.wizard.TextInputs["output_dir"] = outputDir

	// Generate cloud-init checkbox (default true)
	m.wizard.CheckStates["generate_cloudinit"] = true
}

// handleGeneratePhase handles input for the Generate Config options phase
func (m *Model) handleGeneratePhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
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
		if m.wizard.FocusedField < generateFieldCount-1 {
			m.wizard.FocusedField++
		}
		m.focusCurrentInput()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
		// Toggle checkbox
		if m.wizard.FocusedField == generateFieldCloudInit {
			m.wizard.CheckStates["generate_cloudinit"] = !m.wizard.CheckStates["generate_cloudinit"]
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Validate and advance
		m.saveGenerateOptions()
		m.wizard.Advance()
		m.initPhase(m.wizard.Phase)
		return m, nil
	}

	// Forward to text input for text fields
	if m.wizard.FocusedField == generateFieldOutputDir {
		return m.updateActiveTextInput(msg)
	}

	return m, nil
}

// saveGenerateOptions saves the Generate options to wizard data
func (m *Model) saveGenerateOptions() {
	outputDir := m.wizard.GetTextInput("output_dir")
	if outputDir == "" {
		outputDir = "."
	}

	m.wizard.Data.GenerateOpts = GenerateOptions{
		OutputDir:         outputDir,
		GenerateCloudInit: m.wizard.CheckStates["generate_cloudinit"],
	}
}

// viewGeneratePhase renders the Generate Config options phase
func (m *Model) viewGeneratePhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Generate Config Options"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Generate configuration files without deploying."))
	b.WriteString("\n\n")

	// Output directory
	b.WriteString(m.renderGenerateTextField("Output Directory", "output_dir", generateFieldOutputDir))

	// Generate cloud-init checkbox
	b.WriteString(m.renderCheckbox("Generate cloud-init.yaml", "generate_cloudinit", generateFieldCloudInit))

	// Info about what will be generated
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Files that will be generated:"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • config.env - Package enables, git settings"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • cloud-init/secrets.env - SSH keys, auth tokens"))
	b.WriteString("\n")
	if m.wizard.CheckStates["generate_cloudinit"] {
		b.WriteString(dimStyle.Render("  • cloud-init/cloud-init.yaml - Cloud-init configuration"))
		b.WriteString("\n")
	}

	return b.String()
}

// renderGenerateTextField renders a text input field for Generate
func (m *Model) renderGenerateTextField(label, name string, fieldIdx int) string {
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
