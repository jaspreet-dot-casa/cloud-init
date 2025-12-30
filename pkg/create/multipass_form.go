package create

import (
	"fmt"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// OSImage represents an available OS image for Multipass.
type OSImage struct {
	Name  string // Display name
	Image string // Multipass image identifier
}

// osImages contains available OS images for Multipass (from `multipass find`).
var osImages = []OSImage{
	{"Ubuntu 24.04 LTS (Noble Numbat)", "24.04"},
	{"Ubuntu 25.04 (Plucky Puffin)", "25.04"},
	{"Ubuntu 25.10 (Questing Quail)", "25.10"},
	{"Ubuntu 22.04 LTS (Jammy Jellyfish)", "22.04"},
	{"Ubuntu 26.04 LTS Daily (Resolute)", "daily:26.04"},
}

// runMultipassOptions prompts for Multipass-specific options.
func runMultipassOptions() (deploy.MultipassOptions, error) {
	opts := deploy.DefaultMultipassOptions()

	// Generate default VM name
	opts.VMName = fmt.Sprintf("cloud-init-%s", time.Now().Format("20060102-150405"))

	// Build OS image options
	imageOptions := make([]huh.Option[string], len(osImages))
	for i, img := range osImages {
		imageOptions[i] = huh.NewOption(img.Name, img.Image)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("VM Name").
				Description("Name for the Multipass VM").
				Value(&opts.VMName),

			huh.NewSelect[string]().
				Title("OS Image").
				Description("Ubuntu version to install (type to filter)").
				Options(imageOptions...).
				Filtering(true).
				Value(&opts.UbuntuVersion),

			huh.NewSelect[int]().
				Title("CPUs").
				Options(
					huh.NewOption("1 CPU", 1),
					huh.NewOption("2 CPUs (recommended)", 2),
					huh.NewOption("4 CPUs", 4),
				).
				Value(&opts.CPUs),

			huh.NewSelect[int]().
				Title("Memory").
				Options(
					huh.NewOption("2 GB", 2048),
					huh.NewOption("4 GB (recommended)", 4096),
					huh.NewOption("8 GB", 8192),
				).
				Value(&opts.MemoryMB),

			huh.NewSelect[int]().
				Title("Disk Size").
				Options(
					huh.NewOption("10 GB", 10),
					huh.NewOption("20 GB (recommended)", 20),
					huh.NewOption("40 GB", 40),
				).
				Value(&opts.DiskGB),

			huh.NewConfirm().
				Title("Keep VM on failure?").
				Description("Keep the VM for debugging if deployment fails").
				Value(&opts.KeepOnFailure),
		).Title("Multipass VM Options"),
	).WithTheme(tui.Theme())

	if err := form.Run(); err != nil {
		return opts, fmt.Errorf("multipass options cancelled: %w", err)
	}

	return opts, nil
}
