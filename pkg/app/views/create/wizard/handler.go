package wizard

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
)

// PhaseHandler represents a wizard phase that handles its own state, input, and rendering.
// Each phase is responsible for a specific step in the wizard flow.
type PhaseHandler interface {
	// Name returns the display name for this phase (shown in progress indicator)
	Name() string

	// Init initializes the phase state when entering. Called when transitioning to this phase.
	Init(ctx *PhaseContext)

	// Update handles keyboard input and returns:
	// - advance: true if the phase is complete and should advance to next
	// - cmd: any tea.Cmd to execute (e.g., async operations)
	Update(ctx *PhaseContext, msg tea.KeyMsg) (advance bool, cmd tea.Cmd)

	// View renders the phase content
	View(ctx *PhaseContext) string

	// FieldCount returns the number of focusable fields in this phase
	FieldCount() int

	// Save persists phase data to wizard state. Called before advancing to next phase.
	Save(ctx *PhaseContext)
}

// PhaseContext provides dependencies and shared state to phases.
// This decouples phases from the Model and makes them independently testable.
type PhaseContext struct {
	// Wizard holds all wizard state including form inputs and collected data
	Wizard *State

	// ProjectDir is the root directory of the project
	ProjectDir string

	// CloudImages are available cloud images from settings (used by Terraform phase)
	CloudImages []settings.CloudImage

	// Store provides access to settings storage
	Store *settings.Store

	// Message is used to display status messages to the user
	Message *string
}

// BasePhase provides common functionality for phases.
// Embed this in phase implementations to get default behaviors.
type BasePhase struct {
	name       string
	fieldCount int
}

// NewBasePhase creates a new BasePhase with the given name and field count.
func NewBasePhase(name string, fieldCount int) BasePhase {
	return BasePhase{
		name:       name,
		fieldCount: fieldCount,
	}
}

// Name returns the phase display name.
func (p *BasePhase) Name() string {
	return p.name
}

// FieldCount returns the number of focusable fields.
func (p *BasePhase) FieldCount() int {
	return p.fieldCount
}

// HandleTextInput updates the focused text input with the key message.
// Returns the updated model and any command.
func HandleTextInput(ctx *PhaseContext, inputName string, msg tea.KeyMsg) tea.Cmd {
	if ti, ok := ctx.Wizard.TextInputs[inputName]; ok {
		var cmd tea.Cmd
		ti, cmd = ti.Update(msg)
		ctx.Wizard.TextInputs[inputName] = ti
		return cmd
	}
	return nil
}

// FocusInput focuses the text input with the given name.
func FocusInput(ctx *PhaseContext, inputName string) tea.Cmd {
	if ti, ok := ctx.Wizard.TextInputs[inputName]; ok {
		ti.Focus()
		ctx.Wizard.TextInputs[inputName] = ti
	}
	return nil
}

// BlurInput blurs the text input with the given name.
func BlurInput(ctx *PhaseContext, inputName string) {
	if ti, ok := ctx.Wizard.TextInputs[inputName]; ok {
		ti.Blur()
		ctx.Wizard.TextInputs[inputName] = ti
	}
}
