// Package deploy provides deployment abstraction for different targets.
package deploy

import (
	"context"
	"time"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
)

// DeploymentTarget identifies the deployment target type.
type DeploymentTarget string

const (
	TargetMultipass  DeploymentTarget = "multipass"
	TargetUSB        DeploymentTarget = "usb"
	TargetTerragrunt DeploymentTarget = "terragrunt"
	TargetConfigOnly DeploymentTarget = "config"
)

// String returns the string representation of the target.
func (t DeploymentTarget) String() string {
	return string(t)
}

// DisplayName returns a human-readable name for the target.
func (t DeploymentTarget) DisplayName() string {
	switch t {
	case TargetMultipass:
		return "Multipass VM"
	case TargetUSB:
		return "Bootable USB"
	case TargetTerragrunt:
		return "Terragrunt/libvirt"
	case TargetConfigOnly:
		return "Config Only"
	default:
		return string(t)
	}
}

// Description returns a description of the target.
func (t DeploymentTarget) Description() string {
	switch t {
	case TargetMultipass:
		return "Create a local VM using Multipass for testing"
	case TargetUSB:
		return "Generate bootable ISO and optionally write to USB"
	case TargetTerragrunt:
		return "Generate Terragrunt/OpenTofu config for libvirt VM"
	case TargetConfigOnly:
		return "Generate config files only (no deployment)"
	default:
		return ""
	}
}

// AllTargets returns all available deployment targets.
func AllTargets() []DeploymentTarget {
	return []DeploymentTarget{
		TargetTerragrunt,
		TargetMultipass,
		TargetUSB,
	}
}

// DeployOptions contains configuration for deployment.
type DeployOptions struct {
	// Common options
	ProjectRoot   string
	Config        *config.FullConfig
	CloudInitPath string // Path to generated cloud-init.yaml

	// Multipass-specific options
	Multipass MultipassOptions

	// USB-specific options
	USB USBOptions

	// Terragrunt-specific options
	Terragrunt TerragruntOptions
}

// MultipassOptions contains Multipass-specific deployment options.
type MultipassOptions struct {
	VMName        string
	CPUs          int
	MemoryMB      int
	DiskGB        int
	UbuntuVersion string // e.g., "24.04"
	KeepOnFailure bool   // Keep VM for debugging on failure
}

// DefaultMultipassOptions returns sensible defaults for Multipass.
func DefaultMultipassOptions() MultipassOptions {
	return MultipassOptions{
		VMName:        "",   // Will be auto-generated
		CPUs:          2,
		MemoryMB:      2048,
		DiskGB:        20,
		UbuntuVersion: "24.04",
		KeepOnFailure: false,
	}
}

// USBOptions contains USB/ISO-specific deployment options.
type USBOptions struct {
	SourceISO     string // Path to source Ubuntu ISO
	OutputISO     string // Path for output ISO (optional)
	DevicePath    string // USB device path (e.g., /dev/sdb)
	UbuntuVersion string // e.g., "24.04"
	StorageLayout string // "lvm", "direct", or "zfs"
}

// DefaultUSBOptions returns sensible defaults for USB.
func DefaultUSBOptions() USBOptions {
	return USBOptions{
		UbuntuVersion: "24.04",
		StorageLayout: "lvm",
	}
}

// TerragruntOptions contains Terragrunt/OpenTofu generation options.
type TerragruntOptions struct {
	VMName      string
	CPUs        int
	MemoryMB    int
	DiskGB      int
	Autostart   bool   // Start VM automatically on host boot
	LibvirtURI  string // Libvirt connection URI (e.g., "qemu:///system")
	StoragePool string // Libvirt storage pool name
	NetworkName string // Libvirt network name
	UbuntuImage string // Path to Ubuntu cloud image
}

// DefaultTerragruntOptions returns sensible defaults for Terragrunt.
func DefaultTerragruntOptions() TerragruntOptions {
	return TerragruntOptions{
		CPUs:        2,
		MemoryMB:    2048,
		DiskGB:      20,
		LibvirtURI:  "qemu:///system",
		StoragePool: "default",
		NetworkName: "default",
		UbuntuImage: "/var/lib/libvirt/images/noble-server-cloudimg-amd64.img",
	}
}

// DeployResult represents the outcome of a deployment.
type DeployResult struct {
	Success  bool
	Target   DeploymentTarget
	Duration time.Duration
	Outputs  map[string]string // Target-specific outputs (IP, VM name, etc.)
	Logs     []string          // Captured log lines
	Error    error
}

// Deployer executes deployment to a target.
type Deployer interface {
	// Name returns a human-readable name for the deployer.
	Name() string

	// Target returns the deployment target type.
	Target() DeploymentTarget

	// Validate checks if deployment can proceed with the given options.
	Validate(opts *DeployOptions) error

	// Deploy executes the deployment with progress updates.
	Deploy(ctx context.Context, opts *DeployOptions, progress ProgressCallback) (*DeployResult, error)

	// Cleanup performs cleanup on failure (optional, may be no-op).
	Cleanup(ctx context.Context, opts *DeployOptions) error
}
