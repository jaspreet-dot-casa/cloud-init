package phases

import "github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create"

// Registry holds all phase handlers indexed by Phase enum.
// This allows the Model to delegate to the appropriate handler.
type Registry struct {
	handlers map[create.Phase]create.PhaseHandler
}

// NewRegistry creates a new phase registry with all phases registered.
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[create.Phase]create.PhaseHandler),
	}

	// Register all phases
	r.Register(create.PhaseTarget, NewTargetPhase())
	r.Register(create.PhaseGit, NewGitPhase())
	r.Register(create.PhaseHost, NewHostPhase())
	r.Register(create.PhasePackages, NewPackagesPhase())
	r.Register(create.PhaseOptional, NewOptionalPhase())

	// TODO: Register remaining phases as they are converted:
	// r.Register(create.PhaseTargetOptions, NewTargetOptionsPhase())
	// r.Register(create.PhaseSSH, NewSSHPhase())
	// r.Register(create.PhaseReview, NewReviewPhase())
	// r.Register(create.PhaseDeploy, NewDeployPhase())
	// r.Register(create.PhaseComplete, NewCompletePhase())

	return r
}

// Register adds a phase handler to the registry.
func (r *Registry) Register(phase create.Phase, handler create.PhaseHandler) {
	r.handlers[phase] = handler
}

// Get returns the handler for a phase, or nil if not found.
func (r *Registry) Get(phase create.Phase) create.PhaseHandler {
	return r.handlers[phase]
}

// Has returns true if a handler exists for the given phase.
func (r *Registry) Has(phase create.Phase) bool {
	_, ok := r.handlers[phase]
	return ok
}
