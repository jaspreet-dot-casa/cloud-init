// Package globalconfig provides global configuration management for ucli.
// Configuration is stored at ~/.config/ucli/config.yaml and includes
// the project path and all user settings.
package globalconfig

import (
	"os"
	"path/filepath"
)

const (
	// ConfigDirName is the name of the config directory under ~/.config.
	ConfigDirName = "ucli"
	// ConfigFileName is the name of the main config file.
	ConfigFileName = "config.yaml"
	// LegacySettingsFileName is the name of the legacy settings file to migrate from.
	LegacySettingsFileName = "settings.json"
	// StateDirName is the name of the state subdirectory.
	StateDirName = "state"
	// DownloadsFileName is the name of the downloads state file.
	DownloadsFileName = "downloads.json"
)

// GetConfigDir returns the config directory path (~/.config/ucli).
// Respects XDG_CONFIG_HOME if set.
func GetConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, ConfigDirName), nil
}

// GetConfigPath returns the full path to the config file.
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, ConfigFileName), nil
}

// GetLegacySettingsPath returns the path to the legacy settings.json file.
func GetLegacySettingsPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, LegacySettingsFileName), nil
}

// GetStateDir returns the path to the state directory.
func GetStateDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, StateDirName), nil
}

// GetDownloadsPath returns the path to the downloads state file.
func GetDownloadsPath() (string, error) {
	stateDir, err := GetStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, DownloadsFileName), nil
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	// Also ensure state directory exists
	stateDir, err := GetStateDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(stateDir, 0755)
}

// DefaultImagesDir returns the default images directory (~/Downloads).
func DefaultImagesDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "Downloads")
	}
	// Fallback to temp directory
	return filepath.Join(os.TempDir(), ConfigDirName, "images")
}
