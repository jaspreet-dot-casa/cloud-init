package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCmd(t *testing.T) {
	rootCmd := newRootCmd()

	assert.Equal(t, "ucli", rootCmd.Use)
	assert.Equal(t, "Ubuntu Cloud-Init CLI Tool", rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)
}

func TestRootCmdHelp(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"--help"})

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "ucli")
	assert.Contains(t, output, "generate")
	assert.Contains(t, output, "packages")
	assert.Contains(t, output, "validate")
	assert.Contains(t, output, "build")
}

func TestRootCmdVersion(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"--version"})

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "ucli version")
}

func TestGenerateCmd(t *testing.T) {
	// Skip this test as generate command requires an interactive TTY
	// The TUI forms are tested separately in pkg/tui/form_test.go
	t.Skip("generate command requires interactive TTY")
}

func TestPackagesCmd(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"packages"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestValidateCmd(t *testing.T) {
	t.Run("returns error when config files missing", func(t *testing.T) {
		// Save current directory and change to temp dir
		origDir, err := os.Getwd()
		require.NoError(t, err)
		tmpDir := t.TempDir()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(origDir)
			require.NoError(t, err)
		}()

		// Create minimal project structure so findProjectRoot works
		err = os.MkdirAll("scripts/packages", 0755)
		require.NoError(t, err)

		rootCmd := newRootCmd()
		rootCmd.SetArgs([]string{"validate"})

		err = rootCmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("succeeds with valid config files", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)
		tmpDir := t.TempDir()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(origDir)
			require.NoError(t, err)
		}()

		// Create project structure
		err = os.MkdirAll("scripts/packages", 0755)
		require.NoError(t, err)
		err = os.MkdirAll("cloud-init", 0755)
		require.NoError(t, err)

		// Create valid secrets.env
		secretsContent := `
USERNAME="testuser"
HOSTNAME="test-host"
SSH_PUBLIC_KEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest test@example.com"
USER_NAME="Test User"
USER_EMAIL="test@example.com"
`
		err = os.WriteFile("cloud-init/secrets.env", []byte(secretsContent), 0600)
		require.NoError(t, err)

		// Create valid config.env
		configContent := `
GIT_PUSH_AUTO_SETUP_REMOTE=true
DOCKER_ENABLED=true
`
		err = os.WriteFile("config.env", []byte(configContent), 0644)
		require.NoError(t, err)

		rootCmd := newRootCmd()
		rootCmd.SetArgs([]string{"validate"})

		err = rootCmd.Execute()
		assert.NoError(t, err)
	})
}

func TestBuildCmd(t *testing.T) {
	t.Run("returns error when config files missing", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)
		tmpDir := t.TempDir()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(origDir)
			require.NoError(t, err)
		}()

		// Create minimal project structure
		err = os.MkdirAll("scripts/packages", 0755)
		require.NoError(t, err)

		rootCmd := newRootCmd()
		rootCmd.SetArgs([]string{"build"})

		err = rootCmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("succeeds with valid config files and template", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)
		tmpDir := t.TempDir()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(origDir)
			require.NoError(t, err)
		}()

		// Create project structure
		err = os.MkdirAll("scripts/packages", 0755)
		require.NoError(t, err)
		err = os.MkdirAll("cloud-init", 0755)
		require.NoError(t, err)

		// Create valid secrets.env
		secretsContent := `
USERNAME="testuser"
HOSTNAME="test-host"
SSH_PUBLIC_KEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest test@example.com"
USER_NAME="Test User"
USER_EMAIL="test@example.com"
`
		err = os.WriteFile("cloud-init/secrets.env", []byte(secretsContent), 0600)
		require.NoError(t, err)

		// Create valid config.env
		configContent := `
GIT_PUSH_AUTO_SETUP_REMOTE=true
DOCKER_ENABLED=true
`
		err = os.WriteFile("config.env", []byte(configContent), 0644)
		require.NoError(t, err)

		// Create template
		templateContent := `#cloud-config
users:
  - name: ${USERNAME}
hostname: ${HOSTNAME}
`
		err = os.WriteFile("cloud-init/cloud-init.template.yaml", []byte(templateContent), 0644)
		require.NoError(t, err)

		rootCmd := newRootCmd()
		rootCmd.SetArgs([]string{"build"})

		err = rootCmd.Execute()
		assert.NoError(t, err)

		// Verify output file was created
		_, err = os.Stat("cloud-init/cloud-init.yaml")
		assert.NoError(t, err)
	})
}

func TestSubcommandHelp(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		expects []string
	}{
		{
			name:    "generate help",
			args:    []string{"generate", "--help"},
			expects: []string{"TUI", "cloud-init"},
		},
		{
			name:    "packages help",
			args:    []string{"packages", "--help"},
			expects: []string{"packages", "cloud-init"},
		},
		{
			name:    "validate help",
			args:    []string{"validate", "--help"},
			expects: []string{"config.env", "secrets.env"},
		},
		{
			name:    "build help",
			args:    []string{"build", "--help"},
			expects: []string{"Non-interactive", "config.env"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := newRootCmd()
			rootCmd.SetArgs(tt.args)

			var buf bytes.Buffer
			rootCmd.SetOut(&buf)

			err := rootCmd.Execute()
			require.NoError(t, err)

			output := buf.String()
			for _, expect := range tt.expects {
				assert.Contains(t, output, expect)
			}
		})
	}
}
