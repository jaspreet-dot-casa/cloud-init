// Package tfstate manages Terraform state and provides VM lifecycle operations.
// All VM operations are done through Terraform (no virsh) for consistency
// and proper state management.
package tfstate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// VMStatus represents the current state of a VM.
type VMStatus string

const (
	StatusRunning  VMStatus = "running"
	StatusStopped  VMStatus = "stopped"
	StatusPaused   VMStatus = "paused"
	StatusShutoff  VMStatus = "shutoff"
	StatusCrashed  VMStatus = "crashed"
	StatusUnknown  VMStatus = "unknown"
	StatusNotFound VMStatus = "not-found"
)

// VMInfo represents a VM from terraform state.
type VMInfo struct {
	Name      string
	Status    VMStatus
	IP        string
	CPUs      int
	MemoryMB  int
	DiskGB    int
	Autostart bool
	Running   bool   // From terraform tfvars
	TFDir     string // tf/<machine-name>/ path
	CreatedAt time.Time
}

// Manager reads terraform state and provides VM lifecycle operations.
// It manages multiple machines, each in their own tf/<machine-name>/ directory.
type Manager struct {
	baseDir    string // Project root directory (contains tf/)
	libvirtURI string // libvirt connection URI
	verbose    bool
}

// NewManager creates a new terraform state manager.
// baseDir should be the project root (containing tf/ subdirectory).
func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir:    baseDir,
		libvirtURI: "qemu:///system",
	}
}

// SetLibvirtURI sets the libvirt connection URI.
func (m *Manager) SetLibvirtURI(uri string) {
	m.libvirtURI = uri
}

// SetVerbose enables verbose output.
func (m *Manager) SetVerbose(verbose bool) {
	m.verbose = verbose
}

// BaseDir returns the project base directory.
func (m *Manager) BaseDir() string {
	return m.baseDir
}

// TFDir returns the tf/ directory path.
func (m *Manager) TFDir() string {
	return filepath.Join(m.baseDir, "tf")
}

// MachineDir returns the directory for a specific machine.
func (m *Manager) MachineDir(name string) string {
	return filepath.Join(m.TFDir(), name)
}

// ListVMs returns all VMs managed by terraform configurations in tf/.
func (m *Manager) ListVMs(ctx context.Context) ([]VMInfo, error) {
	machines, err := DiscoverMachines(m.TFDir())
	if err != nil {
		return nil, fmt.Errorf("failed to discover machines: %w", err)
	}

	var vms []VMInfo
	for _, name := range machines {
		vm, err := m.GetVM(ctx, name)
		if err != nil {
			// Log but continue - some machines might have incomplete state
			if m.verbose {
				fmt.Printf("Warning: failed to get VM %s: %v\n", name, err)
			}
			continue
		}
		vms = append(vms, *vm)
	}

	return vms, nil
}

// GetVM returns information about a specific VM.
func (m *Manager) GetVM(ctx context.Context, name string) (*VMInfo, error) {
	machineDir := m.MachineDir(name)

	// Check if machine exists
	if !MachineExists(m.TFDir(), name) {
		return nil, fmt.Errorf("machine %q does not exist", name)
	}

	vm := &VMInfo{
		Name:   name,
		Status: StatusUnknown,
		TFDir:  machineDir,
	}

	// Check if terraform state exists
	if !MachineHasState(m.TFDir(), name) {
		vm.Status = StatusNotFound
		return vm, nil
	}

	// Get terraform outputs
	outputs, err := m.getTerraformOutputs(ctx, machineDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get terraform outputs: %w", err)
	}

	// Parse outputs
	if ip, ok := outputs["vm_ip"]; ok {
		vm.IP = ip
	}

	if running, ok := outputs["vm_running"]; ok {
		vm.Running = running == "true"
		if vm.Running {
			vm.Status = StatusRunning
		} else {
			vm.Status = StatusStopped
		}
	}

	if cpus, ok := outputs["vm_vcpu_count"]; ok {
		vm.CPUs, _ = strconv.Atoi(cpus)
	}

	if mem, ok := outputs["vm_memory_mb"]; ok {
		vm.MemoryMB, _ = strconv.Atoi(mem)
	}

	if disk, ok := outputs["vm_disk_size_gb"]; ok {
		vm.DiskGB, _ = strconv.Atoi(disk)
	}

	if autostart, ok := outputs["vm_autostart"]; ok {
		vm.Autostart = autostart == "true"
	}

	return vm, nil
}

// StartVM starts a stopped VM by setting running=true and applying.
func (m *Manager) StartVM(ctx context.Context, name string) error {
	machineDir := m.MachineDir(name)

	if !MachineExists(m.TFDir(), name) {
		return fmt.Errorf("machine %q does not exist", name)
	}

	// Update tfvars
	if err := updateTFVar(machineDir, "running", "true"); err != nil {
		return fmt.Errorf("failed to update running variable: %w", err)
	}

	// Run terraform apply
	return m.terraformApply(ctx, machineDir)
}

// StopVM gracefully stops a running VM by setting running=false and applying.
func (m *Manager) StopVM(ctx context.Context, name string) error {
	machineDir := m.MachineDir(name)

	if !MachineExists(m.TFDir(), name) {
		return fmt.Errorf("machine %q does not exist", name)
	}

	// Update tfvars
	if err := updateTFVar(machineDir, "running", "false"); err != nil {
		return fmt.Errorf("failed to update running variable: %w", err)
	}

	// Run terraform apply
	return m.terraformApply(ctx, machineDir)
}

