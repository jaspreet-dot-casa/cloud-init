package tfstate

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// VirshClient provides direct access to virsh commands.
// Use this for fast operations that don't need terraform state updates.
type VirshClient struct {
	uri string
}

// NewVirshClient creates a new virsh client.
func NewVirshClient(uri string) *VirshClient {
	if uri == "" {
		uri = "qemu:///system"
	}
	return &VirshClient{uri: uri}
}

// VirshVM represents basic VM info from virsh list.
type VirshVM struct {
	ID     int      // -1 if shut off
	Name   string
	Status VMStatus
}

// ListAll returns all VMs (running and shut off) from virsh.
func (c *VirshClient) ListAll(ctx context.Context) ([]VirshVM, error) {
	args := c.baseArgs("list", "--all")
	cmd := exec.CommandContext(ctx, "virsh", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("virsh list failed: %w\n%s", err, stderr.String())
	}

	return parseVirshList(stdout.String()), nil
}

// GetStatus returns the current status of a VM.
func (c *VirshClient) GetStatus(ctx context.Context, name string) (VMStatus, error) {
	args := c.baseArgs("domstate", name)
	cmd := exec.CommandContext(ctx, "virsh", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "failed to get domain") ||
			strings.Contains(stderr.String(), "Domain not found") {
			return StatusNotFound, nil
		}
		return StatusUnknown, fmt.Errorf("virsh domstate failed: %w", err)
	}

	return parseVMStatus(strings.TrimSpace(stdout.String())), nil
}

// Start starts a VM.
func (c *VirshClient) Start(ctx context.Context, name string) error {
	return c.run(ctx, "start", name)
}

// Shutdown gracefully shuts down a VM.
func (c *VirshClient) Shutdown(ctx context.Context, name string) error {
	return c.run(ctx, "shutdown", name)
}

// Destroy forcefully stops a VM (like pulling the power).
func (c *VirshClient) Destroy(ctx context.Context, name string) error {
	return c.run(ctx, "destroy", name)
}

// Reboot reboots a VM.
func (c *VirshClient) Reboot(ctx context.Context, name string) error {
	return c.run(ctx, "reboot", name)
}

// Suspend pauses a VM.
func (c *VirshClient) Suspend(ctx context.Context, name string) error {
	return c.run(ctx, "suspend", name)
}

// Resume resumes a paused VM.
func (c *VirshClient) Resume(ctx context.Context, name string) error {
	return c.run(ctx, "resume", name)
}

// SetAutostart enables or disables autostart for a VM.
func (c *VirshClient) SetAutostart(ctx context.Context, name string, enable bool) error {
	action := "autostart"
	if !enable {
		return c.runWithArgs(ctx, "autostart", "--disable", name)
	}
	return c.run(ctx, action, name)
}

// GetDomainInfo returns detailed information about a VM.
func (c *VirshClient) GetDomainInfo(ctx context.Context, name string) (map[string]string, error) {
	args := c.baseArgs("dominfo", name)
	cmd := exec.CommandContext(ctx, "virsh", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("virsh dominfo failed: %w\n%s", err, stderr.String())
	}

	return parseKeyValue(stdout.String()), nil
}

