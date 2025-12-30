package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/project"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/validation"
)

// runValidate validates existing configuration files.
func runValidate(_ *cobra.Command, _ []string) error {
	projectRoot, err := project.FindRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	validator := validation.NewValidator(projectRoot)
	result := validator.ValidateAll()

	// Print issues
	for _, issue := range result.Issues {
		prefix := "WARNING"
		if issue.Severity == validation.SeverityError {
			prefix = "ERROR"
		}

		if issue.Field != "" {
			fmt.Printf("[%s] %s: %s (%s)\n", prefix, issue.File, issue.Message, issue.Field)
		} else {
			fmt.Printf("[%s] %s: %s\n", prefix, issue.File, issue.Message)
		}
	}

	if result.HasErrors() {
		return fmt.Errorf("validation failed with %d error(s)", result.ErrorCount())
	}

	if len(result.Issues) == 0 {
		fmt.Println("All configuration files are valid.")
	} else {
		fmt.Printf("\nValidation passed with %d warning(s).\n", result.WarningCount())
	}

	return nil
}
