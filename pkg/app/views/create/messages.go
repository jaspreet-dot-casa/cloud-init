package create

import (
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
)

// packagesLoadedMsg is sent when packages have been loaded.
type packagesLoadedMsg struct {
	registry *packages.Registry
	err      error
}

// phaseCompleteMsg is sent when a wizard phase is completed.
type phaseCompleteMsg struct {
	phase wizard.Phase
}

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

// sshKeysLoadedMsg is sent when SSH keys are discovered or fetched.
type sshKeysLoadedMsg struct {
	localKeys  []string
	githubKeys []string
	err        error
}

// githubProfileMsg is sent when GitHub profile is fetched.
type githubProfileMsg struct {
	name  string
	email string
	err   error
}

// errMsg wraps an error for Bubble Tea.
type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}
