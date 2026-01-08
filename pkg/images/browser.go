// Package images provides cloud image management functionality.
package images

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/globalconfig"
)

// BrowserMode represents the current browsing state.
type BrowserMode int

const (
	ModeOSSelection BrowserMode = iota
	ModeVersionSelection
	ModeTypeSelection
	ModeImageSelection
	ModeCurlDialog
)

// Clipboard message types
type clipboardCopiedMsg struct {
	success bool
	err     error
}

// Browser is the image browser tab.
type Browser struct {
	app.BaseTab

	registry *Registry
	config   *globalconfig.Config
	manager  *Manager

	// Navigation state
	mode         BrowserMode
	selectedOS   string
	selectedVer  string
	selectedType ImageType
	cursor       int

	// Current view data
	osList      []string
	versionList []string
	typeList    []ImageType
	imageList   []ImageMetadata

	// Curl command dialog
	showCurlDialog bool
	selectedImage  *ImageMetadata
	pathInput      textinput.Model
	curlCommand    string

	// Status
	message string
	err     error
}

// New creates a new image browser tab.
func New() *Browser {
	pathInput := textinput.New()
	pathInput.Placeholder = "~/.local/share/ucli/images/..."
	pathInput.CharLimit = 256
	pathInput.Width = 70

	return &Browser{
		BaseTab:   app.NewBaseTab(app.TabImages, "Images", "4"),
		registry:  NewRegistry(),
		pathInput: pathInput,
	}
}

// Init initializes the browser.
func (b *Browser) Init() tea.Cmd {
	// Load global config
	cfg, err := globalconfig.LoadOrCreate()
	if err != nil {
		b.err = err
		return nil
	}
	b.config = cfg
	b.manager = NewManager(cfg)

	// Initialize OS list
	b.osList = b.registry.GetOSList()
	sort.Strings(b.osList)
	b.mode = ModeOSSelection

	return nil
}

// Update handles messages.
func (b *Browser) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	if b.showCurlDialog {
		return b.handleCurlDialog(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return b.handleKeyMsg(msg)
	}

	return b, nil
}

// handleKeyMsg handles keyboard input in navigation mode.
func (b *Browser) handleKeyMsg(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up", "k":
		b.moveCursor(-1)
	case "down", "j":
		b.moveCursor(1)
	case "esc":
		b.goBack()
	case "enter":
		return b.handleEnter()
	case "r":
		// Refresh file statuses
		b.refreshStatuses()
	}

	return b, nil
}

// handleCurlDialog handles input when curl dialog is shown.
func (b *Browser) handleCurlDialog(msg tea.Msg) (app.Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "esc":
			b.closeCurlDialog()
			return b, nil

		case "enter":
			// Copy to clipboard and update config
			return b.copyCurlCommand()

		case "tab":
			// Re-generate curl command with new path
			b.curlCommand = generateCurlCommand(b.selectedImage, b.pathInput.Value())
			return b, nil
		}

	case clipboardCopiedMsg:
		if msg.success {
			b.message = "✓ Copied to clipboard! Config updated."
		} else {
			b.message = fmt.Sprintf("Failed to copy: %v", msg.err)
		}
		b.closeCurlDialog()
		return b, nil
	}

	// Update text input
	var cmd tea.Cmd
	b.pathInput, cmd = b.pathInput.Update(msg)
	return b, cmd
}

// moveCursor moves the cursor up or down.
func (b *Browser) moveCursor(delta int) {
	var maxItems int
	switch b.mode {
	case ModeOSSelection:
		maxItems = len(b.osList)
	case ModeVersionSelection:
		maxItems = len(b.versionList)
	case ModeTypeSelection:
		maxItems = len(b.typeList)
	case ModeImageSelection:
		maxItems = len(b.imageList)
	default:
		return
	}

	if maxItems == 0 {
		return
	}

	b.cursor += delta
	if b.cursor < 0 {
		b.cursor = maxItems - 1
	} else if b.cursor >= maxItems {
		b.cursor = 0
	}
}

