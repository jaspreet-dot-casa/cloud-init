// Package main provides the ucli CLI tool for generating cloud-init configurations.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
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
  - Generation of config.env and secrets.env files
  - Creation of bootable ISOs for bare metal installation
  - Creation of seed ISOs for libvirt VMs (future)`,
		Version: version,
	}

	rootCmd.AddCommand(
		newGenerateCmd(),
		newPackagesCmd(),
		newValidateCmd(),
		newBuildCmd(),
	)

	return rootCmd
}

// newGenerateCmd creates the generate subcommand
func newGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Interactive configuration generator",
		Long:  `Launch the interactive TUI to configure cloud-init settings and select packages.`,
		RunE:  runGenerate,
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

// runGenerate launches the interactive TUI for configuration generation.
func runGenerate(_ *cobra.Command, _ []string) error {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	registry, err := packages.DiscoverFromProjectRoot(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}

	result, err := tui.RunForm(registry)
	if err != nil {
		return err
	}

	// Display summary
	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("  Username:  %s\n", result.User.Username)
	fmt.Printf("  Hostname:  %s\n", result.User.Hostname)
	fmt.Printf("  Packages:  %d selected\n", len(result.SelectedPackages))
	fmt.Printf("  Output:    %s\n", result.OutputMode)

	// Generate config files
	cfg := config.NewFullConfigFromFormResult(result)
	writer := config.NewWriter(projectRoot)

	if err := writer.WriteAll(cfg); err != nil {
		return fmt.Errorf("failed to write config files: %w", err)
	}

	fmt.Println("\nGenerated files:")
	fmt.Printf("  - %s/config.env\n", projectRoot)
	fmt.Printf("  - %s/cloud-init/secrets.env\n", projectRoot)

	// Generate cloud-init.yaml if requested
	if result.OutputMode == tui.OutputCloudInit || result.OutputMode == tui.OutputBootableISO {
		fmt.Println("\nTo generate cloud-init.yaml, run:")
		fmt.Printf("  cd %s && ./cloud-init/generate.sh\n", projectRoot)
	}

	// ISO creation placeholder
	if result.OutputMode == tui.OutputBootableISO {
		fmt.Println("\n(Bootable ISO creation will be implemented in Phase 5)")
	}

	return nil
}

// runPackages lists all available packages from scripts/packages/.
func runPackages(_ *cobra.Command, _ []string) error {
	projectRoot, err := findProjectRoot()
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

// findProjectRoot finds the project root by looking for go.mod or scripts/ directory.
func findProjectRoot() (string, error) {
	// Start from current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree
	dir := cwd
	for {
		// Check for scripts/packages directory
		scriptsDir := filepath.Join(dir, "scripts", "packages")
		if _, err := os.Stat(scriptsDir); err == nil {
			return dir, nil
		}

		// Check for go.mod
		goMod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goMod); err == nil {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find project root (looked for scripts/packages or go.mod)")
}

// runValidate validates existing configuration files.
func runValidate(_ *cobra.Command, _ []string) error {
	// TODO(Phase 4): Implement validation
	return nil
}

// runBuild generates cloud-init config from existing files.
func runBuild(_ *cobra.Command, _ []string) error {
	// TODO(Phase 4): Implement build
	return nil
}
