package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSummary(t *testing.T) {
	t.Run("generates valid markdown file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "summary.md")

		cfg := &config.FullConfig{
			Username:        "testuser",
			Hostname:        "testhost",
			FullName:        "Test User",
			Email:           "test@example.com",
			EnabledPackages: []string{"bat", "ripgrep"},
		}
		registry := createTestRegistry()

		err := GenerateSummary(cfg, registry, outputPath)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		// Verify structure
		assert.Contains(t, string(content), "# Configuration Summary")
		assert.Contains(t, string(content), "## User Configuration")
		assert.Contains(t, string(content), "## Packages")
		assert.Contains(t, string(content), "## Services")
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "nested", "deep", "summary.md")

		cfg := &config.FullConfig{Username: "test"}
		registry := createTestRegistry()

		err := GenerateSummary(cfg, registry, outputPath)
		require.NoError(t, err)

		_, err = os.Stat(outputPath)
		require.NoError(t, err)
	})

	t.Run("renders enabled packages with checked boxes", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "summary.md")

		cfg := &config.FullConfig{
			Username:        "testuser",
			EnabledPackages: []string{"bat"},
		}
		registry := createTestRegistry()

		err := GenerateSummary(cfg, registry, outputPath)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		// bat should be checked, fzf should be unchecked
		assert.Contains(t, string(content), "- [x] **bat**")
		assert.Contains(t, string(content), "- [ ] **fzf**")
	})

	t.Run("renders services based on config", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "summary.md")

		cfg := &config.FullConfig{
			Username:         "testuser",
			EnabledPackages:  []string{packages.PackageDocker},
			TailscaleAuthKey: "tskey-xxx",
		}
		registry := createTestRegistry()

		err := GenerateSummary(cfg, registry, outputPath)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		// Docker and Tailscale should be checked
		assert.Contains(t, string(content), "- [x] **Docker**")
		assert.Contains(t, string(content), "- [x] **Tailscale**")
	})

	t.Run("escapes pipe characters in table values", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "summary.md")

		cfg := &config.FullConfig{
			Username: "user|with|pipes",
			FullName: "John | Doe",
		}
		registry := createTestRegistry()

		err := GenerateSummary(cfg, registry, outputPath)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		// Pipes should be escaped in the output
		assert.Contains(t, string(content), "user\\|with\\|pipes")
		assert.Contains(t, string(content), "John \\| Doe")
	})

	t.Run("escapes markdown formatting in package names", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "summary.md")

		registry := packages.NewRegistry()
		registry.Add(packages.Package{
			Name:        "evil",
			DisplayName: "**BOLD** and _italic_",
			Description: "A [link](http://evil.com) here",
			Category:    packages.CategoryCLI,
		})

		cfg := &config.FullConfig{}

		err := GenerateSummary(cfg, registry, outputPath)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		// Markdown characters should be escaped
		assert.Contains(t, string(content), "\\*\\*BOLD\\*\\*")
		assert.Contains(t, string(content), "\\_italic\\_")
		assert.Contains(t, string(content), "\\[link\\]")
	})

	t.Run("handles empty registry", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "summary.md")

		cfg := &config.FullConfig{Username: "test"}
		registry := packages.NewRegistry() // Empty registry

		err := GenerateSummary(cfg, registry, outputPath)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		// Should still have structure but no packages
		assert.Contains(t, string(content), "## Packages")
		assert.Contains(t, string(content), "## Services")
	})

	t.Run("handles newlines in user input", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "summary.md")

		cfg := &config.FullConfig{
			Username: "user\nname",
			FullName: "John\r\nDoe",
		}
		registry := createTestRegistry()

		err := GenerateSummary(cfg, registry, outputPath)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		// Newlines should be replaced with spaces
		assert.Contains(t, string(content), "user name")
		assert.Contains(t, string(content), "John Doe")
	})
}