// goBack navigates to the previous mode.
func (b *Browser) goBack() {
	b.message = ""
	b.cursor = 0

	switch b.mode {
	case ModeVersionSelection:
		b.mode = ModeOSSelection
		b.selectedOS = ""
	case ModeTypeSelection:
		b.mode = ModeVersionSelection
		b.selectedVer = ""
	case ModeImageSelection:
		b.mode = ModeTypeSelection
		b.selectedType = 0
	default:
		// Already at top level
		return
	}
}

// handleEnter processes enter key based on current mode.
func (b *Browser) handleEnter() (app.Tab, tea.Cmd) {
	switch b.mode {
	case ModeOSSelection:
		b.selectOS()
	case ModeVersionSelection:
		b.selectVersion()
	case ModeTypeSelection:
		b.selectType()
	case ModeImageSelection:
		b.showCurlCommandDialog()
	}
	return b, nil
}

// selectOS selects an OS and moves to version selection.
func (b *Browser) selectOS() {
	if b.cursor >= len(b.osList) {
		return
	}

	b.selectedOS = b.osList[b.cursor]
	b.versionList = b.registry.GetVersionsForOS(b.selectedOS)
	sort.Slice(b.versionList, func(i, j int) bool {
		// Sort in reverse order (newer first)
		return b.versionList[i] > b.versionList[j]
	})

	b.cursor = 0
	b.mode = ModeVersionSelection
}

// selectVersion selects a version and moves to type selection.
func (b *Browser) selectVersion() {
	if b.cursor >= len(b.versionList) {
		return
	}

	b.selectedVer = b.versionList[b.cursor]
	b.typeList = b.registry.GetTypesForOSVersion(b.selectedOS, b.selectedVer)

	// Sort types: CloudInit, Desktop, NoCloud
	sort.Slice(b.typeList, func(i, j int) bool {
		return b.typeList[i] < b.typeList[j]
	})

	b.cursor = 0
	b.mode = ModeTypeSelection
}

// selectType selects an image type and moves to image selection.
func (b *Browser) selectType() {
	if b.cursor >= len(b.typeList) {
		return
	}

	b.selectedType = b.typeList[b.cursor]
	b.imageList = b.registry.GetImagesForOSVersionType(b.selectedOS, b.selectedVer, b.selectedType)

	// Sort by arch then variant
	sort.Slice(b.imageList, func(i, j int) bool {
		if b.imageList[i].Arch != b.imageList[j].Arch {
			return b.imageList[i].Arch < b.imageList[j].Arch
		}
		return b.imageList[i].Variant < b.imageList[j].Variant
	})

	b.cursor = 0
	b.mode = ModeImageSelection
}

// showCurlCommandDialog shows the curl command dialog for the selected image.
func (b *Browser) showCurlCommandDialog() {
	if b.cursor >= len(b.imageList) {
		return
	}

	img := b.imageList[b.cursor]
	b.selectedImage = &img

	// Generate default path
	defaultPath := b.manager.DefaultPathForImage(&img)
	b.pathInput.SetValue(defaultPath)
	b.pathInput.Focus()

	// Generate curl command
	b.curlCommand = generateCurlCommand(&img, defaultPath)
	b.showCurlDialog = true
}

// copyCurlCommand copies the command to clipboard and updates config.
func (b *Browser) copyCurlCommand() (app.Tab, tea.Cmd) {
	return b, func() tea.Msg {
		// Copy to clipboard
		if err := clipboard.WriteAll(b.curlCommand); err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}

		// Update config with the path
		path := b.pathInput.Value()
		if path != "" {
			cloudImg := globalconfig.CloudImage{
				ID:            b.selectedImage.ID,
				Name:          b.selectedImage.Description,
				Version:       b.selectedImage.Version,
				Arch:          b.selectedImage.Arch,
				Path:          expandPath(path),
				URL:           b.selectedImage.URL,
				SHA256:        "", // Will be filled after download
				Size:          0,  // Unknown until downloaded
				Source:        b.selectedImage.Source.String(),
				CloudInitType: b.selectedImage.Type.String(),
				Variant:       string(b.selectedImage.Variant),
				CurlCommand:   b.curlCommand,
			}
			cloudImg.UpdateStatus()

			b.config.AddCloudImage(cloudImg)
			if err := b.config.Save(); err != nil {
				return clipboardCopiedMsg{success: false, err: fmt.Errorf("failed to save config: %w", err)}
			}
		}

		return clipboardCopiedMsg{success: true}
	}
}

