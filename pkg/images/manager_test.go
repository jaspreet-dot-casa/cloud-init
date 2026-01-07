package images

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)

	manager := NewManager(store)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.Registry())
}

func TestManager_AddExistingImage(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	// Create a test image file
	imgPath := filepath.Join(tmpDir, "test-image.img")
	err := os.WriteFile(imgPath, []byte("test image content"), 0644)
	require.NoError(t, err)

	// Add the image
	img, err := manager.AddExistingImage(imgPath, "24.04", "amd64")
	require.NoError(t, err)

	assert.Equal(t, "ubuntu-24.04-amd64", img.ID)
	assert.Equal(t, "24.04", img.Version)
	assert.Equal(t, "amd64", img.Arch)
	assert.Contains(t, img.Path, "test-image.img")
	assert.Equal(t, int64(18), img.Size) // "test image content" is 18 bytes
	assert.False(t, img.Verified)

	// Verify it was saved
	images, err := manager.GetImages()
	require.NoError(t, err)
	assert.Len(t, images, 1)
	assert.Equal(t, img.ID, images[0].ID)
}

func TestManager_AddExistingImage_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	_, err := manager.AddExistingImage("/nonexistent/path.img", "24.04", "amd64")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestManager_RemoveImage(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	// Create and add a test image
	imgPath := filepath.Join(tmpDir, "test-image.img")
	err := os.WriteFile(imgPath, []byte("test"), 0644)
	require.NoError(t, err)

	img, err := manager.AddExistingImage(imgPath, "24.04", "amd64")
	require.NoError(t, err)

	// Remove without deleting file
	err = manager.RemoveImage(img.ID, false)
	require.NoError(t, err)

	// Verify removed from registry
	images, err := manager.GetImages()
	require.NoError(t, err)
	assert.Len(t, images, 0)

	// Verify file still exists
	_, err = os.Stat(imgPath)
	assert.NoError(t, err)
}

func TestManager_RemoveImage_WithDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	// Create and add a test image
	imgPath := filepath.Join(tmpDir, "test-image.img")
	err := os.WriteFile(imgPath, []byte("test"), 0644)
	require.NoError(t, err)

	img, err := manager.AddExistingImage(imgPath, "24.04", "amd64")
	require.NoError(t, err)

	// Remove with file deletion
	err = manager.RemoveImage(img.ID, true)
	require.NoError(t, err)

	// Verify file is deleted
	_, err = os.Stat(imgPath)
	assert.True(t, os.IsNotExist(err))
}

func TestManager_RemoveImage_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	err := manager.RemoveImage("nonexistent-id", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image not found")
}

func TestManager_VerifyImage(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	// Create a test image with known content
	content := []byte("test image content for verification")
	imgPath := filepath.Join(tmpDir, "test-image.img")
	err := os.WriteFile(imgPath, content, 0644)
	require.NoError(t, err)

	// Calculate expected SHA256
	h := sha256.New()
	h.Write(content)
	expectedHash := fmt.Sprintf("%x", h.Sum(nil))

	// Add image and set SHA256
	img, err := manager.AddExistingImage(imgPath, "24.04", "amd64")
	require.NoError(t, err)

	// Update with SHA256
	s, err := store.Load()
	require.NoError(t, err)
	storedImg := s.FindCloudImage(img.ID)
	storedImg.SHA256 = expectedHash
	s.AddCloudImage(*storedImg)
	err = store.Save(s)
	require.NoError(t, err)

	// Verify should pass
	verified, err := manager.VerifyImage(img.ID)
	require.NoError(t, err)
	assert.True(t, verified)
}

func TestManager_VerifyImage_NoChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	// Create and add image without SHA256
	imgPath := filepath.Join(tmpDir, "test-image.img")
	err := os.WriteFile(imgPath, []byte("test"), 0644)
	require.NoError(t, err)

	img, err := manager.AddExistingImage(imgPath, "24.04", "amd64")
	require.NoError(t, err)

	// Verify should fail due to no checksum
	_, err = manager.VerifyImage(img.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no checksum available")
}

