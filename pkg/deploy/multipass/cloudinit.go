package multipass

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/generator"
)

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

// InstallInstructions returns installation instructions for multipass.
func InstallInstructions() string {
	return `Multipass is required for VM deployment.

Install with:
  macOS:   brew install multipass
  Linux:   sudo snap install multipass
  Windows: Download from https://multipass.run`
}
