package settings

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettings(t *testing.T) {
	settings := NewSettings()

	assert.Equal(t, Version, settings.Version)
	assert.Empty(t, settings.CloudImages)
	assert.Empty(t, settings.VMConfigs)
	assert.Empty(t, settings.PackagePresets)
}

func TestSettings_AddCloudImage(t *testing.T) {
	settings := NewSettings()

	img := CloudImage{
		ID:      "ubuntu-24.04-amd64",
		Name:    "Ubuntu 24.04 LTS",
		Version: "24.04",
		Arch:    "amd64",
		Path:    "/path/to/image.img",
		AddedAt: time.Now(),
	}

	settings.AddCloudImage(img)

	assert.Len(t, settings.CloudImages, 1)
	assert.Equal(t, img.ID, settings.CloudImages[0].ID)
}

func TestSettings_AddCloudImage_Replace(t *testing.T) {
	settings := NewSettings()

	img1 := CloudImage{ID: "test", Name: "First", Path: "/first"}
	img2 := CloudImage{ID: "test", Name: "Second", Path: "/second"}

	settings.AddCloudImage(img1)
	settings.AddCloudImage(img2)

	assert.Len(t, settings.CloudImages, 1)
	assert.Equal(t, "Second", settings.CloudImages[0].Name)
	assert.Equal(t, "/second", settings.CloudImages[0].Path)
}

func TestSettings_RemoveCloudImage(t *testing.T) {
	settings := NewSettings()
	settings.AddCloudImage(CloudImage{ID: "img1", Name: "Image 1"})
	settings.AddCloudImage(CloudImage{ID: "img2", Name: "Image 2"})

	removed := settings.RemoveCloudImage("img1")

	assert.True(t, removed)
	assert.Len(t, settings.CloudImages, 1)
	assert.Equal(t, "img2", settings.CloudImages[0].ID)
}

func TestSettings_RemoveCloudImage_NotFound(t *testing.T) {
	settings := NewSettings()

	removed := settings.RemoveCloudImage("nonexistent")

	assert.False(t, removed)
}

func TestSettings_FindCloudImage(t *testing.T) {
	settings := NewSettings()
	settings.AddCloudImage(CloudImage{ID: "img1", Name: "Image 1"})
	settings.AddCloudImage(CloudImage{ID: "img2", Name: "Image 2"})

	found := settings.FindCloudImage("img2")

	require.NotNil(t, found)
	assert.Equal(t, "Image 2", found.Name)
}

func TestSettings_FindCloudImage_NotFound(t *testing.T) {
	settings := NewSettings()

	found := settings.FindCloudImage("nonexistent")

	assert.Nil(t, found)
}

func TestSettings_VMConfig(t *testing.T) {
	settings := NewSettings()

	cfg := VMConfig{
		ID:          "cfg1",
		Name:        "Test Config",
		Description: "A test configuration",
		Target:      "terraform",
		CreatedAt:   time.Now(),
		Data: WizardDataSnapshot{
			Username: "testuser",
			Hostname: "testhost",
			Packages: []string{"git", "vim"},
		},
	}

	settings.AddVMConfig(cfg)
	assert.Len(t, settings.VMConfigs, 1)
	assert.Equal(t, "cfg1", settings.VMConfigs[0].ID)

	found := settings.FindVMConfig("cfg1")
	require.NotNil(t, found)
	assert.Equal(t, "Test Config", found.Name)

	removed := settings.RemoveVMConfig("cfg1")
	assert.True(t, removed)
	assert.Len(t, settings.VMConfigs, 0)
}

func TestSettings_PackagePreset(t *testing.T) {
	settings := NewSettings()

	preset := PackagePreset{
		ID:          "preset1",
		Name:        "Dev Tools",
		Description: "Development tools",
		Packages:    []string{"git", "neovim", "ripgrep"},
		CreatedAt:   time.Now(),
	}

	settings.AddPackagePreset(preset)
	assert.Len(t, settings.PackagePresets, 1)

	found := settings.FindPackagePreset("preset1")
	require.NotNil(t, found)
	assert.Equal(t, "Dev Tools", found.Name)

	removed := settings.RemovePackagePreset("preset1")
	assert.True(t, removed)
	assert.Len(t, settings.PackagePresets, 0)
}

func TestStore_SaveAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	store := NewStoreWithDir(tmpDir)

	// Create settings
	settings := NewSettings()
	settings.AddCloudImage(CloudImage{
		ID:   "test-image",
		Name: "Test Image",
		Path: "/path/to/image",
	})
	settings.AppSettings.TerraformDir = "/custom/terraform"
	settings.AppSettings.DefaultTarget = "terraform"

	// Save
	err := store.Save(settings)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(store.SettingsPath())
	require.NoError(t, err)

	// Load
	loaded, err := store.Load()
	require.NoError(t, err)

	assert.Equal(t, settings.AppSettings.TerraformDir, loaded.AppSettings.TerraformDir)
	assert.Len(t, loaded.CloudImages, 1)
	assert.Equal(t, "test-image", loaded.CloudImages[0].ID)
}

func TestStore_Load_DefaultSettings(t *testing.T) {
	// Create temp directory (empty)
	tmpDir := t.TempDir()
	store := NewStoreWithDir(tmpDir)

	// Load from non-existent file
	settings, err := store.Load()
	require.NoError(t, err)

	assert.Equal(t, Version, settings.Version)
	assert.Empty(t, settings.CloudImages)
}

func TestStore_EnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "ucli")
	store := NewStoreWithDir(configDir)

	err := store.EnsureDir()
	require.NoError(t, err)

	// Check directories exist
	_, err = os.Stat(configDir)
	require.NoError(t, err)

	stateDir := filepath.Join(configDir, StateDirName)
	_, err = os.Stat(stateDir)
	require.NoError(t, err)
}

func TestStore_DownloadState(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithDir(tmpDir)

	// Create download state
	state := NewDownloadState()
	state.ActiveDownloads = []Download{
		{
			ID:         "dl1",
			URL:        "https://example.com/image.img",
			DestPath:   "/path/to/dest",
			TotalBytes: 1000,
			Downloaded: 500,
			Status:     StatusDownloading,
		},
	}

	// Save
	err := store.SaveDownloadState(state)
	require.NoError(t, err)

	// Load
	loaded, err := store.LoadDownloadState()
	require.NoError(t, err)

	assert.Len(t, loaded.ActiveDownloads, 1)
	assert.Equal(t, "dl1", loaded.ActiveDownloads[0].ID)
	assert.Equal(t, StatusDownloading, loaded.ActiveDownloads[0].Status)
}

func TestStore_LoadDownloadState_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithDir(tmpDir)

	// Load from non-existent file
	state, err := store.LoadDownloadState()
	require.NoError(t, err)

	assert.NotNil(t, state)
	assert.Empty(t, state.ActiveDownloads)
}

func TestDefaultPackagePresets(t *testing.T) {
	presets := DefaultPackagePresets()

	assert.Len(t, presets, 3)
	assert.Equal(t, "minimal", presets[0].ID)
	assert.Equal(t, "dev-tools", presets[1].ID)
	assert.Equal(t, "full-stack", presets[2].ID)
}
