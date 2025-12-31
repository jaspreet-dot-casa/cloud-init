// Package iso provides the ISO builder view for the TUI application.
package iso

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	isobuilder "github.com/jaspreet-dot-casa/cloud-init/pkg/iso"
)

// BuildISOMsg signals that ISO build should start
type BuildISOMsg struct {
	Options *isobuilder.ISOOptions
}

// BuildCompleteMsg signals that ISO build completed
type BuildCompleteMsg struct {
	Success    bool
	OutputPath string
	Error      error
}

// field represents a form field
type field int

const (
	fieldSourceISO field = iota
	fieldOutputPath
	fieldVersion
	fieldStorage
	fieldTimezone
	fieldSubmit
)

// Model is the ISO builder view model
type Model struct {
	app.BaseTab

	projectDir string

	// Form fields
	sourceInput   textinput.Model
	outputInput   textinput.Model
	versionIdx    int
	storageIdx    int
	timezoneInput textinput.Model

	focusedField field
	building     bool
	message      string
	messageStyle lipgloss.Style
}

var (
	versions = []string{"24.04", "22.04"}
	storages = []string{"lvm", "direct", "zfs"}
)

// New creates a new ISO builder model
func New(projectDir string) *Model {
	// Source ISO input
	sourceInput := textinput.New()
	sourceInput.Placeholder = "/path/to/ubuntu-24.04-live-server-amd64.iso"
	sourceInput.Focus()
	sourceInput.Width = 50

	// Output path input
	outputInput := textinput.New()
	outputInput.Placeholder = "ubuntu-autoinstall.iso (optional)"
	outputInput.Width = 50

	// Timezone input
	timezoneInput := textinput.New()
	timezoneInput.Placeholder = "UTC"
	timezoneInput.Width = 30

	return &Model{
		BaseTab:       app.NewBaseTab(app.TabISO, "ISO", "3"),
		projectDir:    projectDir,
		sourceInput:   sourceInput,
		outputInput:   outputInput,
		timezoneInput: timezoneInput,
		focusedField:  fieldSourceISO,
		messageStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	}
}

// Init initializes the ISO builder view
func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.building {
			return m, nil
		}
		return m.handleKeyMsg(msg)

	case BuildCompleteMsg:
		m.building = false
		if msg.Success {
			m.message = fmt.Sprintf("ISO created: %s", msg.OutputPath)
			m.messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		} else if msg.Error != nil {
			m.message = fmt.Sprintf("Error: %v", msg.Error)
			m.messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		}
		return m, nil
	}

	// Update focused text input
	var cmd tea.Cmd
	switch m.focusedField {
	case fieldSourceISO:
		m.sourceInput, cmd = m.sourceInput.Update(msg)
		cmds = append(cmds, cmd)
	case fieldOutputPath:
		m.outputInput, cmd = m.outputInput.Update(msg)
		cmds = append(cmds, cmd)
	case fieldTimezone:
		m.timezoneInput, cmd = m.timezoneInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles keyboard input
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("tab", "down", "j"))):
		m.nextField()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab", "up", "k"))):
		m.prevField()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		if m.focusedField == fieldVersion {
			if m.versionIdx > 0 {
				m.versionIdx--
			}
		} else if m.focusedField == fieldStorage {
			if m.storageIdx > 0 {
				m.storageIdx--
			}
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
		if m.focusedField == fieldVersion {
			if m.versionIdx < len(versions)-1 {
				m.versionIdx++
			}
		} else if m.focusedField == fieldStorage {
			if m.storageIdx < len(storages)-1 {
				m.storageIdx++
			}
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if m.focusedField == fieldSubmit {
			return m, m.buildISO()
		}
		m.nextField()
		return m, nil
	}

	// Pass through to text input
	var cmd tea.Cmd
	switch m.focusedField {
	case fieldSourceISO:
		m.sourceInput, cmd = m.sourceInput.Update(msg)
	case fieldOutputPath:
		m.outputInput, cmd = m.outputInput.Update(msg)
	case fieldTimezone:
		m.timezoneInput, cmd = m.timezoneInput.Update(msg)
	}

	return m, cmd
}

func (m *Model) nextField() {
	m.blurCurrent()
	m.focusedField++
	if m.focusedField > fieldSubmit {
		m.focusedField = fieldSourceISO
	}
	m.focusCurrent()
}

func (m *Model) prevField() {
	m.blurCurrent()
	if m.focusedField == fieldSourceISO {
		m.focusedField = fieldSubmit
	} else {
		m.focusedField--
	}
	m.focusCurrent()
}

func (m *Model) blurCurrent() {
	switch m.focusedField {
	case fieldSourceISO:
		m.sourceInput.Blur()
	case fieldOutputPath:
		m.outputInput.Blur()
	case fieldTimezone:
		m.timezoneInput.Blur()
	}
}

func (m *Model) focusCurrent() {
	switch m.focusedField {
	case fieldSourceISO:
		m.sourceInput.Focus()
	case fieldOutputPath:
		m.outputInput.Focus()
	case fieldTimezone:
		m.timezoneInput.Focus()
	}
}

