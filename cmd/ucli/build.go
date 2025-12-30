package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/generator"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/project"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/validation"
)

// runBuild generates cloud-init config from existing files.
func runBuild(_ *cobra.Command, _ []string) error {
	projectRoot, err := project.FindRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	// Step 1: Validate configuration files
	fmt.Println("Validating configuration files...")
	validator := validation.NewValidator(projectRoot)
	result := validator.ValidateAll()

	if result.HasErrors() {
		// Print errors
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

	// Print warnings if any
	for _, issue := range result.Issues {
		if issue.Severity == validation.SeverityWarning {
			if issue.Field != "" {
				fmt.Printf("[WARNING] %s: %s (%s)\n", issue.File, issue.Message, issue.Field)
			} else {
				fmt.Printf("[WARNING] %s: %s\n", issue.File, issue.Message)
			}
		}
	}

	fmt.Println("Configuration valid.")

	// Step 2: Check template exists
	templatePath := filepath.Join(projectRoot, "cloud-init", "cloud-init.template.yaml")
	if err := generator.ValidateTemplate(templatePath); err != nil {
		return fmt.Errorf("template error: %w", err)
	}

	// Step 3: Read configuration
	fmt.Println("Reading configuration...")
	reader := config.NewReader(projectRoot)
	cfg, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Step 4: Generate cloud-init.yaml
	fmt.Println("Generating cloud-init.yaml...")
	outputPath := filepath.Join(projectRoot, "cloud-init", "cloud-init.yaml")
	gen := generator.NewGenerator(projectRoot)
	if err := gen.Generate(cfg, templatePath, outputPath); err != nil {
		return fmt.Errorf("failed to generate cloud-init.yaml: %w", err)
	}

	fmt.Printf("\nGenerated: %s\n", outputPath)
	fmt.Println("\nBuild complete!")

	return nil
}
