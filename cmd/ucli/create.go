package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/create"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/project"
)

// runCreate launches the interactive Bubble Tea TUI for the create command.
func runCreate(_ *cobra.Command, _ []string) error {
	projectRoot, err := project.FindRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	return create.Run(projectRoot)
}
