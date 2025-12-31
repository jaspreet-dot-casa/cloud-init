package create

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy/multipass"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy/terraform"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy/usb"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// Run executes the create command workflow.
func Run(projectRoot string) error {
	// Step 1: Welcome and discover packages
	fmt.Println(titleStyle.Render("Ubuntu Cloud-Init Setup"))
	fmt.Println()

	registry, err := packages.DiscoverFromProjectRoot(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}

	// Step 2: TARGET SELECTION FIRST
	target, err := runTargetSelection()
	if err != nil {
		return err
	}

	// Step 3: Target-specific options (before config wizard)
	var targetOpts interface{}
	switch target {
	case deploy.TargetMultipass:
		targetOpts, err = runMultipassOptions()
		if err != nil {
			return err
		}
	case deploy.TargetUSB:
		targetOpts, err = runUSBOptions()
		if err != nil {
			return err
		}
	case deploy.TargetSSH:
		return fmt.Errorf("SSH target not yet implemented")
	case deploy.TargetTerraform:
		targetOpts, err = runTerraformOptions()
		if err != nil {
			return err
		}
	}

	// Step 4: Run the configuration wizard (SSH, Git, Host, Packages, Optional)
	// Skip output mode question since target was already selected
	formResult, err := tui.RunForm(registry, &tui.FormOptions{SkipOutputMode: true})
	if err != nil {
		return err
	}

	// Step 5: Show review and confirm
	if !confirmDeployment(formResult, target, targetOpts) {
		fmt.Println("\n" + dimStyle.Render("Deployment cancelled."))
		return nil
	}

	// Step 6: Generate config and deploy
	cfg := config.NewFullConfigFromFormResult(formResult)

	opts := &deploy.DeployOptions{
		ProjectRoot: projectRoot,
		Config:      cfg,
	}

	// Merge target-specific options
	switch target {
	case deploy.TargetMultipass:
		multipassOpts, ok := targetOpts.(deploy.MultipassOptions)
		if !ok {
			return fmt.Errorf("internal error: expected MultipassOptions but got %T", targetOpts)
		}
		opts.Multipass = multipassOpts
	case deploy.TargetUSB:
		usbOpts, ok := targetOpts.(deploy.USBOptions)
		if !ok {
			return fmt.Errorf("internal error: expected USBOptions but got %T", targetOpts)
		}
		opts.USB = usbOpts
	case deploy.TargetTerraform:
		tfOpts, ok := targetOpts.(deploy.TerraformOptions)
		if !ok {
			return fmt.Errorf("internal error: expected TerraformOptions but got %T", targetOpts)
		}
		opts.Terraform = tfOpts
	}

	// Step 7: Run deployment with progress UI
	return runDeployment(target, opts)
}

// runDeployment runs the deployment with a Bubble Tea progress UI.
func runDeployment(target deploy.DeploymentTarget, opts *deploy.DeployOptions) error {
	// Create deployer
	var deployer deploy.Deployer
	switch target {
	case deploy.TargetMultipass:
		deployer = multipass.New()
	case deploy.TargetUSB:
		deployer = usb.New(opts.ProjectRoot)
	case deploy.TargetTerraform:
		deployer = terraform.New(opts.ProjectRoot)
	default:
		return fmt.Errorf("deployer not implemented for target: %s", target)
	}

	// Run the deployment UI in alt-screen
	m := newDeployModel(deployer, opts)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("deployment UI error: %w", err)
	}

	// Get the result from the model
	model, ok := finalModel.(deployModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	// Print final results outside of alt-screen (so they're scrollable in terminal)
	printDeploymentResults(model.result, deployer.Name())

	// Return error if deployment failed
	if model.result != nil && !model.result.Success {
		return model.result.Error
	}

	return nil
}
