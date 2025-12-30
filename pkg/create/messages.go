// Package create provides the Bubble Tea TUI for the create command.
package create

import (
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
)

// packagesLoadedMsg is sent when packages have been loaded.
type packagesLoadedMsg struct {
	registry *packages.Registry
	err      error
}

// stepCompleteMsg is sent when a wizard step is completed.
type stepCompleteMsg struct {
	step Step
}

// formSubmittedMsg is sent when a form is submitted.
type formSubmittedMsg struct{}

// deployStartMsg is sent to start deployment.
type deployStartMsg struct{}

// deployProgressMsg wraps a deploy.ProgressEvent for Bubble Tea.
type deployProgressMsg deploy.ProgressEvent

// deployCompleteMsg is sent when deployment finishes.
type deployCompleteMsg struct {
	result *deploy.DeployResult
}

// targetValidatedMsg is sent when target validation completes.
type targetValidatedMsg struct {
	err error
}

// errMsg wraps an error for Bubble Tea.
type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

// quitMsg signals the application should quit.
type quitMsg struct{}

// backMsg signals navigation to the previous step.
type backMsg struct{}

// confirmMsg is sent when the user confirms an action.
type confirmMsg struct{}

// cancelMsg is sent when the user cancels an action.
type cancelMsg struct{}

// retryMsg is sent when the user wants to retry deployment.
type retryMsg struct{}

// editMsg is sent when the user wants to edit configuration.
type editMsg struct{}

// viewLogsMsg is sent when the user wants to view logs.
type viewLogsMsg struct{}

// openShellMsg is sent when the user wants to open a shell to the deployed target.
type openShellMsg struct{}
