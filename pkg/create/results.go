package create

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy/usb"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// printDeploymentResults prints the deployment results to the terminal (outside alt-screen).
func printDeploymentResults(result *deploy.DeployResult, deployerName string) {
	fmt.Println()

	if result == nil {
		fmt.Println(errorStyle.Render("Deployment did not complete."))
		return
	}

	if result.Success {
		// Success banner
		fmt.Println(successStyle.Render("  Deployment Complete!"))
		fmt.Println()
		fmt.Printf("  Duration: %s\n", result.Duration.Round(time.Second))
		fmt.Println()

		// Show outputs
		if len(result.Outputs) > 0 {
			fmt.Println(subtitleStyle.Render("  Details:"))

			// Order outputs nicely based on target type
			var orderedKeys []string
			switch result.Target {
			case deploy.TargetUSB:
				orderedKeys = []string{"iso_path", "iso_size", "storage_layout", "source_iso"}
			case deploy.TargetTerraform:
				orderedKeys = []string{"vm_name", "ip", "user", "ssh_command", "console_command", "vnc_port", "cloud_init_path"}
			default:
				orderedKeys = []string{"vm_name", "ip", "user", "ssh_command", "multipass_shell", "cloud_init_path"}
			}
			printedKeys := make(map[string]bool)

			for _, key := range orderedKeys {
				if value, ok := result.Outputs[key]; ok {
					label := formatOutputLabel(key)
					fmt.Printf("    %s: %s\n", label, value)
					printedKeys[key] = true
				}
			}

			// Print any remaining outputs
			for key, value := range result.Outputs {
				if !printedKeys[key] {
					label := formatOutputLabel(key)
					fmt.Printf("    %s: %s\n", label, value)
				}
			}
		}

		// Show helpful commands based on target type
		if vmName, ok := result.Outputs["vm_name"]; ok {
			fmt.Println()
			fmt.Println(dimStyle.Render("  To access the VM:"))
			switch result.Target {
			case deploy.TargetTerraform:
				if ip, ok := result.Outputs["ip"]; ok && ip != "" && ip != "pending" {
					user := result.Outputs["user"]
					if user == "" {
						user = "ubuntu"
					}
					fmt.Printf("    ssh %s@%s\n", user, ip)
				}
				fmt.Printf("    virsh console %s\n", vmName)
			default:
				fmt.Printf("    multipass shell %s\n", vmName)
			}
		}

		if isoPath, ok := result.Outputs["iso_path"]; ok {
			showUSBWriteOptions(isoPath)
		}

		// Show any warnings from logs
		if len(result.Logs) > 0 {
			fmt.Println()
			fmt.Println(dimStyle.Render("  Notes:"))
			for _, log := range result.Logs {
				fmt.Printf("    %s\n", log)
			}
		}
	} else {
		// Failure banner
		fmt.Println(errorStyle.Render("  Deployment Failed"))
		fmt.Println()
		if result.Error != nil {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		fmt.Println()
		fmt.Println(dimStyle.Render("  Run with --verbose for more details"))
	}

	fmt.Println()
}

// formatOutputLabel formats an output key as a human-readable label.
func formatOutputLabel(key string) string {
	switch key {
	case "vm_name":
		return "VM Name"
	case "ip":
		return "IP Address"
	case "user":
		return "Username"
	case "ssh_command":
		return "SSH Command"
	case "multipass_shell":
		return "Shell Command"
	case "cloud_init_path":
		return "Cloud-Init"
	case "iso_path":
		return "ISO Path"
	case "source_iso":
		return "Source ISO"
	case "storage_layout":
		return "Storage"
	case "iso_size":
		return "ISO Size"
	case "console_command":
		return "Console"
	case "vnc_port":
		return "VNC Port"
	default:
		// Use cases.Title instead of deprecated strings.Title
		caser := cases.Title(language.English)
		return caser.String(strings.ReplaceAll(key, "_", " "))
	}
}

// showUSBWriteOptions detects USB devices and shows options to write the ISO.
func showUSBWriteOptions(isoPath string) {
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Write to USB"))
	fmt.Println()

	// Detect USB devices
	devices, err := usb.DetectDevices()
	if err != nil {
		fmt.Println(dimStyle.Render("  Could not detect USB devices: " + err.Error()))
		fmt.Println()
		showManualDDCommand(isoPath)
		return
	}

	if len(devices) == 0 {
		fmt.Println(dimStyle.Render("  No USB devices detected."))
		fmt.Println()
		showManualDDCommand(isoPath)
		return
	}

	// Build device options for selection
	options := make([]huh.Option[string], len(devices)+1)
	for i, dev := range devices {
		label := fmt.Sprintf("%s (%s) - %s", dev.Name, dev.Path, dev.Size)
		options[i] = huh.NewOption(label, dev.Path)
	}
	options[len(devices)] = huh.NewOption("Skip - show manual command", "skip")

	var selectedDevice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select USB device").
				Description("The ISO will be written to this device (all data will be erased)").
				Options(options...).
				Value(&selectedDevice),
		),
	).WithTheme(tui.Theme())

	if err := form.Run(); err != nil {
		showManualDDCommand(isoPath)
		return
	}

	if selectedDevice == "skip" || selectedDevice == "" {
		showManualDDCommand(isoPath)
		return
	}

	// Generate the commands for the selected device
	var unmountCmd, ddCmd, fullCmd string
	if runtime.GOOS == "darwin" {
		// macOS: unmount first, then use rdisk for raw device access (faster)
		diskID := strings.TrimPrefix(selectedDevice, "/dev/")
		rawDevice := strings.Replace(selectedDevice, "/dev/disk", "/dev/rdisk", 1)
		unmountCmd = fmt.Sprintf("diskutil unmountDisk %s", diskID)
		ddCmd = fmt.Sprintf("sudo dd if=%s of=%s bs=4m status=progress", isoPath, rawDevice)
		fullCmd = fmt.Sprintf("%s && %s", unmountCmd, ddCmd)
	} else {
		// Linux: unmount partitions, then write
		unmountCmd = fmt.Sprintf("sudo umount %s?* 2>/dev/null", selectedDevice)
		ddCmd = fmt.Sprintf("sudo dd if=%s of=%s bs=4M status=progress conv=fsync", isoPath, selectedDevice)
		fullCmd = fmt.Sprintf("%s; %s", unmountCmd, ddCmd)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  Commands to write ISO:"))
	fmt.Println()
	fmt.Println(dimStyle.Render("    # Step 1: Unmount the disk"))
	fmt.Printf("    %s\n", unmountCmd)
	fmt.Println()
	fmt.Println(dimStyle.Render("    # Step 2: Write the ISO"))
	fmt.Printf("    %s\n", ddCmd)
	fmt.Println()

	// Try to copy the combined command to clipboard
	if copyToClipboard(fullCmd) {
		fmt.Println(dimStyle.Render("  ✓ Combined command copied to clipboard"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    %s", fullCmd)))
	} else {
		fmt.Println(dimStyle.Render("  Run the commands above in order"))
	}

	fmt.Println()
	fmt.Println(errorStyle.Render("  ⚠ WARNING: This will erase ALL data on " + selectedDevice))
	fmt.Println()
}

// showManualDDCommand shows the generic dd command template.
func showManualDDCommand(isoPath string) {
	fmt.Println(dimStyle.Render("  To write to USB manually:"))
	fmt.Println()
	if runtime.GOOS == "darwin" {
		fmt.Println("    # List disks to find your USB device:")
		fmt.Println("    diskutil list")
		fmt.Println()
		fmt.Println("    # Unmount the disk (replace N with your disk number):")
		fmt.Println("    diskutil unmountDisk diskN")
		fmt.Println()
		fmt.Println("    # Write ISO (replace N with your disk number):")
		fmt.Printf("    sudo dd if=%s of=/dev/rdiskN bs=4m status=progress\n", isoPath)
	} else {
		fmt.Println("    # List disks to find your USB device:")
		fmt.Println("    lsblk")
		fmt.Println()
		fmt.Println("    # Unmount the disk (replace X with your disk letter):")
		fmt.Println("    sudo umount /dev/sdX?*")
		fmt.Println()
		fmt.Println("    # Write ISO (replace X with your disk letter):")
		fmt.Printf("    sudo dd if=%s of=/dev/sdX bs=4M status=progress conv=fsync\n", isoPath)
	}
	fmt.Println()
}

// copyToClipboard attempts to copy text to the system clipboard.
func copyToClipboard(text string) bool {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return false
		}
	default:
		return false
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run() == nil
}
