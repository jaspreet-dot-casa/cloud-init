package tui

import (
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
		{"too long", "a" + string(make([]byte, 64)), true},
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

	assert.Len(t, options, 3)

	// Options should be created for each package
	// Note: We can't easily test the option values without accessing internal fields
	// but we can verify the count
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
			Username:     "testuser",
			Hostname:     "testhost",
			SSHPublicKey: "ssh-ed25519 AAAA...",
			FullName:     "Test User",
			Email:        "test@example.com",
		},
		SelectedPackages: []string{"lazygit", "starship"},
		Optional: OptionalConfig{
			GithubUser: "testgh",
		},
		OutputMode: OutputCloudInit,
	}

	assert.Equal(t, "testuser", result.User.Username)
	assert.Equal(t, "testhost", result.User.Hostname)
	assert.Len(t, result.SelectedPackages, 2)
	assert.Equal(t, "testgh", result.Optional.GithubUser)
	assert.Equal(t, OutputCloudInit, result.OutputMode)
}

func TestTheme(t *testing.T) {
	theme := Theme()
	assert.NotNil(t, theme)
}
