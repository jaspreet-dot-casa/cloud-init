// Package usb provides USB device detection and writing functionality.
package usb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// Device represents a USB storage device.
type Device struct {
	Path       string // Device path: /dev/disk2 (macOS) or /dev/sdb (Linux)
	Name       string // Device name: "SanDisk Ultra"
	Size       string // Human-readable size: "16 GB"
	SizeBytes  int64  // Size in bytes
	IsExternal bool   // True if external/removable
	IsUSB      bool   // True if USB device
	Partitions int    // Number of partitions
}

// DetectDevices returns all external USB storage devices.
func DetectDevices() ([]Device, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectDevicesDarwin()
	case "linux":
		return detectDevicesLinux()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// UnmountDevice unmounts all partitions of a device before writing.
func UnmountDevice(devicePath string) error {
	switch runtime.GOOS {
	case "darwin":
		return unmountDeviceDarwin(devicePath)
	case "linux":
		return unmountDeviceLinux(devicePath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// IsSystemDisk returns true if the device is likely a system disk.
func IsSystemDisk(devicePath string) bool {
	switch runtime.GOOS {
	case "darwin":
		// On macOS, disk0 is typically the system disk
		return devicePath == "/dev/disk0" || devicePath == "disk0"
	case "linux":
		// On Linux, sda or nvme0n1 are typically system disks
		return devicePath == "/dev/sda" || devicePath == "sda" ||
			strings.HasPrefix(devicePath, "/dev/nvme0n1") ||
			strings.HasPrefix(devicePath, "nvme0n1")
	default:
		return false
	}
}

// --- macOS Implementation ---

// diskutilList represents the output of `diskutil list -plist` in JSON format
type diskutilListOutput struct {
	AllDisksAndPartitions []diskutilDisk `json:"AllDisksAndPartitions"`
}

type diskutilDisk struct {
	DeviceIdentifier string            `json:"DeviceIdentifier"`
	Size             int64             `json:"Size"`
	Content          string            `json:"Content"`
	Partitions       []diskutilPartition `json:"Partitions"`
}

type diskutilPartition struct {
	DeviceIdentifier string `json:"DeviceIdentifier"`
	Size             int64  `json:"Size"`
	Content          string `json:"Content"`
}

func detectDevicesDarwin() ([]Device, error) {
	// Get disk list
	cmd := exec.Command("diskutil", "list", "-plist")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run diskutil list: %w", err)
	}

	// Convert plist to JSON using plutil
	plistCmd := exec.Command("plutil", "-convert", "json", "-o", "-", "-")
	plistCmd.Stdin = bytes.NewReader(output)
	jsonOutput, err := plistCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to convert plist to json: %w", err)
	}

	var diskList diskutilListOutput
	if err := json.Unmarshal(jsonOutput, &diskList); err != nil {
		return nil, fmt.Errorf("failed to parse diskutil output: %w", err)
	}

	var devices []Device
	for _, disk := range diskList.AllDisksAndPartitions {
		// Get detailed info for this disk
		info, err := getDiskInfoDarwin(disk.DeviceIdentifier)
		if err != nil {
			continue // Skip disks we can't get info for
		}

		// Only include external/removable devices
		if !info.IsExternal && !info.IsUSB {
			continue
		}

		// Skip system disk
		if IsSystemDisk("/dev/" + disk.DeviceIdentifier) {
			continue
		}

		info.Path = "/dev/" + disk.DeviceIdentifier
		info.SizeBytes = disk.Size
		info.Size = formatSize(disk.Size)
		info.Partitions = len(disk.Partitions)

		devices = append(devices, info)
	}

	return devices, nil
}

func getDiskInfoDarwin(deviceID string) (Device, error) {
	cmd := exec.Command("diskutil", "info", deviceID)
	output, err := cmd.Output()
	if err != nil {
		return Device{}, err
	}

	info := Device{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Device / Media Name:") {
			info.Name = strings.TrimSpace(strings.TrimPrefix(line, "Device / Media Name:"))
		}
		if strings.HasPrefix(line, "Removable Media:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "Removable Media:"))
			info.IsExternal = strings.Contains(strings.ToLower(value), "removable") ||
				strings.Contains(strings.ToLower(value), "yes")
		}
		if strings.HasPrefix(line, "Protocol:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "Protocol:"))
			info.IsUSB = strings.Contains(strings.ToLower(value), "usb")
		}
		if strings.HasPrefix(line, "Device Location:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "Device Location:"))
			if strings.Contains(strings.ToLower(value), "external") {
				info.IsExternal = true
			}
		}
	}

	// If no name found, use a default
	if info.Name == "" {
		info.Name = "Unknown Device"
	}

	return info, nil
}

func unmountDeviceDarwin(devicePath string) error {
	// Remove /dev/ prefix if present for diskutil
	deviceID := strings.TrimPrefix(devicePath, "/dev/")

	cmd := exec.Command("diskutil", "unmountDisk", deviceID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unmount %s: %s", devicePath, string(output))
	}
	return nil
}

// --- Linux Implementation ---

type lsblkOutput struct {
	BlockDevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name       string        `json:"name"`
	Size       string        `json:"size"`
	Type       string        `json:"type"`
	Removable  string        `json:"rm"`
	Tran       string        `json:"tran"` // Transport type (usb, sata, etc.)
	Model      string        `json:"model"`
	Children   []lsblkDevice `json:"children"`
}

func detectDevicesLinux() ([]Device, error) {
	cmd := exec.Command("lsblk", "-J", "-o", "NAME,SIZE,TYPE,RM,TRAN,MODEL")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run lsblk: %w", err)
	}

	var lsblk lsblkOutput
	if err := json.Unmarshal(output, &lsblk); err != nil {
		return nil, fmt.Errorf("failed to parse lsblk output: %w", err)
	}

	var devices []Device
	for _, dev := range lsblk.BlockDevices {
		// Only include disk devices (not partitions)
		if dev.Type != "disk" {
			continue
		}

		// Check if removable or USB
		isRemovable := dev.Removable == "1"
		isUSB := dev.Tran == "usb"

		if !isRemovable && !isUSB {
			continue
		}

		// Skip system disk
		if IsSystemDisk(dev.Name) {
			continue
		}

		name := dev.Model
		if name == "" {
			name = "USB Device"
		}
		name = strings.TrimSpace(name)

		device := Device{
			Path:       "/dev/" + dev.Name,
			Name:       name,
			Size:       dev.Size,
			IsExternal: isRemovable,
			IsUSB:      isUSB,
			Partitions: len(dev.Children),
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func unmountDeviceLinux(devicePath string) error {
	// Get device name without /dev/
	devName := strings.TrimPrefix(devicePath, "/dev/")

	// Find all mounted partitions for this device
	cmd := exec.Command("lsblk", "-n", "-o", "NAME,MOUNTPOINT", devicePath)
	output, err := cmd.Output()
	if err != nil {
		// Device might not have any partitions
		return nil
	}

	// Parse output and unmount each mounted partition
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] != "" {
			// This partition is mounted, unmount it
			partPath := "/dev/" + strings.TrimPrefix(fields[0], "├─")
			partPath = strings.TrimPrefix(partPath, "└─")
			if !strings.HasPrefix(partPath, "/dev/") {
				partPath = "/dev/" + devName + fields[0]
			}

			umountCmd := exec.Command("umount", fields[1])
			if err := umountCmd.Run(); err != nil {
				return fmt.Errorf("failed to unmount %s: %w", fields[1], err)
			}
		}
	}

	return nil
}

// --- Utilities ---

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// ValidateDevice checks if a device path is valid and safe to write to.
func ValidateDevice(devicePath string) error {
	if devicePath == "" {
		return fmt.Errorf("device path is required")
	}

	// Check it's not a system disk
	if IsSystemDisk(devicePath) {
		return fmt.Errorf("refusing to write to system disk: %s", devicePath)
	}

	// Validate path format
	switch runtime.GOOS {
	case "darwin":
		matched, _ := regexp.MatchString(`^/dev/disk\d+$`, devicePath)
		if !matched {
			return fmt.Errorf("invalid macOS device path: %s (expected /dev/diskN)", devicePath)
		}
	case "linux":
		matched, _ := regexp.MatchString(`^/dev/sd[a-z]$|^/dev/nvme\d+n\d+$`, devicePath)
		if !matched {
			return fmt.Errorf("invalid Linux device path: %s", devicePath)
		}
	}

	return nil
}
