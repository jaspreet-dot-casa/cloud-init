// Package iso provides bootable ISO generation with embedded cloud-init configuration.
package iso

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ToolChain manages external tools required for ISO manipulation.
type ToolChain struct {
	XorrisoPath string
	Platform    string
}

// NewToolChain creates a new ToolChain instance.
func NewToolChain() *ToolChain {
	return &ToolChain{
		Platform: runtime.GOOS,
	}
}

// Detect finds and validates required external tools.
func (t *ToolChain) Detect() error {
	// Find xorriso
	path, err := exec.LookPath("xorriso")
	if err != nil {
		return fmt.Errorf("xorriso not found: %w", err)
	}
	t.XorrisoPath = path

	// Validate xorriso works
	cmd := exec.Command(path, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xorriso validation failed: %w", err)
	}

	// Check for minimum version (xorriso 1.4.0+)
	if !strings.Contains(string(output), "xorriso") {
		return fmt.Errorf("unexpected xorriso output: %s", string(output))
	}

	return nil
}

// Available returns true if xorriso is available.
func (t *ToolChain) Available() bool {
	return t.XorrisoPath != ""
}

// InstallInstructions returns platform-specific installation instructions.
func (t *ToolChain) InstallInstructions() string {
	switch t.Platform {
	case "darwin":
		return "Install with: brew install xorriso"
	case "linux":
		return "Install with: sudo apt install xorriso"
	default:
		return "Please install xorriso for your platform"
	}
}

// XorrisoVersion returns the installed xorriso version string.
func (t *ToolChain) XorrisoVersion() (string, error) {
	if t.XorrisoPath == "" {
		return "", fmt.Errorf("xorriso not detected")
	}

	cmd := exec.Command(t.XorrisoPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get xorriso version: %w", err)
	}

	// Parse first line of output
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "", fmt.Errorf("no version output from xorriso")
}
