package iso

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
)

// Builder creates bootable Ubuntu ISOs with cloud-init configuration.
type Builder struct {
	projectRoot string
	tools       *ToolChain
	verbose     bool
}

// NewBuilder creates a new ISO Builder.
func NewBuilder(projectRoot string) *Builder {
	return &Builder{
		projectRoot: projectRoot,
		tools:       NewToolChain(),
		verbose:     false,
	}
}

// SetVerbose enables verbose output during build.
func (b *Builder) SetVerbose(verbose bool) {
	b.verbose = verbose
}

// Build creates a bootable ISO with embedded cloud-init configuration.
func (b *Builder) Build(cfg *config.FullConfig, opts *ISOOptions) error {
	// Step 1: Validate tools
	if err := b.tools.Detect(); err != nil {
		return fmt.Errorf("required tools not available: %w\n%s", err, b.tools.InstallInstructions())
	}

	// Step 2: Validate options
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Step 3: Create temporary work directory in project .tmp
	tmpBase := filepath.Join(b.projectRoot, ".tmp")
	if err := os.MkdirAll(tmpBase, 0755); err != nil {
		return fmt.Errorf("failed to create .tmp directory: %w", err)
	}
	workDir, err := os.MkdirTemp(tmpBase, "iso-*")
	if err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	b.log("Work directory: %s", workDir)

	// Step 4: Extract ISO
	extractDir := filepath.Join(workDir, "iso")
	if err := b.extractISO(opts.SourceISO, extractDir); err != nil {
		return fmt.Errorf("ISO extraction failed: %w", err)
	}

	// Step 5: Generate autoinstall configuration
	generator := NewAutoInstallGenerator(b.projectRoot)
	userData, err := generator.Generate(cfg, opts)
	if err != nil {
		return fmt.Errorf("autoinstall generation failed: %w", err)
	}

	// Step 6: Inject configuration
	if err := b.injectConfiguration(extractDir, userData, generator.GenerateMetaData()); err != nil {
		return fmt.Errorf("configuration injection failed: %w", err)
	}

	// Step 7: Modify boot configuration
	if err := b.modifyGrubConfig(extractDir); err != nil {
		return fmt.Errorf("boot config modification failed: %w", err)
	}

	// Step 8: Ensure output directory exists
	if err := os.MkdirAll(opts.OutputDir(), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Step 9: Repack ISO
	if err := b.repackISO(extractDir, opts); err != nil {
		return fmt.Errorf("ISO repacking failed: %w", err)
	}

	b.log("ISO created: %s", opts.OutputPath)

	return nil
}

// extractISO extracts the source ISO contents.
func (b *Builder) extractISO(isoPath, extractDir string) error {
	b.log("Extracting ISO: %s", isoPath)

	// Create extract directory
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	// Use xorriso to extract ISO contents
	args := []string{
		"-osirrox", "on",
		"-indev", isoPath,
		"-extract", "/", extractDir,
	}

	cmd := exec.Command(b.tools.XorrisoPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xorriso extraction failed: %w\n%s", err, string(output))
	}

	// Make all files writable (xorriso preserves read-only permissions from ISO)
	// Use chmod -R which can handle read-only directories better than filepath.Walk
	chmodCmd := exec.Command("chmod", "-R", "u+w", extractDir)
	if chmodOutput, err := chmodCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fix permissions: %w\n%s", err, string(chmodOutput))
	}

	return nil
}

// injectConfiguration injects the autoinstall configuration into the ISO.
func (b *Builder) injectConfiguration(extractDir string, userData, metaData []byte) error {
	b.log("Injecting autoinstall configuration")

	// Create nocloud directory for cloud-init datasource
	nocloudDir := filepath.Join(extractDir, "nocloud")
	if err := os.MkdirAll(nocloudDir, 0755); err != nil {
		return fmt.Errorf("failed to create nocloud directory: %w", err)
	}

	// Write user-data
	userDataPath := filepath.Join(nocloudDir, "user-data")
	if err := os.WriteFile(userDataPath, userData, 0644); err != nil {
		return fmt.Errorf("failed to write user-data: %w", err)
	}

	// Write meta-data (empty but required)
	metaDataPath := filepath.Join(nocloudDir, "meta-data")
	if err := os.WriteFile(metaDataPath, metaData, 0644); err != nil {
		return fmt.Errorf("failed to write meta-data: %w", err)
	}

	return nil
}

