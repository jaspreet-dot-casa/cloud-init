package packages

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScript(t *testing.T) {
	// Create a temporary test script
	tmpDir := t.TempDir()
	scriptContent := `#!/bin/bash
#==============================================================================
# testpkg Installer
#
# A test package for unit testing
# https://github.com/test/testpkg
#
# Usage: ./testpkg.sh [install|update|verify|version]
#==============================================================================

set -e

PACKAGE_NAME="testpkg"
GITHUB_REPO="test/testpkg"
`
	scriptPath := filepath.Join(tmpDir, "testpkg.sh")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	require.NoError(t, err)

	pkg, err := ParseScript(scriptPath)
	require.NoError(t, err)
	require.NotNil(t, pkg)

	assert.Equal(t, "testpkg", pkg.Name)
	assert.Equal(t, "testpkg", pkg.DisplayName)
	assert.Equal(t, "test/testpkg", pkg.GithubRepo)
	assert.Equal(t, scriptPath, pkg.ScriptPath)
	assert.True(t, pkg.Default)
}

func TestParseScript_WithDescription(t *testing.T) {
	tmpDir := t.TempDir()
	scriptContent := `#!/bin/bash
#==============================================================================
# lazygit Installer
#
# A simple terminal UI for git commands
# https://github.com/jesseduffield/lazygit
#==============================================================================

PACKAGE_NAME="lazygit"
GITHUB_REPO="jesseduffield/lazygit"
`
	scriptPath := filepath.Join(tmpDir, "lazygit.sh")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	require.NoError(t, err)

	pkg, err := ParseScript(scriptPath)
	require.NoError(t, err)
	require.NotNil(t, pkg)

	assert.Equal(t, "lazygit", pkg.Name)
	assert.Equal(t, "lazygit", pkg.DisplayName)
	assert.Equal(t, "jesseduffield/lazygit", pkg.GithubRepo)
	assert.Equal(t, CategoryCLI, pkg.Category)
}

func TestParseScript_NoPackageName(t *testing.T) {
	tmpDir := t.TempDir()
	scriptContent := `#!/bin/bash
# Some random script without PACKAGE_NAME
echo "hello"
`
	scriptPath := filepath.Join(tmpDir, "random.sh")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	require.NoError(t, err)

	pkg, err := ParseScript(scriptPath)
	require.NoError(t, err)
	assert.Nil(t, pkg, "Should return nil for scripts without PACKAGE_NAME")
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	pkg1 := Package{
		Name:        "lazygit",
		DisplayName: "lazygit",
		Category:    CategoryCLI,
	}
	pkg2 := Package{
		Name:        "starship",
		DisplayName: "Starship",
		Category:    CategoryShell,
	}

	registry.Add(pkg1)
	registry.Add(pkg2)

	assert.Len(t, registry.Packages, 2)
	assert.Contains(t, registry.Names(), "lazygit")
	assert.Contains(t, registry.Names(), "starship")

	got := registry.Get("lazygit")
	require.NotNil(t, got)
	assert.Equal(t, "lazygit", got.Name)

	assert.Nil(t, registry.Get("nonexistent"))
}

func TestRegistry_Categories(t *testing.T) {
	registry := NewRegistry()

	registry.Add(Package{Name: "lazygit", Category: CategoryCLI})
	registry.Add(Package{Name: "btop", Category: CategoryCLI})
	registry.Add(Package{Name: "starship", Category: CategoryShell})
	registry.Add(Package{Name: "docker", Category: CategoryDocker})

	categories := registry.Categories()
	assert.Len(t, categories, 3)
	// Should be in defined order
	assert.Equal(t, CategoryCLI, categories[0])
	assert.Equal(t, CategoryShell, categories[1])
	assert.Equal(t, CategoryDocker, categories[2])

	assert.Len(t, registry.ByCategory[CategoryCLI], 2)
	assert.Len(t, registry.ByCategory[CategoryShell], 1)
	assert.Len(t, registry.ByCategory[CategoryDocker], 1)
}

