// Package settings provides the settings/config view for the TUI application.
package settings

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/images"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
)

// Message types for async operations.
type (
	settingsLoadedMsg struct {
		settings *settings.Settings
	}

	settingsErrorMsg struct {
		err error
	}

	downloadStartedMsg struct {
		id string
	}

	downloadProgressMsg struct {
		id         string
		downloaded int64
		total      int64
	}

	downloadCompleteMsg struct {
		id      string
		success bool
		err     error
	}
)

// Section represents a section of the config view.
type Section int

const (
	SectionCloudImages Section = iota
	SectionISOs
	SectionDownloads
)

// Model is the settings/config view model.
type Model struct {
	app.BaseTab

	store      *settings.Store
	manager    *images.Manager
	downloader *images.Downloader
	settings   *settings.Settings

	spinner     spinner.Model
	loading     bool
	err         error
	message     string
	section     Section
	cursor      int
	itemCursors map[Section]int // Cursor per section

	// Download dialog
	showDownloadDialog bool
	downloadVersion    string
	downloadArch       string
	dialogCursor       int
}

// New creates a new settings model.
func New() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	var storeErr error
	var message string

	store, err := settings.NewStore()
	if err != nil {
		storeErr = err
		log.Printf("Failed to create settings store: %v. Using temporary fallback at /tmp/ucli", err)

		// Use a fallback store with temp directory if config dir fails
		store = settings.NewStoreWithDir("/tmp/ucli")
		if store == nil {
			log.Printf("Failed to create fallback store")
			storeErr = fmt.Errorf("failed to create settings store: %w", err)
		} else {
			message = fmt.Sprintf("⚠ Using temporary storage at /tmp/ucli (settings may be lost on reboot). Original error: %v", err)
		}
	}

	return &Model{
		BaseTab:     app.NewBaseTab(app.TabConfig, "Config", "5"),
		store:       store,
		manager:     images.NewManager(store),
		downloader:  images.NewDownloader(store),
		spinner:     s,
		loading:     true,
		err:         storeErr,
		message:     message,
		itemCursors: make(map[Section]int),
	}
}

// Init initializes the settings view.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadSettings,
	)
}

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showDownloadDialog {
			return m.handleDialogKey(msg)
		}
		return m.handleKeyMsg(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case settingsLoadedMsg:
		m.loading = false
		m.settings = msg.settings
		m.err = nil

	case settingsErrorMsg:
		m.loading = false
		m.err = msg.err

	case downloadStartedMsg:
		m.message = fmt.Sprintf("Download started: %s", msg.id)
		m.showDownloadDialog = false

	case downloadProgressMsg:
		// Progress updates handled by polling

	case downloadCompleteMsg:
		if msg.success {
			m.message = fmt.Sprintf("Download complete: %s", msg.id)
			cmds = append(cmds, m.loadSettings)
		} else {
			m.message = fmt.Sprintf("Download failed: %v", msg.err)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles keyboard input.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "tab":
		m.nextSection()
	case "shift+tab":
		m.prevSection()
	case "d":
		return m.openDownloadDialog()
	case "x":
		return m.removeSelected()
	case "r":
		m.loading = true
		m.message = ""
		return m, tea.Batch(m.spinner.Tick, m.loadSettings)
	}
	return m, nil
}

// handleDialogKey handles dialog keyboard input.
func (m *Model) handleDialogKey(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	registry := images.NewRegistry()
	releases := registry.GetLTSReleases()
	maxCursor := len(releases) - 1
	if maxCursor < 0 {
		maxCursor = 0
	}

	switch msg.String() {
	case "up", "k":
		if m.dialogCursor > 0 {
			m.dialogCursor--
		}
	case "down", "j":
		if m.dialogCursor < maxCursor {
			m.dialogCursor++
		}
	case "enter":
		return m.confirmDownload()
	case "esc", "q":
		m.showDownloadDialog = false
	}
	return m, nil
}

// moveCursor moves the cursor within the current section.
func (m *Model) moveCursor(delta int) {
	count := m.getItemCount()
	if count == 0 {
		return
	}

	newPos := m.itemCursors[m.section] + delta
	if newPos < 0 {
		newPos = 0
	}
	if newPos >= count {
		newPos = count - 1
	}
	m.itemCursors[m.section] = newPos
}

// getItemCount returns the number of items in the current section.
func (m *Model) getItemCount() int {
	if m.settings == nil {
		return 0
	}

	switch m.section {
	case SectionCloudImages:
		return len(m.settings.CloudImages) + 1 // +1 for "Add" option
	case SectionISOs:
		return len(m.settings.ISOs) + 1 // +1 for "Add" option
	case SectionDownloads:
		state, _ := m.store.LoadDownloadState()
		if state == nil {
			return 0
		}
		return len(state.ActiveDownloads)
	}
	return 0
}