// ForceStopVM is an alias for StopVM since we use terraform.
// The libvirt provider handles graceful vs forced shutdown.
func (m *Manager) ForceStopVM(ctx context.Context, name string) error {
	return m.StopVM(ctx, name)
}

// DeleteVM destroys the VM using terraform destroy.
func (m *Manager) DeleteVM(ctx context.Context, name string) error {
	machineDir := m.MachineDir(name)

	if !MachineExists(m.TFDir(), name) {
		return fmt.Errorf("machine %q does not exist", name)
	}

	cmd := exec.CommandContext(ctx, "terraform", "destroy", "-auto-approve")
	cmd.Dir = machineDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform destroy failed: %w\n%s", err, stderr.String())
	}

	return nil
}

// DeleteMachineDir removes the machine's terraform directory.
// This should only be called after DeleteVM.
func (m *Manager) DeleteMachineDir(name string) error {
	machineDir := m.MachineDir(name)
	return os.RemoveAll(machineDir)
}

// RefreshState runs terraform apply -refresh-only to update the state.
func (m *Manager) RefreshState(ctx context.Context, name string) error {
	machineDir := m.MachineDir(name)

	if !MachineExists(m.TFDir(), name) {
		return fmt.Errorf("machine %q does not exist", name)
	}

	cmd := exec.CommandContext(ctx, "terraform", "apply", "-refresh-only", "-auto-approve")
	cmd.Dir = machineDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform apply -refresh-only failed: %w\n%s", err, stderr.String())
	}

	return nil
}

// ConsoleCommand returns the virsh console command for a VM.
func (m *Manager) ConsoleCommand(name string) string {
	if m.libvirtURI != "" && m.libvirtURI != "qemu:///system" {
		return fmt.Sprintf("virsh -c %s console %s", m.libvirtURI, name)
	}
	return fmt.Sprintf("virsh console %s", name)
}

// SSHCommand returns the SSH command for a VM.
func (m *Manager) SSHCommand(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" || ip == "pending" {
		return ""
	}
	return fmt.Sprintf("ssh ubuntu@%s", ip)
}

// IsInitialized returns true if any machine has been initialized.
func (m *Manager) IsInitialized() bool {
	machines, err := DiscoverMachines(m.TFDir())
	if err != nil || len(machines) == 0 {
		return false
	}

	for _, name := range machines {
		if MachineIsInitialized(m.TFDir(), name) {
			return true
		}
	}
	return false
}

// HasState returns true if any machine has terraform state.
func (m *Manager) HasState() bool {
	machines, err := DiscoverMachines(m.TFDir())
	if err != nil || len(machines) == 0 {
		return false
	}

	for _, name := range machines {
		if MachineHasState(m.TFDir(), name) {
			return true
		}
	}
	return false
}

// terraformApply runs terraform apply in the given directory.
func (m *Manager) terraformApply(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "terraform", "apply", "-auto-approve")
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform apply failed: %w\n%s", err, stderr.String())
	}

	return nil
}

// getTerraformOutputs runs terraform output -json and parses the results.
func (m *Manager) getTerraformOutputs(ctx context.Context, dir string) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If state doesn't exist yet, return empty
		if strings.Contains(stderr.String(), "No outputs") ||
			strings.Contains(stderr.String(), "no outputs defined") {
			return nil, nil
		}
		return nil, fmt.Errorf("terraform output failed: %w\n%s", err, stderr.String())
	}

	return parseOutputJSON(stdout.Bytes()), nil
}

// parseOutputJSON parses terraform output JSON into a string map.
func parseOutputJSON(data []byte) map[string]string {
	outputs := make(map[string]string)

	var rawOutputs map[string]struct {
		Value interface{} `json:"value"`
		Type  interface{} `json:"type"`
	}

	if err := json.Unmarshal(data, &rawOutputs); err != nil {
		return outputs
	}

	for k, v := range rawOutputs {
		switch val := v.Value.(type) {
		case string:
			outputs[k] = val
		case []interface{}:
			if len(val) > 0 {
				if str, ok := val[0].(string); ok {
					outputs[k] = str
				}
			}
		case float64:
			outputs[k] = strconv.FormatFloat(val, 'f', -1, 64)
		case bool:
			outputs[k] = strconv.FormatBool(val)
		default:
			outputs[k] = fmt.Sprintf("%v", val)
		}
	}

	return outputs
}

// updateTFVar updates a single variable in terraform.tfvars.
// Creates the file if it doesn't exist.
func updateTFVar(machineDir, key, value string) error {
	tfvarsPath := filepath.Join(machineDir, "terraform.tfvars")

	// Read existing tfvars or start with empty
	var lines []string
	data, err := os.ReadFile(tfvarsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read tfvars: %w", err)
	}
	if err == nil {
		lines = strings.Split(string(data), "\n")
	}

	// Find and update the key, or add it
	found := false
	keyPrefix := key + " "
	keyPrefixEq := key + "="
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, keyPrefix) || strings.HasPrefix(trimmed, keyPrefixEq) {
			lines[i] = fmt.Sprintf("%s = %s", key, value)
			found = true
			break
		}
	}

	if !found {
		// Add the variable at the end (before any trailing empty lines)
		// Find last non-empty line
		insertIdx := len(lines)
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				insertIdx = i + 1
				break
			}
		}
		newLine := fmt.Sprintf("%s = %s", key, value)
		if insertIdx >= len(lines) {
			lines = append(lines, newLine)
		} else {
			lines = append(lines[:insertIdx], append([]string{newLine}, lines[insertIdx:]...)...)
		}
	}

	// Write back
	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return os.WriteFile(tfvarsPath, []byte(content), 0644)
}