// closeCurlDialog closes the curl dialog.
func (b *Browser) closeCurlDialog() {
	b.showCurlDialog = false
	b.selectedImage = nil
	b.pathInput.Blur()
	b.pathInput.SetValue("")
}

// refreshStatuses updates download status for all configured images.
func (b *Browser) refreshStatuses() {
	if err := b.manager.UpdateImageStatuses(); err != nil {
		b.message = fmt.Sprintf("Error refreshing status: %v", err)
	} else {
		b.message = "Status refreshed"
	}
}

// getImageStatus checks if an image is downloaded.
func (b *Browser) getImageStatus(img *ImageMetadata) bool {
	configImg := b.config.FindCloudImage(img.ID)
	if configImg != nil {
		return configImg.FileExists()
	}
	return false
}

// View renders the browser.
func (b *Browser) View() string {
	if b.err != nil {
		return errorStyle.Render("Error: " + b.err.Error())
	}

	if b.showCurlDialog {
		return b.viewCurlDialog()
	}

	var s strings.Builder

	// Breadcrumb
	s.WriteString(b.renderBreadcrumb())
	s.WriteString("\n\n")

	// Content based on mode
	switch b.mode {
	case ModeOSSelection:
		s.WriteString(b.viewOSList())
	case ModeVersionSelection:
		s.WriteString(b.viewVersionList())
	case ModeTypeSelection:
		s.WriteString(b.viewTypeList())
	case ModeImageSelection:
		s.WriteString(b.viewImageList())
	}

	// Message
	if b.message != "" {
		s.WriteString("\n\n")
		s.WriteString(dimStyle.Render(b.message))
	}

	// Help
	s.WriteString("\n\n")
	s.WriteString(b.viewHelp())

	return s.String()
}

// renderBreadcrumb renders the navigation breadcrumb.
func (b *Browser) renderBreadcrumb() string {
	parts := []string{"Images"}

	if b.selectedOS != "" {
		parts = append(parts, b.selectedOS)
	}
	if b.selectedVer != "" {
		parts = append(parts, b.selectedVer)
	}
	if b.selectedType != 0 {
		parts = append(parts, b.selectedType.DisplayName())
	}

	return titleStyle.Render(strings.Join(parts, " > "))
}

// viewOSList renders the OS selection view.
func (b *Browser) viewOSList() string {
	var s strings.Builder
	s.WriteString(subtitleStyle.Render("Select Operating System"))
	s.WriteString("\n\n")

	for i, os := range b.osList {
		line := os
		if i == b.cursor {
			s.WriteString(selectedStyle.Render("▶ " + line))
		} else {
			s.WriteString("  " + line)
		}
		s.WriteString("\n")
	}

	return s.String()
}

// viewVersionList renders the version selection view.
func (b *Browser) viewVersionList() string {
	var s strings.Builder
	s.WriteString(subtitleStyle.Render("Select Version"))
	s.WriteString("\n\n")

	for i, ver := range b.versionList {
		line := ver
		if i == b.cursor {
			s.WriteString(selectedStyle.Render("▶ " + line))
		} else {
			s.WriteString("  " + line)
		}
		s.WriteString("\n")
	}

	return s.String()
}

// viewTypeList renders the type selection view.
func (b *Browser) viewTypeList() string {
	var s strings.Builder
	s.WriteString(subtitleStyle.Render("Select Image Type"))
	s.WriteString("\n\n")

	for i, imgType := range b.typeList {
		line := imgType.DisplayName()
		if i == b.cursor {
			s.WriteString(selectedStyle.Render("▶ " + line))
		} else {
			s.WriteString("  " + line)
		}
		s.WriteString("\n")
	}

	return s.String()
}

