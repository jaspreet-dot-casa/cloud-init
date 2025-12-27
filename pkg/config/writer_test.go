package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFullConfigFromFormResult(t *testing.T) {
	result := &tui.FormResult{
		User: tui.UserConfig{
			Username:     "testuser",
			Hostname:     "testhost",
			SSHPublicKey: "ssh-ed25519 AAAA...",
			FullName:     "Test User",
			Email:        "test@example.com",
		},
		SelectedPackages: []string{"lazygit", "docker", "starship"},
		Optional: tui.OptionalConfig{
			GithubUser:   "testgh",
			TailscaleKey: "tskey-test",
			GithubPAT:    "ghp_test",
		},
		OutputMode: tui.OutputCloudInit,
	}

	cfg := NewFullConfigFromFormResult(result)

	assert.Equal(t, "testuser", cfg.Username)
	assert.Equal(t, "testhost", cfg.Hostname)
	assert.Equal(t, "Test User", cfg.FullName)
	assert.Equal(t, "test@example.com", cfg.Email)
	assert.Equal(t, []string{"lazygit", "docker", "starship"}, cfg.EnabledPackages)
	assert.Equal(t, "testgh", cfg.GithubUser)
	assert.Equal(t, "tskey-test", cfg.TailscaleAuthKey)
	assert.Equal(t, "ghp_test", cfg.GithubPAT)
	assert.True(t, cfg.DockerEnabled)
	assert.True(t, cfg.GitPullRebase)
	assert.Equal(t, "main", cfg.GitDefaultBranch)
}

func TestContainsPackage(t *testing.T) {
	packages := []string{"lazygit", "docker", "starship"}

	assert.True(t, containsPackage(packages, "docker"))
	assert.True(t, containsPackage(packages, "lazygit"))
	assert.False(t, containsPackage(packages, "zoxide"))
	assert.False(t, containsPackage(packages, ""))
}

func TestWriteConfigEnv(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &FullConfig{
		Username:               "testuser",
		Hostname:               "testhost",
		FullName:               "Test User",
		Email:                  "test@example.com",
		EnabledPackages:        []string{"lazygit", "docker"},
		GitDefaultBranch:       "main",
		GitPushAutoSetupRemote: true,
		GitPullRebase:          true,
		GitPager:               "delta",
		GitURLRewriteGithub:    true,
		DockerEnabled:          true,
		DockerAddToGroup:       true,
		DockerStartOnBoot:      true,
	}

	writer := NewWriter(tmpDir)
	err := writer.WriteConfigEnv(cfg)
	require.NoError(t, err)

	// Read the file and verify contents
	content, err := os.ReadFile(filepath.Join(tmpDir, "config.env"))
	require.NoError(t, err)

	contentStr := string(content)

	// Check for expected content
	assert.Contains(t, contentStr, `USER_NAME="Test User"`)
	assert.Contains(t, contentStr, `USER_EMAIL="test@example.com"`)
	assert.Contains(t, contentStr, `USER_USERNAME="testuser"`)
	assert.Contains(t, contentStr, `GIT_DEFAULT_BRANCH="main"`)
	assert.Contains(t, contentStr, `GIT_PUSH_AUTO_SETUP_REMOTE=true`)
	assert.Contains(t, contentStr, `PACKAGE_LAZYGIT_ENABLED=true`)
	assert.Contains(t, contentStr, `DOCKER_ENABLED=true`)
}

func TestWriteSecretsEnv(t *testing.T) {
	tmpDir := t.TempDir()

	// Create cloud-init subdirectory
	err := os.MkdirAll(filepath.Join(tmpDir, "cloud-init"), 0755)
	require.NoError(t, err)

	cfg := &FullConfig{
		Username:         "testuser",
		Hostname:         "testhost",
		SSHPublicKey:     "ssh-ed25519 AAAA...",
		FullName:         "Test User",
		Email:            "test@example.com",
		TailscaleAuthKey: "tskey-test",
		GithubPAT:        "ghp_test",
		GithubUser:       "testgh",
		RepoURL:          "https://github.com/test/repo.git",
		RepoBranch:       "main",
	}

	writer := NewWriter(tmpDir)
	err = writer.WriteSecretsEnv(cfg)
	require.NoError(t, err)

	// Read the file and verify contents
	content, err := os.ReadFile(filepath.Join(tmpDir, "cloud-init", "secrets.env"))
	require.NoError(t, err)

	contentStr := string(content)

	// Check for expected content
	assert.Contains(t, contentStr, `USERNAME="testuser"`)
	assert.Contains(t, contentStr, `HOSTNAME="testhost"`)
	assert.Contains(t, contentStr, `SSH_PUBLIC_KEY="ssh-ed25519 AAAA..."`)
	assert.Contains(t, contentStr, `TAILSCALE_AUTH_KEY="tskey-test"`)
	assert.Contains(t, contentStr, `GITHUB_PAT="ghp_test"`)
	assert.Contains(t, contentStr, `GITHUB_USER="testgh"`)

	// Verify file permissions (should be 0600 for secrets)
	info, err := os.Stat(filepath.Join(tmpDir, "cloud-init", "secrets.env"))
	require.NoError(t, err)
	// On Unix, check that file is not world-readable
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestWriteAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create cloud-init subdirectory
	err := os.MkdirAll(filepath.Join(tmpDir, "cloud-init"), 0755)
	require.NoError(t, err)

	cfg := &FullConfig{
		Username:        "testuser",
		Hostname:        "testhost",
		SSHPublicKey:    "ssh-ed25519 AAAA...",
		FullName:        "Test User",
		Email:           "test@example.com",
		EnabledPackages: []string{"lazygit"},
	}

	writer := NewWriter(tmpDir)
	err = writer.WriteAll(cfg)
	require.NoError(t, err)

	// Verify both files exist
	_, err = os.Stat(filepath.Join(tmpDir, "config.env"))
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmpDir, "cloud-init", "secrets.env"))
	assert.NoError(t, err)
}

func TestConfigEnvIsSourceable(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &FullConfig{
		Username:        "testuser",
		Hostname:        "test-host",
		FullName:        "Test User",
		Email:           "test@example.com",
		EnabledPackages: []string{"lazygit"},
	}

	writer := NewWriter(tmpDir)
	err := writer.WriteConfigEnv(cfg)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "config.env"))
	require.NoError(t, err)

	// Check it starts with shebang
	assert.True(t, strings.HasPrefix(string(content), "#!/bin/bash"))
}
