// Package terraform provides a Terraform/libvirt VM deployer implementation.
package terraform

import (
	"context"
	"fmt"
	"io"
	"os"
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
// It creates a new machine directory under tf/<vm-name>/, copies TF templates,
// generates cloud-init.yaml and terraform.tfvars, then runs terraform.
func (d *Deployer) Deploy(ctx context.Context, opts *deploy.DeployOptions, progress deploy.ProgressCallback) (*deploy.DeployResult, error) {
	result := &deploy.DeployResult{
		Target:  deploy.TargetTerraform,
		Outputs: make(map[string]string),
		Logs:    make([]string, 0),
	}
	start := time.Now()

	tfOpts := opts.Terraform

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

	// Stage 2: Determine VM name and create machine directory
	vmName := tfOpts.VMName
	if vmName == "" {
		vmName = d.generateVMName()
		opts.Terraform.VMName = vmName
	}
	result.Outputs["vm_name"] = vmName

	// Create machine directory: tf/<vm-name>/
	machineDir := filepath.Join(opts.ProjectRoot, "tf", vmName)
	templateDir := filepath.Join(opts.ProjectRoot, "terraform")

	progress(deploy.NewProgressEventWithDetail(
		deploy.StagePreparing,
		"Creating machine directory...",
		fmt.Sprintf("tf/%s/", vmName),
		10,
	))

	if err := d.createMachineDir(machineDir, templateDir); err != nil {
		return d.fail(result, err, start), err
	}
	result.Outputs["machine_dir"] = machineDir

	// Stage 3: Generate cloud-init.yaml in machine directory (20%)
	progress(deploy.NewProgressEventWithDetail(
		deploy.StageCloudInit,
		"Generating cloud-init.yaml...",
		fmt.Sprintf("tf/%s/cloud-init.yaml", vmName),
		20,
	))
	cloudInitPath, err := d.generateCloudInitInDir(opts, machineDir)
	if err != nil {
		return d.fail(result, err, start), err
	}
	result.Outputs["cloud_init_path"] = cloudInitPath

	// Stage 4: Generate terraform.tfvars (30%)
	progress(deploy.NewProgressEventWithDetail(
		deploy.StagePreparing,
		"Generating terraform.tfvars...",
		fmt.Sprintf("tf/%s/terraform.tfvars", vmName),
		30,
	))
	if err := d.writeTFVarsInDir(opts, machineDir, cloudInitPath); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 5: Terraform init (40%)
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageValidating,
		"Initializing Terraform...",
		"terraform init",
		40,
	))
	if err := d.terraformInit(ctx, machineDir); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 6: Terraform plan (50%)
	progress(deploy.NewProgressEventWithCommand(
		deploy.StagePlanning,
		"Creating execution plan...",
		"terraform plan",
		50,
	))
	planOutput, err := d.terraformPlan(ctx, machineDir)
	if err != nil {
		return d.fail(result, err, start), err
	}
	result.Logs = append(result.Logs, planOutput)

	// Stage 7: Confirm (60%) - unless AutoApprove is set
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

	// Stage 8: Terraform apply (75%)
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageApplying,
		fmt.Sprintf("Creating VM '%s'...", vmName),
		"terraform apply -auto-approve",
		75,
	))
	if err := d.terraformApply(ctx, machineDir); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 9: Get VM info (90%)
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageVerifying,
		"Retrieving VM information...",
		"terraform output -json",
		90,
	))
	outputs, err := d.terraformOutput(ctx, machineDir)
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

// createMachineDir creates the machine directory and copies TF templates.
func (d *Deployer) createMachineDir(machineDir, templateDir string) error {
	// Create the machine directory
	if err := os.MkdirAll(machineDir, 0755); err != nil {
		return fmt.Errorf("failed to create machine directory: %w", err)
	}

	// Copy terraform files from template directory
	tfFiles := []string{"main.tf", "variables.tf", "outputs.tf"}
	for _, file := range tfFiles {
		src := filepath.Join(templateDir, file)
		dst := filepath.Join(machineDir, file)
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", file, err)
		}
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
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

	vmName := opts.Terraform.VMName
	if vmName == "" {
		return nil // No VM to clean up
	}

	machineDir := filepath.Join(opts.ProjectRoot, "tf", vmName)

	// Check if machine directory exists
	if _, err := os.Stat(machineDir); os.IsNotExist(err) {
		return nil // Nothing to clean up
	}

	// Run terraform destroy
	if err := d.terraformDestroy(ctx, machineDir); err != nil {
		return fmt.Errorf("failed to destroy resources: %w", err)
	}

	// Optionally remove the machine directory
	// For now, we keep it for debugging purposes

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
