// Package settings provides the config manager view for the TUI application.
package settings

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/utils"
)

// Message types for async operations.
type (
	settingsLoadedMsg struct {
		settings *settings.Settings
	}

	settingsErrorMsg struct {
		err error
	}

	settingsSavedMsg struct{}
)

// Section represents a section of the config view.
type Section int

const (
	SectionSavedConfigs Section = iota
	SectionPackagePresets
	SectionAppSettings
	sectionCount
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	activeSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

// Model is the config manager view model.
type Model struct {
	app.BaseTab

	store    *settings.Store
	settings *settings.Settings

	spinner     spinner.Model
	loading     bool
	err         error
	message     string
	section     Section
	itemCursors map[Section]int // Cursor per section

	// Dialog state
	showDialog    bool
	dialogType    string // "new_preset", "edit_preset", "edit_setting"
	dialogInputs  []textinput.Model
	dialogCursor  int
	editingField  string
	editingPreset string // ID of preset being edited
}

// New creates a new config manager model.
func New() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	store, err := settings.NewStore()
	var message string
	if err != nil {
		message = fmt.Sprintf("Warning: Could not create settings store: %v", err)
	}

	return &Model{
		BaseTab:     app.NewBaseTab(app.TabConfig, "Config", "3"),
		store:       store,
		spinner:     s,
		loading:     true,
		message:     message,
		section:     SectionSavedConfigs,
		itemCursors: make(map[Section]int),
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadSettings,
	)
}

// loadSettings loads settings from disk.
func (m *Model) loadSettings() tea.Msg {
	if m.store == nil {
		return settingsErrorMsg{err: fmt.Errorf("settings store not initialized")}
	}

	s, err := m.store.Load()
	if err != nil {
		return settingsErrorMsg{err: err}
	}

	return settingsLoadedMsg{settings: s}
}

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showDialog {
			return m.handleDialogKey(msg)
		}
		return m.handleKeyMsg(msg)

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case settingsLoadedMsg:
		m.loading = false
		m.settings = msg.settings
		m.err = nil
		return m, nil

	case settingsErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case settingsSavedMsg:
		m.message = "Settings saved"
		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles key input for normal navigation.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "left", "h", "[":
		m.prevSection()
	case "right", "l", "]":
		m.nextSection()
	case "enter":
		return m.handleEnter()
	case "n":
		return m.handleNew()
	case "e":
		return m.handleEdit()
	case "x", "delete", "backspace":
		return m.handleDelete()
	case "r":
		m.loading = true
		return m, m.loadSettings
	}
	return m, nil
}

