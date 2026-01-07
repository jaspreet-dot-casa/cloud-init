// Package doctor provides the doctor/dependency check view for the TUI application.
package doctor

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/doctor"
)

// Message types for async operations.
type (
	// checksLoadedMsg indicates checks have completed.
	checksLoadedMsg struct {
		groups []doctor.CheckGroup
	}

	// checksErrorMsg indicates an error during checks.
	checksErrorMsg struct {
		err error
	}

	// fixResultMsg indicates the result of a fix operation.
	fixResultMsg struct {
		checkID string
		success bool
		err     error
	}

	// clipboardResultMsg indicates the result of copying to clipboard.
	clipboardResultMsg struct {
		success bool
		err     error
	}
)

// FlatItem represents a flattened item in the list (either a group header or a check).
type FlatItem struct {
	IsGroup bool
	Group   *doctor.CheckGroup
	Check   *doctor.Check
	GroupID string
}

// Model is the doctor view model.
type Model struct {
	app.BaseTab

	checker *doctor.Checker
	fixer   *doctor.Fixer
	groups  []doctor.CheckGroup
	items   []FlatItem // Flattened list of groups and checks

	spinner  spinner.Model
	loading  bool
	err      error
	cursor   int // Current selection in items list
	message  string

	// Dialog state
	showDialog     bool
	dialogCheckID  string
	dialogFix      *doctor.FixCommand
	dialogCursor   int // 0=Cancel, 1=Copy, 2=Run
	dialogRunning  bool
	dialogMessage  string
}

// New creates a new doctor model.
func New() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &Model{
		BaseTab: app.NewBaseTab(app.TabDoctor, "Doctor", "3"),
		checker: doctor.NewChecker(),
		fixer:   doctor.NewFixer(),
		spinner: s,
		loading: true,
	}
}

// Init initializes the doctor view.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadChecks(),
	)
}

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showDialog {
			return m.handleDialogKey(msg)
		}
		return m.handleKeyMsg(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case checksLoadedMsg:
		m.loading = false
		m.groups = msg.groups
		m.err = nil
		m.flattenItems()

	case checksErrorMsg:
		m.loading = false
		m.err = msg.err

	case fixResultMsg:
		m.dialogRunning = false
		if msg.success {
			m.dialogMessage = "Fix completed successfully!"
			// Refresh checks
			cmds = append(cmds, m.loadChecks())
		} else {
			m.dialogMessage = fmt.Sprintf("Fix failed: %v", msg.err)
		}

	case clipboardResultMsg:
		if msg.success {
			m.message = "Command copied to clipboard"
			m.showDialog = false
		} else {
			m.dialogMessage = fmt.Sprintf("Copy failed: %v", msg.err)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles keyboard input for main view.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "enter":
		return m.openFixDialog()
	case "r":
		m.loading = true
		m.message = ""
		return m, tea.Batch(m.spinner.Tick, m.loadChecks())
	}
	return m, nil
}

// handleDialogKey handles keyboard input for fix dialog.
func (m *Model) handleDialogKey(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	if m.dialogRunning {
		return m, nil // Ignore input while running
	}

	switch msg.String() {
	case "left", "h":
		if m.dialogCursor > 0 {
			m.dialogCursor--
		}
	case "right", "l":
		if m.dialogCursor < 2 {
			m.dialogCursor++
		}
	case "enter":
		return m.executeDialogAction()
	case "esc", "q":
		m.showDialog = false
		m.dialogMessage = ""
	}
	return m, nil
}

// moveCursor moves the selection cursor.
func (m *Model) moveCursor(delta int) {
	newPos := m.cursor + delta

	// Skip group headers when moving
	for newPos >= 0 && newPos < len(m.items) {
		if !m.items[newPos].IsGroup {
			break
		}
		newPos += delta
	}

	if newPos >= 0 && newPos < len(m.items) {
		m.cursor = newPos
	}
}

