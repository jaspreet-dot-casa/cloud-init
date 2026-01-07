package phases

import (
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create/wizard"
)

// Ensure PackagesPhase implements PhaseHandler
var _ wizard.PhaseHandler = (*PackagesPhase)(nil)

// PackagesPhase handles the package selection step.
type PackagesPhase struct {
	wizard.BasePhase
}

// NewPackagesPhase creates a new PackagesPhase.
func NewPackagesPhase() *PackagesPhase {
	return &PackagesPhase{
		BasePhase: wizard.NewBasePhase("Packages", 0), // Dynamic field count
	}
}

// Init initializes the packages phase state.
func (p *PackagesPhase) Init(ctx *wizard.PhaseContext) {
	ctx.Wizard.FocusedField = 0

	// Ensure PackageSelected map is initialized
	if ctx.Wizard.PackageSelected == nil {
		ctx.Wizard.PackageSelected = make(map[string]bool)
	}

	// Initialize package selection to all selected by default
	if ctx.Wizard.Registry != nil && len(ctx.Wizard.PackageSelected) == 0 {
		for _, name := range ctx.Wizard.Registry.Names() {
			ctx.Wizard.PackageSelected[name] = true
		}
	}
}

// getSortedPackages returns a sorted list of package names.
func (p *PackagesPhase) getSortedPackages(ctx *wizard.PhaseContext) []string {
	if ctx.Wizard.Registry == nil {
		return nil
	}

	names := ctx.Wizard.Registry.Names()
	sort.Strings(names)
	return names
}

// FieldCount returns the number of packages.
func (p *PackagesPhase) FieldCount() int {
	// Dynamic count based on packages - return 0 as a sentinel value
	// The actual count depends on the wizard state
	return 0
}

// Update handles keyboard input for the packages phase.
func (p *PackagesPhase) Update(ctx *wizard.PhaseContext, msg tea.KeyMsg) (advance bool, cmd tea.Cmd) {
	// Ensure PackageSelected map is initialized before any writes
	if ctx.Wizard.PackageSelected == nil {
		ctx.Wizard.PackageSelected = make(map[string]bool)
	}

	pkgs := p.getSortedPackages(ctx)
	if len(pkgs) == 0 {
		// No packages, just advance on enter
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			return true, nil
		}
		return false, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if ctx.Wizard.FocusedField > 0 {
			ctx.Wizard.FocusedField--
		}
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if ctx.Wizard.FocusedField < len(pkgs)-1 {
			ctx.Wizard.FocusedField++
		}
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
		// Toggle package selection
		if ctx.Wizard.FocusedField < len(pkgs) {
			pkgName := pkgs[ctx.Wizard.FocusedField]
			ctx.Wizard.PackageSelected[pkgName] = !ctx.Wizard.PackageSelected[pkgName]
		}
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		// Select all
		for _, pkg := range pkgs {
			ctx.Wizard.PackageSelected[pkg] = true
		}
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
		// Select none
		for _, pkg := range pkgs {
			ctx.Wizard.PackageSelected[pkg] = false
		}
		return false, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return true, nil
	}

	return false, nil
}

// View renders the packages phase.
func (p *PackagesPhase) View(ctx *wizard.PhaseContext) string {
	var b strings.Builder

	b.WriteString(wizard.TitleStyle.Render("Package Selection"))
	b.WriteString("\n\n")

	b.WriteString(wizard.DimStyle.Render("Select packages to install. All are selected by default."))
	b.WriteString("\n")
	b.WriteString(wizard.DimStyle.Render("[Space] toggle  [a] all  [n] none"))
	b.WriteString("\n\n")

	pkgs := p.getSortedPackages(ctx)
	if len(pkgs) == 0 {
		b.WriteString(wizard.DimStyle.Render("No packages found."))
		b.WriteString("\n")
		return b.String()
	}

	// Show packages with checkboxes
	for i, pkgName := range pkgs {
		focused := ctx.Wizard.FocusedField == i
		selected := ctx.Wizard.PackageSelected[pkgName]

		cursor := "  "
		if focused {
			cursor = "▸ "
		}

		checkbox := "[ ]"
		if selected {
			checkbox = "[✓]"
		}

		// Get package description
		desc := ""
		if pkg := ctx.Wizard.Registry.Get(pkgName); pkg != nil {
			desc = pkg.Description
		}

		b.WriteString(cursor)
		if focused {
			b.WriteString(wizard.FocusedInputStyle.Render(checkbox + " " + pkgName))
		} else if selected {
			b.WriteString(wizard.SelectedStyle.Render(checkbox + " " + pkgName))
		} else {
			b.WriteString(wizard.LabelStyle.Render(checkbox + " " + pkgName))
		}

		if desc != "" {
			b.WriteString(wizard.DimStyle.Render(" - " + desc))
		}
		b.WriteString("\n")
	}

	// Summary
	selectedCount := 0
	for _, selected := range ctx.Wizard.PackageSelected {
		if selected {
			selectedCount++
		}
	}
	b.WriteString("\n")
	b.WriteString(wizard.DimStyle.Render(strings.Repeat("-", 40)))
	b.WriteString("\n")
	b.WriteString(wizard.ValueStyle.Render(strconv.Itoa(selectedCount) + "/" + strconv.Itoa(len(pkgs)) + " packages selected"))
	b.WriteString("\n")

	return b.String()
}

// Save persists the selected packages to wizard data.
func (p *PackagesPhase) Save(ctx *wizard.PhaseContext) {
	var selected []string
	for pkg, isSelected := range ctx.Wizard.PackageSelected {
		if isSelected {
			selected = append(selected, pkg)
		}
	}
	sort.Strings(selected)
	ctx.Wizard.Data.Packages = selected
}
