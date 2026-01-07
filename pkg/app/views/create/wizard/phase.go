// Package wizard provides shared types for the create wizard.
package wizard

// Phase represents the current step in the create wizard
type Phase int

const (
	// PhaseTarget - Select deployment target (Terraform/Multipass/Generate)
	PhaseTarget Phase = iota
	// PhaseTargetOptions - Target-specific options (VM name, CPU, memory, etc.)
	PhaseTargetOptions
	// PhaseSSH - SSH key configuration
	PhaseSSH
	// PhaseGit - Git configuration (name, email)
	PhaseGit
	// PhaseHost - Host details (username, hostname, display name)
	PhaseHost
	// PhasePackages - Package selection
	PhasePackages
	// PhaseOptional - Optional services (Tailscale, GitHub PAT)
	PhaseOptional
	// PhaseReview - Review and confirm
	PhaseReview
	// PhaseDeploy - Deployment in progress
	PhaseDeploy
	// PhaseComplete - Deployment complete, show results
	PhaseComplete
)

// String returns the display name of the phase
func (p Phase) String() string {
	switch p {
	case PhaseTarget:
		return "Select Target"
	case PhaseTargetOptions:
		return "Target Options"
	case PhaseSSH:
		return "SSH Keys"
	case PhaseGit:
		return "Git Config"
	case PhaseHost:
		return "Host Details"
	case PhasePackages:
		return "Packages"
	case PhaseOptional:
		return "Optional Services"
	case PhaseReview:
		return "Review"
	case PhaseDeploy:
		return "Deploying"
	case PhaseComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// IsConfigPhase returns true if this phase collects configuration data
func (p Phase) IsConfigPhase() bool {
	return p >= PhaseSSH && p <= PhaseOptional
}

// TotalPhases returns the total number of phases for progress display
func TotalPhases() int {
	return int(PhaseComplete) + 1
}
