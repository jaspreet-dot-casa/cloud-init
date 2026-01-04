package tfstate

import (
	"os"
	"path/filepath"
)

// DiscoverMachines scans the tf/ directory for machine subdirectories.
// Each subdirectory with a main.tf file is considered a machine.
func DiscoverMachines(tfDir string) ([]string, error) {
	entries, err := os.ReadDir(tfDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // tf/ doesn't exist yet, no machines
		}
		return nil, err
	}

	var machines []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories
		if entry.Name()[0] == '.' {
			continue
		}

		// Check if this directory has a main.tf file
		mainTF := filepath.Join(tfDir, entry.Name(), "main.tf")
		if _, err := os.Stat(mainTF); err == nil {
			machines = append(machines, entry.Name())
		}
	}

	return machines, nil
}

// MachineExists checks if a machine with the given name exists.
func MachineExists(tfDir, name string) bool {
	machineDir := filepath.Join(tfDir, name)
	mainTF := filepath.Join(machineDir, "main.tf")
	_, err := os.Stat(mainTF)
	return err == nil
}

// GetMachineDir returns the full path to a machine's terraform directory.
func GetMachineDir(tfDir, name string) string {
	return filepath.Join(tfDir, name)
}

// MachineHasState checks if a machine has terraform state.
func MachineHasState(tfDir, name string) bool {
	statePath := filepath.Join(tfDir, name, "terraform.tfstate")
	info, err := os.Stat(statePath)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular() && info.Size() > 0
}

// MachineIsInitialized checks if a machine's terraform has been initialized.
func MachineIsInitialized(tfDir, name string) bool {
	tfProviderDir := filepath.Join(tfDir, name, ".terraform")
	info, err := os.Stat(tfProviderDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}
