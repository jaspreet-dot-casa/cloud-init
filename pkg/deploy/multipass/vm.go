package multipass

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

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
