package main

import "github.com/spf13/cobra"

// newCreateCmd creates the create subcommand (main command)
func newCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create and deploy Ubuntu configuration (interactive)",
		Long: `Launch the interactive TUI to configure cloud-init settings, select packages,
and deploy to a target (Multipass VM, USB, SSH, or Terraform).

This is the main command for end-to-end Ubuntu server configuration and deployment.`,
		RunE: runCreate,
	}
}

// newPackagesCmd creates the packages subcommand
func newPackagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "packages",
		Short: "List available packages",
		Long:  `List all available packages that can be installed via cloud-init.`,
		RunE:  runPackages,
	}
}

// newBuildCmd creates the build subcommand
func newBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build cloud-init.yaml interactively",
		Long: `Launch the interactive TUI to configure cloud-init settings and generate cloud-init.yaml.

This command runs the same configuration wizard as 'create' but only generates
the cloud-init.yaml file without deploying to any target.`,
		RunE: runBuild,
	}
}
