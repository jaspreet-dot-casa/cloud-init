package create

import (
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// OSImage represents an available OS image for Multipass.
type OSImage struct {
	Name  string // Display name
	Image string // Multipass image identifier
}

// Available OS images for Multipass (from `multipass find`)
var osImages = []OSImage{
	{"Ubuntu 24.04 LTS (Noble Numbat)", "24.04"},
	{"Ubuntu 25.04 (Plucky Puffin)", "25.04"},
	{"Ubuntu 25.10 (Questing Quail)", "25.10"},
	{"Ubuntu 22.04 LTS (Jammy Jellyfish)", "22.04"},
	{"Ubuntu 26.04 LTS Daily (Resolute)", "daily:26.04"},
}

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
