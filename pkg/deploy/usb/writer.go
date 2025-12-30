// Package usb provides USB device detection and writing functionality.
package usb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/iso"
)

// Deployer implements deploy.Deployer for USB/ISO creation.
type Deployer struct {
	projectRoot string
	verbose     bool
}

// New creates a new USB deployer.
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
	return "Bootable ISO"
}

// Target returns the deployment target type.
func (d *Deployer) Target() deploy.DeploymentTarget {
	return deploy.TargetUSB
}

// Validate checks if deployment can proceed.
func (d *Deployer) Validate(opts *deploy.DeployOptions) error {
	// Check project root exists
	if d.projectRoot == "" {
		return fmt.Errorf("project root is required")
	}

	// Check config is provided
	if opts.Config == nil {
		return fmt.Errorf("configuration is required")
	}

	// Check source ISO is provided
	if opts.USB.SourceISO == "" {
		return fmt.Errorf("source ISO path is required")
	}

	// Check source ISO exists
	if _, err := os.Stat(opts.USB.SourceISO); os.IsNotExist(err) {
		return fmt.Errorf("source ISO not found: %s", opts.USB.SourceISO)
	}

	// Check ISO tools are available
	builder := iso.NewBuilder(d.projectRoot)
	if err := builder.CheckTools(); err != nil {
		return err
	}

	return nil
}

// Deploy executes the USB/ISO deployment.
func (d *Deployer) Deploy(ctx context.Context, opts *deploy.DeployOptions, progress deploy.ProgressCallback) (*deploy.DeployResult, error) {
	result := &deploy.DeployResult{
		Target:  deploy.TargetUSB,
		Outputs: make(map[string]string),
		Logs:    make([]string, 0),
	}
	start := time.Now()

	// Stage 1: Validate
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageValidating,
		"Validating configuration...",
		"xorriso --version",
		5,
	))
	if err := d.Validate(opts); err != nil {
		progress(deploy.NewErrorEventWithDetail("Validation failed", err.Error()))
		return d.fail(result, err, start), err
	}

	// Stage 2: Build ISO with progress
	builder := iso.NewBuilder(d.projectRoot)
	builder.SetVerbose(d.verbose)

	// Determine output path by appending hostname and timestamp to source ISO name
	outputPath := opts.USB.OutputISO
	if outputPath == "" {
		hostname := "server"
		if opts.Config != nil && opts.Config.Hostname != "" {
			hostname = opts.Config.Hostname
		}
		timestamp := time.Now().Format("200601021504")

		// Get source ISO base name and remove .iso extension
		sourceBase := filepath.Base(opts.USB.SourceISO)
		sourceName := strings.TrimSuffix(sourceBase, ".iso")
		sourceName = strings.TrimSuffix(sourceName, ".ISO")

		// Append hostname, timestamp, and autoinstall suffix
		outputName := fmt.Sprintf("%s-%s-%s-autoinstall.iso", sourceName, hostname, timestamp)
		outputPath = filepath.Join(d.projectRoot, "output", outputName)
	}

	// Build ISO options
	isoOpts := &iso.ISOOptions{
		SourceISO:     opts.USB.SourceISO,
		OutputPath:    outputPath,
		UbuntuVersion: opts.USB.UbuntuVersion,
		StorageLayout: iso.StorageLayout(opts.USB.StorageLayout),
		Timezone:      "UTC",
		Locale:        "en_US.UTF-8",
	}

	// Build ISO with progress callbacks
	if err := d.buildISOWithProgress(ctx, builder, opts.Config, isoOpts, progress); err != nil {
		progress(deploy.NewErrorEventWithDetail("ISO build failed", err.Error()))
		return d.fail(result, err, start), err
	}

	result.Outputs["iso_path"] = outputPath
	result.Outputs["source_iso"] = opts.USB.SourceISO
	result.Outputs["storage_layout"] = opts.USB.StorageLayout

	// Get file size
	if info, err := os.Stat(outputPath); err == nil {
		result.Outputs["iso_size"] = formatSize(info.Size())
	}

	// Success
	progress(deploy.NewProgressEvent(deploy.StageComplete, "ISO created successfully!", 100))
	result.Success = true
	result.Duration = time.Since(start)

	return result, nil
}

// buildISOWithProgress builds the ISO with progress callbacks.
func (d *Deployer) buildISOWithProgress(ctx context.Context, builder *iso.Builder, cfg *config.FullConfig, opts *iso.ISOOptions, progress deploy.ProgressCallback) error {
	// Stage: Config generation
	progress(deploy.NewProgressEventWithDetail(
		deploy.StageConfig,
		"Generating autoinstall configuration...",
		fmt.Sprintf("Storage: %s, Timezone: %s", opts.StorageLayout, opts.Timezone),
		15,
	))

	// Check for cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Stage: Extract ISO
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageInstalling,
		"Extracting source ISO...",
		fmt.Sprintf("xorriso -osirrox on -indev %s", filepath.Base(opts.SourceISO)),
		25,
	))

	// Check for cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Stage: Injecting config
	progress(deploy.NewProgressEventWithDetail(
		deploy.StageInstalling,
		"Injecting autoinstall configuration...",
		"Writing user-data, meta-data, grub.cfg",
		50,
	))

	// Stage: Repacking ISO
	progress(deploy.NewProgressEventWithCommand(
		deploy.StageInstalling,
		"Creating bootable ISO...",
		fmt.Sprintf("xorriso -as mkisofs -o %s", filepath.Base(opts.OutputPath)),
		70,
	))

	// Actually build the ISO
	if err := builder.Build(cfg, opts); err != nil {
		return fmt.Errorf("failed to build ISO: %w", err)
	}

	// Stage: Verification
	progress(deploy.NewProgressEventWithDetail(
		deploy.StageVerifying,
		"Verifying ISO...",
		opts.OutputPath,
		90,
	))

	// Verify output exists
	if _, err := os.Stat(opts.OutputPath); os.IsNotExist(err) {
		return fmt.Errorf("ISO was not created: %s", opts.OutputPath)
	}

	return nil
}

// Cleanup cleans up resources on failure.
func (d *Deployer) Cleanup(ctx context.Context, opts *deploy.DeployOptions) error {
	// Nothing to clean up for ISO generation
	// The temporary work directory is cleaned up by the iso.Builder
	return nil
}

// fail records a failure and returns the result.
func (d *Deployer) fail(result *deploy.DeployResult, err error, start time.Time) *deploy.DeployResult {
	result.Success = false
	result.Error = err
	result.Duration = time.Since(start)
	return result
}

// InstallInstructions returns installation instructions for required tools.
func InstallInstructions() string {
	return `Required tools for ISO generation:

macOS:
  brew install xorriso

Linux (Debian/Ubuntu):
  sudo apt install xorriso

Linux (Fedora/RHEL):
  sudo dnf install xorriso`
}
