package images

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/globalconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBrowser(t *testing.T) {
	b := New()

	assert.NotNil(t, b)
	assert.Equal(t, "Images", b.Name())
	assert.Equal(t, "4", b.ShortKey())
	assert.NotNil(t, b.registry)
	assert.Equal(t, ModeOSSelection, b.mode)
}

func TestBrowser_Init(t *testing.T) {
	b := New()

	// Should not error even if config doesn't exist
	cmd := b.Init()
	assert.Nil(t, cmd)

	// Should have OS list populated
	assert.NotEmpty(t, b.osList)
	assert.Contains(t, b.osList, "Ubuntu")
	assert.Contains(t, b.osList, "Debian")
}

func TestBrowser_Navigation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test config
	cfg := globalconfig.NewConfig()
	cfg.ImagesDir = tmpDir

	// Save to temp location
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	// Create .config/ucli directory
	configDir := filepath.Join(tmpDir, ".config", "ucli")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Write config
	err = os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(`version: "1.0"
project_path: /tmp
images_dir: `+tmpDir+`
cloud_images: []
`), 0644)
	require.NoError(t, err)

	b := New()
	b.Init()

	// Test OS selection
	assert.Equal(t, ModeOSSelection, b.mode)
	assert.Len(t, b.osList, 2) // Ubuntu and Debian

	// Move cursor down
	b.moveCursor(1)
	assert.Equal(t, 1, b.cursor)

	// Move cursor up (should wrap)
	b.moveCursor(-2)
	assert.Equal(t, 1, b.cursor) // wraps to last item

	// Find Ubuntu in the list (OS list is sorted, so Debian comes first)
	ubuntuIdx := -1
	for i, os := range b.osList {
		if os == "Ubuntu" {
			ubuntuIdx = i
			break
		}
	}
	require.NotEqual(t, -1, ubuntuIdx, "Ubuntu should be in OS list")

	// Select Ubuntu
	b.cursor = ubuntuIdx
	b.selectOS()
	assert.Equal(t, ModeVersionSelection, b.mode)
	assert.Equal(t, "Ubuntu", b.selectedOS)
	assert.NotEmpty(t, b.versionList)

	// Select version
	b.cursor = 0 // First version
	b.selectVersion()
	assert.Equal(t, ModeTypeSelection, b.mode)
	assert.NotEmpty(t, b.selectedVer)
	assert.NotEmpty(t, b.typeList)

	// Select type
	b.cursor = 0 // First type
	b.selectType()
	assert.Equal(t, ModeImageSelection, b.mode)
	assert.NotEmpty(t, b.imageList)

	// Go back
	b.goBack()
	assert.Equal(t, ModeTypeSelection, b.mode)

	b.goBack()
	assert.Equal(t, ModeVersionSelection, b.mode)

	b.goBack()
	assert.Equal(t, ModeOSSelection, b.mode)
}

func TestBrowser_CurlCommandGeneration(t *testing.T) {
	img := ImageMetadata{
		ID:          "ubuntu-24.04-amd64-server",
		Source:      SourceUbuntu,
		Type:        TypeCloudInit,
		Variant:     VariantServer,
		OS:          "Ubuntu",
		Version:     "24.04",
		Codename:    "noble",
		Arch:        "amd64",
		URL:         "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
		ChecksumURL: "https://cloud-images.ubuntu.com/noble/current/SHA256SUMS",
		Filename:    "noble-server-cloudimg-amd64.img",
		Description: "Ubuntu 24.04 LTS Server Cloud Image (amd64)",
		Size:        "~700MB",
		LTS:         true,
	}

	path := "/tmp/test/image.img"
	cmd := generateCurlCommand(&img, path)

	// Verify curl command structure
	assert.Contains(t, cmd, "curl -L")
	assert.Contains(t, cmd, "--create-dirs")
	assert.Contains(t, cmd, "--output")
	assert.Contains(t, cmd, path)
	assert.Contains(t, cmd, img.URL)
	assert.Contains(t, cmd, "# Verify checksum:")
	assert.Contains(t, cmd, "sha256sum -c -")
}

