package phases

import "github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"

// Registry holds all phase handlers indexed by Phase enum.
// This allows the Model to delegate to the appropriate handler.
type Registry struct {
	handlers map[wizard.Phase]wizard.PhaseHandler
}

// NewRegistry creates a new phase registry with all phases registered.
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[wizard.Phase]wizard.PhaseHandler),
	}

	// Register all phases
	r.Register(wizard.PhaseTarget, NewTargetPhase())
	r.Register(wizard.PhaseGit, NewGitPhase())
	r.Register(wizard.PhaseHost, NewHostPhase())
	r.Register(wizard.PhasePackages, NewPackagesPhase())
	r.Register(wizard.PhaseOptional, NewOptionalPhase())

	// TODO: Register remaining phases as they are converted:
	// r.Register(wizard.PhaseTargetOptions, NewTargetOptionsPhase())
	// r.Register(wizard.PhaseSSH, NewSSHPhase())
	// r.Register(wizard.PhaseReview, NewReviewPhase())
	// r.Register(wizard.PhaseDeploy, NewDeployPhase())
	// r.Register(wizard.PhaseComplete, NewCompletePhase())

	return r
}

// Register adds a phase handler to the registry.
func (r *Registry) Register(phase wizard.Phase, handler wizard.PhaseHandler) {
	r.handlers[phase] = handler
}

// Get returns the handler for a phase, or nil if not found.
func (r *Registry) Get(phase wizard.Phase) wizard.PhaseHandler {
	return r.handlers[phase]
}

// Has returns true if a handler exists for the given phase.
func (r *Registry) Has(phase wizard.Phase) bool {
	_, ok := r.handlers[phase]
	return ok
}
