package iso

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StorageLayout represents the disk partitioning scheme.
type StorageLayout string

const (
	StorageLVM    StorageLayout = "lvm"
	StorageDirect StorageLayout = "direct"
	StorageZFS    StorageLayout = "zfs"
)

// ISOOptions configures the ISO generation process.
type ISOOptions struct {
	// SourceISO is the path to the source Ubuntu ISO file (required).
	SourceISO string

	// OutputPath is the path for the generated ISO file.
	// If empty, defaults to <source-dir>/ubuntu-autoinstall.iso
	OutputPath string

	// UbuntuVersion specifies the Ubuntu version of the source ISO.
	// Supported values: "22.04", "24.04"
	UbuntuVersion string

	// StorageLayout specifies the disk partitioning scheme.
	// Defaults to "lvm" if not specified.
	StorageLayout StorageLayout

	// Timezone for the installed system. Defaults to "UTC".
	Timezone string

	// Locale for the installed system. Defaults to "en_US.UTF-8".
	Locale string
}

// DefaultOptions returns ISOOptions with sensible defaults.
func DefaultOptions() *ISOOptions {
	return &ISOOptions{
		UbuntuVersion: "24.04",
		StorageLayout: StorageLVM,
		Timezone:      "UTC",
		Locale:        "en_US.UTF-8",
	}
}

// Validate checks that all required options are set and valid.
func (o *ISOOptions) Validate() error {
	// Check source ISO
	if o.SourceISO == "" {
		return fmt.Errorf("source ISO path is required")
	}

	// Check source ISO exists
	info, err := os.Stat(o.SourceISO)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source ISO not found: %s", o.SourceISO)
		}
		return fmt.Errorf("failed to access source ISO: %w", err)
	}

	// Check it's a file, not a directory
	if info.IsDir() {
		return fmt.Errorf("source ISO path is a directory, not a file: %s", o.SourceISO)
	}

	// Check file extension
	if !strings.HasSuffix(strings.ToLower(o.SourceISO), ".iso") {
		return fmt.Errorf("source file does not have .iso extension: %s", o.SourceISO)
	}

	// Validate Ubuntu version
	switch o.UbuntuVersion {
	case "22.04", "24.04":
		// Valid versions
	case "":
		o.UbuntuVersion = "24.04" // Default
	default:
		return fmt.Errorf("unsupported Ubuntu version: %s (supported: 22.04, 24.04)", o.UbuntuVersion)
	}

	// Validate storage layout
	switch o.StorageLayout {
	case StorageLVM, StorageDirect, StorageZFS:
		// Valid layouts
	case "":
		o.StorageLayout = StorageLVM // Default
	default:
		return fmt.Errorf("unsupported storage layout: %s (supported: lvm, direct, zfs)", o.StorageLayout)
	}

	// Set default output path if not specified
	if o.OutputPath == "" {
		dir := filepath.Dir(o.SourceISO)
		o.OutputPath = filepath.Join(dir, "ubuntu-autoinstall.iso")
	}

	// Set defaults for optional fields
	if o.Timezone == "" {
		o.Timezone = "UTC"
	}
	if o.Locale == "" {
		o.Locale = "en_US.UTF-8"
	}

	return nil
}

// OutputDir returns the directory where the output ISO will be created.
func (o *ISOOptions) OutputDir() string {
	return filepath.Dir(o.OutputPath)
}

// OutputFilename returns just the filename of the output ISO.
func (o *ISOOptions) OutputFilename() string {
	return filepath.Base(o.OutputPath)
}
