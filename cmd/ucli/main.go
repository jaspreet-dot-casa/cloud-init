// Package main provides the ucli CLI tool for generating cloud-init configurations.
package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/doctor"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/iso"
	settingsview "github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/settings"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/vmlist"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/globalconfig"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
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
  - Deployment to Multipass VMs, Terraform/libvirt, or bootable ISOs

Run without arguments to launch the full-screen TUI for VM management.`,
		Version: version,
		RunE:    runTUI,
	}

	rootCmd.AddCommand(
		newInitCmd(),
		newPackagesCmd(),
	)

	return rootCmd
}

// runTUI launches the full-screen TUI application.
func runTUI(_ *cobra.Command, _ []string) error {
	// Load global config to get project path
	cfg, err := globalconfig.Load()
	if err != nil {
		if errors.Is(err, globalconfig.ErrNotInitialized) {
			return fmt.Errorf("ucli not initialized. Run 'ucli init <path>' to set up your project path")
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get and validate project directory
	projectDir, err := cfg.ProjectDir()
	if err != nil {
		return fmt.Errorf("invalid project path: %w", err)
	}

	// Create settings store for cloud images
	store, err := settings.NewStore()
	if err != nil {
		// Non-fatal - create will work without cloud images
		store = nil
	}

	// Create the application with tabs
	model := app.New(projectDir).WithTabs(
		vmlist.New(projectDir),
		create.New(projectDir, store),
		iso.New(projectDir),
		doctor.New(),
		settingsview.New(),
	)

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