// handleDialogKey handles key input when a dialog is open.
func (m *Model) handleDialogKey(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showDialog = false
		m.dialogInputs = nil
		return m, nil
	case "tab", "down":
		if m.dialogCursor < len(m.dialogInputs)-1 {
			m.dialogInputs[m.dialogCursor].Blur()
			m.dialogCursor++
			m.dialogInputs[m.dialogCursor].Focus()
		}
		return m, nil
	case "shift+tab", "up":
		if m.dialogCursor > 0 {
			m.dialogInputs[m.dialogCursor].Blur()
			m.dialogCursor--
			m.dialogInputs[m.dialogCursor].Focus()
		}
		return m, nil
	case "enter":
		return m.confirmDialog()
	default:
		// Forward to text input
		if m.dialogCursor < len(m.dialogInputs) {
			var cmd tea.Cmd
			m.dialogInputs[m.dialogCursor], cmd = m.dialogInputs[m.dialogCursor].Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// moveCursor moves the cursor within the current section.
func (m *Model) moveCursor(delta int) {
	count := m.getItemCount()
	if count == 0 {
		return
	}

	cursor := m.itemCursors[m.section]
	newPos := cursor + delta
	if newPos < 0 {
		newPos = 0
	}
	if newPos >= count {
		newPos = count - 1
	}
	m.itemCursors[m.section] = newPos
}

// nextSection moves to the next section.
func (m *Model) nextSection() {
	m.section = (m.section + 1) % sectionCount
}

// prevSection moves to the previous section.
func (m *Model) prevSection() {
	m.section = (m.section - 1 + sectionCount) % sectionCount
}

// getItemCount returns the number of items in the current section.
func (m *Model) getItemCount() int {
	if m.settings == nil {
		return 0
	}
	switch m.section {
	case SectionSavedConfigs:
		return len(m.settings.VMConfigs) + 1 // +1 for "Create new..."
	case SectionPackagePresets:
		return len(m.getAllPresets()) + 1 // +1 for "Create new..."
	case SectionAppSettings:
		return 3 // TerraformDir, DefaultTarget, AutoApprove
	}
	return 0
}

// handleEnter handles the Enter key for the current selection.
func (m *Model) handleEnter() (app.Tab, tea.Cmd) {
	if m.settings == nil {
		return m, nil
	}

	cursor := m.itemCursors[m.section]

	switch m.section {
	case SectionSavedConfigs:
		if cursor == len(m.settings.VMConfigs) {
			// "Create new..." selected
			return m.openNewConfigDialog()
		}
		// Load config - will be implemented in wizard integration
		m.message = "Loading config... (wizard integration pending)"

	case SectionPackagePresets:
		allPresets := m.getAllPresets()
		if cursor == len(allPresets) {
			// "Create new..." selected
			return m.openNewPresetDialog()
		}
		// View preset details
		preset := allPresets[cursor]
		if len(preset.Packages) > 0 {
			pkgList := strings.Join(preset.Packages, ", ")
			if len(pkgList) > 60 {
				pkgList = pkgList[:57] + "..."
			}
			m.message = fmt.Sprintf("Preset '%s': %s", preset.Name, pkgList)
		} else {
			m.message = fmt.Sprintf("Preset '%s': (no packages)", preset.Name)
		}

	case SectionAppSettings:
		return m.editAppSetting(cursor)
	}

	return m, nil
}

// handleNew handles the 'n' key to create new items.
func (m *Model) handleNew() (app.Tab, tea.Cmd) {
	switch m.section {
	case SectionSavedConfigs:
		return m.openNewConfigDialog()
	case SectionPackagePresets:
		return m.openNewPresetDialog()
	}
	return m, nil
}

// handleEdit handles the 'e' key to edit existing items.
func (m *Model) handleEdit() (app.Tab, tea.Cmd) {
	if m.settings == nil {
		return m, nil
	}

	cursor := m.itemCursors[m.section]

	switch m.section {
	case SectionPackagePresets:
		allPresets := m.getAllPresets()
		if cursor < len(allPresets) {
			preset := allPresets[cursor]
			// Prevent editing built-in presets
			if preset.IsBuiltIn {
				m.message = "Cannot edit built-in preset"
				return m, nil
			}
			return m.openEditPresetDialog(&preset)
		}

	case SectionAppSettings:
		return m.editAppSetting(cursor)
	}

	return m, nil
}

// handleDelete handles deletion of the current item.
func (m *Model) handleDelete() (app.Tab, tea.Cmd) {
	if m.settings == nil {
		return m, nil
	}

	cursor := m.itemCursors[m.section]

	switch m.section {
	case SectionSavedConfigs:
		if cursor < len(m.settings.VMConfigs) {
			cfg := m.settings.VMConfigs[cursor]
			m.settings.RemoveVMConfig(cfg.ID)
			if err := m.store.Save(m.settings); err != nil {
				m.err = err
			} else {
				m.message = fmt.Sprintf("Deleted config '%s'", cfg.Name)
			}
			// Adjust cursor if needed
			if m.itemCursors[m.section] >= len(m.settings.VMConfigs) && m.itemCursors[m.section] > 0 {
				m.itemCursors[m.section]--
			}
		}

	case SectionPackagePresets:
		allPresets := m.getAllPresets()
		if cursor < len(allPresets) {
			preset := allPresets[cursor]
			// Prevent deleting built-in presets
			if preset.IsBuiltIn {
				m.message = "Cannot delete built-in preset"
				return m, nil
			}
			m.settings.RemovePackagePreset(preset.ID)
			if err := m.store.Save(m.settings); err != nil {
				m.err = err
			} else {
				m.message = fmt.Sprintf("Deleted preset '%s'", preset.Name)
			}
			// Adjust cursor
			newCount := len(m.getAllPresets())
			if m.itemCursors[m.section] >= newCount && m.itemCursors[m.section] > 0 {
				m.itemCursors[m.section]--
			}
		}
	}

	return m, nil
}

// openNewConfigDialog shows a message directing users to the Create tab.
// Configs should be created via the wizard, not manually, to ensure they have valid data.
func (m *Model) openNewConfigDialog() (app.Tab, tea.Cmd) {
	m.message = "Use the Create tab to build a new VM configuration, then save it from the Review screen"
	return m, nil
}

// openNewPresetDialog opens the dialog to create a new preset.
func (m *Model) openNewPresetDialog() (app.Tab, tea.Cmd) {
	m.showDialog = true
	m.dialogType = "new_preset"
	m.dialogCursor = 0
	m.editingPreset = ""

	nameInput := textinput.New()
	nameInput.Placeholder = "Preset name"
	nameInput.Focus()

	descInput := textinput.New()
	descInput.Placeholder = "Description (optional)"

	packagesInput := textinput.New()
	packagesInput.Placeholder = "Packages (comma-separated)"

	m.dialogInputs = []textinput.Model{nameInput, descInput, packagesInput}
	return m, nil
}

// openEditPresetDialog opens the dialog to edit an existing preset.
func (m *Model) openEditPresetDialog(preset *settings.PackagePreset) (app.Tab, tea.Cmd) {
	m.showDialog = true
	m.dialogType = "edit_preset"
	m.dialogCursor = 0
	m.editingPreset = preset.ID

	nameInput := textinput.New()
	nameInput.Placeholder = "Preset name"
	nameInput.SetValue(preset.Name)
	nameInput.Focus()

	descInput := textinput.New()
	descInput.Placeholder = "Description (optional)"
	descInput.SetValue(preset.Description)

	packagesInput := textinput.New()
	packagesInput.Placeholder = "Packages (comma-separated)"
	packagesInput.SetValue(strings.Join(preset.Packages, ", "))

	m.dialogInputs = []textinput.Model{nameInput, descInput, packagesInput}
	return m, nil
}

// editAppSetting opens a dialog to edit an app setting.
func (m *Model) editAppSetting(index int) (app.Tab, tea.Cmd) {
	m.showDialog = true
	m.dialogType = "edit_setting"
	m.dialogCursor = 0

	input := textinput.New()
	input.Focus()

	switch index {
	case 0: // TerraformDir
		m.editingField = "terraform_dir"
		input.Placeholder = "Terraform directory"
		input.SetValue(m.settings.AppSettings.TerraformDir)
	case 1: // DefaultTarget
		m.editingField = "default_target"
		input.Placeholder = "Default target (terraform/multipass)"
		input.SetValue(m.settings.AppSettings.DefaultTarget)
	case 2: // AutoApprove
		m.editingField = "auto_approve"
		if m.settings.AppSettings.AutoApprove {
			input.SetValue("true")
		} else {
			input.SetValue("false")
		}
		input.Placeholder = "Auto approve (true/false)"
	}

	m.dialogInputs = []textinput.Model{input}
	return m, nil
}

// confirmDialog saves the dialog input.
func (m *Model) confirmDialog() (app.Tab, tea.Cmd) {
	switch m.dialogType {
	case "new_preset":
		name := utils.SanitizeConfigName(m.dialogInputs[0].Value())

		// Validate preset name
		if err := utils.ValidateConfigName(name); err != nil {
			m.message = err.Error()
			return m, nil
		}

		desc := m.dialogInputs[1].Value()
		packagesStr := m.dialogInputs[2].Value()

		var packages []string
		if packagesStr != "" {
			for _, p := range strings.Split(packagesStr, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					packages = append(packages, p)
				}
			}
		}

		preset := settings.PackagePreset{
			ID:          fmt.Sprintf("preset-%d", time.Now().UnixNano()),
			Name:        name,
			Description: desc,
			Packages:    packages,
			CreatedAt:   time.Now(),
		}
		m.settings.AddPackagePreset(preset)
		if err := m.store.Save(m.settings); err != nil {
			m.err = err
		} else {
			m.message = fmt.Sprintf("Created preset '%s'", name)
		}

	case "edit_preset":
		name := utils.SanitizeConfigName(m.dialogInputs[0].Value())

		// Validate preset name
		if err := utils.ValidateConfigName(name); err != nil {
			m.message = err.Error()
			return m, nil
		}

		desc := m.dialogInputs[1].Value()
		packagesStr := m.dialogInputs[2].Value()

		var packages []string
		if packagesStr != "" {
			for _, p := range strings.Split(packagesStr, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					packages = append(packages, p)
				}
			}
		}

		// Find and update the preset
		for i, p := range m.settings.PackagePresets {
			if p.ID == m.editingPreset {
				m.settings.PackagePresets[i].Name = name
				m.settings.PackagePresets[i].Description = desc
				m.settings.PackagePresets[i].Packages = packages
				break
			}
		}

		if err := m.store.Save(m.settings); err != nil {
			m.err = err
		} else {
			m.message = fmt.Sprintf("Updated preset '%s'", name)
		}
		m.editingPreset = ""

	case "edit_setting":
		value := m.dialogInputs[0].Value()
		switch m.editingField {
		case "terraform_dir":
			m.settings.AppSettings.TerraformDir = value
		case "default_target":
			// Validate target
			if value != "" && !settings.IsValidTarget(value) {
				m.message = "Invalid target. Use 'terraform', 'multipass', or 'config'"
				return m, nil
			}
			m.settings.AppSettings.DefaultTarget = value
		case "auto_approve":
			m.settings.AppSettings.AutoApprove = value == "true" || value == "yes" || value == "1"
		}
		if err := m.store.Save(m.settings); err != nil {
			m.err = err
		} else {
			m.message = "Setting updated"
		}
	}

	m.showDialog = false
	m.dialogInputs = nil
	return m, nil
}

