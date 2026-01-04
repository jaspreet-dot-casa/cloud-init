package globalconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// legacySettings represents the old settings.json format.
type legacySettings struct {
	Version     string             `json:"version"`
	ImagesDir   string             `json:"images_dir"`
	CloudImages []legacyCloudImage `json:"cloud_images"`
	ISOs        []legacyISO        `json:"isos"`
	Preferences legacyPreferences  `json:"preferences"`
}

type legacyCloudImage struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Version  string    `json:"version"`
	Arch     string    `json:"arch"`
	Path     string    `json:"path"`
	URL      string    `json:"url,omitempty"`
	SHA256   string    `json:"sha256,omitempty"`
	Size     int64     `json:"size"`
	AddedAt  time.Time `json:"added_at"`
	Verified bool      `json:"verified,omitempty"`
}

type legacyISO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version,omitempty"`
	Path      string    `json:"path"`
	SourceURL string    `json:"source_url,omitempty"`
	AddedAt   time.Time `json:"added_at"`
}

type legacyPreferences struct {
	DefaultCloudImage string `json:"default_cloud_image,omitempty"`
	DefaultISO        string `json:"default_iso,omitempty"`
	AutoVerify        bool   `json:"auto_verify"`
}

// MigrateFromLegacy checks for legacy settings.json and migrates to config.yaml.
// Returns true if migration was performed, false if no migration needed.
func MigrateFromLegacy() (bool, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return false, err
	}

	legacyPath, err := GetLegacySettingsPath()
	if err != nil {
		return false, err
	}

	// Check if config.yaml already exists - no migration needed
	if _, err := os.Stat(configPath); err == nil {
		return false, nil
	}

	// Check if settings.json exists
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // No legacy file, no migration needed
		}
		return false, fmt.Errorf("failed to read legacy settings: %w", err)
	}

	// Parse legacy settings
	var legacy legacySettings
	if err := json.Unmarshal(data, &legacy); err != nil {
		return false, fmt.Errorf("failed to parse legacy settings: %w", err)
	}

	// Convert to new config format
	cfg := convertLegacyToConfig(&legacy)

	// Save new config
	if err := cfg.Save(); err != nil {
		return false, fmt.Errorf("failed to save migrated config: %w", err)
	}

	// Rename legacy file to .bak
	backupPath := legacyPath + ".bak"
	if err := os.Rename(legacyPath, backupPath); err != nil {
		// Log warning but don't fail - config was saved successfully
		fmt.Printf("Warning: could not rename %s to %s: %v\n", legacyPath, backupPath, err)
	}

	return true, nil
}

// convertLegacyToConfig converts legacy settings to new config format.
func convertLegacyToConfig(legacy *legacySettings) *Config {
	cfg := NewConfig()
	cfg.ImagesDir = legacy.ImagesDir

	// Convert cloud images
	for _, img := range legacy.CloudImages {
		cfg.CloudImages = append(cfg.CloudImages, CloudImage{
			ID:       img.ID,
			Name:     img.Name,
			Version:  img.Version,
			Arch:     img.Arch,
			Path:     img.Path,
			URL:      img.URL,
			SHA256:   img.SHA256,
			Size:     img.Size,
			AddedAt:  img.AddedAt,
			Verified: img.Verified,
		})
	}

	// Convert ISOs
	for _, iso := range legacy.ISOs {
		cfg.ISOs = append(cfg.ISOs, ISO{
			ID:        iso.ID,
			Name:      iso.Name,
			Version:   iso.Version,
			Path:      iso.Path,
			SourceURL: iso.SourceURL,
			AddedAt:   iso.AddedAt,
		})
	}

	// Convert preferences
	cfg.Preferences = Preferences{
		DefaultCloudImage: legacy.Preferences.DefaultCloudImage,
		DefaultISO:        legacy.Preferences.DefaultISO,
		AutoVerify:        legacy.Preferences.AutoVerify,
	}

	return cfg
}

// NeedsMigration checks if a migration from settings.json is needed.
func NeedsMigration() (bool, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return false, err
	}

	legacyPath, err := GetLegacySettingsPath()
	if err != nil {
		return false, err
	}

	// If config.yaml exists, no migration needed
	if _, err := os.Stat(configPath); err == nil {
		return false, nil
	}

	// If settings.json exists, migration is needed
	if _, err := os.Stat(legacyPath); err == nil {
		return true, nil
	}

	return false, nil
}
