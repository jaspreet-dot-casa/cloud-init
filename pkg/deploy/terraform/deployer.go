// Package terraform provides a Terraform/libvirt VM deployer implementation.
package terraform

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// Deployer implements deploy.Deployer for Terraform/libvirt VMs.
type Deployer struct {
	projectRoot string
	verbose     bool
}

// New creates a new Terraform deployer.
func New(projectRoot string) *Deployer {
	return &Deployer{
		projectRoot: projectRoot,
	}
}

// SetVerbose enables verbose output.
func (d *Deployer) SetVerbose(v bool) {
	d.verbose = v
}

// Name returns the deployer name.
func (d *Deployer) Name() string {
	return "Terraform/libvirt"
}

// Target returns the deployment target type.
func (d *Deployer) Target() deploy.DeploymentTarget {
	return deploy.TargetTerraform
}

// Validate checks if deployment can proceed.
func (d *Deployer) Validate(opts *deploy.DeployOptions) error {
	// Check terraform is installed
	if err := d.checkTerraformInstalled(); err != nil {
		return err
	}

	// Check project root exists
	if opts.ProjectRoot == "" {
		return fmt.Errorf("project root is required")
	}

	// Check config is provided
	if opts.Config == nil {
		return fmt.Errorf("configuration is required")
	}

	// Check Ubuntu image exists
	if opts.Terraform.UbuntuImage != "" {
		if err := d.checkUbuntuImage(opts.Terraform.UbuntuImage); err != nil {
			return err
		}
	}

	return nil
}

// Deploy executes the Terraform deployment.
func (d *Deployer) Deploy(ctx context.Context, opts *deploy.DeployOptions, progress deploy.ProgressCallback) (*deploy.DeployResult, error) {
	result := &deploy.DeployResult{
		Target:  deploy.TargetTerraform,
		Outputs: make(map[string]string),
		Logs:    make([]string, 0),
	}
	start := time.Now()

	tfOpts := opts.Terraform
	workDir := filepath.Join(opts.ProjectRoot, tfOpts.WorkDir)

	// Stage 1: Validate (5%)
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageValidating,
		"Validating configuration...",
		"terraform version",
		5,
	))
	if err := d.Validate(opts); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 2: Generate config files (15%)
	progress(deploy.NewProgressEventWithDetail(
		deploy.StageConfig,
		"Writing configuration files...",
		fmt.Sprintf("Writing to %s/config.env", opts.ProjectRoot),
		15,
	))
	if err := d.writeConfigs(opts); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 3: Generate cloud-init.yaml (25%)
	progress(deploy.NewProgressEventWithDetail(
		deploy.StageCloudInit,
		"Generating cloud-init.yaml...",
		"Template: cloud-init/cloud-init.template.yaml",
		25,
	))
	cloudInitPath, err := d.generateCloudInit(opts)
	if err != nil {
		return d.fail(result, err, start), err
	}
	result.Outputs["cloud_init_path"] = cloudInitPath

	// Stage 4: Determine VM name
	vmName := tfOpts.VMName
	if vmName == "" {
		vmName = d.generateVMName()
		opts.Terraform.VMName = vmName
	}
	result.Outputs["vm_name"] = vmName

	// Stage 5: Generate terraform.tfvars (35%)
	progress(deploy.NewProgressEventWithDetail(
		deploy.StagePreparing,
		"Generating terraform.tfvars...",
		fmt.Sprintf("Writing to %s/terraform.tfvars", workDir),
		35,
	))
	if err := d.writeTFVars(opts, cloudInitPath); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 6: Terraform init (45%)
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageValidating,
		"Initializing Terraform...",
		"terraform init",
		45,
	))
	if err := d.terraformInit(ctx, workDir); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 7: Terraform plan (55%)
	progress(deploy.NewProgressEventWithCommand(
		deploy.StagePlanning,
		"Creating execution plan...",
		"terraform plan",
		55,
	))
	planOutput, err := d.terraformPlan(ctx, workDir)
	if err != nil {
		return d.fail(result, err, start), err
	}
	result.Logs = append(result.Logs, planOutput)

	// Stage 8: Confirm (60%) - unless AutoApprove is set
	if !tfOpts.AutoApprove {
		progress(deploy.NewProgressEventWithDetail(
			deploy.StageConfirming,
			"Waiting for confirmation...",
			"Review the plan above",
			60,
		))
		confirmed, err := d.confirmApply(planOutput)
		if err != nil {
			return d.fail(result, err, start), err
		}
		if !confirmed {
			err := fmt.Errorf("deployment cancelled by user")
			return d.fail(result, err, start), err
		}
	}

	// Stage 9: Terraform apply (75%)
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageApplying,
		fmt.Sprintf("Creating VM '%s'...", vmName),
		"terraform apply -auto-approve",
		75,
	))
	if err := d.terraformApply(ctx, workDir); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 10: Get VM info (90%)
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageVerifying,
		"Retrieving VM information...",
		"terraform output -json",
		90,
	))
	outputs, err := d.terraformOutput(ctx, workDir)
	if err != nil {
		// Non-fatal, continue without outputs
		result.Logs = append(result.Logs, fmt.Sprintf("Warning: could not get terraform outputs: %v", err))
	} else {
		for k, v := range outputs {
			result.Outputs[k] = v
		}
	}

	// Add console command
	result.Outputs["console_command"] = fmt.Sprintf("virsh console %s", vmName)

	// Get username from config
	username := "ubuntu"
	if opts.Config != nil && opts.Config.Username != "" {
		username = opts.Config.Username
	}
	result.Outputs["user"] = username

	// Generate SSH command if we have an IP
	if ip, ok := result.Outputs["ip"]; ok && ip != "" && ip != "pending" {
		result.Outputs["ssh_command"] = fmt.Sprintf("ssh %s@%s", username, ip)
	}

	// Success
	progress(deploy.NewProgressEvent(deploy.StageComplete, "Deployment complete!", 100))
	result.Success = true
	result.Duration = time.Since(start)

	return result, nil
}

// fail records a failure and returns the result.
func (d *Deployer) fail(result *deploy.DeployResult, err error, start time.Time) *deploy.DeployResult {
	result.Success = false
	result.Error = err
	result.Duration = time.Since(start)
	return result
}

// Cleanup cleans up Terraform resources on failure.
func (d *Deployer) Cleanup(ctx context.Context, opts *deploy.DeployOptions) error {
	if opts.Terraform.KeepOnFailure {
		return nil // Don't cleanup, user wants to debug
	}

	workDir := filepath.Join(opts.ProjectRoot, opts.Terraform.WorkDir)

	// Run terraform destroy
	if err := d.terraformDestroy(ctx, workDir); err != nil {
		return fmt.Errorf("failed to destroy resources: %w", err)
	}

	return nil
}

// generateVMName generates a unique VM name.
func (d *Deployer) generateVMName() string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("cloud-init-%s", timestamp)
}

// InstallInstructions returns installation instructions for terraform and libvirt.
func InstallInstructions() string {
	return `Terraform and libvirt are required for VM deployment.

Install Terraform:
  Linux:   sudo apt install terraform
           or: brew install terraform
  macOS:   brew install terraform

Install libvirt (Linux only):
  Ubuntu:  sudo apt install qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils
  Fedora:  sudo dnf install @virtualization

Note: libvirt is only available on Linux. macOS users can use Multipass instead.`
}