func TestManager_VerifyImage_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	_, err := manager.VerifyImage("nonexistent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image not found")
}

func TestManager_VerifyImage_ChecksumMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	// Create image
	imgPath := filepath.Join(tmpDir, "test-image.img")
	err := os.WriteFile(imgPath, []byte("test content"), 0644)
	require.NoError(t, err)

	img, err := manager.AddExistingImage(imgPath, "24.04", "amd64")
	require.NoError(t, err)

	// Set wrong SHA256
	s, err := store.Load()
	require.NoError(t, err)
	storedImg := s.FindCloudImage(img.ID)
	storedImg.SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	s.AddCloudImage(*storedImg)
	err = store.Save(s)
	require.NoError(t, err)

	// Verify should return false
	verified, err := manager.VerifyImage(img.ID)
	require.NoError(t, err)
	assert.False(t, verified)
}

func TestManager_GetImages_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	images, err := manager.GetImages()
	require.NoError(t, err)
	assert.Empty(t, images)
}

func TestManager_AddExistingISO(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	// Create a test ISO file
	isoPath := filepath.Join(tmpDir, "test.iso")
	err := os.WriteFile(isoPath, []byte("ISO content"), 0644)
	require.NoError(t, err)

	// Add the ISO
	iso, err := manager.AddExistingISO(isoPath, "Ubuntu 24.04 Server", "24.04")
	require.NoError(t, err)

	assert.Contains(t, iso.ID, "iso-24.04")
	assert.Equal(t, "Ubuntu 24.04 Server", iso.Name)
	assert.Equal(t, "24.04", iso.Version)
	assert.Contains(t, iso.Path, "test.iso")

	// Verify it was saved
	isos, err := manager.GetISOs()
	require.NoError(t, err)
	assert.Len(t, isos, 1)
}

func TestManager_AddExistingISO_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	_, err := manager.AddExistingISO("/nonexistent/path.iso", "Test", "24.04")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestManager_RemoveISO(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	// Create and add a test ISO
	isoPath := filepath.Join(tmpDir, "test.iso")
	err := os.WriteFile(isoPath, []byte("ISO"), 0644)
	require.NoError(t, err)

	iso, err := manager.AddExistingISO(isoPath, "Test ISO", "24.04")
	require.NoError(t, err)

	// Remove without deleting file
	err = manager.RemoveISO(iso.ID, false)
	require.NoError(t, err)

	// Verify removed from registry
	isos, err := manager.GetISOs()
	require.NoError(t, err)
	assert.Len(t, isos, 0)

	// Verify file still exists
	_, err = os.Stat(isoPath)
	assert.NoError(t, err)
}

func TestManager_RemoveISO_WithDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	// Create and add a test ISO
	isoPath := filepath.Join(tmpDir, "test.iso")
	err := os.WriteFile(isoPath, []byte("ISO"), 0644)
	require.NoError(t, err)

	iso, err := manager.AddExistingISO(isoPath, "Test ISO", "24.04")
	require.NoError(t, err)

	// Remove with file deletion
	err = manager.RemoveISO(iso.ID, true)
	require.NoError(t, err)

	// Verify file is deleted
	_, err = os.Stat(isoPath)
	assert.True(t, os.IsNotExist(err))
}

func TestManager_RemoveISO_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	err := manager.RemoveISO("nonexistent-id", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ISO not found")
}

func TestManager_GetISOs_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	manager := NewManager(store)

	isos, err := manager.GetISOs()
	require.NoError(t, err)
	assert.Empty(t, isos)
}

func TestCalculateSHA256(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with known content
	content := []byte("hello world")
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, content, 0644)
	require.NoError(t, err)

	// Known SHA256 of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	hash, err := calculateSHA256(path)
	require.NoError(t, err)
	assert.Equal(t, expected, hash)
}

func TestCalculateSHA256_FileNotFound(t *testing.T) {
	_, err := calculateSHA256("/nonexistent/path")
	assert.Error(t, err)
}
