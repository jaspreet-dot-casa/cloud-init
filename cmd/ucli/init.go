package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/globalconfig"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <path>",
		Short: "Initialize ucli with a project path",
		Long: `Initialize ucli by setting the project path in ~/.config/ucli/config.yaml.

The project path should be the root of your cloud-init repository containing
the terraform/ directory with TF templates.

Examples:
  ucli init .                    # Use current directory
  ucli init ~/code/cloud-init    # Use absolute path
  ucli init ../my-homelab        # Use relative path`,
		Args: cobra.ExactArgs(1),
		RunE: runInit,
	}
}

func runInit(_ *cobra.Command, args []string) error {
	path := args[0]

	// Handle "." specially to show a nice message
	if path == "." {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Load existing config or create new one
	cfg, err := globalconfig.LoadOrCreate()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set and validate the project path
	if err := cfg.SetProjectPath(path); err != nil {
		return err
	}

	// Validate the project structure (optional but helpful)
	terraformDir := filepath.Join(cfg.ProjectPath, "terraform")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		fmt.Printf("Warning: %s/terraform/ directory not found\n", cfg.ProjectPath)
		fmt.Println("This directory should contain the Terraform templates (main.tf, variables.tf, outputs.tf)")
	}

	// Save the config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configPath, err := globalconfig.GetConfigPath()
	if err != nil {
		configPath = "~/.config/ucli/config.yaml" // fallback for display
	}
	fmt.Printf("Initialized ucli with project path: %s\n", cfg.ProjectPath)
	fmt.Printf("Config saved to: %s\n", configPath)

	return nil
}