// buildISO starts the ISO build process
func (m *Model) buildISO() tea.Cmd {
	opts := &isobuilder.ISOOptions{
		SourceISO:     m.sourceInput.Value(),
		OutputPath:    m.outputInput.Value(),
		UbuntuVersion: versions[m.versionIdx],
		StorageLayout: isobuilder.StorageLayout(storages[m.storageIdx]),
		Timezone:      m.timezoneInput.Value(),
	}

	// Validate before launching
	if err := opts.Validate(); err != nil {
		m.message = fmt.Sprintf("Validation error: %v", err)
		m.messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		return nil
	}

	m.building = true
	m.message = "Building ISO..."
	m.messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return func() tea.Msg {
		return BuildISOMsg{Options: opts}
	}
}

// View renders the ISO builder view
func (m *Model) View() string {
	if m.Width() == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("ISO Builder"))
	b.WriteString("\n\n")

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	b.WriteString(descStyle.Render("Create a bootable Ubuntu autoinstall ISO"))
	b.WriteString("\n\n")

	// Form fields
	labelStyle := lipgloss.NewStyle().Width(20)
	focusedLabelStyle := labelStyle.Foreground(lipgloss.Color("229")).Bold(true)
	selectStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	focusedSelectStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)

	// Source ISO
	label := labelStyle
	if m.focusedField == fieldSourceISO {
		label = focusedLabelStyle
	}
	b.WriteString(label.Render("Source ISO:"))
	b.WriteString(m.sourceInput.View())
	b.WriteString("\n\n")

	// Output path
	label = labelStyle
	if m.focusedField == fieldOutputPath {
		label = focusedLabelStyle
	}
	b.WriteString(label.Render("Output path:"))
	b.WriteString(m.outputInput.View())
	b.WriteString("\n\n")

	// Ubuntu version (select)
	label = labelStyle
	if m.focusedField == fieldVersion {
		label = focusedLabelStyle
	}
	b.WriteString(label.Render("Ubuntu version:"))
	for i, v := range versions {
		style := selectStyle
		if m.focusedField == fieldVersion && i == m.versionIdx {
			style = focusedSelectStyle
		}
		if i == m.versionIdx {
			b.WriteString(style.Render(fmt.Sprintf("[%s]", v)))
		} else {
			b.WriteString(style.Render(fmt.Sprintf(" %s ", v)))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	// Storage layout (select)
	label = labelStyle
	if m.focusedField == fieldStorage {
		label = focusedLabelStyle
	}
	b.WriteString(label.Render("Storage layout:"))
	for i, s := range storages {
		style := selectStyle
		if m.focusedField == fieldStorage && i == m.storageIdx {
			style = focusedSelectStyle
		}
		if i == m.storageIdx {
			b.WriteString(style.Render(fmt.Sprintf("[%s]", s)))
		} else {
			b.WriteString(style.Render(fmt.Sprintf(" %s ", s)))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	// Timezone
	label = labelStyle
	if m.focusedField == fieldTimezone {
		label = focusedLabelStyle
	}
	b.WriteString(label.Render("Timezone:"))
	b.WriteString(m.timezoneInput.View())
	b.WriteString("\n\n")

	// Submit button
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252"))
	if m.focusedField == fieldSubmit {
		buttonStyle = buttonStyle.
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("229")).
			Bold(true)
	}
	b.WriteString("                    ")
	b.WriteString(buttonStyle.Render("Build ISO"))
	b.WriteString("\n\n")

	// Message
	if m.message != "" {
		b.WriteString(m.messageStyle.Render(m.message))
		b.WriteString("\n")
	}

	// Instructions
	if !m.building {
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Tab: next field • ←/→: change selection • Enter: submit"))
	}

	return b.String()
}

// Focus sets focus on this tab
func (m *Model) Focus() tea.Cmd {
	m.BaseTab.Focus()
	m.message = ""
	m.focusedField = fieldSourceISO
	m.sourceInput.Focus()
	return textinput.Blink
}

// Blur removes focus from this tab
func (m *Model) Blur() {
	m.BaseTab.Blur()
	m.blurCurrent()
}

// SetSize sets the tab dimensions
func (m *Model) SetSize(width, height int) {
	m.BaseTab.SetSize(width, height)
	// Adjust input widths based on available space
	inputWidth := width - 25
	if inputWidth < 30 {
		inputWidth = 30
	}
	if inputWidth > 60 {
		inputWidth = 60
	}
	m.sourceInput.Width = inputWidth
	m.outputInput.Width = inputWidth
}

// KeyBindings returns the key bindings for this tab
func (m *Model) KeyBindings() []string {
	return []string{
		"[Tab] next",
		"[←/→] select",
		"[Enter] build",
	}
}

// GetOptions returns the current ISO options
func (m *Model) GetOptions() *isobuilder.ISOOptions {
	tz := m.timezoneInput.Value()
	if tz == "" {
		tz = "UTC"
	}
	return &isobuilder.ISOOptions{
		SourceISO:     m.sourceInput.Value(),
		OutputPath:    m.outputInput.Value(),
		UbuntuVersion: versions[m.versionIdx],
		StorageLayout: isobuilder.StorageLayout(storages[m.storageIdx]),
		Timezone:      tz,
	}
}

// HasFocusedInput returns true if a text input is currently focused
func (m *Model) HasFocusedInput() bool {
	switch m.focusedField {
	case fieldSourceISO:
		return m.sourceInput.Focused()
	case fieldOutputPath:
		return m.outputInput.Focused()
	case fieldTimezone:
		return m.timezoneInput.Focused()
	}
	return false
}
