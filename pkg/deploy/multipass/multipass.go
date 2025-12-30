// Package multipass provides a Multipass VM deployer implementation.
package multipass

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/generator"
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

// checkInstalled verifies multipass is available.
func (d *Deployer) checkInstalled() error {
	path, err := exec.LookPath(d.binaryPath)
	if err != nil {
		return fmt.Errorf("multipass is not installed; install with: brew install multipass")
	}
	d.binaryPath = path

	// Verify it works
	cmd := exec.Command(d.binaryPath, "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("multipass is installed but not working: %w", err)
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

	// Stage 2: Generate config files
	progress(deploy.NewProgressEventWithDetail(
		deploy.StageConfig,
		"Writing configuration files...",
		fmt.Sprintf("Writing to %s/config.env", opts.ProjectRoot),
		15,
	))
	if err := d.writeConfigs(opts); err != nil {
		return d.fail(result, err, start), err
	}

	// Stage 3: Generate cloud-init.yaml
	progress(deploy.NewProgressEventWithDetail(
		deploy.StageCloudInit,
		"Generating cloud-init.yaml...",
		fmt.Sprintf("Template: cloud-init/cloud-init.template.yaml"),
		25,
	))
	cloudInitPath, err := d.generateCloudInit(opts)
	if err != nil {
		return d.fail(result, err, start), err
	}
	result.Outputs["cloud_init_path"] = cloudInitPath

	// Stage 4: Determine VM name
	vmName := opts.Multipass.VMName
	if vmName == "" {
		vmName = d.generateVMName()
		opts.Multipass.VMName = vmName // Store for Cleanup() to use
	}
	result.Outputs["vm_name"] = vmName

	// Stage 5: Launch VM
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

	// Stage 6: Wait for cloud-init
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

	// Stage 7: Get VM info
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

// writeConfigs writes the configuration files.
func (d *Deployer) writeConfigs(opts *deploy.DeployOptions) error {
	writer := config.NewWriter(opts.ProjectRoot)
	return writer.WriteAll(opts.Config)
}

// generateCloudInit generates the cloud-init.yaml file.
func (d *Deployer) generateCloudInit(opts *deploy.DeployOptions) (string, error) {
	templatePath := filepath.Join(opts.ProjectRoot, "cloud-init", "cloud-init.template.yaml")
	outputPath := filepath.Join(opts.ProjectRoot, "cloud-init", "cloud-init.yaml")

	gen := generator.NewGenerator(opts.ProjectRoot)
	if err := gen.Generate(opts.Config, templatePath, outputPath); err != nil {
		return "", fmt.Errorf("failed to generate cloud-init.yaml: %w", err)
	}

	return outputPath, nil
}

// generateVMName generates a unique VM name.
func (d *Deployer) generateVMName() string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("cloud-init-test-%s", timestamp)
}

// launchVM launches the Multipass VM.
func (d *Deployer) launchVM(ctx context.Context, vmName, cloudInitPath string, opts *deploy.DeployOptions) error {
	mp := opts.Multipass

	// Build command
	args := []string{
		"launch",
		"--name", vmName,
		"--cpus", fmt.Sprintf("%d", mp.CPUs),
		"--memory", fmt.Sprintf("%dM", mp.MemoryMB),
		"--disk", fmt.Sprintf("%dG", mp.DiskGB),
		"--timeout", "900", // 15 minute launch timeout
		"--cloud-init", cloudInitPath,
	}

	// Add Ubuntu version
	version := mp.UbuntuVersion
	if version == "" {
		version = "24.04"
	}
	args = append(args, version)

	cmd := exec.CommandContext(ctx, d.binaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return fmt.Errorf("failed to launch VM: %s", strings.TrimSpace(errMsg))
		}
		return fmt.Errorf("failed to launch VM: %w", err)
	}

	return nil
}

// waitForCloudInit waits for cloud-init to complete.
func (d *Deployer) waitForCloudInit(ctx context.Context, vmName string, progress deploy.ProgressCallback) error {
	timeout := 15 * time.Minute
	pollInterval := 5 * time.Second // Check every 5 seconds for more responsive UI
	deadline := time.Now().Add(timeout)

	progressPct := 50

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check cloud-init status
		cmd := exec.CommandContext(ctx, d.binaryPath, "exec", vmName, "--", "cloud-init", "status")
		output, err := cmd.CombinedOutput()
		status := strings.TrimSpace(string(output))

		// Check for "done" first, even if there was an error
		// (cloud-init status might return non-zero in some cases)
		if strings.Contains(status, "done") {
			return nil
		}

		if err != nil {
			// VM might still be booting or command failed
			detail := "VM is booting..."
			if status != "" {
				detail = fmt.Sprintf("Status: %s", status)
			}
			progress(deploy.NewProgressEventWithDetail(
				deploy.StageWaiting,
				"Waiting for cloud-init...",
				detail,
				progressPct,
			))
			time.Sleep(pollInterval)
			continue
		}

		if strings.Contains(status, "error") {
			// Try to get more info
			logCmd := exec.CommandContext(ctx, d.binaryPath, "exec", vmName, "--", "sudo", "cat", "/var/log/cloud-init-output.log")
			logOutput, _ := logCmd.Output()
			if len(logOutput) > 0 {
				// Get last 20 lines
				lines := strings.Split(string(logOutput), "\n")
				if len(lines) > 20 {
					lines = lines[len(lines)-20:]
				}
				return fmt.Errorf("cloud-init error: %s", strings.Join(lines, "\n"))
			}
			return fmt.Errorf("cloud-init reported an error")
		}

		// Update progress with actual status
		progressPct++
		if progressPct > 85 {
			progressPct = 85
		}
		progress(deploy.NewProgressEventWithDetail(
			deploy.StageWaiting,
			"Waiting for cloud-init...",
			fmt.Sprintf("Status: %s", status),
			progressPct,
		))

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for cloud-init to complete")
}

// getVMInfo retrieves information about the VM.
func (d *Deployer) getVMInfo(vmName string, opts *deploy.DeployOptions) (map[string]string, error) {
	info := make(map[string]string)

	// Get IP address
	cmd := exec.Command(d.binaryPath, "info", vmName)
	output, err := cmd.Output()
	if err != nil {
		return info, fmt.Errorf("failed to get VM info: %w", err)
	}

	// Parse IP from output
	ipRegex := regexp.MustCompile(`IPv4:\s*(\d+\.\d+\.\d+\.\d+)`)
	if matches := ipRegex.FindSubmatch(output); len(matches) > 1 {
		info["ip"] = string(matches[1])
	}

	// Get username from config (falls back to ubuntu if not set)
	username := "ubuntu"
	if opts.Config != nil && opts.Config.Username != "" {
		username = opts.Config.Username
	}
	info["user"] = username

	// Generate SSH command
	if ip, ok := info["ip"]; ok {
		info["ssh_command"] = fmt.Sprintf("ssh %s@%s", username, ip)
		info["multipass_shell"] = fmt.Sprintf("multipass shell %s", vmName)
	}

	return info, nil
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

// InstallInstructions returns installation instructions for multipass.
func InstallInstructions() string {
	return `Multipass is required for VM deployment.

Install with:
  macOS:   brew install multipass
  Linux:   sudo snap install multipass
  Windows: Download from https://multipass.run`
}
