// Package main provides the ucli CLI tool for generating cloud-init configurations.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/iso"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/vmlist"
	createpkg "github.com/jaspreet-dot-casa/cloud-init/pkg/create"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
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

	for {
		// Create the application with tabs
		model := app.New(projectDir).WithTabs(
			vmlist.New(projectDir),
			create.New(projectDir),
			iso.New(projectDir),
			app.NewPlaceholderTab(app.TabConfig, "Config", "4", "  Configuration coming soon...\n\n  Press [1] for VMs, [2] for Create, or [q] to quit."),
		)

		// Run the TUI
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}

		// Check if there's a pending create request
		appModel, ok := finalModel.(app.Model)
		if !ok {
			return nil
		}

		pending := appModel.PendingCreate()
		if pending == nil {
			// Normal exit
			return nil
		}

		// Run the create flow for the selected target
		fmt.Printf("\nLaunching create wizard for %s...\n\n", pending.Target)
		if err := runCreateForTarget(pending.Target, pending.ProjectDir); err != nil {
			fmt.Printf("Create failed: %v\n", err)
			fmt.Println("Press Enter to return to the TUI...")
			fmt.Scanln()
		} else {
			fmt.Println("\nPress Enter to return to the TUI...")
			fmt.Scanln()
		}
		// Loop back to re-launch TUI
	}
}

// runCreateForTarget runs the create flow for a specific target
func runCreateForTarget(_ deploy.DeploymentTarget, projectDir string) error {
	// Use the existing create package which handles target selection
	return createpkg.Run(projectDir)
}
