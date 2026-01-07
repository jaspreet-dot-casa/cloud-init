// Package terragrunt provides a Terragrunt/OpenTofu config generator implementation.
package terragrunt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// vmNamePattern defines valid VM name characters: lowercase alphanumeric and hyphens
var vmNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// Generator implements deploy.Deployer for generating Terragrunt/OpenTofu configs.
type Generator struct {
	projectRoot string
}

// New creates a new Terragrunt config generator.
func New(projectRoot string) *Generator {
	return &Generator{
		projectRoot: projectRoot,
	}
}

// Name returns the generator name.
func (g *Generator) Name() string {
	return "Terragrunt/libvirt"
}

// Target returns the deployment target type.
func (g *Generator) Target() deploy.DeploymentTarget {
	return deploy.TargetTerragrunt
}

// Validate checks if generation can proceed.
func (g *Generator) Validate(opts *deploy.DeployOptions) error {
	// Check project root exists
	if opts.ProjectRoot == "" {
		return fmt.Errorf("project root is required")
	}

	// Check config is provided
	if opts.Config == nil {
		return fmt.Errorf("configuration is required")
	}

	// Check terragrunt modules directory exists
	modulesDir := filepath.Join(opts.ProjectRoot, "terragrunt", "modules", "libvirt-vm")
	if _, err := os.Stat(modulesDir); os.IsNotExist(err) {
		return fmt.Errorf("terragrunt module not found at %s\n\nPossible causes:\n"+
			"  - Running ucli from the wrong directory\n"+
			"  - The terragrunt/modules/libvirt-vm directory was deleted\n\n"+
			"Solution: Run ucli from the project root directory", modulesDir)
	}

	// Validate VM name if provided
	if opts.Terragrunt.VMName != "" {
		if err := ValidateVMName(opts.Terragrunt.VMName); err != nil {
			return err
		}
	}

	return nil
}

// ValidateVMName checks if a VM name is valid for use as a directory and libvirt domain name.
func ValidateVMName(name string) error {
	// Check for empty name
	if name == "" {
		return fmt.Errorf("VM name cannot be empty")
	}

	// Check length (libvirt has a 64-char limit, filesystem usually 255)
	if len(name) > 64 {
		return fmt.Errorf("VM name too long: %d characters (max 64)", len(name))
	}

	// Check for valid UTF-8
	if !utf8.ValidString(name) {
		return fmt.Errorf("VM name contains invalid characters")
	}

	// Check for path traversal attempts
	if filepath.Clean(name) != name || name == "." || name == ".." {
		return fmt.Errorf("VM name cannot contain path separators or be '.' or '..'")
	}

	// Check for slashes explicitly (filepath.Clean might not catch all cases)
	for _, r := range name {
		if r == '/' || r == '\\' {
			return fmt.Errorf("VM name cannot contain slashes")
		}
	}

	// Check against allowed pattern (lowercase alphanumeric and hyphens)
	if !vmNamePattern.MatchString(name) {
		return fmt.Errorf("VM name must contain only lowercase letters, numbers, and hyphens, "+
			"and cannot start or end with a hyphen: %q", name)
	}

	return nil
}