// nextSection moves to the next section.
func (m *Model) nextSection() {
	m.section = (m.section + 1) % 3
}

// prevSection moves to the previous section.
func (m *Model) prevSection() {
	if m.section == 0 {
		m.section = 2
	} else {
		m.section--
	}
}

// openDownloadDialog opens the download dialog.
func (m *Model) openDownloadDialog() (app.Tab, tea.Cmd) {
	m.showDownloadDialog = true
	m.downloadVersion = "24.04"
	m.downloadArch = images.GetDefaultArch()
	m.dialogCursor = 0
	return m, nil
}

// confirmDownload starts the download.
func (m *Model) confirmDownload() (app.Tab, tea.Cmd) {
	m.showDownloadDialog = false

	registry := images.NewRegistry()
	releases := registry.GetLTSReleases()
	if m.dialogCursor >= len(releases) {
		return m, nil
	}

	rel := releases[m.dialogCursor]
	arch := images.GetDefaultArch()
	dl := m.downloader

	return m, func() tea.Msg {
		_, err := dl.DownloadCloudImage(
			context.Background(),
			rel.Version,
			arch,
			nil, // Progress callback
		)
		if err != nil {
			return downloadCompleteMsg{id: rel.Version, success: false, err: err}
		}
		return downloadCompleteMsg{id: rel.Version, success: true}
	}
}

// removeSelected removes the selected item.
func (m *Model) removeSelected() (app.Tab, tea.Cmd) {
	if m.settings == nil {
		return m, nil
	}

	cursor := m.itemCursors[m.section]

	switch m.section {
	case SectionCloudImages:
		if cursor < len(m.settings.CloudImages) {
			img := m.settings.CloudImages[cursor]
			if err := m.manager.RemoveImage(img.ID, false); err != nil {
				m.message = fmt.Sprintf("Failed to remove: %v", err)
			} else {
				m.message = fmt.Sprintf("Removed: %s", img.Name)
				return m, m.loadSettings
			}
		}
	case SectionISOs:
		if cursor < len(m.settings.ISOs) {
			iso := m.settings.ISOs[cursor]
			if err := m.manager.RemoveISO(iso.ID, false); err != nil {
				m.message = fmt.Sprintf("Failed to remove: %v", err)
			} else {
				m.message = fmt.Sprintf("Removed: %s", iso.Name)
				return m, m.loadSettings
			}
		}
	}

	return m, nil
}

// loadSettings loads settings from disk.
func (m *Model) loadSettings() tea.Msg {
	s, err := m.store.Load()
	if err != nil {
		return settingsErrorMsg{err: err}
	}
	return settingsLoadedMsg{settings: s}
}