// GetIPAddress attempts to get the IP address of a VM from DHCP leases.
func (c *VirshClient) GetIPAddress(ctx context.Context, name string) (string, error) {
	// Try domifaddr first (requires qemu-guest-agent or lease lookup)
	args := c.baseArgs("domifaddr", name)
	cmd := exec.CommandContext(ctx, "virsh", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		ip := parseIPFromDomifaddr(stdout.String())
		if ip != "" {
			return ip, nil
		}
	}

	// Try getting from DHCP leases via net-dhcp-leases
	args = c.baseArgs("net-dhcp-leases", "default")
	cmd = exec.CommandContext(ctx, "virsh", args...)
	stdout.Reset()
	stderr.Reset()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		// Look for the VM's MAC and get its IP
		// This requires knowing the MAC address, which we'd get from domiflist
		macArgs := c.baseArgs("domiflist", name)
		macCmd := exec.CommandContext(ctx, "virsh", macArgs...)
		var macOut bytes.Buffer
		macCmd.Stdout = &macOut

		if macCmd.Run() == nil {
			mac := parseMACFromDomiflist(macOut.String())
			if mac != "" {
				ip := parseIPFromLeases(stdout.String(), mac)
				if ip != "" {
					return ip, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not determine IP address for VM %s", name)
}

// Console returns the virsh console command for a VM.
func (c *VirshClient) Console(name string) string {
	if c.uri != "" && c.uri != "qemu:///system" {
		return fmt.Sprintf("virsh -c %s console %s", c.uri, name)
	}
	return fmt.Sprintf("virsh console %s", name)
}

// VNCDisplay returns the VNC display port for a VM.
func (c *VirshClient) VNCDisplay(ctx context.Context, name string) (string, error) {
	args := c.baseArgs("vncdisplay", name)
	cmd := exec.CommandContext(ctx, "virsh", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("virsh vncdisplay failed: %w\n%s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsAvailable checks if virsh is installed and can connect to libvirt.
func (c *VirshClient) IsAvailable(ctx context.Context) error {
	// Check if virsh is installed
	if _, err := exec.LookPath("virsh"); err != nil {
		return fmt.Errorf("virsh is not installed: %w", err)
	}

	// Check if we can connect to libvirt
	args := c.baseArgs("version")
	cmd := exec.CommandContext(ctx, "virsh", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot connect to libvirt at %s: %w\n%s", c.uri, err, stderr.String())
	}

	return nil
}

// baseArgs returns the base arguments for virsh commands.
func (c *VirshClient) baseArgs(args ...string) []string {
	if c.uri != "" {
		return append([]string{"-c", c.uri}, args...)
	}
	return args
}

// run executes a simple virsh command.
func (c *VirshClient) run(ctx context.Context, action, name string) error {
	args := c.baseArgs(action, name)
	cmd := exec.CommandContext(ctx, "virsh", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("virsh %s %s failed: %w\n%s", action, name, err, stderr.String())
	}

	return nil
}

// runWithArgs executes a virsh command with additional arguments.
func (c *VirshClient) runWithArgs(ctx context.Context, args ...string) error {
	fullArgs := c.baseArgs(args...)
	cmd := exec.CommandContext(ctx, "virsh", fullArgs...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("virsh %s failed: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}

	return nil
}

// parseVirshList parses the output of virsh list --all.
func parseVirshList(output string) []VirshVM {
	var vms []VirshVM

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip header and separator lines
		if line == "" || strings.HasPrefix(line, "Id") || strings.HasPrefix(line, "--") {
			continue
		}

		// Parse: " Id   Name           State"
		// Example: " 1    my-vm          running"
		// Example: " -    stopped-vm     shut off"
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		vm := VirshVM{Name: fields[1]}

		// Parse ID
		if fields[0] == "-" {
			vm.ID = -1
		} else {
			vm.ID, _ = strconv.Atoi(fields[0])
		}

		// Parse status (may be multiple words like "shut off")
		status := strings.Join(fields[2:], " ")
		vm.Status = parseVMStatus(status)

		vms = append(vms, vm)
	}

	return vms
}

// parseVMStatus converts a virsh state string to VMStatus.
func parseVMStatus(state string) VMStatus {
	switch strings.ToLower(state) {
	case "running":
		return StatusRunning
	case "shut off", "shutoff":
		return StatusShutoff
	case "paused":
		return StatusPaused
	case "crashed":
		return StatusCrashed
	case "idle":
		return StatusStopped
	default:
		return StatusUnknown
	}
}

// parseKeyValue parses key: value output into a map.
func parseKeyValue(output string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}
	return result
}

// parseIPFromDomifaddr parses IP from virsh domifaddr output.
func parseIPFromDomifaddr(output string) string {
	// Format:
	//  Name       MAC address          Protocol     Address
	// ---------------------------------------------------------------
	//  vnet0      52:54:00:xx:xx:xx    ipv4         192.168.122.10/24
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "ipv4") {
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.Contains(field, ".") && strings.Contains(field, "/") {
					// Extract IP from CIDR notation
					parts := strings.Split(field, "/")
					return parts[0]
				}
			}
		}
	}
	return ""
}

// parseMACFromDomiflist parses MAC address from virsh domiflist output.
func parseMACFromDomiflist(output string) string {
	// Format:
	//  Interface  Type       Source     Model       MAC
	// -------------------------------------------------------
	//  vnet0      network    default    virtio      52:54:00:xx:xx:xx
	macRegex := regexp.MustCompile(`([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}`)
	match := macRegex.FindString(output)
	return strings.ToLower(match)
}

// parseIPFromLeases parses IP from net-dhcp-leases output for a given MAC.
func parseIPFromLeases(output, mac string) string {
	// Format:
	//  Expiry Time          MAC address        Protocol  IP address       Hostname   Client ID
	// -----------------------------------------------------------------------------------------
	//  2024-01-01 00:00:00  52:54:00:xx:xx:xx  ipv4      192.168.122.10/24  my-vm    ...
	mac = strings.ToLower(mac)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), mac) {
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.Contains(field, ".") && strings.Contains(field, "/") {
					// Extract IP from CIDR notation
					parts := strings.Split(field, "/")
					return parts[0]
				}
			}
		}
	}
	return ""
}
