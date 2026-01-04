// Package globalconfig provides global configuration management for ucli.
// Configuration is stored at ~/.config/ucli/config.yaml and includes
// the project path and all user settings.
package globalconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Version is the current config schema version.
const Version = "1.0"

var (
	// ErrNotInitialized is returned when config doesn't exist or has no project path.
	ErrNotInitialized = errors.New("ucli not initialized: run 'ucli init <path>' first")
	// ErrProjectNotFound is returned when the configured project path doesn't exist.
	ErrProjectNotFound = errors.New("configured project path does not exist")
)

// Config represents the global ucli configuration.
type Config struct {
	Version     string       `yaml:"version"`
	ProjectPath string       `yaml:"project_path"` // Set by `ucli init`
	ImagesDir   string       `yaml:"images_dir"`   // Default: ~/Downloads
	CloudImages []CloudImage `yaml:"cloud_images"` // Registered cloud images
	ISOs        []ISO        `yaml:"isos"`         // Registered ISOs
	Preferences Preferences  `yaml:"preferences"`  // User preferences
}

// CloudImage represents a registered cloud image.
type CloudImage struct {
	ID       string    `yaml:"id"`                 // e.g., "ubuntu-24.04-amd64"
	Name     string    `yaml:"name"`               // Display name
	Version  string    `yaml:"version"`            // e.g., "24.04"
	Arch     string    `yaml:"arch"`               // e.g., "amd64"
	Path     string    `yaml:"path"`               // Local file path
	URL      string    `yaml:"url,omitempty"`      // Source URL
	SHA256   string    `yaml:"sha256,omitempty"`   // Expected checksum
	Size     int64     `yaml:"size"`               // File size in bytes
	AddedAt  time.Time `yaml:"added_at"`           // When added
	Verified bool      `yaml:"verified,omitempty"` // Checksum verified
}

// ISO represents a registered ISO file.
type ISO struct {
	ID        string    `yaml:"id"`                   // Unique identifier
	Name      string    `yaml:"name"`                 // Display name
	Version   string    `yaml:"version,omitempty"`    // Ubuntu version if known
	Path      string    `yaml:"path"`                 // Local file path
	SourceURL string    `yaml:"source_url,omitempty"` // Original download URL
	AddedAt   time.Time `yaml:"added_at"`             // When added
}

// Preferences represents user preferences.
type Preferences struct {
	DefaultCloudImage string `yaml:"default_cloud_image,omitempty"` // Default cloud image ID
	DefaultISO        string `yaml:"default_iso,omitempty"`         // Default ISO ID
	AutoVerify        bool   `yaml:"auto_verify"`                   // Auto-verify checksums
}

// NewConfig creates a new Config with defaults.
func NewConfig() *Config {
	return &Config{
		Version:     Version,
		ProjectPath: "",
		ImagesDir:   DefaultImagesDir(),
		CloudImages: []CloudImage{},
		ISOs:        []ISO{},
		Preferences: Preferences{
			AutoVerify: true,
		},
	}
}

// Load loads the config from ~/.config/ucli/config.yaml.
// Returns ErrNotInitialized if config doesn't exist.
// Automatically migrates from legacy settings.json if needed.
func Load() (*Config, error) {
	// Check for and perform migration from legacy settings.json
	migrated, err := MigrateFromLegacy()
	if err != nil {
		return nil, fmt.Errorf("failed to migrate legacy settings: %w", err)
	}
	if migrated {
		fmt.Println("Migrated settings from settings.json to config.yaml")
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotInitialized
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate project path is set
	if cfg.ProjectPath == "" {
		return nil, ErrNotInitialized
	}

	// Set default images dir if empty
	if cfg.ImagesDir == "" {
		cfg.ImagesDir = DefaultImagesDir()
	}

	return &cfg, nil
}

// LoadOrCreate loads the config if it exists, or creates a new one.
// Unlike Load(), this doesn't require the config to be initialized.
func LoadOrCreate() (*Config, error) {
	cfg, err := Load()
	if err != nil {
		if errors.Is(err, ErrNotInitialized) {
			return NewConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}

// Save saves the config to ~/.config/ucli/config.yaml.
func (c *Config) Save() error {
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ProjectDir returns the project path and validates it exists.
func (c *Config) ProjectDir() (string, error) {
	if c.ProjectPath == "" {
		return "", ErrNotInitialized
	}

	info, err := os.Stat(c.ProjectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrProjectNotFound, c.ProjectPath)
		}
		return "", fmt.Errorf("failed to access project path: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", c.ProjectPath)
	}

	return c.ProjectPath, nil
}

// TFDir returns the terraform directory (project_path/tf).
func (c *Config) TFDir() (string, error) {
	projectDir, err := c.ProjectDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectDir, "tf"), nil
}

// TerraformTemplateDir returns the terraform template directory (project_path/terraform).
func (c *Config) TerraformTemplateDir() (string, error) {
	projectDir, err := c.ProjectDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectDir, "terraform"), nil
}

// IsInitialized checks if the config exists and has a valid project path.
func IsInitialized() bool {
	cfg, err := Load()
	if err != nil {
		return false
	}
	_, err = cfg.ProjectDir()
	return err == nil
}

// SetProjectPath sets and validates the project path.
func (c *Config) SetProjectPath(path string) error {
	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Validate path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", absPath)
		}
		return fmt.Errorf("failed to access path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", absPath)
	}

	c.ProjectPath = absPath
	return nil
}

// FindCloudImage finds a cloud image by ID.
func (c *Config) FindCloudImage(id string) *CloudImage {
	for i := range c.CloudImages {
		if c.CloudImages[i].ID == id {
			return &c.CloudImages[i]
		}
	}
	return nil
}

// FindISO finds an ISO by ID.
func (c *Config) FindISO(id string) *ISO {
	for i := range c.ISOs {
		if c.ISOs[i].ID == id {
			return &c.ISOs[i]
		}
	}
	return nil
}

// AddCloudImage adds a cloud image to the config.
// If an image with the same ID exists, it is replaced.
func (c *Config) AddCloudImage(img CloudImage) {
	idx := -1
	for i := range c.CloudImages {
		if c.CloudImages[i].ID == img.ID {
			idx = i
			break
		}
	}
	if idx != -1 {
		c.CloudImages = append(c.CloudImages[:idx], c.CloudImages[idx+1:]...)
	}
	c.CloudImages = append(c.CloudImages, img)
}

// RemoveCloudImage removes a cloud image by ID.
func (c *Config) RemoveCloudImage(id string) bool {
	idx := -1
	for i := range c.CloudImages {
		if c.CloudImages[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}
	c.CloudImages = append(c.CloudImages[:idx], c.CloudImages[idx+1:]...)
	return true
}

// AddISO adds an ISO to the config.
// If an ISO with the same ID exists, it is replaced.
func (c *Config) AddISO(iso ISO) {
	idx := -1
	for i := range c.ISOs {
		if c.ISOs[i].ID == iso.ID {
			idx = i
			break
		}
	}
	if idx != -1 {
		c.ISOs = append(c.ISOs[:idx], c.ISOs[idx+1:]...)
	}
	c.ISOs = append(c.ISOs, iso)
}

// RemoveISO removes an ISO by ID.
func (c *Config) RemoveISO(id string) bool {
	idx := -1
	for i := range c.ISOs {
		if c.ISOs[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}
	c.ISOs = append(c.ISOs[:idx], c.ISOs[idx+1:]...)
	return true
}
