package create

import (
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// confirmDeployment shows a review and asks for confirmation.
func confirmDeployment(result *tui.FormResult, target deploy.DeploymentTarget, targetOpts interface{}) bool {
	fmt.Println()
	fmt.Println(titleStyle.Render("Review Configuration"))
	fmt.Println()

	// Build target-specific details
	var targetDetails string
	switch target {
	case deploy.TargetMultipass:
		var opts deploy.MultipassOptions
		switch v := targetOpts.(type) {
		case deploy.MultipassOptions:
			opts = v
		case *deploy.MultipassOptions:
			opts = *v
		default:
			// Fallback to default display if type assertion fails
			targetDetails = fmt.Sprintf(`
%s
  Target:    %s`,
				successStyle.Render("Deployment"),
				target.DisplayName(),
			)
			break
		}
		if targetDetails == "" {
			targetDetails = fmt.Sprintf(`
%s
  Target:    %s
  VM Name:   %s
  OS Image:  %s
  Resources: %d CPU, %d MB RAM, %d GB disk`,
				successStyle.Render("Deployment"),
				target.DisplayName(),
				opts.VMName,
				opts.UbuntuVersion,
				opts.CPUs,
				opts.MemoryMB,
				opts.DiskGB,
			)
		}
	case deploy.TargetUSB:
		var opts deploy.USBOptions
		switch v := targetOpts.(type) {
		case deploy.USBOptions:
			opts = v
		case *deploy.USBOptions:
			opts = *v
		default:
			// Fallback to default display if type assertion fails
			targetDetails = fmt.Sprintf(`
%s
  Target:    %s`,
				successStyle.Render("Deployment"),
				target.DisplayName(),
			)
			break
		}
		if targetDetails == "" {
			targetDetails = fmt.Sprintf(`
%s
  Target:    %s
  Source:    %s
  Storage:   %s`,
				successStyle.Render("Deployment"),
				target.DisplayName(),
				opts.SourceISO,
				opts.StorageLayout,
			)
		}
	default:
		targetDetails = fmt.Sprintf(`
%s
  Target:    %s`,
			successStyle.Render("Deployment"),
			target.DisplayName(),
		)
	}

	review := fmt.Sprintf(`%s
  Username:  %s
  Hostname:  %s
  Name:      %s
  Email:     %s
  SSH Keys:  %d configured

%s
  Selected:  %d packages%s`,
		successStyle.Render("User Configuration"),
		result.User.Username,
		result.User.Hostname,
		result.User.FullName,
		result.User.Email,
		len(result.User.SSHPublicKeys),
		successStyle.Render("Packages"),
		len(result.SelectedPackages),
		targetDetails,
	)

	fmt.Println(boxStyle.Render(review))
	fmt.Println()

	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Start deployment?").
				Affirmative("Yes, deploy!").
				Negative("Cancel").
				Value(&confirm),
		),
	).WithTheme(tui.Theme())

	if err := confirmForm.Run(); err != nil {
		return false
	}

	return confirm
}