// View renders the view.
func (m *Model) View() string {
	if m.Width() == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Configuration"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", min(40, m.Width()-4)))
	b.WriteString("\n\n")

	// Content
	if m.loading && m.settings == nil {
		b.WriteString(fmt.Sprintf("  %s Loading settings...\n", m.spinner.View()))
	} else if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v\n\n", m.err)))
		b.WriteString(dimStyle.Render("  Press 'r' to retry.\n"))
	} else {
		b.WriteString(m.renderSections())
	}

	// Message
	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s", m.message)))
		b.WriteString("\n")
	}

	// Dialog overlay
	if m.showDialog {
		b.WriteString("\n")
		b.WriteString(m.renderDialog())
	}

	return b.String()
}

// renderSections renders all sections.
func (m *Model) renderSections() string {
	var b strings.Builder

	// Section 1: Saved Configs
	b.WriteString(m.renderSectionHeader("Saved Configs", SectionSavedConfigs))
	b.WriteString(m.renderSavedConfigs())
	b.WriteString("\n")

	// Section 2: Package Presets
	b.WriteString(m.renderSectionHeader("Package Presets", SectionPackagePresets))
	b.WriteString(m.renderPackagePresets())
	b.WriteString("\n")

	// Section 3: App Settings
	b.WriteString(m.renderSectionHeader("App Settings", SectionAppSettings))
	b.WriteString(m.renderAppSettings())

	return b.String()
}