// openFixDialog opens the fix dialog for the selected check.
func (m *Model) openFixDialog() (app.Tab, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return m, nil
	}

	item := m.items[m.cursor]
	if item.IsGroup || item.Check == nil {
		return m, nil
	}

	if item.Check.Status == doctor.StatusOK {
		m.message = "This dependency is already installed"
		return m, nil
	}

	if item.Check.FixCommand == nil {
		m.message = "No fix available for this dependency"
		return m, nil
	}

	m.showDialog = true
	m.dialogCheckID = item.Check.ID
	m.dialogFix = item.Check.FixCommand
	m.dialogCursor = 2 // Default to Run
	m.dialogMessage = ""
	m.message = ""

	return m, nil
}

// executeDialogAction executes the selected dialog action.
func (m *Model) executeDialogAction() (app.Tab, tea.Cmd) {
	switch m.dialogCursor {
	case 0: // Cancel
		m.showDialog = false
		m.dialogMessage = ""
		return m, nil
	case 1: // Copy
		return m, m.copyToClipboard()
	case 2: // Run
		m.dialogRunning = true
		m.dialogMessage = "Running fix..."
		return m, tea.Batch(m.spinner.Tick, m.runFix())
	}
	return m, nil
}

// loadChecks returns a command to load dependency checks.
func (m *Model) loadChecks() tea.Cmd {
	return func() tea.Msg {
		groups := m.checker.CheckAllAsync()
		return checksLoadedMsg{groups: groups}
	}
}

// runFix runs the fix command.
func (m *Model) runFix() tea.Cmd {
	return func() tea.Msg {
		err := m.fixer.RunFix(m.dialogFix)
		return fixResultMsg{
			checkID: m.dialogCheckID,
			success: err == nil,
			err:     err,
		}
	}
}

// copyToClipboard copies the fix command to clipboard.
func (m *Model) copyToClipboard() tea.Cmd {
	return func() tea.Msg {
		err := m.fixer.CopyToClipboard(m.dialogFix)
		return clipboardResultMsg{
			success: err == nil,
			err:     err,
		}
	}
}

// flattenItems flattens groups and checks into a single list for navigation.
func (m *Model) flattenItems() {
	m.items = nil
	for i := range m.groups {
		group := &m.groups[i]
		m.items = append(m.items, FlatItem{
			IsGroup: true,
			Group:   group,
			GroupID: group.ID,
		})
		for j := range group.Checks {
			check := &group.Checks[j]
			m.items = append(m.items, FlatItem{
				IsGroup: false,
				Check:   check,
				GroupID: group.ID,
			})
		}
	}

	// Position cursor on first check (skip first group header)
	m.cursor = 0
	for i, item := range m.items {
		if !item.IsGroup {
			m.cursor = i
			break
		}
	}
}