func TestEscapeMarkdownTable(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escapes pipe characters",
			input:    "foo|bar|baz",
			expected: "foo\\|bar\\|baz",
		},
		{
			name:     "removes newlines",
			input:    "line1\nline2\r\nline3",
			expected: "line1 line2 line3",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no change for safe string",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeMarkdownTable(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeMarkdownText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escapes asterisks",
			input:    "**bold**",
			expected: "\\*\\*bold\\*\\*",
		},
		{
			name:     "escapes underscores",
			input:    "_italic_",
			expected: "\\_italic\\_",
		},
		{
			name:     "escapes brackets",
			input:    "[link](url)",
			expected: "\\[link\\](url)",
		},
		{
			name:     "escapes backslashes",
			input:    "path\\to\\file",
			expected: "path\\\\to\\\\file",
		},
		{
			name:     "escapes backticks",
			input:    "`code` here",
			expected: "\\`code\\` here",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeMarkdownText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildSummaryData(t *testing.T) {
	t.Run("marks enabled packages correctly", func(t *testing.T) {
		cfg := &config.FullConfig{
			Username:        "user",
			EnabledPackages: []string{"bat", "ripgrep"},
		}
		registry := createTestRegistry()

		data := buildSummaryData(cfg, registry)

		// Find bat package - should be enabled
		var batFound, fzfFound bool
		for _, cat := range data.Categories {
			for _, pkg := range cat.Packages {
				if pkg.Name == "bat" {
					batFound = true
					assert.True(t, pkg.Enabled, "bat should be enabled")
				}
				if pkg.Name == "fzf" {
					fzfFound = true
					assert.False(t, pkg.Enabled, "fzf should be disabled")
				}
			}
		}
		assert.True(t, batFound, "bat package should be in data")
		assert.True(t, fzfFound, "fzf package should be in data")
	})

	t.Run("handles empty config fields gracefully", func(t *testing.T) {
		cfg := &config.FullConfig{
			// All fields empty/zero
		}
		registry := createTestRegistry()

		data := buildSummaryData(cfg, registry)

		// Should not panic and should have empty strings
		assert.Equal(t, "", data.Username)
		assert.Equal(t, "", data.Hostname)
		assert.Equal(t, "", data.FullName)
		assert.Equal(t, "", data.Email)
		// Services should be false when nothing enabled
		assert.False(t, data.Tailscale)
		assert.False(t, data.Docker)
	})

	t.Run("handles all packages disabled", func(t *testing.T) {
		cfg := &config.FullConfig{
			Username:        "user",
			EnabledPackages: []string{}, // No packages enabled
		}
		registry := createTestRegistry()

		data := buildSummaryData(cfg, registry)

		// All packages should be disabled
		for _, cat := range data.Categories {
			for _, pkg := range cat.Packages {
				assert.False(t, pkg.Enabled, "package %s should be disabled", pkg.Name)
			}
		}
	})

	t.Run("services false when not enabled", func(t *testing.T) {
		cfg := &config.FullConfig{
			Username:        "user",
			EnabledPackages: []string{"bat", "ripgrep"}, // No docker or tailscale
		}
		registry := createTestRegistry()

		data := buildSummaryData(cfg, registry)

		assert.False(t, data.Tailscale, "tailscale should be disabled")
		assert.False(t, data.Docker, "docker should be disabled")
	})

	t.Run("uses package constants for detection", func(t *testing.T) {
		// Test that detection works using the constant values
		cfg := &config.FullConfig{
			EnabledPackages: []string{packages.PackageDocker, packages.PackageTailscale},
		}
		registry := createTestRegistry()

		data := buildSummaryData(cfg, registry)

		assert.True(t, data.Docker, "docker should be detected via constant")
		assert.True(t, data.Tailscale, "tailscale should be detected via constant")
	})

	t.Run("skips empty categories", func(t *testing.T) {
		cfg := &config.FullConfig{Username: "user"}
		registry := packages.NewRegistry()
		// Add only CLI packages
		registry.Add(packages.Package{
			Name:        "bat",
			DisplayName: "bat",
			Category:    packages.CategoryCLI,
		})

		data := buildSummaryData(cfg, registry)

		// Should only have CLI category
		assert.Len(t, data.Categories, 1)
		assert.Equal(t, "CLI Tools", data.Categories[0].Name)
	})

	t.Run("detects tailscale from enabled packages", func(t *testing.T) {
		cfg := &config.FullConfig{
			EnabledPackages: []string{"tailscale"},
		}
		registry := createTestRegistry()

		data := buildSummaryData(cfg, registry)
		assert.True(t, data.Tailscale)
	})

	t.Run("detects tailscale from auth key", func(t *testing.T) {
		cfg := &config.FullConfig{
			TailscaleAuthKey: "tskey-xxx",
		}
		registry := createTestRegistry()

		data := buildSummaryData(cfg, registry)
		assert.True(t, data.Tailscale)
	})

	t.Run("detects docker from enabled packages", func(t *testing.T) {
		cfg := &config.FullConfig{
			EnabledPackages: []string{"docker"},
		}
		registry := createTestRegistry()

		data := buildSummaryData(cfg, registry)
		assert.True(t, data.Docker)
	})

	t.Run("escapes user input", func(t *testing.T) {
		cfg := &config.FullConfig{
			Username: "user|name",
			FullName: "John | Doe",
			Email:    "test@example.com",
		}
		registry := createTestRegistry()

		data := buildSummaryData(cfg, registry)

		assert.Equal(t, "user\\|name", data.Username)
		assert.Equal(t, "John \\| Doe", data.FullName)
	})

	t.Run("escapes package descriptions", func(t *testing.T) {
		registry := packages.NewRegistry()
		registry.Add(packages.Package{
			Name:        "test",
			DisplayName: "Test *Tool*",
			Description: "A [test] _tool_",
			Category:    packages.CategoryCLI,
		})

		cfg := &config.FullConfig{}
		data := buildSummaryData(cfg, registry)

		assert.Len(t, data.Categories, 1)
		pkg := data.Categories[0].Packages[0]
		assert.Equal(t, "Test \\*Tool\\*", pkg.DisplayName)
		assert.Contains(t, pkg.Description, "\\[test\\]")
	})
}

// createTestRegistry creates a minimal registry for testing
func createTestRegistry() *packages.Registry {
	registry := packages.NewRegistry()

	registry.Add(packages.Package{
		Name:        "bat",
		DisplayName: "bat",
		Description: "Cat with syntax highlighting",
		Category:    packages.CategoryCLI,
	})
	registry.Add(packages.Package{
		Name:        "ripgrep",
		DisplayName: "ripgrep",
		Description: "Fast grep alternative",
		Category:    packages.CategoryCLI,
	})
	registry.Add(packages.Package{
		Name:        "fzf",
		DisplayName: "fzf",
		Description: "Fuzzy finder",
		Category:    packages.CategoryCLI,
	})
	registry.Add(packages.Package{
		Name:        "starship",
		DisplayName: "Starship",
		Description: "Shell prompt",
		Category:    packages.CategoryShell,
	})
	registry.Add(packages.Package{
		Name:        "docker",
		DisplayName: "Docker",
		Description: "Container runtime",
		Category:    packages.CategoryDocker,
	})
	registry.Add(packages.Package{
		Name:        "tailscale",
		DisplayName: "Tailscale",
		Description: "VPN mesh network",
		Category:    packages.CategoryCLI,
	})

	return registry
}
