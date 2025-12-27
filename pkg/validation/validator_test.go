package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/envfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSecretsEnv(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedErrors int
		errorFields    []string
	}{
		{
			name: "valid secrets file",
			content: `
USERNAME="testuser"
HOSTNAME="test-host"
SSH_PUBLIC_KEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest test@example.com"
USER_NAME="Test User"
USER_EMAIL="test@example.com"
`,
			expectedErrors: 0,
		},
		{
			name: "missing required fields",
			content: `
USERNAME="testuser"
`,
			expectedErrors: 4, // HOSTNAME, SSH_PUBLIC_KEY, USER_NAME, USER_EMAIL
			errorFields:    []string{"HOSTNAME", "SSH_PUBLIC_KEY", "USER_NAME", "USER_EMAIL"},
		},
		{
			name: "invalid hostname",
			content: `
USERNAME="testuser"
HOSTNAME="-invalid-hostname-"
SSH_PUBLIC_KEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest test@example.com"
USER_NAME="Test User"
USER_EMAIL="test@example.com"
`,
			expectedErrors: 1,
			errorFields:    []string{"HOSTNAME"},
		},
		{
			name: "invalid SSH key format",
			content: `
USERNAME="testuser"
HOSTNAME="validhost"
SSH_PUBLIC_KEY="not-a-valid-ssh-key"
USER_NAME="Test User"
USER_EMAIL="test@example.com"
`,
			expectedErrors: 1,
			errorFields:    []string{"SSH_PUBLIC_KEY"},
		},
		{
			name: "invalid email format",
			content: `
USERNAME="testuser"
HOSTNAME="validhost"
SSH_PUBLIC_KEY="ssh-rsa AAAAB3NzaC1yc2EAAAATest test@example.com"
USER_NAME="Test User"
USER_EMAIL="not-an-email"
`,
			expectedErrors: 1,
			errorFields:    []string{"USER_EMAIL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and file
			tmpDir := t.TempDir()
			secretsPath := filepath.Join(tmpDir, "secrets.env")
			err := os.WriteFile(secretsPath, []byte(tt.content), 0600)
			require.NoError(t, err)

			validator := NewValidator(tmpDir)
			issues := validator.ValidateSecretsEnv(secretsPath)

			// Count errors only
			errorCount := 0
			var errorFields []string
			for _, issue := range issues {
				if issue.Severity == SeverityError {
					errorCount++
					if issue.Field != "" {
						errorFields = append(errorFields, issue.Field)
					}
				}
			}

			assert.Equal(t, tt.expectedErrors, errorCount, "unexpected error count")
			if tt.errorFields != nil {
				for _, expectedField := range tt.errorFields {
					assert.Contains(t, errorFields, expectedField, "expected error for field %s", expectedField)
				}
			}
		})
	}
}

func TestValidateConfigEnv(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedErrors int
		errorFields    []string
	}{
		{
			name: "valid config file",
			content: `
USER_NAME="Test User"
USER_EMAIL="test@example.com"
GIT_DEFAULT_BRANCH="main"
GIT_PUSH_AUTO_SETUP_REMOTE=true
GIT_PULL_REBASE=false
DOCKER_ENABLED=true
PACKAGE_LAZYGIT_ENABLED=true
`,
			expectedErrors: 0,
		},
		{
			name: "invalid boolean values",
			content: `
GIT_PUSH_AUTO_SETUP_REMOTE=yes
DOCKER_ENABLED=1
`,
			expectedErrors: 2,
			errorFields:    []string{"GIT_PUSH_AUTO_SETUP_REMOTE", "DOCKER_ENABLED"},
		},
		{
			name: "invalid package enabled value",
			content: `
PACKAGE_LAZYGIT_ENABLED=yes
PACKAGE_DOCKER_ENABLED=true
`,
			expectedErrors: 1,
			errorFields:    []string{"PACKAGE_LAZYGIT_ENABLED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.env")
			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			require.NoError(t, err)

			validator := NewValidator(tmpDir)
			issues := validator.ValidateConfigEnv(configPath)

			// Count errors only
			errorCount := 0
			var errorFields []string
			for _, issue := range issues {
				if issue.Severity == SeverityError {
					errorCount++
					if issue.Field != "" {
						errorFields = append(errorFields, issue.Field)
					}
				}
			}

			assert.Equal(t, tt.expectedErrors, errorCount, "unexpected error count")
			if tt.errorFields != nil {
				for _, expectedField := range tt.errorFields {
					assert.Contains(t, errorFields, expectedField, "expected error for field %s", expectedField)
				}
			}
		})
	}
}

func TestValidateAll(t *testing.T) {
	t.Run("missing files", func(t *testing.T) {
		tmpDir := t.TempDir()
		validator := NewValidator(tmpDir)
		result := validator.ValidateAll()

		assert.True(t, result.HasErrors())
		assert.Equal(t, 2, result.ErrorCount()) // Both files missing
	})

	t.Run("valid files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create cloud-init directory
		cloudInitDir := filepath.Join(tmpDir, "cloud-init")
		err := os.MkdirAll(cloudInitDir, 0755)
		require.NoError(t, err)

		// Create valid secrets.env
		secretsContent := `
USERNAME="testuser"
HOSTNAME="test-host"
SSH_PUBLIC_KEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest test@example.com"
USER_NAME="Test User"
USER_EMAIL="test@example.com"
`
		err = os.WriteFile(filepath.Join(cloudInitDir, "secrets.env"), []byte(secretsContent), 0600)
		require.NoError(t, err)

		// Create valid config.env
		configContent := `
USER_NAME="Test User"
GIT_PUSH_AUTO_SETUP_REMOTE=true
DOCKER_ENABLED=true
`
		err = os.WriteFile(filepath.Join(tmpDir, "config.env"), []byte(configContent), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir)
		result := validator.ValidateAll()

		assert.False(t, result.HasErrors())
		assert.Equal(t, 0, result.ErrorCount())
	})
}

func TestParseEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "simple key-value pairs",
			content: `
KEY1=value1
KEY2=value2
`,
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name: "quoted values",
			content: `
KEY1="value with spaces"
KEY2='single quoted'
`,
			expected: map[string]string{"KEY1": "value with spaces", "KEY2": "single quoted"},
		},
		{
			name: "skip comments and empty lines",
			content: `
# This is a comment
KEY1=value1

# Another comment
KEY2=value2
`,
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name: "values with equals sign",
			content: `
KEY1=value=with=equals
`,
			expected: map[string]string{"KEY1": "value=with=equals"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "env-test-*.env")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.content)
			require.NoError(t, err)
			tmpFile.Close()

			result, err := envfile.Parse(tmpFile.Name())
			require.NoError(t, err)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		hostname string
		valid    bool
	}{
		{"valid-host", true},
		{"host123", true},
		{"a", true},
		{"abc", true},
		{"-invalid", false},
		{"invalid-", false},
		{"has space", false},
		{"has.dot", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			err := validateHostname(tt.hostname)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateSSHKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest test@example.com", true},
		{"ssh-rsa AAAAB3NzaC1yc2EAAAATest test@example.com", true},
		{"ecdsa-sha2-nistp256 AAAATest test@example.com", true},
		{"invalid-key", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.key[:min(20, len(tt.key))], func(t *testing.T) {
			err := validateSSHKey(tt.key)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name@domain.org", true},
		{"user+tag@example.co.uk", true},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			err := validateEmail(tt.email)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
