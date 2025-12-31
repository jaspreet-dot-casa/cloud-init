// Package main provides the ucli CLI tool for generating cloud-init configurations.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/vmlist"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/project"
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
		newCreateCmd(),
		newPackagesCmd(),
		newBuildCmd(),
	)

	return rootCmd
}

// runTUI launches the full-screen TUI application.
func runTUI(cmd *cobra.Command, args []string) error {
	// Find project root
	projectDir, err := project.FindRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Create the application with tabs
	model := app.New(projectDir).WithTabs(
		vmlist.New(projectDir),
		app.NewPlaceholderTab(app.TabCreate, "Create", "2", "  Create VM coming soon...\n\n  Press [1] for VMs, [3] for ISO, or [q] to quit."),
		app.NewPlaceholderTab(app.TabISO, "ISO", "3", "  ISO builder coming soon...\n\n  Press [1] for VMs, [2] for Create, or [q] to quit."),
		app.NewPlaceholderTab(app.TabConfig, "Config", "4", "  Configuration coming soon...\n\n  Press [1] for VMs, [2] for Create, or [q] to quit."),
	)

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
