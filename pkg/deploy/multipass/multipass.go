// Package multipass provides a Multipass VM deployer implementation.
package multipass

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// Deployer implements deploy.Deployer for Multipass VMs.
type Deployer struct {
	binaryPath string
	verbose    bool
}

// New creates a new Multipass deployer.
func New() *Deployer {
	return &Deployer{
		binaryPath: "multipass",
	}
}

// SetVerbose enables verbose output.
func (d *Deployer) SetVerbose(v bool) {
	d.verbose = v
}

// Name returns the deployer name.
func (d *Deployer) Name() string {
	return "Multipass VM"
}

// Target returns the deployment target type.
func (d *Deployer) Target() deploy.DeploymentTarget {
	return deploy.TargetMultipass
}

// Validate checks if deployment can proceed.
func (d *Deployer) Validate(opts *deploy.DeployOptions) error {
	// Check multipass is installed
	if err := d.checkInstalled(); err != nil {
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

	return nil
}

// Deploy executes the Multipass deployment.
func (d *Deployer) Deploy(ctx context.Context, opts *deploy.DeployOptions, progress deploy.ProgressCallback) (*deploy.DeployResult, error) {
	result := &deploy.DeployResult{
		Target:  deploy.TargetMultipass,
		Outputs: make(map[string]string),
		Logs:    make([]string, 0),
	}
	start := time.Now()

	// Stage 1: Validate
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageValidating,
		"Validating configuration...",
		"multipass version",
		5,
	))
	if err := d.Validate(opts); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 2: Generate cloud-init.yaml
	progress(deploy.NewProgressEventWithDetail(
		deploy.StageCloudInit,
		"Generating cloud-init.yaml...",
		fmt.Sprintf("Template: cloud-init/cloud-init.template.yaml"),
		15,
	))
	cloudInitPath, err := d.generateCloudInit(opts)
	if err != nil {
		return d.fail(result, err, start), err
	}
	result.Outputs["cloud_init_path"] = cloudInitPath

	// Stage 3: Determine VM name
	vmName := opts.Multipass.VMName
	if vmName == "" {
		vmName = d.generateVMName()
		opts.Multipass.VMName = vmName // Store for Cleanup() to use
	}
	result.Outputs["vm_name"] = vmName

	// Stage 4: Launch VM
	mp := opts.Multipass
	version := mp.UbuntuVersion
	if version == "" {
		version = "24.04"
	}
	launchCmd := fmt.Sprintf("multipass launch --name %s --cpus %d --memory %dM --disk %dG %s",
		vmName, mp.CPUs, mp.MemoryMB, mp.DiskGB, version)

	progress(deploy.NewProgressEventWithCommand(
		deploy.StageLaunching,
		fmt.Sprintf("Launching VM '%s'...", vmName),
		launchCmd,
		35,
	))
	if err := d.launchVM(ctx, vmName, cloudInitPath, opts); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 5: Wait for cloud-init
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageWaiting,
		"Waiting for cloud-init to complete...",
		fmt.Sprintf("multipass exec %s -- cloud-init status", vmName),
		50,
	))
	if err := d.waitForCloudInit(ctx, vmName, progress); err != nil {
		// Don't fail here, just log it
		result.Logs = append(result.Logs, fmt.Sprintf("Warning: %v", err))
	}

	// Stage 6: Get VM info
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageVerifying,
		"Retrieving VM information...",
		fmt.Sprintf("multipass info %s", vmName),
		90,
	))
	info, err := d.getVMInfo(vmName, opts)
	if err != nil {
		return d.fail(result, err, start), err
	}
	for k, v := range info {
		result.Outputs[k] = v
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

// Cleanup cleans up the VM on failure.
func (d *Deployer) Cleanup(ctx context.Context, opts *deploy.DeployOptions) error {
	if opts.Multipass.KeepOnFailure {
		return nil // Don't cleanup, user wants to debug
	}

	vmName := opts.Multipass.VMName
	if vmName == "" {
		return nil // No VM was created
	}

	// Delete VM
	deleteCmd := exec.CommandContext(ctx, d.binaryPath, "delete", vmName)
	if err := deleteCmd.Run(); err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	// Purge
	purgeCmd := exec.CommandContext(ctx, d.binaryPath, "purge")
	if err := purgeCmd.Run(); err != nil {
		return fmt.Errorf("failed to purge VM: %w", err)
	}

	return nil
}
