package images

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
)

// Manager handles image operations.
type Manager struct {
	store    *settings.Store
	registry *Registry
}

// NewManager creates a new image manager.
func NewManager(store *settings.Store) *Manager {
	return &Manager{
		store:    store,
		registry: NewRegistry(),
	}
}

// Registry returns the image registry.
func (m *Manager) Registry() *Registry {
	return m.registry
}

// AddExistingImage adds an existing image file to the registry.
func (m *Manager) AddExistingImage(path string, version, arch string) (*settings.CloudImage, error) {
	// Verify file exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Generate ID
	id := GenerateImageID(version, arch)

	// Get release info
	rel := m.registry.FindRelease(version)
	name := fmt.Sprintf("Ubuntu %s (%s)", version, arch)
	if rel != nil {
		name = fmt.Sprintf("%s (%s)", rel.Name, arch)
	}

	// Create cloud image entry
	img := &settings.CloudImage{
		ID:       id,
		Name:     name,
		Version:  version,
		Arch:     arch,
		Path:     absPath,
		Size:     info.Size(),
		AddedAt:  time.Now(),
		Verified: false,
	}

	// Load current settings
	s, err := m.store.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	// Add image
	s.AddCloudImage(*img)

	// Save settings
	if err := m.store.Save(s); err != nil {
		return nil, fmt.Errorf("failed to save settings: %w", err)
	}

	return img, nil
}

// RemoveImage removes an image from the registry.
func (m *Manager) RemoveImage(id string, deleteFile bool) error {
	// Load current settings
	s, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Find image
	img := s.FindCloudImage(id)
	if img == nil {
		return fmt.Errorf("image not found: %s", id)
	}

	// Optionally delete file
	if deleteFile && img.Path != "" {
		if err := os.Remove(img.Path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	// Remove from settings
	s.RemoveCloudImage(id)

	// Save settings
	if err := m.store.Save(s); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}

// VerifyImage verifies an image's checksum.
func (m *Manager) VerifyImage(id string) (bool, error) {
	// Load current settings
	s, err := m.store.Load()
	if err != nil {
		return false, fmt.Errorf("failed to load settings: %w", err)
	}

	// Find image
	img := s.FindCloudImage(id)
	if img == nil {
		return false, fmt.Errorf("image not found: %s", id)
	}

	// If no SHA256 is set, we can't verify
	if img.SHA256 == "" {
		return false, fmt.Errorf("no checksum available for verification")
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
		s.AddCloudImage(*img) // Update
		if err := m.store.Save(s); err != nil {
			return verified, fmt.Errorf("failed to save settings: %w", err)
		}
	}

	return verified, nil
}

// GetImages returns all registered images.
func (m *Manager) GetImages() ([]settings.CloudImage, error) {
	s, err := m.store.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}
	return s.CloudImages, nil
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
