// Package settings provides persistent user settings storage.
package settings

import (
	"time"
)

// Version is the current settings schema version.
const Version = "2.0"

// Settings represents the user's persistent settings.
type Settings struct {
	Version        string          `json:"version"`
	CloudImages    []CloudImage    `json:"cloud_images"`              // Registered cloud images (for Terraform)
	VMConfigs      []VMConfig      `json:"vm_configs,omitempty"`      // Saved VM configurations
	PackagePresets []PackagePreset `json:"package_presets,omitempty"` // Package preset groups
	AppSettings    AppSettings     `json:"app_settings"`              // Application settings
}

// CloudImage represents a registered cloud image.
type CloudImage struct {
	ID       string    `json:"id"`                 // e.g., "ubuntu-24.04-amd64"
	Name     string    `json:"name"`               // Display name
	Version  string    `json:"version"`            // e.g., "24.04"
	Arch     string    `json:"arch"`               // e.g., "amd64"
	Path     string    `json:"path"`               // Local file path
	URL      string    `json:"url,omitempty"`      // Source URL
	SHA256   string    `json:"sha256,omitempty"`   // Expected checksum
	Size     int64     `json:"size"`               // File size in bytes
	AddedAt  time.Time `json:"added_at"`           // When added
	Verified bool      `json:"verified,omitempty"` // Checksum verified
}

// VMConfig represents a saved VM configuration.
type VMConfig struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Target      string             `json:"target"` // "terraform", "multipass"
	CreatedAt   time.Time          `json:"created_at"`
	LastUsedAt  time.Time          `json:"last_used_at,omitempty"`
	Data        WizardDataSnapshot `json:"data"`
}

// WizardDataSnapshot is a serializable subset of wizard data.
type WizardDataSnapshot struct {
	Username    string   `json:"username"`
	Hostname    string   `json:"hostname"`
	DisplayName string   `json:"display_name,omitempty"`
	GitName     string   `json:"git_name,omitempty"`
	GitEmail    string   `json:"git_email,omitempty"`
	GitHubUser  string   `json:"github_user,omitempty"`
	SSHKeys     []string `json:"ssh_keys,omitempty"`
	Packages    []string `json:"packages"`
	// Target-specific options
	TerragruntOpts *TerragruntOptsSnapshot `json:"terragrunt_opts,omitempty"`
	MultipassOpts  *MultipassOptsSnapshot  `json:"multipass_opts,omitempty"`
}

// TerragruntOptsSnapshot captures Terragrunt-specific options.
type TerragruntOptsSnapshot struct {
	CPUs        int    `json:"cpus"`
	MemoryMB    int    `json:"memory_mb"`
	DiskGB      int    `json:"disk_gb"`
	Autostart   bool   `json:"autostart"`
	UbuntuImage string `json:"ubuntu_image,omitempty"`
}

// MultipassOptsSnapshot captures Multipass-specific options.
type MultipassOptsSnapshot struct {
	CPUs          int    `json:"cpus"`
	MemoryMB      int    `json:"memory_mb"`
	DiskGB        int    `json:"disk_gb"`
	UbuntuVersion string `json:"ubuntu_version"`
}

// PackagePreset represents a named group of packages.
type PackagePreset struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Packages    []string  `json:"packages"`
	CreatedAt   time.Time `json:"created_at"`
	IsBuiltIn   bool      `json:"is_built_in,omitempty"` // True for default presets
}

// AppSettings represents ucli application preferences.
type AppSettings struct {
	TerraformDir  string `json:"terraform_dir,omitempty"`
	DefaultTarget string `json:"default_target,omitempty"` // "terraform" or "multipass"
	AutoApprove   bool   `json:"auto_approve"`
}

// DownloadState represents active downloads state.
type DownloadState struct {
	ActiveDownloads []Download `json:"active"`
}

// DownloadStatus represents the status of a download.
type DownloadStatus string

const (
	StatusDownloading DownloadStatus = "downloading"
	StatusPaused      DownloadStatus = "paused"
	StatusComplete    DownloadStatus = "complete"
	StatusError       DownloadStatus = "error"
)

