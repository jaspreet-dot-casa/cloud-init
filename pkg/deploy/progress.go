package deploy

import "time"

// Stage represents a deployment stage.
type Stage string

const (
	StageValidating  Stage = "validating"
	StageConfig      Stage = "config"
	StageCloudInit   Stage = "cloudinit"
	StageLaunching   Stage = "launching"
	StageWaiting     Stage = "waiting"
	StageConnecting  Stage = "connecting"
	StageTransfer    Stage = "transfer"
	StageInstalling  Stage = "installing"
	StageVerifying   Stage = "verifying"
	StageComplete    Stage = "complete"
	StageCleanup     Stage = "cleanup"
	StageError       Stage = "error"
)

// String returns the string representation of the stage.
func (s Stage) String() string {
	return string(s)
}

// DisplayName returns a human-readable name for the stage.
func (s Stage) DisplayName() string {
	switch s {
	case StageValidating:
		return "Validating"
	case StageConfig:
		return "Generating Config"
	case StageCloudInit:
		return "Generating Cloud-Init"
	case StageLaunching:
		return "Launching"
	case StageWaiting:
		return "Waiting"
	case StageConnecting:
		return "Connecting"
	case StageTransfer:
		return "Transferring Files"
	case StageInstalling:
		return "Installing"
	case StageVerifying:
		return "Verifying"
	case StageComplete:
		return "Complete"
	case StageCleanup:
		return "Cleaning Up"
	case StageError:
		return "Error"
	default:
		return string(s)
	}
}

// ProgressEvent represents a deployment progress update.
type ProgressEvent struct {
	Stage     Stage     // Current stage
	Message   string    // Human-readable message
	Command   string    // Command being executed (e.g., "multipass launch...")
	Detail    string    // Additional detail or output
	Percent   int       // 0-100, -1 for indeterminate
	IsError   bool      // True if this is an error message
	Timestamp time.Time // When this event occurred
}

// NewProgressEvent creates a new progress event.
func NewProgressEvent(stage Stage, message string, percent int) ProgressEvent {
	return ProgressEvent{
		Stage:     stage,
		Message:   message,
		Percent:   percent,
		IsError:   false,
		Timestamp: time.Now(),
	}
}

// NewProgressEventWithCommand creates a progress event with a command.
func NewProgressEventWithCommand(stage Stage, message, command string, percent int) ProgressEvent {
	return ProgressEvent{
		Stage:     stage,
		Message:   message,
		Command:   command,
		Percent:   percent,
		IsError:   false,
		Timestamp: time.Now(),
	}
}

// NewProgressEventWithDetail creates a progress event with detail.
func NewProgressEventWithDetail(stage Stage, message, detail string, percent int) ProgressEvent {
	return ProgressEvent{
		Stage:     stage,
		Message:   message,
		Detail:    detail,
		Percent:   percent,
		IsError:   false,
		Timestamp: time.Now(),
	}
}

// NewErrorEvent creates a new error progress event.
func NewErrorEvent(message string) ProgressEvent {
	return ProgressEvent{
		Stage:     StageError,
		Message:   message,
		Percent:   -1,
		IsError:   true,
		Timestamp: time.Now(),
	}
}

// NewErrorEventWithDetail creates an error event with detail.
func NewErrorEventWithDetail(message, detail string) ProgressEvent {
	return ProgressEvent{
		Stage:     StageError,
		Message:   message,
		Detail:    detail,
		Percent:   -1,
		IsError:   true,
		Timestamp: time.Now(),
	}
}

// ProgressCallback is called with progress updates during deployment.
type ProgressCallback func(ProgressEvent)

// NoOpProgress is a progress callback that does nothing.
func NoOpProgress(_ ProgressEvent) {}

// ProgressTracker collects progress events for later review.
type ProgressTracker struct {
	events []ProgressEvent
}

// NewProgressTracker creates a new progress tracker.
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		events: make([]ProgressEvent, 0),
	}
}

// Callback returns a ProgressCallback that records events.
func (t *ProgressTracker) Callback() ProgressCallback {
	return func(e ProgressEvent) {
		t.events = append(t.events, e)
	}
}

// Events returns all recorded events.
func (t *ProgressTracker) Events() []ProgressEvent {
	return t.events
}

// LastEvent returns the most recent event, or nil if none.
func (t *ProgressTracker) LastEvent() *ProgressEvent {
	if len(t.events) == 0 {
		return nil
	}
	return &t.events[len(t.events)-1]
}

// HasErrors returns true if any error events were recorded.
func (t *ProgressTracker) HasErrors() bool {
	for _, e := range t.events {
		if e.IsError {
			return true
		}
	}
	return false
}

// Errors returns all error events.
func (t *ProgressTracker) Errors() []ProgressEvent {
	var errors []ProgressEvent
	for _, e := range t.events {
		if e.IsError {
			errors = append(errors, e)
		}
	}
	return errors
}
