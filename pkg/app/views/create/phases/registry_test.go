package phases

import (
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/stretchr/testify/assert"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)
}

func TestRegistry_HasAllExpectedPhases(t *testing.T) {
	r := NewRegistry()

	// Check that all expected phases are registered
	assert.True(t, r.Has(wizard.PhaseTarget), "PhaseTarget should be registered")
	assert.True(t, r.Has(wizard.PhaseGit), "PhaseGit should be registered")
	assert.True(t, r.Has(wizard.PhaseHost), "PhaseHost should be registered")
	assert.True(t, r.Has(wizard.PhasePackages), "PhasePackages should be registered")
	assert.True(t, r.Has(wizard.PhaseOptional), "PhaseOptional should be registered")
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		phase        wizard.Phase
		expectedName string
	}{
		{wizard.PhaseTarget, "Select Target"},
		{wizard.PhaseGit, "Git Config"},
		{wizard.PhaseHost, "Host Details"},
		{wizard.PhasePackages, "Packages"},
		{wizard.PhaseOptional, "Optional Services"},
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
	handler := r.Get(wizard.PhaseSSH)
	assert.Nil(t, handler)
}

func TestRegistry_Has_UnregisteredPhase(t *testing.T) {
	r := NewRegistry()

	assert.False(t, r.Has(wizard.PhaseSSH))
	assert.False(t, r.Has(wizard.PhaseReview))
	assert.False(t, r.Has(wizard.PhaseDeploy))
}

func TestRegistry_Register(t *testing.T) {
	r := &Registry{
		handlers: make(map[wizard.Phase]wizard.PhaseHandler),
	}

	// Register a phase
	r.Register(wizard.PhaseTarget, NewTargetPhase())

	assert.True(t, r.Has(wizard.PhaseTarget))
	assert.Equal(t, "Select Target", r.Get(wizard.PhaseTarget).Name())
}

func TestRegistry_Register_Override(t *testing.T) {
	r := NewRegistry()

	// Create a custom host phase with a different name
	customPhase := &HostPhase{
		BasePhase: wizard.NewBasePhase("Custom Host", 3),
	}

	// Override the existing registration
	r.Register(wizard.PhaseHost, customPhase)

	assert.Equal(t, "Custom Host", r.Get(wizard.PhaseHost).Name())
}

func TestRegistry_HandlersAreIndependent(t *testing.T) {
	r := NewRegistry()

	// Get handlers
	target := r.Get(wizard.PhaseTarget)
	git := r.Get(wizard.PhaseGit)

	// They should be different instances
	assert.NotSame(t, target, git)
	assert.NotEqual(t, target.Name(), git.Name())
}
