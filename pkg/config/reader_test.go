package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadAll(t *testing.T) {
	t.Run("reads valid config files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create cloud-init directory
		cloudInitDir := filepath.Join(tmpDir, "cloud-init")
		err := os.MkdirAll(cloudInitDir, 0755)
		require.NoError(t, err)

		// Create secrets.env
		secretsContent := `
USERNAME="testuser"
HOSTNAME="test-host"
SSH_PUBLIC_KEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest test@example.com"
USER_NAME="Test User"
USER_EMAIL="test@example.com"
MACHINE_USER_NAME="My Machine"
TAILSCALE_AUTH_KEY="tskey-auth-test"
GITHUB_USER="testgithub"
GITHUB_PAT="ghp_test123"
REPO_URL="https://github.com/test/repo.git"
REPO_BRANCH="develop"
`
		err = os.WriteFile(filepath.Join(cloudInitDir, "secrets.env"), []byte(secretsContent), 0600)
		require.NoError(t, err)

		// Create config.env
		configContent := `
USER_NAME="Test User"
USER_EMAIL="test@example.com"
GIT_DEFAULT_BRANCH="main"
GIT_PUSH_AUTO_SETUP_REMOTE=true
GIT_PULL_REBASE=false
GIT_PAGER="less"
TAILSCALE_SSH_ENABLED=true
DOCKER_ENABLED=true
PACKAGE_LAZYGIT_ENABLED=true
PACKAGE_DOCKER_ENABLED=true
PACKAGE_STARSHIP_ENABLED=true
`
		err = os.WriteFile(filepath.Join(tmpDir, "config.env"), []byte(configContent), 0644)
		require.NoError(t, err)

		reader := NewReader(tmpDir)
		cfg, err := reader.ReadAll()
		require.NoError(t, err)

		// Check secrets.env values
		assert.Equal(t, "testuser", cfg.Username)
		assert.Equal(t, "test-host", cfg.Hostname)
		assert.Len(t, cfg.SSHPublicKeys, 1)
		assert.Contains(t, cfg.SSHPublicKeys[0], "ssh-ed25519")
		assert.Equal(t, "My Machine", cfg.MachineName)
		assert.Equal(t, "tskey-auth-test", cfg.TailscaleAuthKey)
		assert.Equal(t, "testgithub", cfg.GithubUser)
		assert.Equal(t, "ghp_test123", cfg.GithubPAT)
		assert.Equal(t, "https://github.com/test/repo.git", cfg.RepoURL)
		assert.Equal(t, "develop", cfg.RepoBranch)

		// Check config.env values
		assert.Equal(t, "Test User", cfg.FullName)
		assert.Equal(t, "test@example.com", cfg.Email)
		assert.Equal(t, "main", cfg.GitDefaultBranch)
		assert.True(t, cfg.GitPushAutoSetupRemote)
		assert.False(t, cfg.GitPullRebase)
		assert.Equal(t, "less", cfg.GitPager)
		assert.True(t, cfg.TailscaleSSHEnabled)
		assert.True(t, cfg.DockerEnabled)

		// Check enabled packages
		assert.Contains(t, cfg.EnabledPackages, "lazygit")
		assert.Contains(t, cfg.EnabledPackages, "docker")
		assert.Contains(t, cfg.EnabledPackages, "starship")
	})

	t.Run("uses defaults for missing values", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create minimal files
		cloudInitDir := filepath.Join(tmpDir, "cloud-init")
		err := os.MkdirAll(cloudInitDir, 0755)
		require.NoError(t, err)

		secretsContent := `
USERNAME="user"
HOSTNAME="host"
SSH_PUBLIC_KEY="ssh-ed25519 test"
USER_NAME="Name"
USER_EMAIL="email@test.com"
`
		err = os.WriteFile(filepath.Join(cloudInitDir, "secrets.env"), []byte(secretsContent), 0600)
		require.NoError(t, err)

		// Empty config.env
		err = os.WriteFile(filepath.Join(tmpDir, "config.env"), []byte(""), 0644)
		require.NoError(t, err)

		reader := NewReader(tmpDir)
		cfg, err := reader.ReadAll()
		require.NoError(t, err)

		// Check defaults
		assert.Equal(t, "main", cfg.GitDefaultBranch)
		assert.True(t, cfg.GitPushAutoSetupRemote)
		assert.True(t, cfg.GitPullRebase)
		assert.Equal(t, "delta", cfg.GitPager)
		assert.True(t, cfg.TailscaleSSHEnabled)
		assert.True(t, cfg.DockerEnabled)
		assert.Equal(t, "12h", cfg.TailscaleSSHCheckPeriod)
		assert.Equal(t, "main", cfg.RepoBranch)
	})

	t.Run("fails when secrets.env is missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Only create config.env, not secrets.env
		err := os.WriteFile(filepath.Join(tmpDir, "config.env"), []byte(""), 0644)
		require.NoError(t, err)

		reader := NewReader(tmpDir)
		_, err = reader.ReadAll()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secrets.env")
	})

	t.Run("fails when config.env is missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Only create secrets.env, not config.env
		cloudInitDir := filepath.Join(tmpDir, "cloud-init")
		err := os.MkdirAll(cloudInitDir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(cloudInitDir, "secrets.env"), []byte("USERNAME=test"), 0600)
		require.NoError(t, err)

		reader := NewReader(tmpDir)
		_, err = reader.ReadAll()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config.env")
	})
}

func TestParseEnabledPackages(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected []string
	}{
		{
			name: "parses enabled packages",
			envVars: map[string]string{
				"PACKAGE_LAZYGIT_ENABLED": "true",
				"PACKAGE_DOCKER_ENABLED":  "true",
				"PACKAGE_NVIM_ENABLED":    "false",
			},
			expected: []string{"lazygit", "docker"},
		},
		{
			name: "handles underscores in package names",
			envVars: map[string]string{
				"PACKAGE_LAZY_DOCKER_ENABLED": "true",
			},
			expected: []string{"lazy-docker"},
		},
		{
			name:     "empty when no packages enabled",
			envVars:  map[string]string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEnabledPackages(tt.envVars)
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				for _, pkg := range tt.expected {
					assert.Contains(t, result, pkg)
				}
			}
		})
	}
}

func TestGetStringOrDefault(t *testing.T) {
	envVars := map[string]string{
		"KEY1": "value1",
		"KEY2": "",
	}

	assert.Equal(t, "value1", getStringOrDefault(envVars, "KEY1", "default"))
	assert.Equal(t, "default", getStringOrDefault(envVars, "KEY2", "default"))
	assert.Equal(t, "default", getStringOrDefault(envVars, "KEY3", "default"))
}

func TestGetBoolOrDefault(t *testing.T) {
	envVars := map[string]string{
		"TRUE_VAL":    "true",
		"FALSE_VAL":   "false",
		"INVALID_VAL": "yes",
	}

	assert.True(t, getBoolOrDefault(envVars, "TRUE_VAL", false))
	assert.False(t, getBoolOrDefault(envVars, "FALSE_VAL", true))
	assert.True(t, getBoolOrDefault(envVars, "INVALID_VAL", true))   // Falls back to default
	assert.False(t, getBoolOrDefault(envVars, "MISSING_VAL", false)) // Missing, uses default
}
