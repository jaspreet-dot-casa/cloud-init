package create

// Step represents a wizard step in the create flow.
type Step int

const (
	StepLoading Step = iota
	StepSSHSource
	StepSSHKeys
	StepGitConfig
	StepHostDetails
	StepPackages
	StepOptional
	StepTarget
	StepTargetOptions
	StepReview
	StepDeploy
	StepComplete
)

// String returns the string representation of the step.
func (s Step) String() string {
	switch s {
	case StepLoading:
		return "loading"
	case StepSSHSource:
		return "ssh-source"
	case StepSSHKeys:
		return "ssh-keys"
	case StepGitConfig:
		return "git-config"
	case StepHostDetails:
		return "host-details"
	case StepPackages:
		return "packages"
	case StepOptional:
		return "optional"
	case StepTarget:
		return "target"
	case StepTargetOptions:
		return "target-options"
	case StepReview:
		return "review"
	case StepDeploy:
		return "deploy"
	case StepComplete:
		return "complete"
	default:
		return "unknown"
	}
}

// Title returns the display title for the step.
func (s Step) Title() string {
	switch s {
	case StepLoading:
		return "Loading"
	case StepSSHSource:
		return "SSH Key Source"
	case StepSSHKeys:
		return "SSH Keys"
	case StepGitConfig:
		return "Git Configuration"
	case StepHostDetails:
		return "Host Details"
	case StepPackages:
		return "Package Selection"
	case StepOptional:
		return "Optional Services"
	case StepTarget:
		return "Deployment Target"
	case StepTargetOptions:
		return "Target Options"
	case StepReview:
		return "Review Configuration"
	case StepDeploy:
		return "Deploying"
	case StepComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// IsFormStep returns true if this step displays a form.
func (s Step) IsFormStep() bool {
	switch s {
	case StepSSHSource, StepSSHKeys, StepGitConfig, StepHostDetails,
		StepPackages, StepOptional, StepTarget, StepTargetOptions:
		return true
	default:
		return false
	}
}

// CanGoBack returns true if the user can navigate back from this step.
func (s Step) CanGoBack() bool {
	switch s {
	case StepLoading, StepSSHSource, StepDeploy, StepComplete:
		return false
	default:
		return true
	}
}

// StepCount returns the total number of form steps for progress display.
func StepCount() int {
	return int(StepReview) // Steps 1-9 (SSHSource through Review)
}

// StepNumber returns the 1-based step number for progress display.
func (s Step) StepNumber() int {
	if s <= StepLoading {
		return 0
	}
	if s > StepReview {
		return StepCount()
	}
	return int(s)
}
