package create

import (
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// handlePackagesPhase handles input for the Packages selection phase
func (m *Model) handlePackagesPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	pkgs := m.getSortedPackages()
	if len(pkgs) == 0 {
		// No packages, just advance
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			m.savePackagesOptions()
			m.wizard.Advance()
			m.initPhase(m.wizard.Phase)
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.wizard.FocusedField > 0 {
			m.wizard.FocusedField--
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.wizard.FocusedField < len(pkgs)-1 {
			m.wizard.FocusedField++
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
		// Toggle package selection
		if m.wizard.FocusedField < len(pkgs) {
			pkgName := pkgs[m.wizard.FocusedField]
			m.wizard.PackageSelected[pkgName] = !m.wizard.PackageSelected[pkgName]
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		// Select all
		for _, pkg := range pkgs {
			m.wizard.PackageSelected[pkg] = true
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
		// Select none
		for _, pkg := range pkgs {
			m.wizard.PackageSelected[pkg] = false
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		m.savePackagesOptions()
		m.wizard.Advance()
		m.initPhase(m.wizard.Phase)
		return m, nil
	}

	return m, nil
}

// getSortedPackages returns a sorted list of package names
func (m *Model) getSortedPackages() []string {
	if m.wizard.Registry == nil {
		return nil
	}

	names := m.wizard.Registry.Names()
	sort.Strings(names)
	return names
}

// savePackagesOptions saves the selected packages to wizard data
func (m *Model) savePackagesOptions() {
	var selected []string
	for pkg, isSelected := range m.wizard.PackageSelected {
		if isSelected {
			selected = append(selected, pkg)
		}
	}
	sort.Strings(selected)
	m.wizard.Data.Packages = selected
}

// viewPackagesPhase renders the Packages selection phase
func (m *Model) viewPackagesPhase() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Package Selection"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Select packages to install. All are selected by default."))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("[Space] toggle  [a] all  [n] none"))
	b.WriteString("\n\n")

	pkgs := m.getSortedPackages()
	if len(pkgs) == 0 {
		b.WriteString(dimStyle.Render("No packages found."))
		b.WriteString("\n")
		return b.String()
	}

	// Show packages with checkboxes
	for i, pkgName := range pkgs {
		focused := m.wizard.FocusedField == i
		selected := m.wizard.PackageSelected[pkgName]

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
		if pkg := m.wizard.Registry.Get(pkgName); pkg != nil {
			desc = pkg.Description
		}

		b.WriteString(cursor)
		if focused {
			b.WriteString(focusedInputStyle.Render(checkbox + " " + pkgName))
		} else if selected {
			b.WriteString(selectedStyle.Render(checkbox + " " + pkgName))
		} else {
			b.WriteString(labelStyle.Render(checkbox + " " + pkgName))
		}

		if desc != "" {
			b.WriteString(dimStyle.Render(" - " + desc))
		}
		b.WriteString("\n")
	}

	// Summary
	selectedCount := 0
	for _, selected := range m.wizard.PackageSelected {
		if selected {
			selectedCount++
		}
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("-", 40)))
	b.WriteString("\n")
	b.WriteString(valueStyle.Render(strconv.Itoa(selectedCount) + "/" + strconv.Itoa(len(pkgs)) + " packages selected"))
	b.WriteString("\n")

	return b.String()
}
