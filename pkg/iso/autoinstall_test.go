package iso

import (
	"strings"
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAutoInstallGenerator_Generate(t *testing.T) {
	generator := NewAutoInstallGenerator("/tmp/project")

	cfg := &config.FullConfig{
		Username:      "testuser",
		Hostname:      "test-host",
		SSHPublicKeys: []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest test@example.com"},
		FullName:      "Test User",
		Email:         "test@example.com",
		DockerEnabled: true,
	}

	opts := &ISOOptions{
		UbuntuVersion: "24.04",
		StorageLayout: StorageLVM,
		Timezone:      "America/New_York",
		Locale:        "en_US.UTF-8",
	}

	content, err := generator.Generate(cfg, opts)
	require.NoError(t, err)
	require.NotEmpty(t, content)

	// Should start with cloud-config header
	assert.True(t, strings.HasPrefix(string(content), "#cloud-config\n"))

	// Parse the YAML to verify structure
	var wrapper CloudConfigWrapper
	err = yaml.Unmarshal(content[len("#cloud-config\n"):], &wrapper)
	require.NoError(t, err)

	// Verify autoinstall version
	assert.Equal(t, 1, wrapper.AutoInstall.Version)

	// Verify identity
	assert.Equal(t, "test-host", wrapper.AutoInstall.Identity.Hostname)
	assert.Equal(t, "testuser", wrapper.AutoInstall.Identity.Username)

	// Verify SSH config
	assert.True(t, wrapper.AutoInstall.SSH.InstallServer)
	assert.Len(t, wrapper.AutoInstall.SSH.AuthorizedKeys, 1)
	assert.Contains(t, wrapper.AutoInstall.SSH.AuthorizedKeys[0], "ssh-ed25519")

	// Verify storage layout
	assert.Equal(t, "lvm", wrapper.AutoInstall.Storage.Layout.Name)

	// Verify timezone
	assert.Equal(t, "America/New_York", wrapper.AutoInstall.Timezone)

	// Verify late-commands exist
	assert.NotEmpty(t, wrapper.AutoInstall.LateCommands)
}

func TestAutoInstallGenerator_Generate_MultipleSSHKeys(t *testing.T) {
	generator := NewAutoInstallGenerator("/tmp/project")

	cfg := &config.FullConfig{
		Username: "testuser",
		Hostname: "test-host",
		SSHPublicKeys: []string{
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKey1 key1@example.com",
			"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAKey2 key2@example.com",
		},
	}

	opts := DefaultOptions()

	content, err := generator.Generate(cfg, opts)
	require.NoError(t, err)

	var wrapper CloudConfigWrapper
	err = yaml.Unmarshal(content[len("#cloud-config\n"):], &wrapper)
	require.NoError(t, err)

	assert.Len(t, wrapper.AutoInstall.SSH.AuthorizedKeys, 2)
}

func TestAutoInstallGenerator_Generate_NilConfig(t *testing.T) {
	generator := NewAutoInstallGenerator("/tmp/project")

	_, err := generator.Generate(nil, DefaultOptions())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is nil")
}

func TestAutoInstallGenerator_Generate_NilOptions(t *testing.T) {
	generator := NewAutoInstallGenerator("/tmp/project")

	cfg := &config.FullConfig{
		Username: "testuser",
		Hostname: "test-host",
	}

	_, err := generator.Generate(cfg, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "options is nil")
}

func TestAutoInstallGenerator_Generate_StorageLayouts(t *testing.T) {
	tests := []struct {
		layout   StorageLayout
		expected string
	}{
		{StorageLVM, "lvm"},
		{StorageDirect, "direct"},
		{StorageZFS, "zfs"},
	}

	generator := NewAutoInstallGenerator("/tmp/project")

	for _, tt := range tests {
		t.Run(string(tt.layout), func(t *testing.T) {
			cfg := &config.FullConfig{
				Username: "testuser",
				Hostname: "test-host",
			}

			opts := &ISOOptions{
				StorageLayout: tt.layout,
				Timezone:      "UTC",
				Locale:        "en_US.UTF-8",
			}

			content, err := generator.Generate(cfg, opts)
			require.NoError(t, err)

			var wrapper CloudConfigWrapper
			err = yaml.Unmarshal(content[len("#cloud-config\n"):], &wrapper)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, wrapper.AutoInstall.Storage.Layout.Name)
		})
	}
}

func TestAutoInstallGenerator_Generate_LateCommands(t *testing.T) {
	generator := NewAutoInstallGenerator("/tmp/project")

	cfg := &config.FullConfig{
		Username:      "testuser",
		Hostname:      "test-host",
		DockerEnabled: true,
		FullName:      "Test User",
		Email:         "test@example.com",
	}

	opts := DefaultOptions()

	content, err := generator.Generate(cfg, opts)
	require.NoError(t, err)

	var wrapper CloudConfigWrapper
	err = yaml.Unmarshal(content[len("#cloud-config\n"):], &wrapper)
	require.NoError(t, err)

	commands := wrapper.AutoInstall.LateCommands

	// Check Docker installation is included
	hasDocker := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "docker") {
			hasDocker = true
			break
		}
	}
	assert.True(t, hasDocker, "Docker installation should be in late-commands")

	// Check git config is included
	hasGitConfig := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "git config") {
			hasGitConfig = true
			break
		}
	}
	assert.True(t, hasGitConfig, "Git config should be in late-commands")
}

func TestAutoInstallGenerator_Generate_NoDocker(t *testing.T) {
	generator := NewAutoInstallGenerator("/tmp/project")

	cfg := &config.FullConfig{
		Username:      "testuser",
		Hostname:      "test-host",
		DockerEnabled: false,
	}

	opts := DefaultOptions()

	content, err := generator.Generate(cfg, opts)
	require.NoError(t, err)

	var wrapper CloudConfigWrapper
	err = yaml.Unmarshal(content[len("#cloud-config\n"):], &wrapper)
	require.NoError(t, err)

	// Check Docker installation is NOT included
	for _, cmd := range wrapper.AutoInstall.LateCommands {
		assert.NotContains(t, cmd, "get.docker.com", "Docker installation should not be in late-commands when disabled")
	}
}

func TestAutoInstallGenerator_GenerateMetaData(t *testing.T) {
	generator := NewAutoInstallGenerator("/tmp/project")

	metaData := generator.GenerateMetaData()

	// meta-data should be empty but not nil
	assert.NotNil(t, metaData)
	assert.Empty(t, metaData)
}

func TestAutoInstallGenerator_Generate_Packages(t *testing.T) {
	generator := NewAutoInstallGenerator("/tmp/project")

	cfg := &config.FullConfig{
		Username: "testuser",
		Hostname: "test-host",
	}

	opts := DefaultOptions()

	content, err := generator.Generate(cfg, opts)
	require.NoError(t, err)

	var wrapper CloudConfigWrapper
	err = yaml.Unmarshal(content[len("#cloud-config\n"):], &wrapper)
	require.NoError(t, err)

	// Verify essential packages are included
	packages := wrapper.AutoInstall.Packages
	assert.Contains(t, packages, "curl")
	assert.Contains(t, packages, "git")
	assert.Contains(t, packages, "zsh")
	assert.Contains(t, packages, "jq")
}