// viewImageList renders the image selection view.
func (b *Browser) viewImageList() string {
	var s strings.Builder
	s.WriteString(subtitleStyle.Render("Select Image"))
	s.WriteString("\n\n")

	for i, img := range b.imageList {
		// Check if downloaded
		status := b.getImageStatus(&img)
		var statusIcon string
		if status {
			statusIcon = successStyle.Render("✓")
		} else {
			statusIcon = errorStyle.Render("✗")
		}

		// Main line
		line := fmt.Sprintf("%s %s", statusIcon, img.Description)
		if i == b.cursor {
			s.WriteString(selectedStyle.Render("▶ " + line))
		} else {
			s.WriteString("  " + line)
		}
		s.WriteString("\n")

		// Metadata
		meta := fmt.Sprintf("    %s | %s", img.Arch, img.Size)
		s.WriteString(dimStyle.Render(meta))
		s.WriteString("\n")
	}

	return s.String()
}

// viewCurlDialog renders the curl command dialog.
func (b *Browser) viewCurlDialog() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Download Command"))
	s.WriteString("\n\n")

	// Image info
	s.WriteString(labelStyle.Render("Image: "))
	s.WriteString(b.selectedImage.Description)
	s.WriteString("\n\n")

	// Path input
	s.WriteString(labelStyle.Render("Download Path:"))
	s.WriteString("\n")
	s.WriteString(b.pathInput.View())
	s.WriteString("\n")
	s.WriteString(dimStyle.Render("Press Tab to update command with edited path"))
	s.WriteString("\n\n")

	// Curl command
	s.WriteString(labelStyle.Render("Command:"))
	s.WriteString("\n")
	s.WriteString(codeStyle.Render(b.curlCommand))
	s.WriteString("\n\n")

	s.WriteString(dimStyle.Render("[Enter] Copy to clipboard  [Esc] Cancel"))

	return dialogStyle.Render(s.String())
}

// viewHelp renders the help text.
func (b *Browser) viewHelp() string {
	if b.showCurlDialog {
		return ""
	}

	help := []string{"↑/↓ navigate", "Enter select", "r refresh"}
	if b.mode != ModeOSSelection {
		help = append(help, "Esc back")
	}

	return dimStyle.Render(strings.Join(help, "  •  "))
}

// KeyBindings returns the key bindings for the footer.
func (b *Browser) KeyBindings() []string {
	return []string{
		"↑/↓ navigate",
		"enter select",
		"r refresh",
		"esc back",
	}
}

// HasFocusedInput returns true if the text input is focused.
func (b *Browser) HasFocusedInput() bool {
	return b.showCurlDialog && b.pathInput.Focused()
}

// Styles
var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	subtitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	labelStyle    = lipgloss.NewStyle().Bold(true)
	codeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2).
			Width(80)
)

// generateCurlCommand creates a curl command for downloading an image.
func generateCurlCommand(img *ImageMetadata, destPath string) string {
	expanded := expandPath(destPath)

	var s strings.Builder

	s.WriteString("curl -L \\\n")
	s.WriteString("  --create-dirs \\\n")
	s.WriteString(fmt.Sprintf("  --output \"%s\" \\\n", expanded))
	s.WriteString("  --progress-bar")

	// Add resume support for large ISOs
	if img.Type == TypeDesktop {
		s.WriteString(" \\\n  --continue-at -")
	}

	s.WriteString(" \\\n")
	s.WriteString(fmt.Sprintf("  \"%s\"", img.URL))

	// Add checksum verification if available
	if img.ChecksumURL != "" {
		s.WriteString("\n\n# Verify checksum:\n")
		s.WriteString(fmt.Sprintf("curl -sL \"%s\" | grep -F '%s'", img.ChecksumURL, img.Filename))

		// Use appropriate checksum tool
		if strings.Contains(img.ChecksumURL, "SHA512") {
			s.WriteString(" | sha512sum -c -")
		} else {
			s.WriteString(" | sha256sum -c -")
		}
	}

	return s.String()
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