// View renders the settings view.
func (m *Model) View() string {
	if m.Width() == 0 {
		return "Loading..."
	}

	var content string

	// Header
	header := m.renderHeader()

	// Main content
	if m.loading && m.settings == nil {
		content = fmt.Sprintf("\n  %s Loading settings...\n", m.spinner.View())
	} else if m.err != nil {
		content = fmt.Sprintf("\n  Error: %v\n\n  Press 'r' to retry.\n", m.err)
	} else {
		content = m.renderContent()
	}

	// Status message
	statusBar := m.renderStatusBar()

	// Dialog overlay
	if m.showDownloadDialog {
		return m.renderWithDialog(header, content, statusBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}

// renderHeader renders the view header.
func (m *Model) renderHeader() string {
	title := "Configuration"
	if m.loading {
		title += fmt.Sprintf(" %s", m.spinner.View())
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))

	return headerStyle.Render(title)
}

// renderContent renders the main content.
func (m *Model) renderContent() string {
	var sections []string

	// Cloud Images section
	sections = append(sections, m.renderSection("Cloud Images", SectionCloudImages, m.renderCloudImages))

	// ISOs section
	sections = append(sections, m.renderSection("ISOs", SectionISOs, m.renderISOs))

	// Downloads section
	sections = append(sections, m.renderSection("Active Downloads", SectionDownloads, m.renderDownloads))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderSection renders a section with header.
func (m *Model) renderSection(title string, section Section, renderItems func() string) string {
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	header := sectionStyle.Render(title)
	if m.section == section {
		header = activeStyle.Render("▸ " + title)
	}

	border := borderStyle.Render("  " + strings.Repeat("─", m.Width()-4))

	items := renderItems()

	return fmt.Sprintf("\n%s\n%s\n%s", header, border, items)
}

// renderCloudImages renders the cloud images list.
func (m *Model) renderCloudImages() string {
	if m.settings == nil || len(m.settings.CloudImages) == 0 {
		return m.renderAddOption(SectionCloudImages)
	}

	var lines []string
	cursor := m.itemCursors[SectionCloudImages]

	for i, img := range m.settings.CloudImages {
		line := m.renderImageLine(i, cursor, img.Name, filepath.Base(img.Path), img.Verified)
		lines = append(lines, line)
	}

	// Add option
	lines = append(lines, m.renderAddLine(len(m.settings.CloudImages), cursor, "Download cloud image..."))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderISOs renders the ISOs list.
func (m *Model) renderISOs() string {
	if m.settings == nil || len(m.settings.ISOs) == 0 {
		return m.renderAddOption(SectionISOs)
	}

	var lines []string
	cursor := m.itemCursors[SectionISOs]

	for i, iso := range m.settings.ISOs {
		line := m.renderImageLine(i, cursor, iso.Name, filepath.Base(iso.Path), true)
		lines = append(lines, line)
	}

	// Add option
	lines = append(lines, m.renderAddLine(len(m.settings.ISOs), cursor, "Add ISO..."))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderDownloads renders active downloads.
func (m *Model) renderDownloads() string {
	state, _ := m.store.LoadDownloadState()
	if state == nil || len(state.ActiveDownloads) == 0 {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		return dimStyle.Render("  No active downloads")
	}

	var lines []string
	for _, dl := range state.ActiveDownloads {
		progress := float64(dl.Downloaded) / float64(dl.TotalBytes) * 100
		if dl.TotalBytes == 0 {
			progress = 0
		}

		statusIcon := "⏳"
		if dl.Status == settings.StatusComplete {
			statusIcon = "✓"
		} else if dl.Status == settings.StatusError {
			statusIcon = "✗"
		}

		line := fmt.Sprintf("  %s %-30s %5.1f%%", statusIcon, filepath.Base(dl.DestPath), progress)
		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderAddOption renders a simple add option for empty sections.
func (m *Model) renderAddOption(section Section) string {
	cursor := m.itemCursors[section]
	return m.renderAddLine(0, cursor, "Add...")
}

// renderImageLine renders a single image line.
func (m *Model) renderImageLine(idx, cursor int, name, filename string, verified bool) string {
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("237"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	cursorStr := "  "
	if idx == cursor && m.section != SectionDownloads {
		cursorStr = "▸ "
	}

	icon := okStyle.Render("✓")
	if !verified {
		icon = dimStyle.Render("?")
	}

	line := fmt.Sprintf("  %s%s %-25s %s", cursorStr, icon, name, dimStyle.Render(filename))

	if idx == cursor {
		line = selectedStyle.Render(line)
	}

	return line
}

// renderAddLine renders an "Add" line.
func (m *Model) renderAddLine(idx, cursor int, text string) string {
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("237"))
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))

	cursorStr := "  "
	if idx == cursor {
		cursorStr = "▸ "
	}

	line := fmt.Sprintf("  %s%s", cursorStr, addStyle.Render("+ "+text))

	if idx == cursor {
		line = selectedStyle.Render(line)
	}

	return line
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

// renderWithDialog renders the view with download dialog overlay.
func (m *Model) renderWithDialog(_, _, _ string) string {
	dialog := m.renderDownloadDialog()

	return lipgloss.Place(
		m.Width(),
		m.Height(),
		lipgloss.Center,
		lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
	)
}

// renderDownloadDialog renders the download dialog.
func (m *Model) renderDownloadDialog() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(50)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("237"))

	title := titleStyle.Render("Download Cloud Image")

	registry := images.NewRegistry()
	releases := registry.GetLTSReleases()

	var options []string
	for i, rel := range releases {
		line := fmt.Sprintf("  %s (%s)", rel.Name, rel.Codename)
		if i == m.dialogCursor {
			line = selectedStyle.Render("▸" + line[1:])
		}
		options = append(options, line)
	}

	optionsStr := lipgloss.JoinVertical(lipgloss.Left, options...)

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("\n[Enter] download  [Esc] cancel")

	content := title + "\n\n" + optionsStr + hint

	return boxStyle.Render(content)
}

// Focus sets focus on this tab.
func (m *Model) Focus() tea.Cmd {
	m.BaseTab.Focus()
	return tea.Batch(
		m.spinner.Tick,
		m.loadSettings,
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
	if m.showDownloadDialog {
		return []string{
			"[↑/↓] select",
			"[Enter] download",
			"[Esc] cancel",
		}
	}
	return []string{
		"[↑/↓] navigate",
		"[Tab] section",
		"[d] download",
		"[x] remove",
		"[r] refresh",
	}
}

// HasFocusedInput returns true when dialog is open.
func (m *Model) HasFocusedInput() bool {
	return m.showDownloadDialog
}