// renderSectionHeader renders a section header.
func (m *Model) renderSectionHeader(title string, section Section) string {
	style := sectionStyle
	prefix := "  "
	if m.section == section {
		style = activeSectionStyle
		prefix = "▸ "
	}
	return prefix + style.Render(title) + "\n  " + strings.Repeat("─", min(30, m.Width()-6)) + "\n"
}

// renderSavedConfigs renders the saved configs section.
func (m *Model) renderSavedConfigs() string {
	var b strings.Builder

	if m.settings == nil {
		return ""
	}

	cursor := m.itemCursors[SectionSavedConfigs]
	isActive := m.section == SectionSavedConfigs

	for i, cfg := range m.settings.VMConfigs {
		prefix := "    "
		style := itemStyle
		if isActive && i == cursor {
			prefix = "  ▸ "
			style = selectedItemStyle
		}

		// Format last used
		lastUsed := "Never used"
		if !cfg.LastUsedAt.IsZero() {
			lastUsed = "Last used: " + utils.FormatTimeAgo(cfg.LastUsedAt)
		}

		b.WriteString(prefix)
		b.WriteString(style.Render(cfg.Name))
		if cfg.Target != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf(" (%s)", cfg.Target)))
		}
		b.WriteString("\n")
		b.WriteString("      ")
		b.WriteString(dimStyle.Render(lastUsed))
		b.WriteString("\n")
	}

	// "Create new..." option
	prefix := "    "
	style := dimStyle
	if isActive && cursor == len(m.settings.VMConfigs) {
		prefix = "  ▸ "
		style = selectedItemStyle
	}
	b.WriteString(prefix)
	b.WriteString(style.Render("+ Create new..."))
	b.WriteString("\n")

	return b.String()
}

// getAllPresets returns built-in presets merged with user presets.
func (m *Model) getAllPresets() []settings.PackagePreset {
	builtIn := settings.DefaultPackagePresets()
	if m.settings == nil {
		return builtIn
	}

	// Build a set of built-in IDs to avoid duplicates
	builtInIDs := make(map[string]bool)
	for _, p := range builtIn {
		builtInIDs[p.ID] = true
	}

	// Append user presets (skip any that duplicate built-in IDs)
	result := builtIn
	for _, p := range m.settings.PackagePresets {
		if !builtInIDs[p.ID] {
			result = append(result, p)
		}
	}
	return result
}

