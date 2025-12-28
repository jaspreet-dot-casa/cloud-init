package iso

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("/tmp/project")

	assert.NotNil(t, builder)
	assert.Equal(t, "/tmp/project", builder.projectRoot)
	assert.NotNil(t, builder.tools)
	assert.False(t, builder.verbose)
}

func TestBuilder_SetVerbose(t *testing.T) {
	builder := NewBuilder("/tmp/project")

	builder.SetVerbose(true)
	assert.True(t, builder.verbose)

	builder.SetVerbose(false)
	assert.False(t, builder.verbose)
}

func TestBuilder_CheckTools(t *testing.T) {
	builder := NewBuilder("/tmp/project")
	err := builder.CheckTools()

	// This test depends on whether xorriso is installed
	if err != nil {
		assert.Contains(t, err.Error(), "xorriso")
	} else {
		assert.True(t, builder.ToolsAvailable())
	}
}

func TestBuilder_InstallInstructions(t *testing.T) {
	builder := NewBuilder("/tmp/project")
	instructions := builder.InstallInstructions()

	assert.NotEmpty(t, instructions)
	assert.Contains(t, instructions, "xorriso")
}

func TestBuilder_Build_MissingSourceISO(t *testing.T) {
	builder := NewBuilder("/tmp/project")

	cfg := &config.FullConfig{
		Username: "testuser",
		Hostname: "test-host",
	}

	opts := &ISOOptions{
		SourceISO: "/nonexistent/ubuntu.iso",
	}

	err := builder.Build(cfg, opts)
	assert.Error(t, err)
	// Either tools not available or source ISO not found
	assert.True(t,
		strings.Contains(err.Error(), "xorriso") ||
			strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "source ISO"))
}

func TestBuilder_Build_InvalidOptions(t *testing.T) {
	builder := NewBuilder("/tmp/project")

	// Skip if xorriso not available
	if err := builder.CheckTools(); err != nil {
		t.Skip("xorriso not installed")
	}

	cfg := &config.FullConfig{
		Username: "testuser",
		Hostname: "test-host",
	}

	// Create a temp file that looks like an ISO
	tmpDir := t.TempDir()
	fakeISO := filepath.Join(tmpDir, "fake.iso")
	err := os.WriteFile(fakeISO, []byte("fake iso content"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name    string
		opts    *ISOOptions
		errText string
	}{
		{
			name:    "empty source ISO",
			opts:    &ISOOptions{},
			errText: "source ISO path is required",
		},
		{
			name: "invalid Ubuntu version",
			opts: &ISOOptions{
				SourceISO:     fakeISO,
				UbuntuVersion: "20.04",
			},
			errText: "unsupported Ubuntu version",
		},
		{
			name: "invalid storage layout",
			opts: &ISOOptions{
				SourceISO:     fakeISO,
				UbuntuVersion: "24.04",
				StorageLayout: StorageLayout("invalid"),
			},
			errText: "unsupported storage layout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := builder.Build(cfg, tt.opts)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errText)
		})
	}
}

func TestBuilder_injectConfiguration(t *testing.T) {
	builder := NewBuilder("/tmp/project")

	// Create temp directory structure
	tmpDir := t.TempDir()
	extractDir := filepath.Join(tmpDir, "iso")
	err := os.MkdirAll(extractDir, 0755)
	require.NoError(t, err)

	userData := []byte("#cloud-config\nautoinstall:\n  version: 1\n")
	metaData := []byte("")

	err = builder.injectConfiguration(extractDir, userData, metaData)
	require.NoError(t, err)

	// Verify nocloud directory was created
	nocloudDir := filepath.Join(extractDir, "nocloud")
	info, err := os.Stat(nocloudDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify user-data was written
	content, err := os.ReadFile(filepath.Join(nocloudDir, "user-data"))
	require.NoError(t, err)
	assert.Equal(t, userData, content)

	// Verify meta-data was written
	content, err = os.ReadFile(filepath.Join(nocloudDir, "meta-data"))
	require.NoError(t, err)
	assert.Equal(t, metaData, content)
}

func TestBuilder_modifyGrubConfig(t *testing.T) {
	builder := NewBuilder("/tmp/project")

	// Create temp directory with mock grub.cfg
	tmpDir := t.TempDir()
	extractDir := filepath.Join(tmpDir, "iso")
	grubDir := filepath.Join(extractDir, "boot", "grub")
	err := os.MkdirAll(grubDir, 0755)
	require.NoError(t, err)

	// Write mock grub.cfg
	originalContent := `menuentry "Ubuntu Server" {
    linux /casper/vmlinuz quiet ---
    initrd /casper/initrd
}`
	grubCfgPath := filepath.Join(grubDir, "grub.cfg")
	err = os.WriteFile(grubCfgPath, []byte(originalContent), 0644)
	require.NoError(t, err)

	// Modify grub config
	err = builder.modifyGrubConfig(extractDir)
	require.NoError(t, err)

	// Verify autoinstall was added
	content, err := os.ReadFile(grubCfgPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "autoinstall")
	assert.Contains(t, string(content), "ds=nocloud")
	assert.Contains(t, string(content), "/cdrom/nocloud/")
}

func TestBuilder_modifyGrubConfig_NoGrubCfg(t *testing.T) {
	builder := NewBuilder("/tmp/project")

	// Create temp directory without grub.cfg
	tmpDir := t.TempDir()
	extractDir := filepath.Join(tmpDir, "iso")
	err := os.MkdirAll(extractDir, 0755)
	require.NoError(t, err)

	err = builder.modifyGrubConfig(extractDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grub.cfg not found")
}

func TestBuilder_modifyGrubConfig_AlreadyHasAutoinstall(t *testing.T) {
	builder := NewBuilder("/tmp/project")

	// Create temp directory with mock grub.cfg that already has autoinstall
	tmpDir := t.TempDir()
	extractDir := filepath.Join(tmpDir, "iso")
	grubDir := filepath.Join(extractDir, "boot", "grub")
	err := os.MkdirAll(grubDir, 0755)
	require.NoError(t, err)

	originalContent := `menuentry "Ubuntu Server" {
    linux /casper/vmlinuz quiet autoinstall ds=nocloud;s=/cdrom/nocloud/ ---
    initrd /casper/initrd
}`
	grubCfgPath := filepath.Join(grubDir, "grub.cfg")
	err = os.WriteFile(grubCfgPath, []byte(originalContent), 0644)
	require.NoError(t, err)

	err = builder.modifyGrubConfig(extractDir)
	require.NoError(t, err)

	// Should not add duplicate autoinstall
	content, err := os.ReadFile(grubCfgPath)
	require.NoError(t, err)

	// Count occurrences of autoinstall
	count := strings.Count(string(content), "autoinstall")
	assert.Equal(t, 1, count, "Should not duplicate autoinstall parameter")
}
