package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
)

// runPackages lists all available packages from embedded scripts.
func runPackages(_ *cobra.Command, _ []string) error {
	registry, err := packages.DiscoverEmbedded()
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
