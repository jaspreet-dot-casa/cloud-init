package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/generator"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/project"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// runBuild runs the interactive TUI and generates cloud-init.yaml.
func runBuild(_ *cobra.Command, _ []string) error {
	projectRoot, err := project.FindRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	// Step 1: Discover available packages
	fmt.Println("Discovering packages...")
	registry, err := packages.DiscoverFromProjectRoot(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}
	fmt.Printf("Found %d packages\n\n", len(registry.Packages))

	// Step 2: Run interactive TUI
	formResult, err := tui.RunForm(registry, nil)
	if err != nil {
		return fmt.Errorf("form cancelled: %w", err)
	}

	// Step 3: Convert to config
	cfg := config.NewFullConfigFromFormResult(formResult)

	// Step 4: Check template exists
	templatePath := filepath.Join(projectRoot, "cloud-init", "cloud-init.template.yaml")
	if err := generator.ValidateTemplate(templatePath); err != nil {
		return fmt.Errorf("template error: %w", err)
	}

	// Step 5: Generate cloud-init.yaml
	fmt.Println("\nGenerating cloud-init.yaml...")
	outputPath := filepath.Join(projectRoot, "cloud-init", "cloud-init.yaml")
	gen := generator.NewGenerator(projectRoot)
	if err := gen.Generate(cfg, templatePath, outputPath); err != nil {
		return fmt.Errorf("failed to generate cloud-init.yaml: %w", err)
	}

	fmt.Printf("\nGenerated: %s\n", outputPath)
	fmt.Println("\nBuild complete!")

	return nil
}
