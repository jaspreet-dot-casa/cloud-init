package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/iso"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/project"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// runGenerate launches the interactive TUI for configuration generation.
func runGenerate(_ *cobra.Command, _ []string) error {
	projectRoot, err := project.FindRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	registry, err := packages.DiscoverFromProjectRoot(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}

	result, err := tui.RunForm(registry, nil) // nil = show all questions including output mode
	if err != nil {
		return err
	}

	// Display summary
	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("  Username:  %s\n", result.User.Username)
	fmt.Printf("  Hostname:  %s\n", result.User.Hostname)
	fmt.Printf("  Packages:  %d selected\n", len(result.SelectedPackages))
	fmt.Printf("  Output:    %s\n", result.OutputMode)

	// Generate config files
	cfg := config.NewFullConfigFromFormResult(result)
	writer := config.NewWriter(projectRoot)

	if err := writer.WriteAll(cfg); err != nil {
		return fmt.Errorf("failed to write config files: %w", err)
	}

	fmt.Println("\nGenerated files:")
	fmt.Printf("  - %s/config.env\n", projectRoot)
	fmt.Printf("  - %s/cloud-init/secrets.env\n", projectRoot)

	// Generate cloud-init.yaml if requested
	if result.OutputMode == tui.OutputCloudInit || result.OutputMode == tui.OutputBootableISO {
		fmt.Println("\nTo generate cloud-init.yaml, run:")
		fmt.Printf("  cd %s && ./cloud-init/generate.sh\n", projectRoot)
	}

	// Bootable ISO creation
	if result.OutputMode == tui.OutputBootableISO {
		fmt.Println("\nBuilding bootable ISO...")

		builder := iso.NewBuilder(projectRoot)
		builder.SetVerbose(true)

		// Check if tools are available
		if err := builder.CheckTools(); err != nil {
			fmt.Printf("\nError: %v\n", err)
			fmt.Printf("\n%s\n", builder.InstallInstructions())
			return fmt.Errorf("required tools not installed")
		}

		outputPath := filepath.Join(projectRoot, "output", fmt.Sprintf("ubuntu-%s-autoinstall.iso", result.ISO.UbuntuVersion))

		opts := &iso.ISOOptions{
			SourceISO:     result.ISO.SourcePath,
			OutputPath:    outputPath,
			UbuntuVersion: result.ISO.UbuntuVersion,
			StorageLayout: iso.StorageLayout(result.ISO.StorageLayout),
			Timezone:      "UTC",
			Locale:        "en_US.UTF-8",
		}

		if err := builder.Build(cfg, opts); err != nil {
			return fmt.Errorf("failed to build ISO: %w", err)
		}

		fmt.Printf("\nBootable ISO created: %s\n", outputPath)
		fmt.Println("\nTo write to USB:")
		fmt.Printf("  sudo dd if=%s of=/dev/sdX bs=4M status=progress\n", outputPath)
	}

	return nil
}