func TestBrowser_CurlCommandGeneration_Desktop(t *testing.T) {
	img := ImageMetadata{
		ID:          "ubuntu-24.04-amd64-desktop",
		Source:      SourceUbuntu,
		Type:        TypeDesktop,
		Variant:     VariantDesktop,
		OS:          "Ubuntu",
		Version:     "24.04",
		Codename:    "noble",
		Arch:        "amd64",
		URL:         "https://releases.ubuntu.com/24.04/ubuntu-24.04-desktop-amd64.iso",
		ChecksumURL: "https://releases.ubuntu.com/24.04/SHA256SUMS",
		Filename:    "ubuntu-24.04-desktop-amd64.iso",
		Description: "Ubuntu 24.04 LTS Desktop (amd64)",
		Size:        "~5.7GB",
		LTS:         true,
	}

	path := "/tmp/test/image.iso"
	cmd := generateCurlCommand(&img, path)

	// Desktop ISOs should have --continue-at - for resume support
	assert.Contains(t, cmd, "--continue-at -")
}

func TestBrowser_CurlCommandGeneration_Debian(t *testing.T) {
	img := ImageMetadata{
		ID:          "debian-12-amd64-generic",
		Source:      SourceDebian,
		Type:        TypeCloudInit,
		Variant:     VariantGeneric,
		OS:          "Debian",
		Version:     "12",
		Codename:    "bookworm",
		Arch:        "amd64",
		URL:         "https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-amd64.qcow2",
		ChecksumURL: "https://cloud.debian.org/images/cloud/bookworm/latest/SHA512SUMS",
		Filename:    "debian-12-generic-amd64.qcow2",
		Description: "Debian 12 Bookworm Generic Cloud Image (amd64)",
		Size:        "~500MB",
		LTS:         false,
	}

	path := "/tmp/test/image.qcow2"
	cmd := generateCurlCommand(&img, path)

	// Debian uses SHA512SUMS
	assert.Contains(t, cmd, "sha512sum -c -")
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHome bool
	}{
		{
			name:     "tilde expansion",
			input:    "~/Downloads/image.img",
			wantHome: true,
		},
		{
			name:     "absolute path",
			input:    "/tmp/image.img",
			wantHome: false,
		},
		{
			name:     "relative path",
			input:    "images/test.img",
			wantHome: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)

			if tt.wantHome {
				home, err := os.UserHomeDir()
				if err == nil && home != "" {
					// Should not contain tilde
					assert.NotContains(t, result, "~")
					// Should contain user's home directory
					assert.Contains(t, result, home)
				} else {
					// If HOME is not available, path is returned as-is
					assert.Equal(t, tt.input, result)
				}
			} else {
				assert.Equal(t, tt.input, result)
			}
		})
	}
}

func TestBrowser_DefaultPathGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := globalconfig.NewConfig()
	cfg.ImagesDir = tmpDir

	manager := NewManager(cfg)

	img := &ImageMetadata{
		OS:       "Ubuntu",
		Version:  "24.04",
		Variant:  VariantServer,
		Filename: "noble-server-cloudimg-amd64.img",
	}

	path := manager.DefaultPathForImage(img)

	expected := filepath.Join(tmpDir, "ubuntu", "24.04", "server", "noble-server-cloudimg-amd64.img")
	assert.Equal(t, expected, path)
}

func TestBrowser_KeyBindings(t *testing.T) {
	b := New()
	b.Init()

	bindings := b.KeyBindings()
	assert.NotEmpty(t, bindings)
	assert.Contains(t, bindings[0], "navigate")
}

func TestBrowser_HasFocusedInput(t *testing.T) {
	b := New()
	b.Init()

	// Initially no focused input
	assert.False(t, b.HasFocusedInput())

	// When curl dialog is shown with focused input
	b.showCurlDialog = true
	b.pathInput.Focus()
	assert.True(t, b.HasFocusedInput())

	// When dialog shown but input not focused
	b.pathInput.Blur()
	assert.False(t, b.HasFocusedInput())
}

func TestBrowser_GetImageStatus(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test image file
	imgPath := filepath.Join(tmpDir, "test-image.img")
	err := os.WriteFile(imgPath, []byte("test"), 0644)
	require.NoError(t, err)

	cfg := globalconfig.NewConfig()
	cfg.ImagesDir = tmpDir

	// Add image to config
	cfg.AddCloudImage(globalconfig.CloudImage{
		ID:   "test-image",
		Path: imgPath,
	})

	b := New()
	b.config = cfg

	img := &ImageMetadata{
		ID: "test-image",
	}

	// Should return true since file exists
	status := b.getImageStatus(img)
	assert.True(t, status)

	// Test non-existent image
	img2 := &ImageMetadata{
		ID: "non-existent",
	}
	status2 := b.getImageStatus(img2)
	assert.False(t, status2)
}

