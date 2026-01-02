package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// ConfigDirName is the name of the config directory.
	ConfigDirName = "ucli"
	// SettingsFileName is the name of the settings file.
	SettingsFileName = "settings.json"
	// DownloadsFileName is the name of the downloads state file.
	DownloadsFileName = "downloads.json"
	// StateDirName is the name of the state subdirectory.
	StateDirName = "state"
)

// Store manages persistent settings storage.
type Store struct {
	configDir string
	settings  *Settings
}

// NewStore creates a new settings store.
func NewStore() (*Store, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	return &Store{
		configDir: configDir,
	}, nil
}

// NewStoreWithDir creates a new settings store with a custom directory.
func NewStoreWithDir(dir string) *Store {
	return &Store{
		configDir: dir,
	}
}

// getConfigDir returns the config directory path.
func getConfigDir() (string, error) {
	// Use XDG_CONFIG_HOME if set, otherwise ~/.config
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

// EnsureDir ensures the config directory exists.
func (s *Store) EnsureDir() error {
	// Create main config dir
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create state subdirectory
	stateDir := filepath.Join(s.configDir, StateDirName)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	return nil
}

// ConfigDir returns the config directory path.
func (s *Store) ConfigDir() string {
	return s.configDir
}

// SettingsPath returns the path to the settings file.
func (s *Store) SettingsPath() string {
	return filepath.Join(s.configDir, SettingsFileName)
}

// DownloadsPath returns the path to the downloads state file.
func (s *Store) DownloadsPath() string {
	return filepath.Join(s.configDir, StateDirName, DownloadsFileName)
}

// Load loads settings from disk.
func (s *Store) Load() (*Settings, error) {
	path := s.SettingsPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default settings if file doesn't exist
			settings := NewSettings()
			settings.ImagesDir = s.defaultImagesDir()
			// Ensure the images directory exists
			if err := os.MkdirAll(settings.ImagesDir, 0755); err != nil {
				// Log but don't fail - the directory will be created on first download
			}
			return settings, nil
		}
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	// Set default images dir if not set or empty
	if settings.ImagesDir == "" {
		settings.ImagesDir = s.defaultImagesDir()
	}

	// Ensure the images directory exists
	if err := os.MkdirAll(settings.ImagesDir, 0755); err != nil {
		// Log but don't fail - the directory will be created on first download
	}

	s.settings = &settings
	return &settings, nil
}

// Save saves settings to disk.
func (s *Store) Save(settings *Settings) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}

	path := s.SettingsPath()

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	s.settings = settings
	return nil
}

// LoadDownloadState loads the download state from disk.
func (s *Store) LoadDownloadState() (*DownloadState, error) {
	path := s.DownloadsPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewDownloadState(), nil
		}
		return nil, fmt.Errorf("failed to read downloads file: %w", err)
	}

	var state DownloadState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse downloads file: %w", err)
	}

	return &state, nil
}

// SaveDownloadState saves the download state to disk.
func (s *Store) SaveDownloadState(state *DownloadState) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}

	path := s.DownloadsPath()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal download state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write downloads file: %w", err)
	}

	return nil
}

// defaultImagesDir returns the default images directory.
// Falls back through multiple options to ensure a valid absolute path is always returned.
func (s *Store) defaultImagesDir() string {
	// Try user's Downloads folder first
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "Downloads")
	}

	// Fall back to XDG config directory
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		return filepath.Join(configDir, ConfigDirName, "images")
	}

	// Fall back to store's config directory if set
	if s.configDir != "" {
		return filepath.Join(s.configDir, "images")
	}

	// Last resort: use /tmp with our app name
	return filepath.Join(os.TempDir(), ConfigDirName, "images")
}

// Settings returns the currently loaded settings.
func (s *Store) Settings() *Settings {
	return s.settings
}
