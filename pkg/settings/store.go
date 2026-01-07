package settings

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	// MaxVMConfigs is the maximum number of saved VM configurations.
	MaxVMConfigs = 100
	// MaxPackagePresets is the maximum number of package presets.
	MaxPackagePresets = 50
)

// Store manages persistent settings storage.
type Store struct {
	configDir string
	settings  *Settings
	mu        sync.RWMutex
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadInternal()
}

// loadInternal loads settings without locking (caller must hold lock).
func (s *Store) loadInternal() (*Settings, error) {
	path := s.SettingsPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default settings if file doesn't exist
			settings := NewSettings()
			return settings, nil
		}
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	// Migrate if needed
	if err := s.migrate(&settings); err != nil {
		return nil, fmt.Errorf("failed to migrate settings: %w", err)
	}

	s.settings = &settings
	return &settings, nil
}

// migrate handles settings migration between versions.
func (s *Store) migrate(settings *Settings) error {
	if settings.Version == Version {
		return nil
	}

	currentMajor, currentMinor := parseVersion(Version)
	fileMajor, fileMinor := parseVersion(settings.Version)

	// Warn if file is from a newer version (forward compatibility)
	if fileMajor > currentMajor || (fileMajor == currentMajor && fileMinor > currentMinor) {
		log.Printf("Warning: settings file version %s is newer than supported version %s",
			settings.Version, Version)
		// Continue anyway - we'll do our best
	}

	// Handle migration from older versions
	if fileMajor < 2 {
		// Migration from 1.x to 2.0
		// - ISOs field removed (ignored during unmarshal)
		// - Preferences field removed (ignored during unmarshal)
		// - New fields have zero values which are fine
		settings.Version = Version
	}

	// Ensure required fields are initialized
	if settings.CloudImages == nil {
		settings.CloudImages = []CloudImage{}
	}
	if settings.VMConfigs == nil {
		settings.VMConfigs = []VMConfig{}
	}
	if settings.PackagePresets == nil {
		settings.PackagePresets = []PackagePreset{}
	}

	return nil
}

// parseVersion extracts major and minor version numbers.
// Returns (0, 0) for invalid versions.
func parseVersion(v string) (major, minor int) {
	if v == "" {
		return 0, 0
	}
	parts := strings.Split(v, ".")
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor
}

// Save saves settings to disk atomically.
func (s *Store) Save(settings *Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveInternal(settings)
}

// saveInternal saves settings without locking (caller must hold lock).
func (s *Store) saveInternal(settings *Settings) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}

	// Enforce limits using LRU eviction (keep most recently used)
	s.enforceLimits(settings)

	path := s.SettingsPath()
	tmpPath := path + ".tmp"

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write to temp file first
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		// Try harder to clean up temp file
		if removeErr := os.Remove(tmpPath); removeErr != nil {
			log.Printf("Warning: failed to clean up temp file %s: %v", tmpPath, removeErr)
		}
		return fmt.Errorf("failed to save settings file: %w", err)
	}

	s.settings = settings
	return nil
}

// enforceLimits applies LRU eviction to keep collections within limits.
func (s *Store) enforceLimits(settings *Settings) {
	// Enforce VM configs limit using LRU (sort by LastUsedAt, keep most recent)
	if len(settings.VMConfigs) > MaxVMConfigs {
		// Sort by LastUsedAt descending (most recent first)
		sort.Slice(settings.VMConfigs, func(i, j int) bool {
			ti := settings.VMConfigs[i].LastUsedAt
			tj := settings.VMConfigs[j].LastUsedAt
			// If LastUsedAt is zero, fall back to CreatedAt
			if ti.IsZero() {
				ti = settings.VMConfigs[i].CreatedAt
			}
			if tj.IsZero() {
				tj = settings.VMConfigs[j].CreatedAt
			}
			return ti.After(tj)
		})
		// Keep only the most recent MaxVMConfigs
		settings.VMConfigs = settings.VMConfigs[:MaxVMConfigs]
	}

	// Enforce presets limit - keep user presets, evict oldest by CreatedAt
	// Built-in presets don't count toward limit and are never evicted
	if len(settings.PackagePresets) > MaxPackagePresets {
		var userPresets []PackagePreset
		var builtInPresets []PackagePreset
		for _, p := range settings.PackagePresets {
			if p.IsBuiltIn {
				builtInPresets = append(builtInPresets, p)
			} else {
				userPresets = append(userPresets, p)
			}
		}

		maxUserPresets := MaxPackagePresets - len(builtInPresets)
		if maxUserPresets < 0 {
			maxUserPresets = 0
		}

		if len(userPresets) > maxUserPresets {
			// Sort by CreatedAt descending (most recent first)
			sort.Slice(userPresets, func(i, j int) bool {
				return userPresets[i].CreatedAt.After(userPresets[j].CreatedAt)
			})
			userPresets = userPresets[:maxUserPresets]
		}

		// Combine: built-in first, then user presets
		settings.PackagePresets = append(builtInPresets, userPresets...)
	}
}

// LoadAndSave atomically loads, modifies, and saves settings.
// This prevents race conditions when multiple operations need to
// read-modify-write the settings file.
func (s *Store) LoadAndSave(modify func(*Settings) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.loadInternal()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	if err := modify(settings); err != nil {
		return err
	}

	return s.saveInternal(settings)
}

// LoadDownloadState loads the download state from disk.
func (s *Store) LoadDownloadState() (*DownloadState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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

// SaveDownloadState saves the download state to disk atomically.
func (s *Store) SaveDownloadState(state *DownloadState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureDir(); err != nil {
		return err
	}

	path := s.DownloadsPath()
	tmpPath := path + ".tmp"

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal download state: %w", err)
	}

	// Write to temp file first
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write downloads file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		// Try harder to clean up temp file
		if removeErr := os.Remove(tmpPath); removeErr != nil {
			log.Printf("Warning: failed to clean up temp file %s: %v", tmpPath, removeErr)
		}
		return fmt.Errorf("failed to save downloads file: %w", err)
	}

	return nil
}

// Settings returns a copy of the currently loaded settings.
// Returns nil if no settings have been loaded yet.
func (s *Store) Settings() *Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.settings == nil {
		return nil
	}
	// Return a deep copy to prevent external mutation
	return s.settings.Clone()
}
