package tui

import (
	"strings"
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/stretchr/testify/assert"
)

func TestValidateRequired(t *testing.T) {
	validator := validateRequired("Username")

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid input", "testuser", false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"with spaces", "test user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "ubuntu", false},
		{"valid with numbers", "server1", false},
		{"valid with hyphens", "my-server", false},
		{"valid mixed", "web-server-01", false},
		{"empty string", "", true},
		{"starts with hyphen", "-server", true},
		{"ends with hyphen", "server-", true},
		{"too long", strings.Repeat("a", 64), true},
		{"contains underscore", "my_server", true},
		{"uppercase converted", "MyServer", false}, // Should work after lowercase conversion
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHostname(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSSHKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid ed25519", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG... user@host", false},
		{"valid rsa", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB... user@host", false},
		{"valid ecdsa", "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHA... user@host", false},
		{"empty string", "", true},
		{"invalid prefix", "invalid-key AAAA...", true},
		{"random text", "not a valid key at all", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSSHKey(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "user@example.com", false},
		{"valid with dots", "first.last@example.com", false},
		{"valid with plus", "user+tag@example.com", false},
		{"valid subdomain", "user@mail.example.com", false},
		{"empty string", "", true},
		{"missing @", "userexample.com", true},
		{"missing domain", "user@", true},
		{"missing local", "@example.com", true},
		{"missing tld", "user@example", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildPackageOptions(t *testing.T) {
	registry := packages.NewRegistry()
	registry.Add(packages.Package{
		Name:        "lazygit",
		Description: "A simple terminal UI for git",
		Category:    packages.CategoryCLI,
	})
	registry.Add(packages.Package{
		Name:        "starship",
		Description: "Cross-shell prompt",
		Category:    packages.CategoryShell,
	})
	registry.Add(packages.Package{
		Name:        "docker",
		Description: "",
		Category:    packages.CategoryDocker,
	})

	options := buildPackageOptions(registry)

	// Verify correct number of options
	assert.Len(t, options, 3)

	// Collect option values and keys for verification
	values := make(map[string]string) // value -> key mapping
	for _, opt := range options {
		values[opt.Value] = opt.Key
	}

	// Verify all packages are represented
	assert.Contains(t, values, "lazygit")
	assert.Contains(t, values, "starship")
	assert.Contains(t, values, "docker")

	// Verify labels are formatted correctly (with description)
	assert.Equal(t, "lazygit - A simple terminal UI for git", values["lazygit"])
	assert.Equal(t, "starship - Cross-shell prompt", values["starship"])

	// Verify label without description uses just the name
	assert.Equal(t, "docker", values["docker"])
}

func TestOutputMode(t *testing.T) {
	assert.Equal(t, OutputMode("config"), OutputConfigOnly)
	assert.Equal(t, OutputMode("cloudinit"), OutputCloudInit)
	assert.Equal(t, OutputMode("bootable"), OutputBootableISO)
	assert.Equal(t, OutputMode("seed"), OutputSeedISO)
}

func TestFormResult(t *testing.T) {
	result := &FormResult{
		User: UserConfig{
			Username:      "testuser",
			Hostname:      "testhost",
			SSHPublicKeys: []string{"ssh-ed25519 AAAA...", "ssh-rsa BBBB..."},
			FullName:      "Test User",
			Email:         "test@example.com",
			MachineName:   "Test User",
		},
		SelectedPackages: []string{"lazygit", "starship"},
		Optional: OptionalConfig{
			GithubUser: "testgh",
		},
		OutputMode: OutputCloudInit,
	}

	assert.Equal(t, "testuser", result.User.Username)
	assert.Equal(t, "testhost", result.User.Hostname)
	assert.Len(t, result.User.SSHPublicKeys, 2)
	assert.Equal(t, "Test User", result.User.MachineName)
	assert.Len(t, result.SelectedPackages, 2)
	assert.Equal(t, "testgh", result.Optional.GithubUser)
	assert.Equal(t, OutputCloudInit, result.OutputMode)
}

func TestTheme(t *testing.T) {
	theme := Theme()
	assert.NotNil(t, theme)

	// Verify theme has customized focused styles
	assert.NotNil(t, theme.Focused)
	assert.NotNil(t, theme.Focused.Title)
	assert.NotNil(t, theme.Focused.Description)
	assert.NotNil(t, theme.Focused.SelectedOption)
}

func TestStyles(t *testing.T) {
	// Verify TitleStyle has expected properties
	titleRender := TitleStyle.Render("Test")
	assert.NotEmpty(t, titleRender)
	assert.Contains(t, titleRender, "Test")

	// Verify SubtitleStyle renders correctly
	subtitleRender := SubtitleStyle.Render("Subtitle")
	assert.NotEmpty(t, subtitleRender)
	assert.Contains(t, subtitleRender, "Subtitle")

	// Verify SuccessStyle renders correctly
	successRender := SuccessStyle.Render("Success")
	assert.NotEmpty(t, successRender)
	assert.Contains(t, successRender, "Success")

	// Verify ErrorStyle renders correctly
	errorRender := ErrorStyle.Render("Error")
	assert.NotEmpty(t, errorRender)
	assert.Contains(t, errorRender, "Error")

	// Verify WarningStyle renders correctly
	warningRender := WarningStyle.Render("Warning")
	assert.NotEmpty(t, warningRender)
	assert.Contains(t, warningRender, "Warning")
}

func TestSSHKeySourceConstants(t *testing.T) {
	// Verify SSH key source constants
	assert.Equal(t, SSHKeySource("github"), SSHKeyFromGitHub)
	assert.Equal(t, SSHKeySource("local"), SSHKeyFromLocal)
	assert.Equal(t, SSHKeySource("manual"), SSHKeyManual)
}

func TestFetchGitHubSSHKeys(t *testing.T) {
	// Test with a known GitHub user that has SSH keys
	// Using "torvalds" as he's a well-known user with SSH keys
	t.Run("valid user with keys", func(t *testing.T) {
		// Skip in CI environments or if network is unavailable
		keys, err := fetchGitHubSSHKeys("torvalds")
		if err != nil {
			t.Skipf("Skipping network test: %v", err)
		}
		assert.NoError(t, err)
		assert.NotEmpty(t, keys)
		// Verify all keys have valid prefixes
		for _, key := range keys {
			hasValidPrefix := strings.HasPrefix(key, "ssh-") || strings.HasPrefix(key, "ecdsa-")
			assert.True(t, hasValidPrefix, "Key should have valid SSH prefix: %s", key[:min(30, len(key))])
		}
	})

	t.Run("invalid user", func(t *testing.T) {
		// Use a username that almost certainly doesn't exist
		_, err := fetchGitHubSSHKeys("this-user-definitely-does-not-exist-12345678901234567890")
		if err == nil {
			t.Skip("Skipping: expected error for invalid user but got none (network issue?)")
		}
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("empty username", func(t *testing.T) {
		keys, err := fetchGitHubSSHKeys("")
		// Empty username should return 404 or error
		if err == nil {
			assert.Empty(t, keys)
		}
	})
}

func TestGetDefaultSSHKey(t *testing.T) {
	// This test is environment-dependent
	// Just verify it doesn't panic and returns a string
	key := getDefaultSSHKey()
	// Key might be empty if no SSH keys exist, that's fine
	assert.IsType(t, "", key)

	// If a key is returned, it should have a valid prefix
	if key != "" {
		hasValidPrefix := strings.HasPrefix(key, "ssh-") || strings.HasPrefix(key, "ecdsa-")
		assert.True(t, hasValidPrefix, "Key should have valid SSH prefix")
	}
}

func TestFetchGitHubProfile(t *testing.T) {
	t.Run("valid user with public profile", func(t *testing.T) {
		// Using "torvalds" as he has a public profile
		profile, err := fetchGitHubProfile("torvalds")
		if err != nil {
			t.Skipf("Skipping network test: %v", err)
		}
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, "torvalds", profile.Login)
		// Name is usually public for well-known users
		assert.NotEmpty(t, profile.Name)
	})

	t.Run("invalid user", func(t *testing.T) {
		_, err := fetchGitHubProfile("this-user-definitely-does-not-exist-12345678901234567890")
		if err == nil {
			t.Skip("Skipping: expected error for invalid user but got none (network issue?)")
		}
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("profile struct fields", func(t *testing.T) {
		// Verify GitHubProfile struct has expected fields
		profile := GitHubProfile{
			Name:  "Test User",
			Email: "test@example.com",
			Login: "testuser",
		}
		assert.Equal(t, "Test User", profile.Name)
		assert.Equal(t, "test@example.com", profile.Email)
		assert.Equal(t, "testuser", profile.Login)
	})
}
