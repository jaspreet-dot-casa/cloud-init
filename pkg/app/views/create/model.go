// Package create provides the Create VM view for the TUI application.
package create

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// CreateCompleteMsg signals that the create wizard completed
type CreateCompleteMsg struct {
	Success bool
	Error   error
}

// Model is the Create VM view model
type Model struct {
	app.BaseTab

	projectDir string
	targets    []targetItem
	selected   int
	launching  bool
	message    string
}

type targetItem struct {
	target      deploy.DeploymentTarget
	name        string
	description string
	icon        string
}

// New creates a new Create VM model
func New(projectDir string) *Model {
	return &Model{
		BaseTab:    app.NewBaseTab(app.TabCreate, "Create", "2"),
		projectDir: projectDir,
		targets: []targetItem{
			{
				target:      deploy.TargetTerraform,
				name:        "Terraform/libvirt",
				description: "Create VM using Terraform with libvirt provider",
				icon:        "🖥️ ",
			},
			{
				target:      deploy.TargetMultipass,
				name:        "Multipass",
				description: "Create VM using Canonical Multipass",
				icon:        "☁️ ",
			},
			{
				target:      deploy.TargetUSB,
				name:        "Bootable USB",
				description: "Create bootable USB installer",
				icon:        "💾",
			},
		},
		selected: 0,
	}
}

// Init initializes the create view
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.launching {
			return m, nil
		}
		return m.handleKeyMsg(msg)

	case CreateCompleteMsg:
		m.launching = false
		if msg.Success {
			m.message = "VM created successfully! Press [1] to view VMs."
		} else if msg.Error != nil {
			m.message = fmt.Sprintf("Error: %v", msg.Error)
		}
		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles keyboard input
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.selected > 0 {
			m.selected--
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.selected < len(m.targets)-1 {
			m.selected++
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		m.launching = true
		m.message = ""
		return m, m.launchCreate()
	}

	return m, nil
}

// launchCreate launches the create wizard
func (m *Model) launchCreate() tea.Cmd {
	target := m.targets[m.selected].target
	projectDir := m.projectDir
	return func() tea.Msg {
		return app.RunCreateMsg{Target: target, ProjectDir: projectDir}
	}
}

// View renders the create view
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

	b.WriteString(titleStyle.Render("Create New VM"))
	b.WriteString("\n\n")

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	b.WriteString(descStyle.Render("Select a deployment target:"))
	b.WriteString("\n\n")

	// Target list
	for i, target := range m.targets {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

		if i == m.selected {
			cursor = "▸ "
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Bold(true)
		}

		nameStyle := style
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginLeft(4)

		b.WriteString(cursor)
		b.WriteString(target.icon)
		b.WriteString(" ")
		b.WriteString(nameStyle.Render(target.name))
		b.WriteString("\n")
		b.WriteString(descStyle.Render(target.description))
		b.WriteString("\n\n")
	}

	// Message
	if m.message != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			MarginTop(1)
		b.WriteString(msgStyle.Render(m.message))
		b.WriteString("\n")
	}

	// Instructions
	if m.launching {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("Launching create wizard..."))
	} else {
		b.WriteString("\n")
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		b.WriteString(helpStyle.Render("Press Enter to start the create wizard"))
	}

	return b.String()
}

// Focus sets focus on this tab
func (m *Model) Focus() tea.Cmd {
	m.BaseTab.Focus()
	m.message = ""
	return nil
}

// Blur removes focus from this tab
func (m *Model) Blur() {
	m.BaseTab.Blur()
}

// SetSize sets the tab dimensions
func (m *Model) SetSize(width, height int) {
	m.BaseTab.SetSize(width, height)
}

// KeyBindings returns the key bindings for this tab
func (m *Model) KeyBindings() []string {
	return []string{
		"[↑/↓] navigate",
		"[Enter] create",
	}
}

// ProjectDir returns the project directory
func (m *Model) ProjectDir() string {
	return m.projectDir
}

// SelectedTarget returns the currently selected deployment target
func (m *Model) SelectedTarget() deploy.DeploymentTarget {
	if m.selected >= 0 && m.selected < len(m.targets) {
		return m.targets[m.selected].target
	}
	return deploy.TargetTerraform
}
