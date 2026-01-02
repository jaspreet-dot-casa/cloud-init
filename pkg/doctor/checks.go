package doctor

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// EnvGetter is an interface for getting environment variables (allows testing).
type EnvGetter interface {
	Getenv(key string) string
}

// RealEnvGetter gets environment variables from the real environment.
type RealEnvGetter struct{}

// Getenv gets an environment variable.
func (e *RealEnvGetter) Getenv(key string) string {
	return os.Getenv(key)
}

// CommandExecutor is an interface for executing commands, allowing for testing.
type CommandExecutor interface {
	LookPath(file string) (string, error)
	Run(name string, args ...string) (string, error)
	CombinedOutput(name string, args ...string) ([]byte, error)
	FileExists(path string) bool
}

// RealExecutor is the default command executor that uses the real system.
type RealExecutor struct{}

// LookPath finds the path to an executable.
func (e *RealExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Run executes a command and returns its output.
func (e *RealExecutor) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// Some tools output version to stderr
		if stderr.Len() > 0 {
			return stderr.String(), err
		}
		return stdout.String(), err
	}
	// Prefer stdout, fall back to stderr (some tools output version to stderr)
	output := stdout.String()
	if output == "" {
		output = stderr.String()
	}
	return output, nil
}

// CombinedOutput runs a command and returns combined stdout and stderr.
func (e *RealExecutor) CombinedOutput(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// FileExists checks if a file exists.
func (e *RealExecutor) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// checkTool checks if a tool is installed and gets its version.
func checkTool(exec CommandExecutor, id, name, desc string, versionArgs []string, versionRegex *regexp.Regexp, fixCmd *FixCommand) Check {
	check := Check{
		ID:          id,
		Name:        name,
		Description: desc,
		FixCommand:  fixCmd,
	}

	path, err := exec.LookPath(id)
	if err != nil {
		check.Status = StatusMissing
		check.Message = "not installed"
		return check
	}

	// Try to get version
	output, err := exec.Run(path, versionArgs...)
	if err != nil {
		// Tool exists but version check failed - still consider it OK
		check.Status = StatusOK
		check.Message = "installed (version unknown)"
		return check
	}

	// Extract version from output
	version := extractVersion(output, versionRegex)
	if version != "" {
		check.Status = StatusOK
		check.Message = version
	} else {
		check.Status = StatusOK
		check.Message = "installed"
	}

	return check
}

// extractVersion extracts version string from command output.
func extractVersion(output string, regex *regexp.Regexp) string {
	if regex == nil {
		// Default: look for common version patterns
		defaultRegex := regexp.MustCompile(`v?(\d+\.\d+(?:\.\d+)?(?:-[a-zA-Z0-9]+)?)`)
		matches := defaultRegex.FindStringSubmatch(output)
		if len(matches) >= 2 {
			return matches[1]
		}
		return ""
	}

	matches := regex.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// CheckTerraform checks if terraform is installed.
func CheckTerraform(exec CommandExecutor) Check {
	return checkTool(
		exec,
		IDTerraform,
		"Terraform",
		"Infrastructure as Code tool",
		[]string{"version"},
		regexp.MustCompile(`Terraform v(\d+\.\d+\.\d+)`),
		GetFixCommand(IDTerraform, runtime.GOOS),
	)
}

// CheckMultipass checks if multipass is installed.
func CheckMultipass(exec CommandExecutor) Check {
	return checkTool(
		exec,
		IDMultipass,
		"Multipass",
		"Ubuntu VM manager",
		[]string{"version"},
		regexp.MustCompile(`multipass\s+(\d+\.\d+\.\d+)`),
		GetFixCommand(IDMultipass, runtime.GOOS),
	)
}

// CheckXorriso checks if xorriso is installed.
func CheckXorriso(exec CommandExecutor) Check {
	return checkTool(
		exec,
		IDXorriso,
		"xorriso",
		"ISO manipulation tool",
		[]string{"-version"},
		regexp.MustCompile(`xorriso\s+(\d+\.\d+\.\d+)`),
		GetFixCommand(IDXorriso, runtime.GOOS),
	)
}

// CheckLibvirt checks if libvirt is installed and running.
func CheckLibvirt(exec CommandExecutor) Check {
	check := Check{
		ID:          IDLibvirt,
		Name:        "libvirt",
		Description: "Virtualization API",
		FixCommand:  GetFixCommand(IDLibvirt, runtime.GOOS),
	}

	// Check for libvirtd or virtqemud
	_, err := exec.LookPath("libvirtd")
	if err != nil {
		_, err = exec.LookPath("virtqemud")
		if err != nil {
			check.Status = StatusMissing
			check.Message = "not installed"
			return check
		}
	}

	// Try to check if service is running
	output, err := exec.Run("systemctl", "is-active", "libvirtd")
	if err != nil {
		// Try virtqemud for newer systems
		output, err = exec.Run("systemctl", "is-active", "virtqemud")
	}

	if err != nil || !strings.Contains(strings.TrimSpace(output), "active") {
		check.Status = StatusWarning
		check.Message = "installed but service not running"
		return check
	}

	check.Status = StatusOK
	check.Message = "running"
	return check
}

// CheckVirsh checks if virsh CLI is installed.
func CheckVirsh(exec CommandExecutor) Check {
	return checkTool(
		exec,
		IDVirsh,
		"virsh",
		"Libvirt CLI",
		[]string{"--version"},
		regexp.MustCompile(`(\d+\.\d+\.\d+)`),
		GetFixCommand(IDVirsh, runtime.GOOS),
	)
}

// CheckQemuKVM checks if QEMU/KVM is installed.
func CheckQemuKVM(exec CommandExecutor) Check {
	check := Check{
		ID:          IDQemuKVM,
		Name:        "QEMU/KVM",
		Description: "Hardware virtualization",
		FixCommand:  GetFixCommand(IDQemuKVM, runtime.GOOS),
	}

	// Check for qemu-system-x86_64 or kvm
	path, err := exec.LookPath("qemu-system-x86_64")
	if err != nil {
		path, err = exec.LookPath("kvm")
		if err != nil {
			check.Status = StatusMissing
			check.Message = "not installed"
			return check
		}
	}

	output, err := exec.Run(path, "--version")
	if err != nil {
		check.Status = StatusOK
		check.Message = "installed"
		return check
	}

	version := extractVersion(output, regexp.MustCompile(`QEMU.*version (\d+\.\d+(?:\.\d+)?)`))
	if version != "" {
		check.Status = StatusOK
		check.Message = version
	} else {
		check.Status = StatusOK
		check.Message = "installed"
	}

	return check
}

// CheckCloudImage checks if a cloud image exists at the default path.
func CheckCloudImage(exec CommandExecutor, imagePath string) Check {
	check := Check{
		ID:          IDCloudImage,
		Name:        "Ubuntu Cloud Image",
		Description: "Base image for VMs",
		FixCommand:  nil, // Handled by image management
	}

	if imagePath == "" {
		imagePath = "/var/lib/libvirt/images/jammy-server-cloudimg-amd64.img"
	}

	if exec.FileExists(imagePath) {
		check.Status = StatusOK
		check.Message = imagePath
	} else {
		check.Status = StatusMissing
		check.Message = "no image at " + imagePath
	}

	return check
}

// CheckGhostty checks if Ghostty terminal is installed and/or running.
func CheckGhostty(exec CommandExecutor, env EnvGetter) Check {
	check := Check{
		ID:          IDGhostty,
		Name:        "Ghostty",
		Description: "Modern GPU-accelerated terminal",
		FixCommand:  GetFixCommand(IDGhostty, runtime.GOOS),
	}

	// Check if currently running in Ghostty via environment variables
	termProgram := env.Getenv("TERM_PROGRAM")
	term := env.Getenv("TERM")

	if termProgram == "ghostty" || strings.Contains(strings.ToLower(term), "ghostty") {
		check.Status = StatusOK
		check.Message = "running in Ghostty"
		return check
	}

	// Check if Ghostty is installed (even if not current terminal)
	_, err := exec.LookPath("ghostty")
	if err == nil {
		check.Status = StatusWarning
		check.Message = "installed (not current terminal)"
		return check
	}

	check.Status = StatusMissing
	check.Message = "not installed"
	return check
}
