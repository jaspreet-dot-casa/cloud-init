package phases

import (
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create"
	"github.com/stretchr/testify/assert"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)
}

func TestRegistry_HasAllExpectedPhases(t *testing.T) {
	r := NewRegistry()

	// Check that all expected phases are registered
	assert.True(t, r.Has(create.PhaseTarget), "PhaseTarget should be registered")
	assert.True(t, r.Has(create.PhaseGit), "PhaseGit should be registered")
	assert.True(t, r.Has(create.PhaseHost), "PhaseHost should be registered")
	assert.True(t, r.Has(create.PhasePackages), "PhasePackages should be registered")
	assert.True(t, r.Has(create.PhaseOptional), "PhaseOptional should be registered")
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		phase        create.Phase
		expectedName string
	}{
		{create.PhaseTarget, "Select Target"},
		{create.PhaseGit, "Git Config"},
		{create.PhaseHost, "Host Details"},
		{create.PhasePackages, "Packages"},
		{create.PhaseOptional, "Optional Services"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			handler := r.Get(tt.phase)
			assert.NotNil(t, handler)
			assert.Equal(t, tt.expectedName, handler.Name())
		})
	}
}

func TestRegistry_Get_UnregisteredPhase(t *testing.T) {
	r := NewRegistry()

	// PhaseSSH is not registered yet
	handler := r.Get(create.PhaseSSH)
	assert.Nil(t, handler)
}

func TestRegistry_Has_UnregisteredPhase(t *testing.T) {
	r := NewRegistry()

	assert.False(t, r.Has(create.PhaseSSH))
	assert.False(t, r.Has(create.PhaseReview))
	assert.False(t, r.Has(create.PhaseDeploy))
}

func TestRegistry_Register(t *testing.T) {
	r := &Registry{
		handlers: make(map[create.Phase]create.PhaseHandler),
	}

	// Register a phase
	r.Register(create.PhaseTarget, NewTargetPhase())

	assert.True(t, r.Has(create.PhaseTarget))
	assert.Equal(t, "Select Target", r.Get(create.PhaseTarget).Name())
}

func TestRegistry_Register_Override(t *testing.T) {
	r := NewRegistry()

	// Create a custom host phase with a different name
	customPhase := &HostPhase{
		BasePhase: create.NewBasePhase("Custom Host", 3),
	}

	// Override the existing registration
	r.Register(create.PhaseHost, customPhase)

	assert.Equal(t, "Custom Host", r.Get(create.PhaseHost).Name())
}

func TestRegistry_HandlersAreIndependent(t *testing.T) {
	r := NewRegistry()

	// Get handlers
	target := r.Get(create.PhaseTarget)
	git := r.Get(create.PhaseGit)

	// They should be different instances
	assert.NotSame(t, target, git)
	assert.NotEqual(t, target.Name(), git.Name())
}
