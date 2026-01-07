package wizard

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
)

// WizardData holds all data collected across phases
type WizardData struct {
	// Target selection
	Target deploy.DeploymentTarget

	// Target-specific options
	MultipassOpts  deploy.MultipassOptions
	TerragruntOpts deploy.TerragruntOptions
	GenerateOpts   GenerateOptions

	// SSH configuration
	SSHKeys       []string
	GitHubUser    string
	LocalSSHKeys  []string
	GitHubSSHKeys []string

	// Git configuration
	GitName  string
	GitEmail string

	// Host details
	DisplayName string
	Username    string
	Hostname    string

	// Package selection
	Packages []string

	// Optional services
	TailscaleKey string
	GitHubPAT    string

	// GitHub profile (fetched from API)
	GitHubID int64
}

// GenerateOptions holds config-only generation options
type GenerateOptions struct {
	GenerateCloudInit bool
	OutputDir         string
}

// State holds the current state of the wizard
type State struct {
	// Current phase
	Phase Phase

	// Collected data
	Data WizardData

	// Package registry
	Registry *packages.Registry

	// Phase-specific state
	TargetSelected int // Index in target list

	// Form field state (like ISO tab pattern)
	FocusedField int
	TextInputs   map[string]textinput.Model
	SelectIdxs   map[string]int
	CheckStates  map[string]bool

	// Multi-select state for packages
	PackageSelected map[string]bool

	// SSH keys multi-select
	SSHKeySelected map[string]bool

	// Deployment state (type is any to allow create package to use its own deployState)
	Deploying   bool
	DeployState any

	// Error state
	LastError error
}

// NewState creates a new wizard state
func NewState() *State {
	return &State{
		Phase:           PhaseTarget,
		Data:            WizardData{},
		TextInputs:      make(map[string]textinput.Model),
		SelectIdxs:      make(map[string]int),
		CheckStates:     make(map[string]bool),
		PackageSelected: make(map[string]bool),
		SSHKeySelected:  make(map[string]bool),
	}
}

// CanGoBack returns true if the user can go back from the current phase
func (s *State) CanGoBack() bool {
	switch s.Phase {
	case PhaseTarget, PhaseDeploy, PhaseComplete:
		return false
	default:
		return true
	}
}

// NextPhase advances to the next phase
func (s *State) NextPhase() Phase {
	switch s.Phase {
	case PhaseTarget:
		return PhaseTargetOptions
	case PhaseTargetOptions:
		return PhaseSSH
	case PhaseSSH:
		return PhaseGit
	case PhaseGit:
		return PhaseHost
	case PhaseHost:
		return PhasePackages
	case PhasePackages:
		return PhaseOptional
	case PhaseOptional:
		return PhaseReview
	case PhaseReview:
		return PhaseDeploy
	case PhaseDeploy:
		return PhaseComplete
	default:
		return PhaseComplete
	}
}

// PrevPhase goes back to the previous phase
func (s *State) PrevPhase() Phase {
	switch s.Phase {
	case PhaseTargetOptions:
		return PhaseTarget
	case PhaseSSH:
		return PhaseTargetOptions
	case PhaseGit:
		return PhaseSSH
	case PhaseHost:
		return PhaseGit
	case PhasePackages:
		return PhaseHost
	case PhaseOptional:
		return PhasePackages
	case PhaseReview:
		return PhaseOptional
	default:
		return s.Phase
	}
}

// Advance moves to the next phase and returns any initialization commands
func (s *State) Advance() tea.Cmd {
	s.Phase = s.NextPhase()
	s.FocusedField = 0
	return nil
}

// GoBack moves to the previous phase
func (s *State) GoBack() tea.Cmd {
	if s.CanGoBack() {
		s.Phase = s.PrevPhase()
		s.FocusedField = 0
	}
	return nil
}

// Reset resets the wizard to the initial state
func (s *State) Reset() {
	s.Phase = PhaseTarget
	s.Data = WizardData{}
	s.FocusedField = 0
	s.TargetSelected = 0
	s.Deploying = false
	s.LastError = nil

	// Clear form state but keep registry
	s.TextInputs = make(map[string]textinput.Model)
	s.SelectIdxs = make(map[string]int)
	s.CheckStates = make(map[string]bool)
	s.PackageSelected = make(map[string]bool)
	s.SSHKeySelected = make(map[string]bool)
}

// PhaseProgress returns the current progress as a fraction
func (s *State) PhaseProgress() float64 {
	total := float64(PhaseComplete)
	current := float64(s.Phase)
	return current / total
}

// InitTextInput initializes a text input field
func (s *State) InitTextInput(name, placeholder string, charLimit int) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	s.TextInputs[name] = ti
}

// GetTextInput gets the value of a text input
func (s *State) GetTextInput(name string) string {
	if ti, ok := s.TextInputs[name]; ok {
		return ti.Value()
	}
	return ""
}

// SetTextInput sets the value of a text input
func (s *State) SetTextInput(name, value string) {
	if ti, ok := s.TextInputs[name]; ok {
		ti.SetValue(value)
		s.TextInputs[name] = ti
	}
}

// GetSelectIdx gets the selected index for a select field
func (s *State) GetSelectIdx(name string) int {
	return s.SelectIdxs[name]
}

// SetSelectIdx sets the selected index for a select field
func (s *State) SetSelectIdx(name string, idx int) {
	s.SelectIdxs[name] = idx
}

// GetCheckState gets the checked state for a checkbox
func (s *State) GetCheckState(name string) bool {
	return s.CheckStates[name]
}

// SetCheckState sets the checked state for a checkbox
func (s *State) SetCheckState(name string, checked bool) {
	s.CheckStates[name] = checked
}

// NavigateField moves focus up/down with bounds checking.
// delta should be -1 for up or +1 for down.
func (s *State) NavigateField(delta int, maxField int) {
	newField := s.FocusedField + delta
	if newField < 0 {
		newField = 0
	} else if newField > maxField {
		newField = maxField
	}
	s.FocusedField = newField
}

// CycleSelect cycles through options with wrap-around.
// delta should be -1 for left or +1 for right.
func (s *State) CycleSelect(name string, optionCount, delta int) {
	if optionCount <= 0 {
		return
	}
	idx := s.SelectIdxs[name] + delta
	if idx < 0 {
		idx = optionCount - 1
	} else if idx >= optionCount {
		idx = 0
	}
	s.SelectIdxs[name] = idx
}
