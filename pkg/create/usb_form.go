package create

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/huh"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// runUSBOptions prompts for USB/ISO-specific options.
func runUSBOptions() (deploy.USBOptions, error) {
	opts := deploy.DefaultUSBOptions()

	// Check for required tools before showing the form
	if _, err := exec.LookPath("xorriso"); err != nil {
		if err := offerToInstallXorriso(); err != nil {
			return opts, err
		}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Source Ubuntu ISO").
				Description("Path to Ubuntu Server ISO file").
				Placeholder("/path/to/ubuntu-24.04-live-server-amd64.iso").
				Value(&opts.SourceISO).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("source ISO path is required")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Storage Layout").
				Description("Disk partitioning scheme for installation").
				Options(
					huh.NewOption("LVM - Flexible partitions, snapshots, easy resizing", "lvm"),
					huh.NewOption("Direct - Simple partitions, no overhead, full disk access", "direct"),
					huh.NewOption("ZFS - Advanced filesystem, built-in snapshots, compression", "zfs"),
				).
				Value(&opts.StorageLayout),
		).Title("Bootable ISO Options"),
	).WithTheme(tui.Theme())

	if err := form.Run(); err != nil {
		return opts, fmt.Errorf("USB options cancelled: %w", err)
	}

	return opts, nil
}

// offerToInstallXorriso checks for xorriso and offers to install it.
func offerToInstallXorriso() error {
	fmt.Println()
	fmt.Println(errorStyle.Render("  Missing Required Tool"))
	fmt.Println()
	fmt.Println("  xorriso is required to create bootable ISOs but was not found.")
	fmt.Println()

	// Determine install command based on platform
	var installCmd string
	var installArgs []string
	var manualInstructions string

	switch runtime.GOOS {
	case "darwin":
		// Check if brew is available
		if _, err := exec.LookPath("brew"); err == nil {
			installCmd = "brew"
			installArgs = []string{"install", "xorriso"}
			manualInstructions = "brew install xorriso"
		} else {
			fmt.Println("  Homebrew is not installed. Please install it first:")
			fmt.Println("    /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
			fmt.Println()
			fmt.Println("  Then run:")
			fmt.Println("    brew install xorriso")
			fmt.Println()
			return fmt.Errorf("xorriso not found - please install Homebrew and xorriso")
		}
	case "linux":
		// Try to detect package manager
		if _, err := exec.LookPath("apt"); err == nil {
			installCmd = "sudo"
			installArgs = []string{"apt", "install", "-y", "xorriso"}
			manualInstructions = "sudo apt install xorriso"
		} else if _, err := exec.LookPath("dnf"); err == nil {
			installCmd = "sudo"
			installArgs = []string{"dnf", "install", "-y", "xorriso"}
			manualInstructions = "sudo dnf install xorriso"
		} else if _, err := exec.LookPath("pacman"); err == nil {
			installCmd = "sudo"
			installArgs = []string{"pacman", "-S", "--noconfirm", "libisoburn"}
			manualInstructions = "sudo pacman -S libisoburn"
		} else {
			fmt.Println("  Could not detect package manager. Please install xorriso manually.")
			fmt.Println()
			return fmt.Errorf("xorriso not found - please install it manually")
		}
	default:
		fmt.Println("  Unsupported platform. Please install xorriso manually.")
		fmt.Println()
		return fmt.Errorf("xorriso not found - unsupported platform")
	}

	// Ask user if they want to install
	var install bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Install xorriso?").
				Description(fmt.Sprintf("Run: %s", manualInstructions)).
				Value(&install),
		),
	).WithTheme(tui.Theme())

	if err := form.Run(); err != nil {
		return fmt.Errorf("cancelled")
	}

	if !install {
		fmt.Println()
		fmt.Println("  To install manually, run:")
		fmt.Println("    " + manualInstructions)
		fmt.Println()
		return fmt.Errorf("xorriso not found - please install it and try again")
	}

	// Run the install command
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Installing xorriso..."))
	fmt.Println()

	cmd := exec.Command(installCmd, installArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("  Installation failed"))
		fmt.Println()
		fmt.Println("  Please install manually:")
		fmt.Println("    " + manualInstructions)
		fmt.Println()
		return fmt.Errorf("failed to install xorriso: %w", err)
	}

	// Verify installation
	if _, err := exec.LookPath("xorriso"); err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("  Installation completed but xorriso not found in PATH"))
		fmt.Println()
		return fmt.Errorf("xorriso installed but not found in PATH")
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  ✓ xorriso installed successfully"))
	fmt.Println()

	return nil
}
