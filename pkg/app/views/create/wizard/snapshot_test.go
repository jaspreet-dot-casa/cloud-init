package wizard

import (
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSnapshot_Terragrunt(t *testing.T) {
	data := &WizardData{
		Target:      deploy.TargetTerragrunt,
		Username:    "testuser",
		Hostname:    "testhost",
		DisplayName: "Test User",
		GitName:     "Test User",
		GitEmail:    "test@example.com",
		GitHubUser:  "testgh",
		SSHKeys:     []string{"ssh-rsa AAAA..."},
		Packages:    []string{"git", "vim", "neovim"},
		TerragruntOpts: deploy.TerragruntOptions{
			CPUs:        4,
			MemoryMB:    8192,
			DiskGB:      50,
			Autostart:   true,
			UbuntuImage: "/path/to/image.img",
		},
	}

	snapshot := ToSnapshot(data)

	assert.Equal(t, "testuser", snapshot.Username)
	assert.Equal(t, "testhost", snapshot.Hostname)
	assert.Equal(t, "Test User", snapshot.DisplayName)
	assert.Equal(t, "Test User", snapshot.GitName)
	assert.Equal(t, "test@example.com", snapshot.GitEmail)
	assert.Equal(t, "testgh", snapshot.GitHubUser)
	assert.Equal(t, []string{"ssh-rsa AAAA..."}, snapshot.SSHKeys)
	assert.Equal(t, []string{"git", "vim", "neovim"}, snapshot.Packages)

	require.NotNil(t, snapshot.TerragruntOpts)
	assert.Equal(t, 4, snapshot.TerragruntOpts.CPUs)
	assert.Equal(t, 8192, snapshot.TerragruntOpts.MemoryMB)
	assert.Equal(t, 50, snapshot.TerragruntOpts.DiskGB)
	assert.True(t, snapshot.TerragruntOpts.Autostart)
	assert.Equal(t, "/path/to/image.img", snapshot.TerragruntOpts.UbuntuImage)

	assert.Nil(t, snapshot.MultipassOpts)
}

func TestToSnapshot_Multipass(t *testing.T) {
	data := &WizardData{
		Target:   deploy.TargetMultipass,
		Username: "testuser",
		Hostname: "testhost",
		Packages: []string{"git"},
		MultipassOpts: deploy.MultipassOptions{
			CPUs:          2,
			MemoryMB:      4096,
			DiskGB:        20,
			UbuntuVersion: "24.04",
		},
	}

	snapshot := ToSnapshot(data)

	assert.Nil(t, snapshot.TerragruntOpts)
	require.NotNil(t, snapshot.MultipassOpts)
	assert.Equal(t, 2, snapshot.MultipassOpts.CPUs)
	assert.Equal(t, 4096, snapshot.MultipassOpts.MemoryMB)
	assert.Equal(t, 20, snapshot.MultipassOpts.DiskGB)
	assert.Equal(t, "24.04", snapshot.MultipassOpts.UbuntuVersion)
}

func TestToSnapshot_ConfigOnly(t *testing.T) {
	data := &WizardData{
		Target:   deploy.TargetConfigOnly,
		Username: "testuser",
		Hostname: "testhost",
		Packages: []string{"git"},
	}

	snapshot := ToSnapshot(data)

	assert.Nil(t, snapshot.TerragruntOpts)
	assert.Nil(t, snapshot.MultipassOpts)
	assert.Equal(t, "testuser", snapshot.Username)
}

func TestFromSnapshot_Terragrunt(t *testing.T) {
	snapshot := &settings.WizardDataSnapshot{
		Username:    "testuser",
		Hostname:    "testhost",
		DisplayName: "Test User",
		GitName:     "Test User",
		GitEmail:    "test@example.com",
		GitHubUser:  "testgh",
		SSHKeys:     []string{"ssh-rsa AAAA..."},
		Packages:    []string{"git", "vim"},
		TerragruntOpts: &settings.TerragruntOptsSnapshot{
			CPUs:        4,
			MemoryMB:    8192,
			DiskGB:      50,
			Autostart:   true,
			UbuntuImage: "/path/to/image.img",
		},
	}

	data := &WizardData{}
	FromSnapshot(snapshot, deploy.TargetTerragrunt, data)

	assert.Equal(t, deploy.TargetTerragrunt, data.Target)
	assert.Equal(t, "testuser", data.Username)
	assert.Equal(t, "testhost", data.Hostname)
	assert.Equal(t, "Test User", data.DisplayName)
	assert.Equal(t, "Test User", data.GitName)
	assert.Equal(t, "test@example.com", data.GitEmail)
	assert.Equal(t, "testgh", data.GitHubUser)
	assert.Equal(t, []string{"ssh-rsa AAAA..."}, data.SSHKeys)
	assert.Equal(t, []string{"git", "vim"}, data.Packages)

	assert.Equal(t, 4, data.TerragruntOpts.CPUs)
	assert.Equal(t, 8192, data.TerragruntOpts.MemoryMB)
	assert.Equal(t, 50, data.TerragruntOpts.DiskGB)
	assert.True(t, data.TerragruntOpts.Autostart)
	assert.Equal(t, "/path/to/image.img", data.TerragruntOpts.UbuntuImage)
}

func TestFromSnapshot_Multipass(t *testing.T) {
	snapshot := &settings.WizardDataSnapshot{
		Username: "testuser",
		Hostname: "testhost",
		Packages: []string{"git"},
		MultipassOpts: &settings.MultipassOptsSnapshot{
			CPUs:          2,
			MemoryMB:      4096,
			DiskGB:        20,
			UbuntuVersion: "24.04",
		},
	}

	data := &WizardData{}
	FromSnapshot(snapshot, deploy.TargetMultipass, data)

	assert.Equal(t, deploy.TargetMultipass, data.Target)
	assert.Equal(t, 2, data.MultipassOpts.CPUs)
	assert.Equal(t, 4096, data.MultipassOpts.MemoryMB)
	assert.Equal(t, 20, data.MultipassOpts.DiskGB)
	assert.Equal(t, "24.04", data.MultipassOpts.UbuntuVersion)
}

func TestToVMConfig(t *testing.T) {
	data := &WizardData{
		Target:   deploy.TargetTerragrunt,
		Username: "testuser",
		Hostname: "testhost",
		Packages: []string{"git"},
		TerragruntOpts: deploy.TerragruntOptions{
			CPUs:     4,
			MemoryMB: 8192,
		},
	}

	cfg := ToVMConfig(data, "my-config", "A test config")

	assert.NotEmpty(t, cfg.ID)
	assert.Equal(t, "my-config", cfg.Name)
	assert.Equal(t, "A test config", cfg.Description)
	assert.Equal(t, "terragrunt", cfg.Target)
	assert.False(t, cfg.CreatedAt.IsZero())
	assert.False(t, cfg.LastUsedAt.IsZero())
	assert.Equal(t, "testuser", cfg.Data.Username)
	assert.Equal(t, "testhost", cfg.Data.Hostname)
}

func TestLoadFromConfig(t *testing.T) {
	cfg := &settings.VMConfig{
		ID:          "test-id",
		Name:        "test-config",
		Description: "Test description",
		Target:      "terragrunt",
		Data: settings.WizardDataSnapshot{
			Username: "testuser",
			Hostname: "testhost",
			Packages: []string{"git", "vim"},
			TerragruntOpts: &settings.TerragruntOptsSnapshot{
				CPUs:     4,
				MemoryMB: 8192,
			},
		},
	}

	state := NewState()
	LoadFromConfig(cfg, state)

	assert.Equal(t, deploy.TargetTerragrunt, state.Data.Target)
	assert.Equal(t, 0, state.TargetSelected) // Terragrunt is index 0
	assert.Equal(t, "testuser", state.Data.Username)
	assert.Equal(t, "testhost", state.Data.Hostname)
	assert.Equal(t, 4, state.Data.TerragruntOpts.CPUs)

	// Check package selection map was populated
	assert.True(t, state.PackageSelected["git"])
	assert.True(t, state.PackageSelected["vim"])
}

func TestLoadFromConfig_Multipass(t *testing.T) {
	cfg := &settings.VMConfig{
		ID:     "test-id",
		Name:   "test-config",
		Target: "multipass",
		Data: settings.WizardDataSnapshot{
			Username: "testuser",
			Packages: []string{"git"},
		},
	}

	state := NewState()
	LoadFromConfig(cfg, state)

	assert.Equal(t, deploy.TargetMultipass, state.Data.Target)
	assert.Equal(t, 1, state.TargetSelected) // Multipass is index 1
}

func TestLoadFromConfig_ConfigOnly(t *testing.T) {
	cfg := &settings.VMConfig{
		ID:     "test-id",
		Name:   "test-config",
		Target: "config",
		Data: settings.WizardDataSnapshot{
			Username: "testuser",
			Packages: []string{"git"},
		},
	}

	state := NewState()
	LoadFromConfig(cfg, state)

	assert.Equal(t, deploy.TargetConfigOnly, state.Data.Target)
	assert.Equal(t, 2, state.TargetSelected) // ConfigOnly is index 2
}

func TestApplyPackagePreset(t *testing.T) {
	state := NewState()
	state.PackageSelected = map[string]bool{
		"git":     true,
		"vim":     true,
		"neovim":  false,
		"ripgrep": true,
	}

	preset := &settings.PackagePreset{
		ID:       "test-preset",
		Name:     "Test Preset",
		Packages: []string{"neovim", "fzf"},
	}

	result := ApplyPackagePreset(preset, state)

	// Previous selections should be cleared
	assert.False(t, state.PackageSelected["git"])
	assert.False(t, state.PackageSelected["vim"])
	assert.False(t, state.PackageSelected["ripgrep"])

	// Preset packages should be selected (no registry, so all applied)
	assert.True(t, state.PackageSelected["neovim"])
	assert.True(t, state.PackageSelected["fzf"])

	// Verify result (no registry means all packages applied)
	assert.Equal(t, 2, result.Applied)
	assert.Equal(t, 0, result.Skipped)
}

func TestApplyPackagePreset_EmptyPreset(t *testing.T) {
	state := NewState()
	state.PackageSelected = map[string]bool{
		"git": true,
		"vim": true,
	}

	preset := &settings.PackagePreset{
		ID:       "empty",
		Name:     "Empty",
		Packages: []string{},
	}

	result := ApplyPackagePreset(preset, state)

	// All should be deselected
	assert.False(t, state.PackageSelected["git"])
	assert.False(t, state.PackageSelected["vim"])

	// Verify result
	assert.Equal(t, 0, result.Applied)
	assert.Equal(t, 0, result.Skipped)
}

func TestRoundTrip_TerragruntConfig(t *testing.T) {
	// Create original data
	original := &WizardData{
		Target:      deploy.TargetTerragrunt,
		Username:    "testuser",
		Hostname:    "testhost",
		DisplayName: "Test User",
		GitName:     "Git User",
		GitEmail:    "git@example.com",
		GitHubUser:  "ghuser",
		SSHKeys:     []string{"key1", "key2"},
		Packages:    []string{"git", "vim", "neovim"},
		TerragruntOpts: deploy.TerragruntOptions{
			CPUs:        4,
			MemoryMB:    8192,
			DiskGB:      100,
			Autostart:   true,
			UbuntuImage: "/path/to/image.img",
		},
	}

	// Convert to VMConfig
	cfg := ToVMConfig(original, "round-trip-test", "Testing round trip")

	// Load back into a new state
	state := NewState()
	LoadFromConfig(&cfg, state)

	// Verify data matches
	assert.Equal(t, original.Target, state.Data.Target)
	assert.Equal(t, original.Username, state.Data.Username)
	assert.Equal(t, original.Hostname, state.Data.Hostname)
	assert.Equal(t, original.DisplayName, state.Data.DisplayName)
	assert.Equal(t, original.GitName, state.Data.GitName)
	assert.Equal(t, original.GitEmail, state.Data.GitEmail)
	assert.Equal(t, original.GitHubUser, state.Data.GitHubUser)
	assert.Equal(t, original.SSHKeys, state.Data.SSHKeys)
	assert.Equal(t, original.Packages, state.Data.Packages)
	assert.Equal(t, original.TerragruntOpts.CPUs, state.Data.TerragruntOpts.CPUs)
	assert.Equal(t, original.TerragruntOpts.MemoryMB, state.Data.TerragruntOpts.MemoryMB)
	assert.Equal(t, original.TerragruntOpts.DiskGB, state.Data.TerragruntOpts.DiskGB)
	assert.Equal(t, original.TerragruntOpts.Autostart, state.Data.TerragruntOpts.Autostart)
	assert.Equal(t, original.TerragruntOpts.UbuntuImage, state.Data.TerragruntOpts.UbuntuImage)
}
