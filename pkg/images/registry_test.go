package images

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()

	assert.NotNil(t, reg)
	assert.NotEmpty(t, reg.GetReleases())
}

func TestRegistry_GetReleases(t *testing.T) {
	reg := NewRegistry()
	releases := reg.GetReleases()

	assert.Len(t, releases, 3)

	// Check that 24.04 is included
	found := false
	for _, rel := range releases {
		if rel.Version == "24.04" {
			found = true
			assert.Equal(t, "noble", rel.Codename)
			assert.True(t, rel.LTS)
		}
	}
	assert.True(t, found, "Ubuntu 24.04 should be in releases")
}

func TestRegistry_GetLTSReleases(t *testing.T) {
	reg := NewRegistry()
	lts := reg.GetLTSReleases()

	// All known releases in our test data are LTS
	assert.Len(t, lts, 3)
	for _, rel := range lts {
		assert.True(t, rel.LTS)
	}
}

func TestRegistry_FindRelease(t *testing.T) {
	reg := NewRegistry()

	tests := []struct {
		input    string
		expected string
	}{
		{"24.04", "noble"},
		{"noble", "noble"},
		{"22.04", "jammy"},
		{"jammy", "jammy"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			rel := reg.FindRelease(tt.input)
			if tt.expected == "" {
				assert.Nil(t, rel)
			} else {
				require.NotNil(t, rel)
				assert.Equal(t, tt.expected, rel.Codename)
			}
		})
	}
}

func TestRegistry_GetCloudImageInfo(t *testing.T) {
	reg := NewRegistry()

	tests := []struct {
		version  string
		arch     string
		expected bool
	}{
		{"24.04", "amd64", true},
		{"24.04", "arm64", true},
		{"22.04", "amd64", true},
		{"24.04", "i386", false},  // Unsupported arch
		{"99.99", "amd64", false}, // Unknown version
	}

	for _, tt := range tests {
		t.Run(tt.version+"_"+tt.arch, func(t *testing.T) {
			info := reg.GetCloudImageInfo(tt.version, tt.arch)
			if tt.expected {
				require.NotNil(t, info)
				assert.Equal(t, tt.version, info.Version)
				assert.Equal(t, tt.arch, info.Arch)
				assert.Contains(t, info.URL, "cloud-images.ubuntu.com")
				assert.NotEmpty(t, info.Filename)
			} else {
				assert.Nil(t, info)
			}
		})
	}
}

func TestRegistry_GetAllCloudImages(t *testing.T) {
	reg := NewRegistry()
	images := reg.GetAllCloudImages()

	// 3 releases x 2 archs = 6 images
	assert.Len(t, images, 6)
}

func TestGenerateImageID(t *testing.T) {
	id := GenerateImageID("24.04", "amd64")
	assert.Equal(t, "ubuntu-24.04-amd64", id)
}

func TestGetDefaultArch(t *testing.T) {
	arch := GetDefaultArch()
	// Should return a valid arch
	assert.Contains(t, []string{"amd64", "arm64"}, arch)
}
