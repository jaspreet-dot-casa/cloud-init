// Package tfstate manages Terraform state and provides VM lifecycle operations.
// It reads VM information from terraform state/outputs and uses virsh for
// fast start/stop operations.
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
	CreatedAt time.Time
}

// Manager reads terraform state and provides VM lifecycle operations.
type Manager struct {
	workDir    string // terraform/ directory
	libvirtURI string // libvirt connection URI
	verbose    bool
}

// NewManager creates a new terraform state manager.
func NewManager(workDir string) *Manager {
	return &Manager{
		workDir:    workDir,
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

// WorkDir returns the terraform working directory.
func (m *Manager) WorkDir() string {
	return m.workDir
}

// ListVMs returns all VMs managed by this terraform configuration.
// It combines terraform output with virsh status for accurate state.
func (m *Manager) ListVMs(ctx context.Context) ([]VMInfo, error) {
	// First, check if terraform state exists
	statePath := filepath.Join(m.workDir, "terraform.tfstate")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return nil, nil // No VMs yet
	}

	// Get terraform outputs
	outputs, err := m.getTerraformOutputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get terraform outputs: %w", err)
	}

	// If no outputs, no VMs
	if len(outputs) == 0 {
		return nil, nil
	}

	// Get VM name from outputs
	vmName, ok := outputs["vm_name"]
	if !ok || vmName == "" {
		return nil, nil
	}

	// Get VM info from virsh for accurate status
	vm, err := m.getVMInfo(ctx, vmName, outputs)
	if err != nil {
		return nil, err
	}

	return []VMInfo{*vm}, nil
}

// GetVM returns information about a specific VM.
func (m *Manager) GetVM(ctx context.Context, name string) (*VMInfo, error) {
	outputs, err := m.getTerraformOutputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get terraform outputs: %w", err)
	}

	vmName, ok := outputs["vm_name"]
	if !ok || vmName != name {
		return nil, fmt.Errorf("VM %q not found in terraform state", name)
	}

	return m.getVMInfo(ctx, name, outputs)
}

// StartVM starts a stopped VM using virsh.
func (m *Manager) StartVM(ctx context.Context, name string) error {
	return m.virshCommand(ctx, "start", name)
}

// StopVM gracefully shuts down a running VM using virsh.
func (m *Manager) StopVM(ctx context.Context, name string) error {
	return m.virshCommand(ctx, "shutdown", name)
}

// ForceStopVM immediately stops a VM (like pulling the power cord).
func (m *Manager) ForceStopVM(ctx context.Context, name string) error {
	return m.virshCommand(ctx, "destroy", name)
}

// DeleteVM destroys the VM using terraform destroy.
func (m *Manager) DeleteVM(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "terraform", "destroy", "-auto-approve")
	cmd.Dir = m.workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform destroy failed: %w\n%s", err, stderr.String())
	}

	return nil
}

// RefreshState runs terraform refresh to update the state.
func (m *Manager) RefreshState(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "terraform", "refresh")
	cmd.Dir = m.workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform refresh failed: %w\n%s", err, stderr.String())
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
	if ip == "" || ip == "pending" {
		return ""
	}
	return fmt.Sprintf("ssh ubuntu@%s", ip)
}

// getTerraformOutputs runs terraform output -json and parses the results.
func (m *Manager) getTerraformOutputs(ctx context.Context) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = m.workDir

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

// getVMInfo gets VM info by combining terraform outputs with virsh status.
func (m *Manager) getVMInfo(ctx context.Context, name string, outputs map[string]string) (*VMInfo, error) {
	vm := &VMInfo{
		Name:   name,
		Status: StatusUnknown,
	}

	// Get IP from outputs
	if ip, ok := outputs["vm_ip"]; ok {
		vm.IP = ip
	}

	// Get status from virsh
	status, err := m.getVirshStatus(ctx, name)
	if err != nil {
		// VM might not exist in virsh yet
		vm.Status = StatusNotFound
	} else {
		vm.Status = status
	}

	// Get detailed info from virsh dominfo
	info, err := m.getVirshDominfo(ctx, name)
	if err == nil {
		if cpus, ok := info["CPU(s)"]; ok {
			vm.CPUs, _ = strconv.Atoi(cpus)
		}
		if mem, ok := info["Max memory"]; ok {
			// Parse "2097152 KiB" to MB
			mem = strings.TrimSuffix(mem, " KiB")
			if kib, err := strconv.ParseInt(mem, 10, 64); err == nil {
				vm.MemoryMB = int(kib / 1024)
			}
		}
		if autostart, ok := info["Autostart"]; ok {
			vm.Autostart = autostart == "enable"
		}
	}

	return vm, nil
}

// getVirshStatus gets the current status of a VM from virsh.
func (m *Manager) getVirshStatus(ctx context.Context, name string) (VMStatus, error) {
	args := []string{"domstate", name}
	if m.libvirtURI != "" {
		args = append([]string{"-c", m.libvirtURI}, args...)
	}

	cmd := exec.CommandContext(ctx, "virsh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "failed to get domain") {
			return StatusNotFound, fmt.Errorf("VM not found")
		}
		return StatusUnknown, err
	}

	state := strings.TrimSpace(stdout.String())
	switch state {
	case "running":
		return StatusRunning, nil
	case "shut off":
		return StatusShutoff, nil
	case "paused":
		return StatusPaused, nil
	case "crashed":
		return StatusCrashed, nil
	default:
		return StatusUnknown, nil
	}
}

// getVirshDominfo gets detailed VM information from virsh dominfo.
func (m *Manager) getVirshDominfo(ctx context.Context, name string) (map[string]string, error) {
	args := []string{"dominfo", name}
	if m.libvirtURI != "" {
		args = append([]string{"-c", m.libvirtURI}, args...)
	}

	cmd := exec.CommandContext(ctx, "virsh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("virsh dominfo failed: %w\n%s", err, stderr.String())
	}

	info := make(map[string]string)
	for _, line := range strings.Split(stdout.String(), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			info[key] = value
		}
	}

	return info, nil
}

// virshCommand runs a virsh command with the configured libvirt URI.
func (m *Manager) virshCommand(ctx context.Context, action, name string) error {
	args := []string{action, name}
	if m.libvirtURI != "" {
		args = append([]string{"-c", m.libvirtURI}, args...)
	}

	cmd := exec.CommandContext(ctx, "virsh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("virsh %s failed: %w\n%s", action, err, stderr.String())
	}

	return nil
}

// IsInitialized returns true if the terraform directory has been initialized.
func (m *Manager) IsInitialized() bool {
	tfDir := filepath.Join(m.workDir, ".terraform")
	_, err := os.Stat(tfDir)
	return err == nil
}

// HasState returns true if a terraform state file exists.
func (m *Manager) HasState() bool {
	statePath := filepath.Join(m.workDir, "terraform.tfstate")
	info, err := os.Stat(statePath)
	if err != nil {
		return false
	}
	// Must be a regular file with content (not a directory)
	return info.Mode().IsRegular() && info.Size() > 0
}