func TestCategorize(t *testing.T) {
	tests := []struct {
		name     string
		pkg      *Package
		expected Category
	}{
		{
			name:     "lazygit is CLI",
			pkg:      &Package{Name: "lazygit"},
			expected: CategoryCLI,
		},
		{
			name:     "lazydocker is Docker",
			pkg:      &Package{Name: "lazydocker"},
			expected: CategoryDocker,
		},
		{
			name:     "docker is Docker",
			pkg:      &Package{Name: "docker"},
			expected: CategoryDocker,
		},
		{
			name:     "starship is Shell",
			pkg:      &Package{Name: "starship"},
			expected: CategoryShell,
		},
		{
			name:     "zoxide is Shell",
			pkg:      &Package{Name: "zoxide"},
			expected: CategoryShell,
		},
		{
			name:     "zellij is Shell",
			pkg:      &Package{Name: "zellij"},
			expected: CategoryShell,
		},
		{
			name:     "github-cli is Git",
			pkg:      &Package{Name: "github-cli"},
			expected: CategoryGit,
		},
		{
			name:     "delta is Git",
			pkg:      &Package{Name: "delta"},
			expected: CategoryGit,
		},
		{
			name:     "apt is System",
			pkg:      &Package{Name: "apt"},
			expected: CategorySystem,
		},
		{
			name:     "btop is CLI (default)",
			pkg:      &Package{Name: "btop"},
			expected: CategoryCLI,
		},
		{
			name:     "yq is CLI (default)",
			pkg:      &Package{Name: "yq"},
			expected: CategoryCLI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorize(tt.pkg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDiscover(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	packagesDir := filepath.Join(tmpDir, "packages")
	err := os.MkdirAll(packagesDir, 0755)
	require.NoError(t, err)

	// Create test scripts
	scripts := map[string]string{
		"lazygit.sh": `#!/bin/bash
# lazygit Installer
# A terminal UI for git
PACKAGE_NAME="lazygit"
GITHUB_REPO="jesseduffield/lazygit"
`,
		"starship.sh": `#!/bin/bash
# starship Installer
# Cross-shell prompt
PACKAGE_NAME="starship"
GITHUB_REPO="starship/starship"
`,
		"_template.sh": `#!/bin/bash
# Template file - should be skipped
PACKAGE_NAME="template"
`,
		"not-a-script.txt": `This is not a shell script`,
	}

	for name, content := range scripts {
		path := filepath.Join(packagesDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	registry, err := Discover(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, registry)

	// Should find lazygit and starship, skip template and txt file
	assert.Len(t, registry.Packages, 2)
	assert.NotNil(t, registry.Get("lazygit"))
	assert.NotNil(t, registry.Get("starship"))
	assert.Nil(t, registry.Get("template"))
}

func TestDiscoverFromProjectRoot(t *testing.T) {
	// Get the actual project root (go up from pkg/packages to root)
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate to project root (../../ from pkg/packages)
	projectRoot := filepath.Join(cwd, "..", "..")

	// Check if scripts/packages exists
	packagesDir := filepath.Join(projectRoot, "scripts", "packages")
	if _, err := os.Stat(packagesDir); os.IsNotExist(err) {
		t.Skip("scripts/packages not found, skipping integration test")
	}

	registry, err := DiscoverFromProjectRoot(projectRoot)
	require.NoError(t, err)
	require.NotNil(t, registry)

	// Should find at least some packages from the real project
	assert.Greater(t, len(registry.Packages), 0, "Should discover at least one package")

	// Check for some expected packages (just verify they exist)
	expectedPackages := []string{"lazygit", "starship", "docker", "btop"}
	for _, name := range expectedPackages {
		if pkg := registry.Get(name); pkg != nil {
			assert.Equal(t, name, pkg.Name)
			assert.NotEmpty(t, pkg.ScriptPath)
		}
	}
}
