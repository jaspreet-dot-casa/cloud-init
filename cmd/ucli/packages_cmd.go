package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/project"
)

// runPackages lists all available packages from scripts/packages/.
func runPackages(_ *cobra.Command, _ []string) error {
	projectRoot, err := project.FindRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	registry, err := packages.DiscoverFromProjectRoot(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}

	fmt.Printf("Found %d packages:\n\n", len(registry.Packages))

	for _, category := range registry.Categories() {
		fmt.Printf("%s:\n", category)
		for _, pkg := range registry.ByCategory[category] {
			desc := pkg.Description
			if desc == "" {
				desc = "(no description)"
			}
			fmt.Printf("  - %s: %s\n", pkg.Name, desc)
		}
		fmt.Println()
	}

	return nil
}