// renderPackagePresets renders the package presets section.
func (m *Model) renderPackagePresets() string {
	var b strings.Builder

	allPresets := m.getAllPresets()
	cursor := m.itemCursors[SectionPackagePresets]
	isActive := m.section == SectionPackagePresets

	for i, preset := range allPresets {
		prefix := "    "
		style := itemStyle
		if isActive && i == cursor {
			prefix = "  ▸ "
			style = selectedItemStyle
		}

		b.WriteString(prefix)
		b.WriteString(style.Render(preset.Name))
		b.WriteString(dimStyle.Render(fmt.Sprintf(" (%d packages)", len(preset.Packages))))

		// Show [built-in] indicator
		if preset.IsBuiltIn {
			b.WriteString(dimStyle.Render(" [built-in]"))
		}
		b.WriteString("\n")

		if preset.Description != "" {
			b.WriteString("      ")
			b.WriteString(dimStyle.Render(preset.Description))
			b.WriteString("\n")
		}
	}

	// "Create new..." option
	prefix := "    "
	style := dimStyle
	if isActive && cursor == len(allPresets) {
		prefix = "  ▸ "
		style = selectedItemStyle
	}
	b.WriteString(prefix)
	b.WriteString(style.Render("+ Create new..."))
	b.WriteString("\n")

	return b.String()
}

// renderAppSettings renders the app settings section.
func (m *Model) renderAppSettings() string {
	var b strings.Builder

	if m.settings == nil {
		return ""
	}

	cursor := m.itemCursors[SectionAppSettings]
	isActive := m.section == SectionAppSettings

	settings := []struct {
		label string
		value string
	}{
		{"Terraform Dir", m.settings.AppSettings.TerraformDir},
		{"Default Target", m.settings.AppSettings.DefaultTarget},
		{"Auto Approve", fmt.Sprintf("%v", m.settings.AppSettings.AutoApprove)},
	}

	for i, s := range settings {
		prefix := "    "
		style := labelStyle
		if isActive && i == cursor {
			prefix = "  ▸ "
			style = selectedItemStyle
		}

		value := s.value
		if value == "" {
			value = "(not set)"
		}

		b.WriteString(prefix)
		b.WriteString(style.Render(s.label + ": "))
		b.WriteString(valueStyle.Render(value))
		b.WriteString("\n")
	}

	return b.String()
}

// renderDialog renders the dialog overlay.
func (m *Model) renderDialog() string {
	var b strings.Builder

	b.WriteString("  ┌")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("┐\n")

	var title string
	switch m.dialogType {
	case "new_preset":
		title = "New Package Preset"
	case "edit_preset":
		title = "Edit Package Preset"
	case "edit_setting":
		title = "Edit Setting"
	}

	b.WriteString(fmt.Sprintf("  │ %-38s │\n", title))
	b.WriteString("  ├")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("┤\n")

	for i, input := range m.dialogInputs {
		prefix := "  "
		if i == m.dialogCursor {
			prefix = "▸ "
		}
		b.WriteString(fmt.Sprintf("  │ %s%-36s │\n", prefix, input.View()))
	}

	b.WriteString("  ├")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("┤\n")
	b.WriteString("  │ [Enter] Save  [Esc] Cancel           │\n")
	b.WriteString("  └")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("┘\n")

	return b.String()
}

// Focus is called when the tab becomes active.
func (m *Model) Focus() tea.Cmd {
	m.BaseTab.Focus()
	return tea.Batch(
		m.spinner.Tick,
		m.loadSettings,
	)
}

// Blur is called when the tab becomes inactive.
func (m *Model) Blur() {
	m.BaseTab.Blur()
	m.showDialog = false
	m.dialogInputs = nil
}

// KeyBindings returns the key bindings for the footer.
func (m *Model) KeyBindings() []string {
	if m.showDialog {
		return []string{
			"[Enter] save",
			"[Esc] cancel",
			"[Tab] next field",
		}
	}
	return []string{
		"[↑/↓] navigate",
		"[h/l] section",
		"[Enter] select",
		"[n] new",
		"[e] edit",
		"[x] delete",
		"[r] refresh",
	}
}

// HasFocusedInput returns true if dialog is open.
func (m *Model) HasFocusedInput() bool {
	return m.showDialog
}


func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
