// Package main provides the ucli CLI tool for generating cloud-init configurations.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/generator"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/iso"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/validation"
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
		newBuildISOCmd(),
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

	// Bootable ISO creation
	if result.OutputMode == tui.OutputBootableISO {
		fmt.Println("\nBuilding bootable ISO...")

		builder := iso.NewBuilder(projectRoot)
		builder.SetVerbose(true)

		// Check if tools are available
		if err := builder.CheckTools(); err != nil {
			fmt.Printf("\nError: %v\n", err)
			fmt.Printf("\n%s\n", builder.InstallInstructions())
			return fmt.Errorf("required tools not installed")
		}

		outputPath := filepath.Join(projectRoot, "output", fmt.Sprintf("ubuntu-%s-autoinstall.iso", result.ISO.UbuntuVersion))

		opts := &iso.ISOOptions{
			SourceISO:     result.ISO.SourcePath,
			OutputPath:    outputPath,
			UbuntuVersion: result.ISO.UbuntuVersion,
			StorageLayout: iso.StorageLayout(result.ISO.StorageLayout),
			Timezone:      "UTC",
			Locale:        "en_US.UTF-8",
		}

		if err := builder.Build(cfg, opts); err != nil {
			return fmt.Errorf("failed to build ISO: %w", err)
		}

		fmt.Printf("\nBootable ISO created: %s\n", outputPath)
		fmt.Println("\nTo write to USB:")
		fmt.Printf("  sudo dd if=%s of=/dev/sdX bs=4M status=progress\n", outputPath)
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
	projectRoot, err := findProjectRoot()
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

// runBuild generates cloud-init config from existing files.
func runBuild(_ *cobra.Command, _ []string) error {
	projectRoot, err := findProjectRoot()
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

// runBuildISO builds a bootable ISO from existing config files.
func runBuildISO(sourceISO, outputPath, version, storage string, verbose bool) error {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	// Step 1: Check tools
	builder := iso.NewBuilder(projectRoot)
	builder.SetVerbose(verbose)

	if err := builder.CheckTools(); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("\n%s\n", builder.InstallInstructions())
		return fmt.Errorf("required tools not installed")
	}

	// Step 2: Validate configuration files
	fmt.Println("Validating configuration files...")
	validator := validation.NewValidator(projectRoot)
	result := validator.ValidateAll()

	if result.HasErrors() {
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

	fmt.Println("Configuration valid.")

	// Step 3: Read configuration
	fmt.Println("Reading configuration...")
	reader := config.NewReader(projectRoot)
	cfg, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Step 4: Set output path if not specified
	if outputPath == "" {
		outputPath = filepath.Join(projectRoot, "output", fmt.Sprintf("ubuntu-%s-autoinstall.iso", version))
	}

	// Step 5: Build ISO
	fmt.Println("Building bootable ISO...")

	opts := &iso.ISOOptions{
		SourceISO:     sourceISO,
		OutputPath:    outputPath,
		UbuntuVersion: version,
		StorageLayout: iso.StorageLayout(storage),
		Timezone:      "UTC",
		Locale:        "en_US.UTF-8",
	}

	if err := builder.Build(cfg, opts); err != nil {
		return fmt.Errorf("failed to build ISO: %w", err)
	}

	fmt.Printf("\nBootable ISO created: %s\n", outputPath)
	fmt.Println("\nTo write to USB:")
	fmt.Printf("  sudo dd if=%s of=/dev/sdX bs=4M status=progress\n", outputPath)
	fmt.Println("\nBuild complete!")

	return nil
}
