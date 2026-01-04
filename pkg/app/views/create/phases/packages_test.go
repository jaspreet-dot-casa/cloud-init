package phases

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app/views/create"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/stretchr/testify/assert"
)

// createTestContextWithPackages creates a test context with a real package registry
func createTestContextWithPackages() *create.PhaseContext {
	ctx := newTestContext()

	// Create a real registry with test packages
	registry := &packages.Registry{
		Packages: []packages.Package{
			{Name: "docker", Description: "Container runtime"},
			{Name: "git", Description: "Version control"},
			{Name: "neovim", Description: "Text editor"},
		},
		ByName: map[string]packages.Package{
			"docker": {Name: "docker", Description: "Container runtime"},
			"git":    {Name: "git", Description: "Version control"},
			"neovim": {Name: "neovim", Description: "Text editor"},
		},
	}
	ctx.Wizard.Registry = registry

	return ctx
}

func TestPackagesPhase_Name(t *testing.T) {
	p := NewPackagesPhase()
	assert.Equal(t, "Packages", p.Name())
}

func TestPackagesPhase_Init(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()

	p.Init(ctx)

	// All packages should be selected by default
	assert.Equal(t, 0, ctx.Wizard.FocusedField)
	assert.True(t, ctx.Wizard.PackageSelected["docker"])
	assert.True(t, ctx.Wizard.PackageSelected["git"])
	assert.True(t, ctx.Wizard.PackageSelected["neovim"])
}

func TestPackagesPhase_Init_NoPackages(t *testing.T) {
	p := NewPackagesPhase()
	ctx := newTestContext() // No registry

	p.Init(ctx)

	assert.Equal(t, 0, ctx.Wizard.FocusedField)
}

func TestPackagesPhase_Update_NavigateDown(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyDown}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)
}

func TestPackagesPhase_Update_NavigateUp(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)
	ctx.Wizard.FocusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.Equal(t, 0, ctx.Wizard.FocusedField)
}

func TestPackagesPhase_Update_VimKeys(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	// Test j for down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	advance, _ := p.Update(ctx, msg)
	assert.False(t, advance)
	assert.Equal(t, 1, ctx.Wizard.FocusedField)

	// Test k for up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	advance, _ = p.Update(ctx, msg)
	assert.False(t, advance)
	assert.Equal(t, 0, ctx.Wizard.FocusedField)
}

func TestPackagesPhase_Update_ToggleSelection(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	// Get sorted packages to know which is at index 0
	pkgs := p.getSortedPackages(ctx)
	firstPkg := pkgs[0]

	// Initially selected
	assert.True(t, ctx.Wizard.PackageSelected[firstPkg])

	// Toggle with space
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	assert.False(t, ctx.Wizard.PackageSelected[firstPkg])

	// Toggle again
	advance, _ = p.Update(ctx, msg)

	assert.False(t, advance)
	assert.True(t, ctx.Wizard.PackageSelected[firstPkg])
}

func TestPackagesPhase_Update_SelectAll(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	// Deselect all first
	for pkg := range ctx.Wizard.PackageSelected {
		ctx.Wizard.PackageSelected[pkg] = false
	}

	// Select all with 'a'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	for _, selected := range ctx.Wizard.PackageSelected {
		assert.True(t, selected)
	}
}

func TestPackagesPhase_Update_SelectNone(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	// All selected initially
	for _, selected := range ctx.Wizard.PackageSelected {
		assert.True(t, selected)
	}

	// Select none with 'n'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	advance, _ := p.Update(ctx, msg)

	assert.False(t, advance)
	for _, selected := range ctx.Wizard.PackageSelected {
		assert.False(t, selected)
	}
}

func TestPackagesPhase_Update_Enter(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	advance, _ := p.Update(ctx, msg)

	assert.True(t, advance)
}

func TestPackagesPhase_Update_EnterNoPackages(t *testing.T) {
	p := NewPackagesPhase()
	ctx := newTestContext() // No registry

	p.Init(ctx)
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	advance, _ := p.Update(ctx, msg)

	assert.True(t, advance)
}

func TestPackagesPhase_View(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	view := p.View(ctx)

	assert.Contains(t, view, "Package Selection")
	assert.Contains(t, view, "[Space] toggle")
	assert.Contains(t, view, "[a] all")
	assert.Contains(t, view, "[n] none")
	assert.Contains(t, view, "docker")
	assert.Contains(t, view, "git")
	assert.Contains(t, view, "neovim")
	assert.Contains(t, view, "packages selected")
}

func TestPackagesPhase_View_NoPackages(t *testing.T) {
	p := NewPackagesPhase()
	ctx := newTestContext() // No registry

	p.Init(ctx)
	view := p.View(ctx)

	assert.Contains(t, view, "No packages found")
}

func TestPackagesPhase_View_ShowsCheckboxes(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	view := p.View(ctx)

	// Should show checkmarks for selected packages
	assert.Contains(t, view, "[✓]")
}

func TestPackagesPhase_View_ShowsCursor(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	view := p.View(ctx)

	// Should show cursor on focused item
	assert.Contains(t, view, "▸")
}

func TestPackagesPhase_Save(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	// Deselect git
	ctx.Wizard.PackageSelected["git"] = false

	p.Save(ctx)

	assert.Len(t, ctx.Wizard.Data.Packages, 2)
	assert.Contains(t, ctx.Wizard.Data.Packages, "docker")
	assert.Contains(t, ctx.Wizard.Data.Packages, "neovim")
	assert.NotContains(t, ctx.Wizard.Data.Packages, "git")
}

func TestPackagesPhase_Save_None(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	// Deselect all
	for pkg := range ctx.Wizard.PackageSelected {
		ctx.Wizard.PackageSelected[pkg] = false
	}

	p.Save(ctx)

	assert.Empty(t, ctx.Wizard.Data.Packages)
}

func TestPackagesPhase_Save_IsSorted(t *testing.T) {
	p := NewPackagesPhase()
	ctx := createTestContextWithPackages()
	p.Init(ctx)

	p.Save(ctx)

	// Packages should be sorted alphabetically
	for i := 1; i < len(ctx.Wizard.Data.Packages); i++ {
		assert.True(t, ctx.Wizard.Data.Packages[i-1] < ctx.Wizard.Data.Packages[i],
			"packages should be sorted, but %s comes before %s",
			ctx.Wizard.Data.Packages[i-1], ctx.Wizard.Data.Packages[i])
	}
}

func TestPackagesPhase_ImplementsPhaseHandler(t *testing.T) {
	var _ create.PhaseHandler = (*PackagesPhase)(nil)
}