// modifyGrubConfig modifies GRUB configuration to enable autoinstall.
func (b *Builder) modifyGrubConfig(extractDir string) error {
	b.log("Modifying GRUB configuration")

	// Find grub.cfg - location varies between Ubuntu versions
	grubPaths := []string{
		filepath.Join(extractDir, "boot", "grub", "grub.cfg"),
		filepath.Join(extractDir, "EFI", "boot", "grub.cfg"),
	}

	var grubCfgPath string
	for _, path := range grubPaths {
		if _, err := os.Stat(path); err == nil {
			grubCfgPath = path
			break
		}
	}

	if grubCfgPath == "" {
		return fmt.Errorf("grub.cfg not found in expected locations")
	}

	content, err := os.ReadFile(grubCfgPath)
	if err != nil {
		return fmt.Errorf("failed to read grub.cfg: %w", err)
	}

	// Add autoinstall parameter to kernel command line
	// The pattern "---" separates kernel arguments from init arguments
	modified := string(content)

	// Add autoinstall datasource parameter before "---"
	// Note: semicolon must be escaped with backslash for GRUB
	autoinstallParam := "autoinstall ds=nocloud\\;s=/cdrom/nocloud/"
	if !strings.Contains(modified, "autoinstall") {
		modified = strings.ReplaceAll(modified, "---", autoinstallParam+" ---")
	}

	if err := os.WriteFile(grubCfgPath, []byte(modified), 0644); err != nil {
		return fmt.Errorf("failed to write grub.cfg: %w", err)
	}

	// Also modify the loopback.cfg if it exists (used for some boot scenarios)
	loopbackPath := filepath.Join(extractDir, "boot", "grub", "loopback.cfg")
	if _, err := os.Stat(loopbackPath); err == nil {
		content, err := os.ReadFile(loopbackPath)
		if err == nil {
			modified := strings.ReplaceAll(string(content), "---", autoinstallParam+" ---")
			if err := os.WriteFile(loopbackPath, []byte(modified), 0644); err != nil {
				b.log("Warning: failed to modify loopback.cfg: %v", err)
			}
		}
	}

	return nil
}

// repackISO creates the final bootable ISO.
func (b *Builder) repackISO(extractDir string, opts *ISOOptions) error {
	b.log("Repacking ISO: %s", opts.OutputPath)

	// Build xorriso command for creating bootable ISO
	// Volume ID must be max 32 chars, uppercase, limited charset for ISO 9660 compliance
	volumeID := fmt.Sprintf("UBUNTU_AUTOINSTALL_%s", strings.ReplaceAll(opts.UbuntuVersion, ".", "_"))
	if len(volumeID) > 32 {
		volumeID = volumeID[:32]
	}

	args := []string{
		"-as", "mkisofs",
		"-r",              // Rock Ridge extensions
		"-V", volumeID,    // Volume ID
		"-J",              // Joliet extensions
		"-joliet-long",    // Allow long Joliet names
		"-l",              // Allow full 31-character filenames
		"-iso-level", "3", // ISO 9660 level 3
	}

	// Check what boot files exist in the extracted ISO
	eltoritoImg := filepath.Join(extractDir, "boot", "grub", "i386-pc", "eltorito.img")
	bootHybridImg := filepath.Join(extractDir, "boot", "grub", "i386-pc", "boot_hybrid.img")
	efiBootImg := filepath.Join(extractDir, "boot", "grub", "efi.img")

	// Log what we find
	b.log("Checking boot files:")
	b.log("  eltorito.img: %v", fileExists(eltoritoImg))
	b.log("  boot_hybrid.img: %v", fileExists(bootHybridImg))
	b.log("  efi.img: %v", fileExists(efiBootImg))

	// Add BIOS boot support (only if both required files exist)
	if fileExists(eltoritoImg) && fileExists(bootHybridImg) {
		args = append(args,
			"-partition_offset", "16",
			"-b", "boot/grub/i386-pc/eltorito.img",
			"-c", "boot.catalog",
			"-no-emul-boot",
			"-boot-load-size", "4",
			"-boot-info-table",
			"--grub2-boot-info",
			"--grub2-mbr", bootHybridImg,
		)
	} else if fileExists(eltoritoImg) {
		// BIOS boot without hybrid MBR
		args = append(args,
			"-b", "boot/grub/i386-pc/eltorito.img",
			"-c", "boot.catalog",
			"-no-emul-boot",
			"-boot-load-size", "4",
			"-boot-info-table",
		)
	}

	// Add EFI boot support
	if fileExists(efiBootImg) {
		args = append(args,
			"-eltorito-alt-boot",
			"-e", "boot/grub/efi.img",
			"-no-emul-boot",
		)
	}

	// Add output path and source directory
	args = append(args, "-o", opts.OutputPath, extractDir)

	b.log("Running xorriso with args: %v", args)

	cmd := exec.Command(b.tools.XorrisoPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xorriso repacking failed: %w\n%s", err, string(output))
	}

	return nil
}

// fileExists returns true if the file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// log prints a message if verbose mode is enabled.
func (b *Builder) log(format string, args ...interface{}) {
	if b.verbose {
		fmt.Printf(format+"\n", args...)
	}
}

// CheckTools verifies that required tools are available.
func (b *Builder) CheckTools() error {
	return b.tools.Detect()
}

// ToolsAvailable returns true if all required tools are installed.
func (b *Builder) ToolsAvailable() bool {
	return b.tools.Detect() == nil
}

// InstallInstructions returns instructions for installing required tools.
func (b *Builder) InstallInstructions() string {
	return b.tools.InstallInstructions()
}
