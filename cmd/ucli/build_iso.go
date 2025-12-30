package main

import (
	"fmt"
	"path/filepath"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/iso"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/project"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/validation"
)

// runBuildISO builds a bootable ISO from existing config files.
func runBuildISO(sourceISO, outputPath, version, storage string, verbose bool) error {
	projectRoot, err := project.FindRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	// Step 1: Check tools
	builder := iso.NewBuilder(projectRoot)
	builder.SetVerbose(verbose)

	if err := builder.CheckTools(); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("\n%s\n", builder.InstallInstructions())
		return fmt.Errorf("required tools not installed")
	}

	// Step 2: Validate configuration files
	fmt.Println("Validating configuration files...")
	validator := validation.NewValidator(projectRoot)
	result := validator.ValidateAll()

	if result.HasErrors() {
		for _, issue := range result.Issues {
			if issue.Severity == validation.SeverityError {
				if issue.Field != "" {
					fmt.Printf("[ERROR] %s: %s (%s)\n", issue.File, issue.Message, issue.Field)
				} else {
					fmt.Printf("[ERROR] %s: %s\n", issue.File, issue.Message)
				}
			}
		}
		return fmt.Errorf("validation failed with %d error(s), fix errors before building", result.ErrorCount())
	}

	fmt.Println("Configuration valid.")

	// Step 3: Read configuration
	fmt.Println("Reading configuration...")
	reader := config.NewReader(projectRoot)
	cfg, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Step 4: Set output path if not specified
	if outputPath == "" {
		outputPath = filepath.Join(projectRoot, "output", fmt.Sprintf("ubuntu-%s-autoinstall.iso", version))
	}

	// Step 5: Build ISO
	fmt.Println("Building bootable ISO...")

	opts := &iso.ISOOptions{
		SourceISO:     sourceISO,
		OutputPath:    outputPath,
		UbuntuVersion: version,
		StorageLayout: iso.StorageLayout(storage),
		Timezone:      "UTC",
		Locale:        "en_US.UTF-8",
	}

	if err := builder.Build(cfg, opts); err != nil {
		return fmt.Errorf("failed to build ISO: %w", err)
	}

	fmt.Printf("\nBootable ISO created: %s\n", outputPath)
	fmt.Println("\nTo write to USB:")
	fmt.Printf("  sudo dd if=%s of=/dev/sdX bs=4M status=progress\n", outputPath)
	fmt.Println("\nBuild complete!")

	return nil
}
