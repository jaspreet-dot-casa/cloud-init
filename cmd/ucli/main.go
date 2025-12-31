// Package main provides the ucli CLI tool for generating cloud-init configurations.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

// version is set via -ldflags during build
var version = "dev"

func main() {
	rootCmd := newRootCmd()

	// Cobra handles error printing
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// newRootCmd creates the root command for ucli
func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ucli",
		Short: "Ubuntu Cloud-Init CLI Tool",
		Long: `ucli is an interactive CLI tool for generating cloud-init configurations
with package selection for Ubuntu server installations.

It supports:
  - Interactive package selection from available installers
  - Direct generation of cloud-init.yaml (no config files needed)
  - Deployment to Multipass VMs, Terraform/libvirt, or bootable ISOs`,
		Version: version,
	}

	rootCmd.AddCommand(
		newCreateCmd(),
		newPackagesCmd(),
		newBuildCmd(),
	)

	return rootCmd
}
