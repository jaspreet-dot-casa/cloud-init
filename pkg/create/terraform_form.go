package create

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy/terraform"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// runTerraformOptions prompts for Terraform-specific options.
func runTerraformOptions() (deploy.TerraformOptions, error) {
	opts := deploy.DefaultTerraformOptions()

	// Check if we're on Linux
	if runtime.GOOS != "linux" {
		fmt.Println()
		fmt.Println(warningStyle.Render(" Note: Terraform/libvirt requires Linux with KVM support."))
		fmt.Println(dimStyle.Render("       macOS users should use Multipass for local VMs."))
		fmt.Println()

		var proceed bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Continue anyway?").
					Description("You can still generate terraform files for use on a Linux machine").
					Value(&proceed),
			),
		).WithTheme(tui.Theme())

		if err := confirmForm.Run(); err != nil {
			return opts, err
		}
		if !proceed {
			return opts, fmt.Errorf("cancelled: Terraform/libvirt requires Linux")
		}
	}

	// Check if terraform is installed
	if _, err := exec.LookPath("terraform"); err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render(" Terraform is not installed."))
		fmt.Println()
		fmt.Println(terraform.InstallInstructions())
		fmt.Println()

		var proceed bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Continue without terraform?").
					Description("You can install terraform later and run the deployment manually").
					Value(&proceed),
			),
		).WithTheme(tui.Theme())

		if err := confirmForm.Run(); err != nil {
			return opts, err
		}
		if !proceed {
			return opts, fmt.Errorf("cancelled: terraform not installed")
		}
	}

	// Generate default VM name
	opts.VMName = fmt.Sprintf("cloud-init-%s", time.Now().Format("20060102-150405"))

	// Check for common Ubuntu cloud image locations
	defaultImagePaths := []string{
		"/var/lib/libvirt/images/jammy-server-cloudimg-amd64.img",
		"/var/lib/libvirt/images/noble-server-cloudimg-amd64.img",
		"~/images/jammy-server-cloudimg-amd64.img",
	}
	for _, path := range defaultImagePaths {
		if _, err := os.Stat(path); err == nil {
			opts.UbuntuImage = path
			break
		}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("VM Name").
				Description("Name for the libvirt VM").
				Value(&opts.VMName),

			huh.NewSelect[int]().
				Title("CPUs").
				Options(
					huh.NewOption("1 CPU", 1),
					huh.NewOption("2 CPUs (recommended)", 2),
					huh.NewOption("4 CPUs", 4),
					huh.NewOption("8 CPUs", 8),
				).
				Value(&opts.CPUs),

			huh.NewSelect[int]().
				Title("Memory").
				Options(
					huh.NewOption("2 GB", 2048),
					huh.NewOption("4 GB (recommended)", 4096),
					huh.NewOption("8 GB", 8192),
					huh.NewOption("16 GB", 16384),
				).
				Value(&opts.MemoryMB),

			huh.NewSelect[int]().
				Title("Disk Size").
				Options(
					huh.NewOption("10 GB", 10),
					huh.NewOption("20 GB (recommended)", 20),
					huh.NewOption("40 GB", 40),
					huh.NewOption("80 GB", 80),
				).
				Value(&opts.DiskGB),

			huh.NewInput().
				Title("Ubuntu Cloud Image").
				Description("Path to Ubuntu cloud image (.img or .qcow2)").
				Placeholder("/var/lib/libvirt/images/jammy-server-cloudimg-amd64.img").
				Value(&opts.UbuntuImage).
				Validate(validateImagePath),
		).Title("VM Configuration"),

		huh.NewGroup(
			huh.NewInput().
				Title("Libvirt URI").
				Description("Connection URI for libvirt").
				Placeholder("qemu:///system").
				Value(&opts.LibvirtURI),

			huh.NewInput().
				Title("Storage Pool").
				Description("Libvirt storage pool name").
				Placeholder("default").
				Value(&opts.StoragePool),

			huh.NewInput().
				Title("Network Name").
				Description("Libvirt network name").
				Placeholder("default").
				Value(&opts.NetworkName),

			huh.NewConfirm().
				Title("Keep on failure?").
				Description("Keep resources for debugging if deployment fails").
				Value(&opts.KeepOnFailure),
		).Title("Advanced Options"),
	).WithTheme(tui.Theme())

	if err := form.Run(); err != nil {
		return opts, fmt.Errorf("terraform options cancelled: %w", err)
	}

	return opts, nil
}

// validateImagePath validates the Ubuntu cloud image path.
func validateImagePath(path string) error {
	if path == "" {
		return fmt.Errorf("image path is required")
	}

	// Skip validation on non-Linux systems (they're just generating files)
	if runtime.GOOS != "linux" {
		return nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("image not found: %s\n\nDownload with:\n  wget https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img -O %s", path, path)
	}

	return nil
}
