package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Run("substitutes variables correctly", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create template
		templateContent := `#cloud-config
users:
  - name: ${USERNAME}
    ssh_authorized_keys:
      - ${SSH_PUBLIC_KEY}
hostname: ${HOSTNAME}
write_files:
  - path: /home/${USERNAME}/.config
    content: |
      USER_NAME="${USER_NAME}"
      USER_EMAIL="${USER_EMAIL}"
`
		templatePath := filepath.Join(tmpDir, "template.yaml")
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		outputPath := filepath.Join(tmpDir, "output.yaml")

		cfg := &config.FullConfig{
			Username:      "testuser",
			Hostname:      "test-host",
			SSHPublicKeys: []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest test@example.com"},
			FullName:      "Test User",
			Email:         "test@example.com",
		}

		gen := NewGenerator(tmpDir)
		err = gen.Generate(cfg, templatePath, outputPath)
		require.NoError(t, err)

		// Read output
		output, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		content := string(output)
		assert.Contains(t, content, "name: testuser")
		assert.Contains(t, content, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest")
		assert.Contains(t, content, "hostname: test-host")
		assert.Contains(t, content, `USER_NAME="Test User"`)
		assert.Contains(t, content, `USER_EMAIL="test@example.com"`)
		assert.NotContains(t, content, "${USERNAME}")
		assert.NotContains(t, content, "${HOSTNAME}")
	})

	t.Run("handles missing optional values", func(t *testing.T) {
		tmpDir := t.TempDir()

		templateContent := `
TAILSCALE_AUTH_KEY="${TAILSCALE_AUTH_KEY}"
GITHUB_USER="${GITHUB_USER}"
`
		templatePath := filepath.Join(tmpDir, "template.yaml")
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		outputPath := filepath.Join(tmpDir, "output.yaml")

		cfg := &config.FullConfig{
			Username:      "user",
			Hostname:      "host",
			SSHPublicKeys: []string{"ssh-ed25519 test"},
		}

		gen := NewGenerator(tmpDir)
		err = gen.Generate(cfg, templatePath, outputPath)
		require.NoError(t, err)

		output, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		content := string(output)
		// Empty values should result in empty strings
		assert.Contains(t, content, `TAILSCALE_AUTH_KEY=""`)
		assert.Contains(t, content, `GITHUB_USER=""`)
	})

	t.Run("sets default repo values", func(t *testing.T) {
		tmpDir := t.TempDir()

		templateContent := `REPO_URL="${REPO_URL}"
REPO_BRANCH="${REPO_BRANCH}"`
		templatePath := filepath.Join(tmpDir, "template.yaml")
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		outputPath := filepath.Join(tmpDir, "output.yaml")

		cfg := &config.FullConfig{
			Username:      "user",
			Hostname:      "host",
			SSHPublicKeys: []string{"ssh-ed25519 test"},
			// No REPO_URL or REPO_BRANCH set
		}

		gen := NewGenerator(tmpDir)
		err = gen.Generate(cfg, templatePath, outputPath)
		require.NoError(t, err)

		output, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		content := string(output)
		assert.Contains(t, content, "REPO_BRANCH=\"main\"")
		assert.Contains(t, content, "https://github.com/jaspreet-dot-casa/cloud-init.git")
	})
}

func TestSubstituteVars(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     *TemplateVars
		expected string
	}{
		{
			name:     "replaces ${VAR} format",
			template: "Hello ${USERNAME}!",
			vars:     &TemplateVars{USERNAME: "world"},
			expected: "Hello world!",
		},
		{
			name:     "replaces $VAR format",
			template: "Hello $USERNAME!",
			vars:     &TemplateVars{USERNAME: "world"},
			expected: "Hello world!",
		},
		{
			name:     "handles multiple occurrences",
			template: "${USERNAME} and ${USERNAME} again",
			vars:     &TemplateVars{USERNAME: "test"},
			expected: "test and test again",
		},
		{
			name:     "handles multiple variables",
			template: "${USERNAME}@${HOSTNAME}",
			vars:     &TemplateVars{USERNAME: "user", HOSTNAME: "host"},
			expected: "user@host",
		},
		{
			name:     "preserves unmatched patterns",
			template: "Hello ${UNKNOWN}!",
			vars:     &TemplateVars{USERNAME: "test"},
			expected: "Hello ${UNKNOWN}!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteVars(tt.template, tt.vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigToVars(t *testing.T) {
	t.Run("converts config to vars", func(t *testing.T) {
		cfg := &config.FullConfig{
			Username:         "testuser",
			Hostname:         "testhost",
			SSHPublicKeys:    []string{"ssh-ed25519 key1", "ssh-ed25519 key2"},
			FullName:         "Test User",
			Email:            "test@example.com",
			MachineName:      "My Machine",
			TailscaleAuthKey: "tskey-test",
			GithubUser:       "github-user",
			GithubPAT:        "ghp-test",
			RepoURL:          "https://github.com/test/repo.git",
			RepoBranch:       "develop",
		}

		vars := configToVars(cfg)

		assert.Equal(t, "testuser", vars.USERNAME)
		assert.Equal(t, "testhost", vars.HOSTNAME)
		assert.Equal(t, "ssh-ed25519 key1", vars.SSH_PUBLIC_KEY)
		assert.Contains(t, vars.SSH_PUBLIC_KEYS, "- ssh-ed25519 key1")
		assert.Contains(t, vars.SSH_PUBLIC_KEYS, "- ssh-ed25519 key2")
		assert.Equal(t, "Test User", vars.USER_NAME)
		assert.Equal(t, "test@example.com", vars.USER_EMAIL)
		assert.Equal(t, "My Machine", vars.MACHINE_USER_NAME)
		assert.Equal(t, "tskey-test", vars.TAILSCALE_AUTH_KEY)
		assert.Equal(t, "github-user", vars.GITHUB_USER)
		assert.Equal(t, "ghp-test", vars.GITHUB_PAT)
		assert.Equal(t, "https://github.com/test/repo.git", vars.REPO_URL)
		assert.Equal(t, "develop", vars.REPO_BRANCH)
	})

	t.Run("uses FullName as fallback for MachineName", func(t *testing.T) {
		cfg := &config.FullConfig{
			Username:  "user",
			Hostname:  "host",
			FullName:  "Test User",
			Email:     "test@example.com",
			// MachineName not set
		}

		vars := configToVars(cfg)
		assert.Equal(t, "Test User", vars.MACHINE_USER_NAME)
	})

	t.Run("sets default repo values", func(t *testing.T) {
		cfg := &config.FullConfig{
			Username: "user",
			Hostname: "host",
			// REPO_URL and REPO_BRANCH not set
		}

		vars := configToVars(cfg)
		assert.Equal(t, "main", vars.REPO_BRANCH)
		assert.Contains(t, vars.REPO_URL, "github.com/jaspreet-dot-casa/cloud-init")
	})
}

func TestValidateTemplate(t *testing.T) {
	t.Run("valid template", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "template.yaml")
		err := os.WriteFile(path, []byte("content"), 0644)
		require.NoError(t, err)

		err = ValidateTemplate(path)
		assert.NoError(t, err)
	})

	t.Run("missing template", func(t *testing.T) {
		err := ValidateTemplate("/nonexistent/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("directory instead of file", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := ValidateTemplate(tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory")
	})
}
