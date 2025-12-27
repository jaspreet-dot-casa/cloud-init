package packages

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Pre-compiled regex patterns for parsing shell scripts
var (
	packageNameRe = regexp.MustCompile(`^PACKAGE_NAME="([^"]+)"`)
	githubRepoRe  = regexp.MustCompile(`GITHUB_REPO="([^"]+)"`)
	headerRe      = regexp.MustCompile(`^#\s*(\w+)\s+[Ii]nstaller`)
	descRe        = regexp.MustCompile(`^#\s+([A-Z].+)$`)
	separatorRe   = regexp.MustCompile(`^#[=\-]*$|^#\s*$`)
)

// Discover scans a directory for package installer scripts and returns a registry.
func Discover(scriptsDir string) (*Registry, error) {
	registry := NewRegistry()

	packagesDir := filepath.Join(scriptsDir, "packages")

	// Validate directory exists
	info, err := os.Stat(packagesDir)
	if err != nil {
		return nil, fmt.Errorf("packages directory not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("packages path is not a directory: %s", packagesDir)
	}

	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read packages directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Skip non-shell files and template
		if !strings.HasSuffix(name, ".sh") || name == "_template.sh" {
			continue
		}

		scriptPath := filepath.Join(packagesDir, name)
		pkg, err := ParseScript(scriptPath)
		if err != nil {
			// Log error but continue processing other scripts
			// In production, consider collecting these as warnings
			continue
		}

		if pkg != nil {
			registry.Add(*pkg)
		}
	}

	return registry, nil
}

// ParseScript parses a package installer script and extracts metadata.
// Returns (nil, nil) if the script doesn't contain a PACKAGE_NAME.
// Returns (nil, error) on I/O errors.
// Returns (*Package, nil) on successful parsing.
func ParseScript(path string) (*Package, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open script: %w", err)
	}
	defer file.Close()

	pkg := &Package{
		ScriptPath: path,
		Default:    true, // All packages enabled by default
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	lookingForDesc := false

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Look for PACKAGE_NAME
		if matches := packageNameRe.FindStringSubmatch(line); len(matches) > 1 {
			pkg.Name = strings.TrimSpace(matches[1])
		}

		// Look for GITHUB_REPO
		if matches := githubRepoRe.FindStringSubmatch(line); len(matches) > 1 {
			pkg.GithubRepo = strings.TrimSpace(matches[1])
		}

		// Look for header comment (e.g., "# lazygit Installer")
		if matches := headerRe.FindStringSubmatch(line); len(matches) > 1 {
			pkg.DisplayName = matches[1]
			lookingForDesc = true
			continue
		}

		// After finding header, look for description
		if lookingForDesc {
			// Skip separator and empty comment lines
			if separatorRe.MatchString(line) {
				continue
			}

			// Found description line
			if matches := descRe.FindStringSubmatch(line); len(matches) > 1 {
				desc := strings.TrimSpace(matches[1])
				// Skip URL lines
				if !strings.HasPrefix(desc, "http") {
					pkg.Description = desc
					lookingForDesc = false
				}
			} else {
				// Not a description line, stop looking
				lookingForDesc = false
			}
		}

		// Stop after finding all we need (first 50 lines should be enough)
		if lineNum > 50 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading script: %w", err)
	}

	// Skip if no package name found
	if pkg.Name == "" {
		return nil, nil
	}

	// Use name as display name if not found
	if pkg.DisplayName == "" {
		pkg.DisplayName = pkg.Name
	}

	// Determine category based on package name or other hints
	pkg.Category = categorize(pkg)

	return pkg, nil
}

// categorize determines the category for a package based on its name and other metadata.
func categorize(pkg *Package) Category {
	name := strings.ToLower(pkg.Name)

	// Docker-related
	if strings.Contains(name, "docker") || strings.Contains(name, "lazydocker") {
		return CategoryDocker
	}

	// Git-related
	if strings.Contains(name, "git") || strings.Contains(name, "delta") {
		if name == "lazygit" {
			return CategoryCLI // lazygit is a CLI tool
		}
		return CategoryGit
	}

	// Shell/terminal tools
	shellTools := []string{"starship", "zoxide", "zsh", "oh-my-zsh", "zellij"}
	for _, tool := range shellTools {
		if strings.Contains(name, tool) {
			return CategoryShell
		}
	}

	// System packages (apt-based)
	if name == "apt" {
		return CategorySystem
	}

	// Default to CLI tools
	return CategoryCLI
}

// DiscoverFromProjectRoot discovers packages from the project root directory.
// It looks for scripts/packages/ relative to the given root.
func DiscoverFromProjectRoot(projectRoot string) (*Registry, error) {
	scriptsDir := filepath.Join(projectRoot, "scripts")
	return Discover(scriptsDir)
}
