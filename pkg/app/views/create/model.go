// Package create provides the Create VM view for the TUI application.
// This implements a multi-phase wizard for creating VMs or generating configs.
package create

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
)

// CreateCompleteMsg signals that the create wizard completed
type CreateCompleteMsg struct {
	Success bool
	Error   error
}

// Model is the Create VM view model with embedded wizard
type Model struct {
	app.BaseTab

	projectDir string
	store      *settings.Store
	wizard     *WizardState
	message    string

	// For loading packages
	loadingPackages bool
	packagesLoaded  bool

	// For fetching GitHub data
	fetchingGitHub bool

	// Available cloud images from settings
	cloudImages []settings.CloudImage
}

// New creates a new Create VM model
func New(projectDir string, store *settings.Store) *Model {
	m := &Model{
		BaseTab:    app.NewBaseTab(app.TabCreate, "Create", "2"),
		projectDir: projectDir,
		store:      store,
		wizard:     NewWizardState(),
	}

	// Load cloud images from settings
	if store != nil {
		if s, err := store.Load(); err == nil {
			m.cloudImages = s.CloudImages
		}
	}

	return m
}

// Init initializes the create view
func (m *Model) Init() tea.Cmd {
	// Packages are loaded lazily in Focus() when tab becomes active
	return nil
}

// loadPackages loads the package registry from embedded scripts
func (m *Model) loadPackages() tea.Cmd {
	m.loadingPackages = true
	return func() tea.Msg {
		registry, err := packages.DiscoverEmbedded()
		return packagesLoadedMsg{registry: registry, err: err}
	}
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case packagesLoadedMsg:
		m.loadingPackages = false
		m.packagesLoaded = true
		if msg.err != nil {
			m.message = "Error loading packages: " + msg.err.Error()
		} else {
			m.wizard.Registry = msg.registry
			// Initialize all packages as selected by default
			for _, name := range msg.registry.Names() {
				m.wizard.PackageSelected[name] = true
			}
		}
		return m, nil

	case CreateCompleteMsg:
		m.wizard.Deploying = false
		if msg.Success {
			m.message = "Deployment complete! Press [1] to view VMs."
			m.wizard.Phase = PhaseComplete
		} else if msg.Error != nil {
			m.message = "Error: " + msg.Error.Error()
		}
		return m, nil

	case githubDataMsg:
		m.fetchingGitHub = false
		m.message = ""

		// Store SSH keys from GitHub
		if msg.keysErr != nil {
			m.message = "Warning: " + msg.keysErr.Error()
		} else if len(msg.keys) > 0 {
			m.wizard.Data.GitHubSSHKeys = msg.keys
			// Select all GitHub keys by default
			for _, key := range msg.keys {
				m.wizard.SSHKeySelected[key] = true
			}
		}

		// Store profile data for Git phase
		if msg.profile != nil {
			if msg.profile.Name != "" {
				m.wizard.Data.GitName = msg.profile.Name
			}
			if email := msg.profile.bestEmail(); email != "" {
				m.wizard.Data.GitEmail = email
			}
			m.wizard.Data.GitHubID = msg.profile.ID
		}

		// Now advance to next phase
		m.saveSSHOptions()
		m.wizard.Advance()
		m.initPhase(m.wizard.Phase)
		return m, nil

	case deployProgressMsg, deployCompleteMsg, spinner.TickMsg, progress.FrameMsg:
		// Route deploy messages to the deploy phase handler
		if m.wizard.Phase == PhaseDeploy {
			return m.handleDeployPhase(msg)
		}
		return m, nil
	}

	return m, nil
}

// updateActiveTextInput updates the currently focused text input
func (m *Model) updateActiveTextInput(msg tea.Msg) (app.Tab, tea.Cmd) {
	// Get the active text input name based on current phase and field
	inputName := m.getActiveInputName()
	if inputName == "" {
		return m, nil
	}

	if ti, ok := m.wizard.TextInputs[inputName]; ok {
		var cmd tea.Cmd
		ti, cmd = ti.Update(msg)
		m.wizard.TextInputs[inputName] = ti
		return m, cmd
	}
	return m, nil
}