// Download represents an active or completed download.
type Download struct {
	ID         string         `json:"id"`
	URL        string         `json:"url"`
	DestPath   string         `json:"dest_path"`
	TotalBytes int64          `json:"total_bytes"`
	Downloaded int64          `json:"downloaded"`
	StartedAt  time.Time      `json:"started_at"`
	Status     DownloadStatus `json:"status"`
	Error      string         `json:"error,omitempty"`
}

// NewSettings creates a new Settings with defaults.
func NewSettings() *Settings {
	return &Settings{
		Version:        Version,
		CloudImages:    []CloudImage{},
		VMConfigs:      []VMConfig{},
		PackagePresets: []PackagePreset{},
		AppSettings:    AppSettings{},
	}
}

// NewDownloadState creates a new empty download state.
func NewDownloadState() *DownloadState {
	return &DownloadState{
		ActiveDownloads: []Download{},
	}
}

// FindCloudImage finds a cloud image by ID.
func (s *Settings) FindCloudImage(id string) *CloudImage {
	for i := range s.CloudImages {
		if s.CloudImages[i].ID == id {
			return &s.CloudImages[i]
		}
	}
	return nil
}

// AddCloudImage adds a cloud image to the settings.
// If an image with the same ID exists, it is replaced.
func (s *Settings) AddCloudImage(img CloudImage) {
	idx := -1
	for i := range s.CloudImages {
		if s.CloudImages[i].ID == img.ID {
			idx = i
			break
		}
	}
	if idx != -1 {
		s.CloudImages = append(s.CloudImages[:idx], s.CloudImages[idx+1:]...)
	}
	s.CloudImages = append(s.CloudImages, img)
}

