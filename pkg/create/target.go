package create

import (
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// runTargetSelection prompts the user to select a deployment target.
func runTargetSelection() (deploy.DeploymentTarget, error) {
	var target deploy.DeploymentTarget

	targetForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[deploy.DeploymentTarget]().
				Title("What would you like to create?").
				Description("Select your deployment target").
				Options(
					huh.NewOption("Multipass VM (local testing)", deploy.TargetMultipass),
					huh.NewOption("Bootable ISO (bare metal install)", deploy.TargetUSB),
					huh.NewOption("Remote SSH (existing server)", deploy.TargetSSH),
					huh.NewOption("Terraform/libvirt (local KVM)", deploy.TargetTerraform),
				).
				Value(&target),
		).Title("Deployment Target"),
	).WithTheme(tui.Theme())

	if err := targetForm.Run(); err != nil {
		return "", fmt.Errorf("target selection cancelled: %w", err)
	}

	return target, nil
}