func TestBrowser_Update_KeyMessages(t *testing.T) {
	b := New()
	b.Init()

	tests := []struct {
		name     string
		key      string
		mode     BrowserMode
		wantMode BrowserMode
	}{
		{
			name:     "down key moves cursor",
			key:      "down",
			mode:     ModeOSSelection,
			wantMode: ModeOSSelection,
		},
		{
			name:     "up key moves cursor",
			key:      "up",
			mode:     ModeOSSelection,
			wantMode: ModeOSSelection,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.mode = tt.mode
			b.cursor = 0

			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "down" {
				msg.Type = tea.KeyDown
			} else if tt.key == "up" {
				msg.Type = tea.KeyUp
			}

			tab, _ := b.Update(msg)
			browser := tab.(*Browser)
			assert.Equal(t, tt.wantMode, browser.mode)
		})
	}
}

func TestBrowser_View(t *testing.T) {
	// Set HOME to avoid config migration issues
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	// Create config directory
	configDir := filepath.Join(tmpDir, ".config", "ucli")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	b := New()
	b.Init()

	// Test view in different modes
	b.mode = ModeOSSelection
	view := b.View()
	assert.Contains(t, view, "Images")
	assert.Contains(t, view, "Ubuntu")
	assert.Contains(t, view, "Debian")

	// Test with error
	b.err = assert.AnError
	view = b.View()
	assert.Contains(t, view, "Error")
}

func TestBrowser_CurlDialog(t *testing.T) {
	tmpDir := t.TempDir()

	// Set HOME to avoid config migration issues
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	// Create config directory
	configDir := filepath.Join(tmpDir, ".config", "ucli")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	cfg := globalconfig.NewConfig()
	cfg.ImagesDir = tmpDir

	b := New()
	b.config = cfg
	b.manager = NewManager(cfg)
	b.Init()

	// Navigate to an image
	b.osList = []string{"Ubuntu"}
	b.selectedOS = "Ubuntu"
	b.versionList = []string{"24.04"}
	b.selectedVer = "24.04"
	b.typeList = []ImageType{TypeCloudInit}
	b.selectedType = TypeCloudInit
	b.imageList = []ImageMetadata{
		{
			ID:          "ubuntu-24.04-amd64-server",
			Source:      SourceUbuntu,
			Type:        TypeCloudInit,
			Variant:     VariantServer,
			OS:          "Ubuntu",
			Version:     "24.04",
			Codename:    "noble",
			Arch:        "amd64",
			URL:         "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
			ChecksumURL: "https://cloud-images.ubuntu.com/noble/current/SHA256SUMS",
			Filename:    "noble-server-cloudimg-amd64.img",
			Description: "Ubuntu 24.04 LTS Server Cloud Image (amd64)",
			Size:        "~700MB",
			LTS:         true,
		},
	}
	b.mode = ModeImageSelection
	b.cursor = 0

	// Show curl dialog
	b.showCurlCommandDialog()

	assert.True(t, b.showCurlDialog)
	assert.NotNil(t, b.selectedImage)
	assert.NotEmpty(t, b.pathInput.Value())
	assert.NotEmpty(t, b.curlCommand)

	// Verify view shows dialog
	view := b.View()
	assert.Contains(t, view, "Download Command")
	assert.Contains(t, view, "curl")

	// Close dialog
	b.closeCurlDialog()
	assert.False(t, b.showCurlDialog)
	assert.Nil(t, b.selectedImage)
}

func TestBrowser_TextInputInCurlDialog(t *testing.T) {
	tmpDir := t.TempDir()

	b := New()
	b.Init()

	// Setup dialog state
	b.showCurlDialog = true
	img := &ImageMetadata{
		ID:       "test-img",
		Filename: "test.img",
		URL:      "https://example.com/test.img",
	}
	b.selectedImage = img
	b.pathInput = textinput.New()
	initialPath := filepath.Join(tmpDir, "test.img")
	b.pathInput.SetValue(initialPath)
	b.pathInput.Focus()

	// Generate initial command
	b.curlCommand = generateCurlCommand(b.selectedImage, initialPath)

	// Test Tab key regenerates command
	initialCmd := b.curlCommand
	newPath := filepath.Join(tmpDir, "new-path.img")
	b.pathInput.SetValue(newPath)

	msg := tea.KeyMsg{Type: tea.KeyTab}
	tab, _ := b.handleCurlDialog(msg)

	browser := tab.(*Browser)
	assert.NotEqual(t, initialCmd, browser.curlCommand)
	assert.Contains(t, browser.curlCommand, newPath)
}