// RemoveCloudImage removes a cloud image by ID.
func (s *Settings) RemoveCloudImage(id string) bool {
	idx := -1
	for i := range s.CloudImages {
		if s.CloudImages[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}
	s.CloudImages = append(s.CloudImages[:idx], s.CloudImages[idx+1:]...)
	return true
}

// FindVMConfig finds a VM config by ID.
func (s *Settings) FindVMConfig(id string) *VMConfig {
	for i := range s.VMConfigs {
		if s.VMConfigs[i].ID == id {
			return &s.VMConfigs[i]
		}
	}
	return nil
}

// AddVMConfig adds a VM config to the settings.
// If a config with the same ID exists, it is replaced.
func (s *Settings) AddVMConfig(cfg VMConfig) {
	idx := -1
	for i := range s.VMConfigs {
		if s.VMConfigs[i].ID == cfg.ID {
			idx = i
			break
		}
	}
	if idx != -1 {
		s.VMConfigs = append(s.VMConfigs[:idx], s.VMConfigs[idx+1:]...)
	}
	s.VMConfigs = append(s.VMConfigs, cfg)
}

// RemoveVMConfig removes a VM config by ID.
func (s *Settings) RemoveVMConfig(id string) bool {
	idx := -1
	for i := range s.VMConfigs {
		if s.VMConfigs[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}
	s.VMConfigs = append(s.VMConfigs[:idx], s.VMConfigs[idx+1:]...)
	return true
}

// UpdateVMConfigLastUsed updates the LastUsedAt field of a VM config.
func (s *Settings) UpdateVMConfigLastUsed(id string) bool {
	for i := range s.VMConfigs {
		if s.VMConfigs[i].ID == id {
			s.VMConfigs[i].LastUsedAt = time.Now()
			return true
		}
	}
	return false
}

// FindPackagePreset finds a package preset by ID.
func (s *Settings) FindPackagePreset(id string) *PackagePreset {
	for i := range s.PackagePresets {
		if s.PackagePresets[i].ID == id {
			return &s.PackagePresets[i]
		}
	}
	return nil
}

// AddPackagePreset adds a package preset to the settings.
// If a preset with the same ID exists, it is replaced.
func (s *Settings) AddPackagePreset(preset PackagePreset) {
	idx := -1
	for i := range s.PackagePresets {
		if s.PackagePresets[i].ID == preset.ID {
			idx = i
			break
		}
	}
	if idx != -1 {
		s.PackagePresets = append(s.PackagePresets[:idx], s.PackagePresets[idx+1:]...)
	}
	s.PackagePresets = append(s.PackagePresets, preset)
}

// RemovePackagePreset removes a package preset by ID.
func (s *Settings) RemovePackagePreset(id string) bool {
	idx := -1
	for i := range s.PackagePresets {
		if s.PackagePresets[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}
	s.PackagePresets = append(s.PackagePresets[:idx], s.PackagePresets[idx+1:]...)
	return true
}

// Clone returns a deep copy of the settings.
func (s *Settings) Clone() *Settings {
	if s == nil {
		return nil
	}

	clone := &Settings{
		Version:     s.Version,
		AppSettings: s.AppSettings,
	}

	// Deep copy slices
	if s.CloudImages != nil {
		clone.CloudImages = make([]CloudImage, len(s.CloudImages))
		copy(clone.CloudImages, s.CloudImages)
	}

	if s.VMConfigs != nil {
		clone.VMConfigs = make([]VMConfig, len(s.VMConfigs))
		for i, cfg := range s.VMConfigs {
			clone.VMConfigs[i] = cfg.Clone()
		}
	}

	if s.PackagePresets != nil {
		clone.PackagePresets = make([]PackagePreset, len(s.PackagePresets))
		for i, p := range s.PackagePresets {
			clone.PackagePresets[i] = p.Clone()
		}
	}

	return clone
}

// Clone returns a deep copy of the VM config.
func (c VMConfig) Clone() VMConfig {
	clone := c
	// Deep copy slices in Data
	if c.Data.SSHKeys != nil {
		clone.Data.SSHKeys = make([]string, len(c.Data.SSHKeys))
		copy(clone.Data.SSHKeys, c.Data.SSHKeys)
	}
	if c.Data.Packages != nil {
		clone.Data.Packages = make([]string, len(c.Data.Packages))
		copy(clone.Data.Packages, c.Data.Packages)
	}
	// Deep copy pointers
	if c.Data.TerragruntOpts != nil {
		opts := *c.Data.TerragruntOpts
		clone.Data.TerragruntOpts = &opts
	}
	if c.Data.MultipassOpts != nil {
		opts := *c.Data.MultipassOpts
		clone.Data.MultipassOpts = &opts
	}
	return clone
}

// Clone returns a deep copy of the package preset.
func (p PackagePreset) Clone() PackagePreset {
	clone := p
	if p.Packages != nil {
		clone.Packages = make([]string, len(p.Packages))
		copy(clone.Packages, p.Packages)
	}
	return clone
}

// IsValidTarget checks if a target string is valid.
func IsValidTarget(target string) bool {
	switch target {
	case "terragrunt", "multipass", "config":
		return true
	}
	return false
}

// DefaultPackagePresets returns a set of built-in package presets.
// Built-in presets use zero time for CreatedAt to avoid timestamp drift.
func DefaultPackagePresets() []PackagePreset {
	return []PackagePreset{
		{
			ID:          "minimal",
			Name:        "Minimal",
			Description: "Essential tools only",
			Packages:    []string{"git", "vim", "curl"},
			IsBuiltIn:   true,
		},
		{
			ID:          "dev-tools",
			Name:        "Dev Tools",
			Description: "Common development tools",
			Packages:    []string{"git", "lazygit", "neovim", "bat", "fd", "ripgrep", "fzf", "zoxide", "delta"},
			IsBuiltIn:   true,
		},
		{
			ID:          "full-stack",
			Name:        "Full Stack",
			Description: "Complete development environment",
			Packages:    []string{"git", "lazygit", "neovim", "bat", "fd", "ripgrep", "fzf", "zoxide", "delta", "starship", "docker", "mise"},
			IsBuiltIn:   true,
		},
	}
}