// Deploy generates the Terragrunt configuration files.
// It creates a new directory under tf/<vm-name>/ with terragrunt.hcl and cloud-init.yaml.
func (g *Generator) Deploy(ctx context.Context, opts *deploy.DeployOptions, progress deploy.ProgressCallback) (*deploy.DeployResult, error) {
	result := &deploy.DeployResult{
		Target:  deploy.TargetTerragrunt,
		Outputs: make(map[string]string),
		Logs:    make([]string, 0),
	}
	start := time.Now()

	tgOpts := opts.Terragrunt

	// Stage 1: Validate (10%)
	progress(deploy.NewProgressEvent(deploy.StageValidating, "Validating configuration...", 10))
	if err := g.Validate(opts); err != nil {
		return g.fail(result, err, start), err
	}

	// Validate Ubuntu image path (warning only, don't fail)
	// Report warning through progress so user sees it
	if warning := g.checkUbuntuImage(tgOpts.UbuntuImage); warning != "" {
		result.Logs = append(result.Logs, warning)
		progress(deploy.NewProgressEventWithDetail(
			deploy.StageValidating,
			"Warning: Ubuntu image issue detected",
			warning,
			15,
		))
	}

	// Stage 2: Determine VM name and create directory
	vmName := tgOpts.VMName
	if vmName == "" {
		vmName = g.generateVMName()
		opts.Terragrunt.VMName = vmName
	}
	result.Outputs["vm_name"] = vmName

	// Ensure tf/ directory and root terragrunt.hcl exist
	tfDir := filepath.Join(opts.ProjectRoot, "tf")
	if err := g.ensureRootConfig(opts.ProjectRoot, tfDir); err != nil {
		return g.fail(result, err, start), err
	}

	// Create machine directory: tf/<vm-name>/
	machineDir := filepath.Join(tfDir, vmName)

	progress(deploy.NewProgressEventWithDetail(
		deploy.StagePreparing,
		"Creating config directory...",
		fmt.Sprintf("tf/%s/", vmName),
		20,
	))

	// Use atomic directory creation to avoid TOCTOU race condition
	// os.Mkdir fails if directory exists, which is what we want
	dirCreated := false
	if err := os.Mkdir(machineDir, 0755); err != nil {
		if os.IsExist(err) {
			// Directory exists - check if it has config files
			if exists, checkErr := g.configExists(machineDir); checkErr != nil {
				return g.fail(result, checkErr, start), checkErr
			} else if exists {
				err := fmt.Errorf("config '%s' already exists at %s\n\nTo regenerate, first delete the directory:\n  rm -rf %s", vmName, machineDir, machineDir)
				return g.fail(result, err, start), err
			}
			// Directory exists but is empty/has no config - we can use it
		} else {
			return g.fail(result, fmt.Errorf("failed to create directory %s: %w", machineDir, err), start), err
		}
	} else {
		// We successfully created the directory
		dirCreated = true
	}

	// Cleanup on failure only if we created the directory
	defer func() {
		if !result.Success && dirCreated {
			_ = os.RemoveAll(machineDir)
		}
	}()

	result.Outputs["config_dir"] = machineDir

	// Stage 3: Generate cloud-init.yaml (50%)
	progress(deploy.NewProgressEventWithDetail(
		deploy.StageCloudInit,
		"Generating cloud-init.yaml...",
		fmt.Sprintf("tf/%s/cloud-init.yaml", vmName),
		50,
	))
	cloudInitPath, err := g.generateCloudInitInDir(opts, machineDir)
	if err != nil {
		return g.fail(result, err, start), err
	}
	result.Outputs["cloud_init_path"] = cloudInitPath

	// Stage 4: Generate terragrunt.hcl (80%)
	progress(deploy.NewProgressEventWithDetail(
		deploy.StagePreparing,
		"Generating terragrunt.hcl...",
		fmt.Sprintf("tf/%s/terragrunt.hcl", vmName),
		80,
	))
	if err := g.writeTerragruntHCL(opts, machineDir); err != nil {
		return g.fail(result, err, start), err
	}
	result.Outputs["terragrunt_path"] = filepath.Join(machineDir, "terragrunt.hcl")

	// Stage 5: Complete (100%)
	progress(deploy.NewProgressEvent(deploy.StageComplete, "Configuration generated!", 100))
	result.Success = true
	result.Duration = time.Since(start)

	// Add helpful outputs
	result.Outputs["next_steps"] = fmt.Sprintf("cd tf/%s && terragrunt init && terragrunt apply", vmName)

	return result, nil
}

