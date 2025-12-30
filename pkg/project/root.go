// Package project provides utilities for working with the project structure.
package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRoot finds the project root by looking for go.mod or scripts/ directory.
func FindRoot() (string, error) {
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