// getActiveInputName returns the name of the currently active text input
func (m *Model) getActiveInputName() string {
	switch m.wizard.Phase {
	case PhaseTargetOptions:
		return m.getTargetOptionsInputName()
	case PhaseSSH:
		return "github_user"
	case PhaseGit:
		switch m.wizard.FocusedField {
		case 0:
			return "git_name"
		case 1:
			return "git_email"
		}
	case PhaseHost:
		switch m.wizard.FocusedField {
		case 0:
			return "display_name"
		case 1:
			return "username"
		case 2:
			return "hostname"
		}
	case PhaseOptional:
		switch m.wizard.FocusedField {
		case 0:
			return "tailscale_key"
		case 1:
			return "github_pat"
		}
	}
	return ""
}

// getTargetOptionsInputName returns the input name for target options phase
func (m *Model) getTargetOptionsInputName() string {
	switch m.wizard.Data.Target {
	case deploy.TargetMultipass:
		if m.wizard.FocusedField == 0 {
			return "vm_name"
		}
	case deploy.TargetTerraform:
		if m.wizard.FocusedField == 0 {
			return "vm_name"
		}
	case deploy.TargetUSB:
		switch m.wizard.FocusedField {
		case 0:
			return "source_iso"
		case 1:
			return "output_path"
		}
	case TargetConfigOnly:
		if m.wizard.FocusedField == 0 {
			return "output_dir"
		}
	}
	return ""
}

// handleKeyMsg handles keyboard input based on current phase
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	// Global escape to go back (if possible)
	if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
		if m.wizard.CanGoBack() {
			m.wizard.GoBack()
			return m, nil
		}
	}

	// Handle based on current phase
	switch m.wizard.Phase {
	case PhaseTarget:
		cmd, advance := m.handleTargetPhase(msg)
		if advance {
			m.wizard.Advance()
			m.initPhase(m.wizard.Phase)
		}
		return m, cmd

	case PhaseTargetOptions:
		return m.handleTargetOptionsPhase(msg)

	case PhaseSSH:
		return m.handleSSHPhase(msg)

	case PhaseGit:
		return m.handleGitPhase(msg)

	case PhaseHost:
		return m.handleHostPhase(msg)

	case PhasePackages:
		return m.handlePackagesPhase(msg)

	case PhaseOptional:
		return m.handleOptionalPhase(msg)

	case PhaseReview:
		return m.handleReviewPhase(msg)

	case PhaseDeploy:
		// Route key messages to deploy phase handler
		return m.handleDeployPhase(msg)

	case PhaseComplete:
		return m.handleCompletePhase(msg)
	}

	return m, nil
}

// initPhase initializes a new phase
func (m *Model) initPhase(phase Phase) {
	m.wizard.FocusedField = 0

	switch phase {
	case PhaseTargetOptions:
		m.initTargetOptionsPhase()
	case PhaseSSH:
		m.initSSHPhase()
	case PhaseGit:
		m.initGitPhase()
	case PhaseHost:
		m.initHostPhase()
	case PhaseOptional:
		m.initOptionalPhase()
	case PhaseReview:
		m.initReviewPhase()
	case PhaseDeploy:
		m.initDeployPhase()
	case PhaseComplete:
		m.initCompletePhase()
	}
}

// View renders the create view
func (m *Model) View() string {
	if m.Width() == 0 {
		return "Loading..."
	}

	if m.loadingPackages {
		return "Loading packages..."
	}

	var b strings.Builder

	// Phase indicator
	b.WriteString(m.viewPhaseIndicator())
	b.WriteString("\n")

	// Phase content
	switch m.wizard.Phase {
	case PhaseTarget:
		b.WriteString(m.viewTargetPhase())
	case PhaseTargetOptions:
		b.WriteString(m.viewTargetOptionsPhase())
	case PhaseSSH:
		b.WriteString(m.viewSSHPhase())
	case PhaseGit:
		b.WriteString(m.viewGitPhase())
	case PhaseHost:
		b.WriteString(m.viewHostPhase())
	case PhasePackages:
		b.WriteString(m.viewPackagesPhase())
	case PhaseOptional:
		b.WriteString(m.viewOptionalPhase())
	case PhaseReview:
		b.WriteString(m.viewReviewPhase())
	case PhaseDeploy:
		b.WriteString(m.viewDeployPhase())
	case PhaseComplete:
		b.WriteString(m.viewCompletePhase())
	}

	// Message
	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(m.message))
	}

	return b.String()
}

