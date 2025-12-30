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

// newGenerateCmd creates the generate subcommand (deprecated, use create instead)
func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "generate",
		Short:      "Interactive configuration generator (deprecated: use 'create' instead)",
		Long:       `Launch the interactive TUI to configure cloud-init settings and select packages.`,
		RunE:       runGenerate,
		Deprecated: "use 'ucli create' instead for end-to-end configuration and deployment",
	}
	return cmd
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

// newValidateCmd creates the validate subcommand
func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration files",
		Long:  `Validate existing config.env and secrets.env files for correctness.`,
		RunE:  runValidate,
	}
}

// newBuildCmd creates the build subcommand
func newBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build cloud-init config from existing files",
		Long:  `Non-interactive build using existing config.env and secrets.env files.`,
		RunE:  runBuild,
	}
}

// newBuildISOCmd creates the build-iso subcommand
func newBuildISOCmd() *cobra.Command {
	var sourceISO, outputPath, version, storage string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "build-iso",
		Short: "Build bootable ISO from existing config files",
		Long: `Build a bootable Ubuntu autoinstall ISO using existing config.env and secrets.env files.

Requires xorriso to be installed:
  macOS: brew install xorriso
  Linux: sudo apt install xorriso`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuildISO(sourceISO, outputPath, version, storage, verbose)
		},
	}

	cmd.Flags().StringVarP(&sourceISO, "source", "s", "", "Source Ubuntu ISO path (required)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output ISO path (defaults to ./output/ubuntu-<version>-autoinstall.iso)")
	cmd.Flags().StringVarP(&version, "version", "v", "24.04", "Ubuntu version (22.04 or 24.04)")
	cmd.Flags().StringVar(&storage, "storage", "lvm", "Storage layout (lvm, direct, zfs)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	if err := cmd.MarkFlagRequired("source"); err != nil {
		panic(err)
	}

	return cmd
}
