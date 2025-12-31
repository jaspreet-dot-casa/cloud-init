package main

import "github.com/spf13/cobra"

// newPackagesCmd creates the packages subcommand
func newPackagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "packages",
		Short: "List available packages",
		Long:  `List all available packages that can be installed via cloud-init.`,
		RunE:  runPackages,
	}
}
