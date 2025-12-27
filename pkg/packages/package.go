// Package packages provides functionality for discovering and managing
// installable packages from shell scripts.
package packages

// Category represents a grouping of related packages.
type Category string

const (
	CategoryCLI    Category = "CLI Tools"
	CategoryShell  Category = "Shell & Terminal"
	CategoryGit    Category = "Git & Version Control"
	CategoryDocker Category = "Docker & Containers"
	CategorySystem Category = "System"
)

// Package represents a discoverable package from scripts/packages/.
type Package struct {
	// Name is the package identifier (e.g., "lazygit")
	Name string

	// DisplayName is a human-readable name
	DisplayName string

	// Description is a brief description of the package
	Description string

	// ScriptPath is the path to the installer script
	ScriptPath string

	// GithubRepo is the GitHub repository if applicable (e.g., "jesseduffield/lazygit")
	GithubRepo string

	// Category is the package category for grouping in TUI
	Category Category

	// Default indicates whether the package is enabled by default
	Default bool
}

// Registry holds all discovered packages.
// Note: Registry is not thread-safe and should not be modified concurrently.
type Registry struct {
	// Packages is an ordered list of all discovered packages
	Packages []Package

	// ByName provides quick lookup by package name (stores copies, not pointers)
	ByName map[string]Package

	// ByCategory groups packages by their category
	ByCategory map[Category][]Package
}

// NewRegistry creates an empty package registry.
func NewRegistry() *Registry {
	return &Registry{
		Packages:   make([]Package, 0, 16), // Preallocate for typical package count
		ByName:     make(map[string]Package),
		ByCategory: make(map[Category][]Package),
	}
}

// Add adds a package to the registry.
func (r *Registry) Add(pkg Package) {
	r.Packages = append(r.Packages, pkg)
	r.ByName[pkg.Name] = pkg // Store copy, not pointer

	if _, ok := r.ByCategory[pkg.Category]; !ok {
		r.ByCategory[pkg.Category] = make([]Package, 0)
	}
	r.ByCategory[pkg.Category] = append(r.ByCategory[pkg.Category], pkg)
}

// Get returns a package by name, or nil if not found.
func (r *Registry) Get(name string) *Package {
	if pkg, ok := r.ByName[name]; ok {
		return &pkg
	}
	return nil
}

// Names returns a list of all package names.
func (r *Registry) Names() []string {
	names := make([]string, len(r.Packages))
	for i, pkg := range r.Packages {
		names[i] = pkg.Name
	}
	return names
}

// Categories returns all categories that have packages.
func (r *Registry) Categories() []Category {
	// Return in a consistent order
	order := []Category{CategoryCLI, CategoryShell, CategoryGit, CategoryDocker, CategorySystem}
	result := make([]Category, 0)
	for _, cat := range order {
		if pkgs, ok := r.ByCategory[cat]; ok && len(pkgs) > 0 {
			result = append(result, cat)
		}
	}
	return result
}
