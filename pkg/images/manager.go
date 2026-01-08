package images

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/globalconfig"
)

// Manager handles image config operations (NO downloads).
type Manager struct {
	config   *globalconfig.Config
	registry *Registry
}

// NewManager creates a new image manager.
func NewManager(cfg *globalconfig.Config) *Manager {
	return &Manager{
		config:   cfg,
		registry: NewRegistry(),
	}
}

// Registry returns the image registry.
func (m *Manager) Registry() *Registry {
	return m.registry
}

// DefaultPathForImage generates the default download path for an image.
// Path structure: {imagesDir}/{os}/{version}/{variant}/{filename}
func (m *Manager) DefaultPathForImage(img *ImageMetadata) string {
	baseDir := m.config.ImagesDir
	if baseDir == "" {
		baseDir = globalconfig.DefaultImagesDir()
	}

	// Normalize OS name to lowercase
	osName := strings.ToLower(img.OS)

	// Normalize variant to lowercase
	variant := strings.ToLower(string(img.Variant))

	// Structure: {imagesDir}/{os}/{version}/{variant}/{filename}
	return filepath.Join(baseDir, osName, img.Version, variant, img.Filename)
}

// CheckImageExists verifies if an image file exists.
func (m *Manager) CheckImageExists(id string) (bool, error) {
	img := m.config.FindCloudImage(id)
	if img == nil {
		return false, fmt.Errorf("image not found in config: %s", id)
	}
	return img.FileExists(), nil
}

// UpdateImageStatuses refreshes download status for all images in config.
func (m *Manager) UpdateImageStatuses() error {
	for i := range m.config.CloudImages {
		m.config.CloudImages[i].UpdateStatus()
	}
	return m.config.Save()
}

// RemoveImage removes an image from config (optionally deletes file).
func (m *Manager) RemoveImage(id string, deleteFile bool) error {
	img := m.config.FindCloudImage(id)
	if img == nil {
		return fmt.Errorf("image not found: %s", id)
	}

	// Optionally delete file
	if deleteFile && img.Path != "" {
		if err := os.Remove(img.Path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	// Remove from config
	if !m.config.RemoveCloudImage(id) {
		return fmt.Errorf("failed to remove from config")
	}

	return m.config.Save()
}

// VerifyImage verifies an image's checksum.
func (m *Manager) VerifyImage(id string) (bool, error) {
	img := m.config.FindCloudImage(id)
	if img == nil {
		return false, fmt.Errorf("image not found: %s", id)
	}

	// If no SHA256 is set, we can't verify
	if img.SHA256 == "" {
		return false, fmt.Errorf("no checksum available for verification")
	}

	// Check file exists
	if !img.FileExists() {
		return false, fmt.Errorf("image file not found: %s", img.Path)
	}

	// Calculate file hash
	hash, err := calculateSHA256(img.Path)
	if err != nil {
		return false, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Compare
	verified := hash == img.SHA256
	if verified != img.Verified {
		img.Verified = verified
		m.config.AddCloudImage(*img) // Update in config
		if err := m.config.Save(); err != nil {
			return verified, fmt.Errorf("failed to save config: %w", err)
		}
	}

	return verified, nil
}

// GetImages returns all registered images from config.
func (m *Manager) GetImages() []globalconfig.CloudImage {
	return m.config.CloudImages
}

// AddImage adds or updates an image in the config.
func (m *Manager) AddImage(img globalconfig.CloudImage) error {
	m.config.AddCloudImage(img)
	return m.config.Save()
}

// FindImage finds an image by ID.
func (m *Manager) FindImage(id string) *globalconfig.CloudImage {
	return m.config.FindCloudImage(id)
}

// calculateSHA256 calculates the SHA256 hash of a file.
func calculateSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
