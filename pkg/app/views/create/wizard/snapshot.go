package wizard

import (
	"time"

	"github.com/google/uuid"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
)

// ToSnapshot converts WizardData to a serializable WizardDataSnapshot.
func ToSnapshot(data *WizardData) settings.WizardDataSnapshot {
	snapshot := settings.WizardDataSnapshot{
		Username:    data.Username,
		Hostname:    data.Hostname,
		DisplayName: data.DisplayName,
		GitName:     data.GitName,
		GitEmail:    data.GitEmail,
		GitHubUser:  data.GitHubUser,
		SSHKeys:     data.SSHKeys,
		Packages:    data.Packages,
	}

	// Target-specific options
	switch data.Target {
	case deploy.TargetTerragrunt:
		snapshot.TerragruntOpts = &settings.TerragruntOptsSnapshot{
			CPUs:        data.TerragruntOpts.CPUs,
			MemoryMB:    data.TerragruntOpts.MemoryMB,
			DiskGB:      data.TerragruntOpts.DiskGB,
			Autostart:   data.TerragruntOpts.Autostart,
			UbuntuImage: data.TerragruntOpts.UbuntuImage,
		}
	case deploy.TargetMultipass:
		snapshot.MultipassOpts = &settings.MultipassOptsSnapshot{
			CPUs:          data.MultipassOpts.CPUs,
			MemoryMB:      data.MultipassOpts.MemoryMB,
			DiskGB:        data.MultipassOpts.DiskGB,
			UbuntuVersion: data.MultipassOpts.UbuntuVersion,
		}
	}

	return snapshot
}

// FromSnapshot populates WizardData from a WizardDataSnapshot.
func FromSnapshot(snapshot *settings.WizardDataSnapshot, target deploy.DeploymentTarget, data *WizardData) {
	data.Username = snapshot.Username
	data.Hostname = snapshot.Hostname
	data.DisplayName = snapshot.DisplayName
	data.GitName = snapshot.GitName
	data.GitEmail = snapshot.GitEmail
	data.GitHubUser = snapshot.GitHubUser
	data.SSHKeys = snapshot.SSHKeys
	data.Packages = snapshot.Packages
	data.Target = target

	// Target-specific options
	if snapshot.TerragruntOpts != nil {
		data.TerragruntOpts = deploy.TerragruntOptions{
			CPUs:        snapshot.TerragruntOpts.CPUs,
			MemoryMB:    snapshot.TerragruntOpts.MemoryMB,
			DiskGB:      snapshot.TerragruntOpts.DiskGB,
			Autostart:   snapshot.TerragruntOpts.Autostart,
			UbuntuImage: snapshot.TerragruntOpts.UbuntuImage,
		}
	}
	if snapshot.MultipassOpts != nil {
		data.MultipassOpts = deploy.MultipassOptions{
			CPUs:          snapshot.MultipassOpts.CPUs,
			MemoryMB:      snapshot.MultipassOpts.MemoryMB,
			DiskGB:        snapshot.MultipassOpts.DiskGB,
			UbuntuVersion: snapshot.MultipassOpts.UbuntuVersion,
		}
	}
}

// ToVMConfig creates a VMConfig from the current wizard state.
func ToVMConfig(data *WizardData, name, description string) settings.VMConfig {
	targetStr := string(data.Target)

	return settings.VMConfig{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Target:      targetStr,
		CreatedAt:   time.Now(),
		LastUsedAt:  time.Now(),
		Data:        ToSnapshot(data),
	}
}

// LoadFromConfig populates wizard state from a VMConfig.
// Returns an error if the config has an invalid or unsupported target.
func LoadFromConfig(cfg *settings.VMConfig, state *State) error {
	// Validate target before proceeding
	if !settings.IsValidTarget(cfg.Target) {
		// Fall back to terragrunt as default for unknown targets (e.g., old "usb" configs)
		cfg.Target = string(deploy.TargetTerragrunt)
	}

	target := deploy.DeploymentTarget(cfg.Target)
	FromSnapshot(&cfg.Data, target, &state.Data)

	// Set the target selection index based on target
	switch target {
	case deploy.TargetTerragrunt:
		state.TargetSelected = 0
	case deploy.TargetMultipass:
		state.TargetSelected = 1
	case deploy.TargetConfigOnly:
		state.TargetSelected = 2
	default:
		// Fallback to Terragrunt for any unrecognized target
		state.TargetSelected = 0
		state.Data.Target = deploy.TargetTerragrunt
	}

	// Initialize package selection map from packages list
	state.PackageSelected = make(map[string]bool)
	for _, pkg := range cfg.Data.Packages {
		state.PackageSelected[pkg] = true
	}

	return nil
}

// PresetApplyResult contains the result of applying a package preset.
type PresetApplyResult struct {
	Applied  int      // Number of packages successfully applied
	Skipped  int      // Number of packages skipped (not in registry)
	Invalid  []string // List of package names that were not in registry
}

// ApplyPackagePreset applies a package preset to the wizard state.
// Only packages that exist in the registry will be applied.
// Returns the result indicating which packages were applied.
func ApplyPackagePreset(preset *settings.PackagePreset, state *State) PresetApplyResult {
	result := PresetApplyResult{}

	// Clear existing selections
	for pkg := range state.PackageSelected {
		state.PackageSelected[pkg] = false
	}

	// Apply preset packages, but only if they exist in the registry
	for _, pkg := range preset.Packages {
		if state.Registry != nil && state.Registry.Get(pkg) != nil {
			state.PackageSelected[pkg] = true
			result.Applied++
		} else if state.Registry != nil {
			// Package not found in registry
			result.Skipped++
			result.Invalid = append(result.Invalid, pkg)
		} else {
			// No registry, apply blindly (for testing)
			state.PackageSelected[pkg] = true
			result.Applied++
		}
	}

	return result
}