// viewPhaseIndicator renders the phase progress indicator
func (m *Model) viewPhaseIndicator() string {
	var b strings.Builder

	// Current phase name
	b.WriteString(phaseIndicatorStyle.Render(m.wizard.Phase.String()))

	// Progress dots
	b.WriteString("  ")
	for i := 0; i <= int(PhaseComplete); i++ {
		if i == int(m.wizard.Phase) {
			b.WriteString(activeStyle.Render("●"))
		} else if i < int(m.wizard.Phase) {
			b.WriteString(successStyle.Render("●"))
		} else {
			b.WriteString(dimStyle.Render("○"))
		}
		if i < int(PhaseComplete) {
			b.WriteString(" ")
		}
	}

	return b.String()
}

// Focus sets focus on this tab
func (m *Model) Focus() tea.Cmd {
	m.BaseTab.Focus()
	m.message = ""
	// Focus the current text input if applicable
	m.focusCurrentInput()

	// Load packages on first focus (lazy loading)
	if !m.packagesLoaded && !m.loadingPackages {
		return m.loadPackages()
	}
	return nil
}

// focusCurrentInput focuses the current text input
func (m *Model) focusCurrentInput() {
	inputName := m.getActiveInputName()
	if inputName != "" {
		if ti, ok := m.wizard.TextInputs[inputName]; ok {
			ti.Focus()
			m.wizard.TextInputs[inputName] = ti
		}
	}
}

// blurCurrentInput blurs the current text input
func (m *Model) blurCurrentInput() {
	inputName := m.getActiveInputName()
	if inputName != "" {
		if ti, ok := m.wizard.TextInputs[inputName]; ok {
			ti.Blur()
			m.wizard.TextInputs[inputName] = ti
		}
	}
}

// Blur removes focus from this tab
func (m *Model) Blur() {
	m.BaseTab.Blur()
	m.blurCurrentInput()
}

// SetSize sets the tab dimensions
func (m *Model) SetSize(width, height int) {
	m.BaseTab.SetSize(width, height)
}

// KeyBindings returns the key bindings for this tab
func (m *Model) KeyBindings() []string {
	switch m.wizard.Phase {
	case PhaseTarget:
		return m.targetKeyBindings()
	case PhasePackages:
		return []string{"[↑/↓] navigate", "[Space] toggle", "[Enter] continue", "[Esc] back"}
	case PhaseReview:
		return []string{"[Enter] deploy", "[Esc] back"}
	case PhaseDeploy:
		return []string{"Deploying..."}
	case PhaseComplete:
		return []string{"[Enter] new", "[1] view VMs"}
	default:
		bindings := []string{"[↑/↓] navigate", "[Enter] continue"}
		if m.wizard.CanGoBack() {
			bindings = append(bindings, "[Esc] back")
		}
		return bindings
	}
}

// ProjectDir returns the project directory
func (m *Model) ProjectDir() string {
	return m.projectDir
}

// SelectedTarget returns the currently selected deployment target
func (m *Model) SelectedTarget() deploy.DeploymentTarget {
	return m.wizard.Data.Target
}

// HasFocusedInput returns true if a text input is currently focused
func (m *Model) HasFocusedInput() bool {
	// Check if we're in a phase with text inputs
	switch m.wizard.Phase {
	case PhaseTargetOptions, PhaseSSH, PhaseGit, PhaseHost, PhaseOptional:
		inputName := m.getActiveInputName()
		if inputName != "" {
			if ti, ok := m.wizard.TextInputs[inputName]; ok {
				return ti.Focused()
			}
		}
	}
	return false
}