// View renders the doctor view.
func (m *Model) View() string {
	if m.Width() == 0 {
		return "Loading..."
	}

	var content string

	// Header
	header := m.renderHeader()

	// Main content
	if m.loading && len(m.groups) == 0 {
		content = fmt.Sprintf("\n  %s Checking dependencies...\n", m.spinner.View())
	} else if m.err != nil {
		content = fmt.Sprintf("\n  Error: %v\n\n  Press 'r' to retry.\n", m.err)
	} else if len(m.groups) == 0 {
		content = "\n  No dependency groups found.\n"
	} else {
		content = m.renderChecks()
	}

	// Status message
	statusBar := m.renderStatusBar()

	// Dialog overlay
	if m.showDialog {
		return m.renderWithDialog(header, content, statusBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}

// renderHeader renders the view header.
func (m *Model) renderHeader() string {
	title := "Dependency Health Check"
	if m.loading {
		title += fmt.Sprintf(" %s", m.spinner.View())
	}

	summary := m.renderSummary()

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229"))

	left := headerStyle.Render(title)
	right := summary

	gap := m.Width() - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}

	return fmt.Sprintf("%s%s%s", left, lipgloss.NewStyle().Width(gap).Render(""), right)
}

// renderSummary renders the check summary.
func (m *Model) renderSummary() string {
	if len(m.groups) == 0 {
		return ""
	}

	summary := m.checker.GetSummary(m.groups)

	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	parts := []string{}
	if summary.OK > 0 {
		parts = append(parts, okStyle.Render(fmt.Sprintf("✓ %d", summary.OK)))
	}
	if summary.Missing > 0 {
		parts = append(parts, errStyle.Render(fmt.Sprintf("✗ %d", summary.Missing)))
	}
	if summary.Warnings > 0 {
		parts = append(parts, warnStyle.Render(fmt.Sprintf("⚠ %d", summary.Warnings)))
	}

	if len(parts) == 0 {
		return dimStyle.Render("No checks")
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

// renderChecks renders the check list.
func (m *Model) renderChecks() string {
	var lines []string

	groupStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("237"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	for i, item := range m.items {
		var line string

		if item.IsGroup {
			line = "\n  " + groupStyle.Render(item.Group.Name)
		} else {
			check := item.Check

			// Status icon
			var icon string
			var iconStyle lipgloss.Style
			switch check.Status {
			case doctor.StatusOK:
				icon = "✓"
				iconStyle = okStyle
			case doctor.StatusMissing:
				icon = "✗"
				iconStyle = errStyle
			case doctor.StatusWarning:
				icon = "⚠"
				iconStyle = warnStyle
			case doctor.StatusError:
				icon = "!"
				iconStyle = errStyle
			}

			// Cursor indicator
			cursor := "  "
			if i == m.cursor {
				cursor = "▸ "
			}

			// Format line
			name := check.Name
			msg := dimStyle.Render(check.Message)
			line = fmt.Sprintf("    %s%s %-16s %s", cursor, iconStyle.Render(icon), name, msg)

			// Highlight selected
			if i == m.cursor {
				line = selectedStyle.Render(line)
			}
		}

		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderStatusBar renders the status bar.
func (m *Model) renderStatusBar() string {
	if m.message != "" {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("\n  " + m.message)
	}
	return ""
}

// renderWithDialog renders the view with fix dialog overlay.
func (m *Model) renderWithDialog(_, _, _ string) string {
	// Dialog box centered on screen
	dialog := m.renderDialog()

	return lipgloss.Place(
		m.Width(),
		m.Height(),
		lipgloss.Center,
		lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
	)
}

// renderDialog renders the fix dialog.
func (m *Model) renderDialog() string {
	if m.dialogFix == nil {
		return ""
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(60)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	cmdStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)
	noteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Title
	title := titleStyle.Render(fmt.Sprintf("Install %s", m.dialogCheckID))

	// Description
	desc := fmt.Sprintf("\n%s\n", m.dialogFix.Description)

	// Command
	cmd := "\n" + cmdStyle.Render(m.dialogFix.Command) + "\n"

	// Sudo note
	var note string
	if m.dialogFix.Sudo {
		note = "\n" + noteStyle.Render("Note: Requires sudo password") + "\n"
	}

	// Message
	var msg string
	if m.dialogMessage != "" {
		msg = "\n" + msgStyle.Render(m.dialogMessage) + "\n"
	}

	// Buttons
	buttons := m.renderDialogButtons()

	content := title + desc + cmd + note + msg + "\n" + buttons

	return boxStyle.Render(content)
}

// renderDialogButtons renders the dialog buttons.
func (m *Model) renderDialogButtons() string {
	normalStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	selectedStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Bold(true)

	buttons := []string{"Cancel", "Copy", "Run"}
	rendered := make([]string, 3)

	for i, btn := range buttons {
		if i == m.dialogCursor {
			rendered[i] = selectedStyle.Render(btn)
		} else {
			rendered[i] = normalStyle.Render(btn)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, rendered...)
}

// Focus sets focus on this tab.
func (m *Model) Focus() tea.Cmd {
	m.BaseTab.Focus()
	return tea.Batch(
		m.spinner.Tick,
		m.loadChecks(),
	)
}

// Blur removes focus from this tab.
func (m *Model) Blur() {
	m.BaseTab.Blur()
}

// SetSize sets the tab dimensions.
func (m *Model) SetSize(width, height int) {
	m.BaseTab.SetSize(width, height)
}

// KeyBindings returns the key bindings for this tab.
func (m *Model) KeyBindings() []string {
	if m.showDialog {
		return []string{
			"[←/→] select",
			"[Enter] confirm",
			"[Esc] cancel",
		}
	}
	return []string{
		"[↑/↓] navigate",
		"[Enter] fix",
		"[r] refresh",
	}
}

// HasFocusedInput returns true when dialog is open.
func (m *Model) HasFocusedInput() bool {
	return m.showDialog
}