// ensureRootConfig ensures the tf/ directory exists with a root terragrunt.hcl.
func (g *Generator) ensureRootConfig(projectRoot, tfDir string) error {
	// Create tf/ directory if needed
	if err := os.MkdirAll(tfDir, 0755); err != nil {
		return fmt.Errorf("failed to create tf directory: %w", err)
	}

	// Check if root terragrunt.hcl exists
	rootHCL := filepath.Join(tfDir, "terragrunt.hcl")
	if _, err := os.Stat(rootHCL); err == nil {
		return nil // Already exists
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check root terragrunt.hcl: %w", err)
	}

	// Calculate relative path from tf/ to terragrunt/modules/libvirt-vm
	relModulePath := "../terragrunt/modules/libvirt-vm"

	// Generate root terragrunt.hcl
	content := fmt.Sprintf(`# =============================================================================
# Root Terragrunt Configuration for VM Deployments
#
# This file is auto-generated by ucli. It provides common configuration
# inherited by all VM configs in subdirectories.
#
# Each VM config (e.g., tf/<vm-name>/terragrunt.hcl) includes this file.
# =============================================================================

# Local values available to all children
locals {
  module_path = "%s"
}

# Configure OpenTofu/Terraform settings
generate "versions" {
  path      = "versions.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<-EOF
    terraform {
      required_version = ">= 1.0"

      required_providers {
        libvirt = {
          source  = "dmacvicar/libvirt"
          version = "~> 0.9.0"
        }
      }
    }
  EOF
}
`, relModulePath)

	if err := os.WriteFile(rootHCL, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write root terragrunt.hcl: %w", err)
	}

	return nil
}

// configExists checks if a config already exists in the directory.
func (g *Generator) configExists(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check directory: %w", err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("path exists but is not a directory: %s", dir)
	}

	// Check for any config files that indicate existing setup
	configFiles := []string{"terragrunt.hcl", "cloud-init.yaml"}
	for _, file := range configFiles {
		path := filepath.Join(dir, file)
		if _, err := os.Stat(path); err == nil {
			return true, nil
		} else if !os.IsNotExist(err) {
			return false, fmt.Errorf("failed to check for %s: %w", file, err)
		}
	}

	// Directory exists but has no config files - safe to use
	return false, nil
}

// checkUbuntuImage validates the Ubuntu image path and returns a warning if invalid.
func (g *Generator) checkUbuntuImage(imagePath string) string {
	if imagePath == "" {
		return "Warning: No Ubuntu image path specified. You'll need to set ubuntu_image_path in terragrunt.hcl"
	}

	// Check if path exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return fmt.Sprintf("Warning: Ubuntu image not found at %s. Download it with:\n"+
			"  wget -O %s https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
			imagePath, imagePath)
	}

	return ""
}

// fail records a failure and returns the result.
func (g *Generator) fail(result *deploy.DeployResult, err error, start time.Time) *deploy.DeployResult {
	result.Success = false
	result.Error = err
	result.Duration = time.Since(start)
	return result
}

// Cleanup is a no-op for the generator (cleanup happens in defer).
func (g *Generator) Cleanup(_ context.Context, _ *deploy.DeployOptions) error {
	return nil
}

// generateVMName generates a unique VM name with timestamp and random suffix.
func (g *Generator) generateVMName() string {
	timestamp := time.Now().Format("0102-1504") // MMDD-HHMM

	// Add random suffix to prevent collisions
	randomBytes := make([]byte, 2)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to just timestamp if random fails
		return fmt.Sprintf("vm-%s", timestamp)
	}
	randomSuffix := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("vm-%s-%s", timestamp, randomSuffix)
}

// InstallInstructions returns installation instructions for terragrunt and opentofu.
func InstallInstructions() string {
	return `Terragrunt and OpenTofu are required to apply the generated configuration.

Install OpenTofu:
  macOS:   brew install opentofu
  Linux:   curl -fsSL https://get.opentofu.org/install-opentofu.sh | sh

Install Terragrunt:
  macOS:   brew install terragrunt
  Linux:   See https://terragrunt.gruntwork.io/docs/getting-started/install/

Install libvirt (Linux only):
  Ubuntu:  sudo apt install qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils
  Fedora:  sudo dnf install @virtualization

Note: libvirt is only available on Linux. macOS users can use Multipass instead.`
}
