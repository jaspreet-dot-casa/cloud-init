// Package settings provides persistent user settings storage.
package settings

import (
	"time"
)

// Version is the current settings schema version.
const Version = "1.0"

// Settings represents the user's persistent settings.
type Settings struct {
	Version     string       `json:"version"`
	ImagesDir   string       `json:"images_dir"`   // Default: ~/Downloads
	CloudImages []CloudImage `json:"cloud_images"` // Registered cloud images
	ISOs        []ISO        `json:"isos"`         // Registered ISOs
	Preferences Preferences  `json:"preferences"`  // User preferences
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

// ISO represents a registered ISO file.
type ISO struct {
	ID        string    `json:"id"`                   // Unique identifier
	Name      string    `json:"name"`                 // Display name
	Version   string    `json:"version,omitempty"`    // Ubuntu version if known
	Path      string    `json:"path"`                 // Local file path
	SourceURL string    `json:"source_url,omitempty"` // Original download URL
	AddedAt   time.Time `json:"added_at"`             // When added
}

// Preferences represents user preferences.
type Preferences struct {
	DefaultCloudImage string `json:"default_cloud_image,omitempty"` // Default cloud image ID
	DefaultISO        string `json:"default_iso,omitempty"`         // Default ISO ID
	AutoVerify        bool   `json:"auto_verify"`                   // Auto-verify checksums
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
		Version:     Version,
		ImagesDir:   "", // Will be set to ~/Downloads on first load
		CloudImages: []CloudImage{},
		ISOs:        []ISO{},
		Preferences: Preferences{
			AutoVerify: true,
		},
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

// FindISO finds an ISO by ID.
func (s *Settings) FindISO(id string) *ISO {
	for i := range s.ISOs {
		if s.ISOs[i].ID == id {
			return &s.ISOs[i]
		}
	}
	return nil
}

// AddCloudImage adds a cloud image to the settings.
func (s *Settings) AddCloudImage(img CloudImage) {
	// Remove existing with same ID
	for i := range s.CloudImages {
		if s.CloudImages[i].ID == img.ID {
			s.CloudImages = append(s.CloudImages[:i], s.CloudImages[i+1:]...)
			break
		}
	}
	s.CloudImages = append(s.CloudImages, img)
}

// RemoveCloudImage removes a cloud image by ID.
func (s *Settings) RemoveCloudImage(id string) bool {
	for i := range s.CloudImages {
		if s.CloudImages[i].ID == id {
			s.CloudImages = append(s.CloudImages[:i], s.CloudImages[i+1:]...)
			return true
		}
	}
	return false
}

// AddISO adds an ISO to the settings.
func (s *Settings) AddISO(iso ISO) {
	// Remove existing with same ID
	for i := range s.ISOs {
		if s.ISOs[i].ID == iso.ID {
			s.ISOs = append(s.ISOs[:i], s.ISOs[i+1:]...)
			break
		}
	}
	s.ISOs = append(s.ISOs, iso)
}

// RemoveISO removes an ISO by ID.
func (s *Settings) RemoveISO(id string) bool {
	for i := range s.ISOs {
		if s.ISOs[i].ID == id {
			s.ISOs = append(s.ISOs[:i], s.ISOs[i+1:]...)
			return true
		}
	}
	return false
}
