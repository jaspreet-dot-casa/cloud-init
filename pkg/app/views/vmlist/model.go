// Package vmlist provides the VM list view for the TUI application.
package vmlist

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tfstate"
)

const (
	// RefreshInterval is the auto-refresh interval for VM list
	RefreshInterval = 5 * time.Second
)

// RefreshMsg triggers a VM list refresh
type RefreshMsg struct{}

// VMsLoadedMsg indicates VMs were loaded successfully
type VMsLoadedMsg struct {
	VMs []tfstate.VMInfo
}

// VMsErrorMsg indicates an error loading VMs
type VMsErrorMsg struct {
	Err error
}

// ActionResultMsg indicates result of a VM action
type ActionResultMsg struct {
	Action  string
	VMName  string
	Success bool
	Err     error
}

// Model is the VM list view model
type Model struct {
	app.BaseTab

	manager    *tfstate.Manager
	vms        []tfstate.VMInfo
	table      table.Model
	spinner    spinner.Model
	loading    bool
	err        error
	lastUpdate time.Time

	// Action state
	actionInProgress bool
	actionMessage    string

	// Auto-refresh
	autoRefresh bool
}

// New creates a new VM list model
func New(projectDir string) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := &Model{
		BaseTab:     app.NewBaseTab(app.TabVMs, "VMs", "1"),
		manager:     tfstate.NewManager(projectDir),
		spinner:     s,
		loading:     true,
		autoRefresh: true,
	}

	m.table = m.createTable()
	return m
}

// createTable creates the table model with columns
func (m *Model) createTable() table.Model {
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "STATUS", Width: 12},
		{Title: "IP", Width: 16},
		{Title: "CPU", Width: 5},
		{Title: "MEM", Width: 8},
		{Title: "DISK", Width: 8},
		{Title: "AUTO", Width: 6},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t
}

// Init initializes the VM list view
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadVMs,
	)
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.actionInProgress {
			return m, nil
		}
		return m.handleKeyMsg(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case RefreshMsg:
		if !m.loading && !m.actionInProgress {
			m.loading = true
			cmds = append(cmds, m.loadVMs)
		}
		if m.autoRefresh && m.IsFocused() {
			cmds = append(cmds, m.scheduleRefresh())
		}

	case VMsLoadedMsg:
		m.loading = false
		m.vms = msg.VMs
		m.lastUpdate = time.Now()
		m.err = nil
		m.updateTableRows()

	case VMsErrorMsg:
		m.loading = false
		m.err = msg.Err

	case ActionResultMsg:
		m.actionInProgress = false
		if msg.Success {
			m.actionMessage = fmt.Sprintf("%s %s: success", msg.Action, msg.VMName)
			cmds = append(cmds, m.loadVMs)
		} else {
			m.actionMessage = fmt.Sprintf("%s %s: %v", msg.Action, msg.VMName, msg.Err)
		}
	}

	// Update table
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles keyboard input
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	keys := app.DefaultVMListKeyMap()

	switch {
	case msg.String() == "s":
		return m.startSelectedVM()
	case msg.String() == "S":
		return m.stopSelectedVM()
	case msg.String() == "d":
		return m.deleteSelectedVM()
	case msg.String() == "c":
		return m.openConsole()
	case msg.String() == "x":
		return m.sshToVM()
	case msg.String() == "r":
		m.loading = true
		return m, m.loadVMs
	case msg.String() == "enter":
		return m.showDetails()
	default:
		_ = keys // silence unused warning for now
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
}

// View renders the VM list
func (m *Model) View() string {
	if m.Width() == 0 {
		return "Loading..."
	}

	var content string

	// Header with last update time
	header := m.renderHeader()

	// Main content
	if m.loading && len(m.vms) == 0 {
		content = fmt.Sprintf("\n  %s Loading VMs...\n", m.spinner.View())
	} else if m.err != nil {
		content = fmt.Sprintf("\n  Error: %v\n\n  Press 'r' to retry.\n", m.err)
	} else if len(m.vms) == 0 {
		content = "\n  No VMs found.\n\n  Press '2' to create a new VM.\n"
	} else {
		content = m.table.View()
	}

	// Status bar
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}

// renderHeader renders the view header
func (m *Model) renderHeader() string {
	title := "VMs (Terraform/libvirt)"
	if m.loading {
		title += fmt.Sprintf(" %s", m.spinner.View())
	}

	updateInfo := ""
	if !m.lastUpdate.IsZero() {
		updateInfo = fmt.Sprintf("Last update: %s", m.lastUpdate.Format("15:04:05"))
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	left := headerStyle.Render(title)
	right := dimStyle.Render(updateInfo)

	gap := m.Width() - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}

	return fmt.Sprintf("%s%s%s", left, lipgloss.NewStyle().Width(gap).Render(""), right)
}

// renderStatusBar renders the status bar at the bottom
func (m *Model) renderStatusBar() string {
	if m.actionMessage != "" {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(m.actionMessage)
	}

	if m.actionInProgress {
		return fmt.Sprintf("%s Action in progress...", m.spinner.View())
	}

	return ""
}

// updateTableRows updates the table with current VM data
func (m *Model) updateTableRows() {
	rows := make([]table.Row, len(m.vms))
	for i, vm := range m.vms {
		autostart := "no"
		if vm.Autostart {
			autostart = "yes"
		}
		rows[i] = table.Row{
			vm.Name,
			string(vm.Status),
			m.formatIP(vm.IP),
			fmt.Sprintf("%d", vm.CPUs),
			m.formatMemory(vm.MemoryMB),
			m.formatDisk(vm.DiskGB),
			autostart,
		}
	}
	m.table.SetRows(rows)
}

// formatIP formats IP address for display
func (m *Model) formatIP(ip string) string {
	if ip == "" || ip == "pending" {
		return "-"
	}
	return ip
}

// formatMemory formats memory in human-readable form
func (m *Model) formatMemory(mb int) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.1fGB", float64(mb)/1024)
	}
	return fmt.Sprintf("%dMB", mb)
}

// formatDisk formats disk size in human-readable form
func (m *Model) formatDisk(gb int) string {
	return fmt.Sprintf("%dGB", gb)
}

// loadVMs returns a command to load VMs
func (m *Model) loadVMs() tea.Msg {
	ctx := context.Background()
	vms, err := m.manager.ListVMs(ctx)
	if err != nil {
		return VMsErrorMsg{Err: err}
	}
	return VMsLoadedMsg{VMs: vms}
}

// scheduleRefresh returns a command to schedule the next refresh
func (m *Model) scheduleRefresh() tea.Cmd {
	return tea.Tick(RefreshInterval, func(t time.Time) tea.Msg {
		return RefreshMsg{}
	})
}

// selectedVM returns the currently selected VM, if any
func (m *Model) selectedVM() *tfstate.VMInfo {
	if len(m.vms) == 0 {
		return nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.vms) {
		return nil
	}
	return &m.vms[idx]
}

// startSelectedVM starts the selected VM
func (m *Model) startSelectedVM() (app.Tab, tea.Cmd) {
	vm := m.selectedVM()
	if vm == nil {
		return m, nil
	}
	m.actionInProgress = true
	m.actionMessage = ""
	return m, m.performAction("start", vm.Name, m.manager.StartVM)
}

// stopSelectedVM stops the selected VM
func (m *Model) stopSelectedVM() (app.Tab, tea.Cmd) {
	vm := m.selectedVM()
	if vm == nil {
		return m, nil
	}
	m.actionInProgress = true
	m.actionMessage = ""
	return m, m.performAction("stop", vm.Name, m.manager.StopVM)
}

// deleteSelectedVM deletes the selected VM
func (m *Model) deleteSelectedVM() (app.Tab, tea.Cmd) {
	vm := m.selectedVM()
	if vm == nil {
		return m, nil
	}
	m.actionInProgress = true
	m.actionMessage = ""
	return m, m.performAction("delete", vm.Name, m.manager.DeleteVM)
}

// performAction performs a VM action asynchronously
func (m *Model) performAction(action, vmName string, fn func(context.Context, string) error) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := fn(ctx, vmName)
		return ActionResultMsg{
			Action:  action,
			VMName:  vmName,
			Success: err == nil,
			Err:     err,
		}
	}
}

// openConsole opens virsh console for the selected VM
func (m *Model) openConsole() (app.Tab, tea.Cmd) {
	vm := m.selectedVM()
	if vm == nil {
		return m, nil
	}
	// TODO: Implement console opening (requires exec)
	m.actionMessage = fmt.Sprintf("Console for %s: virsh console %s", vm.Name, vm.Name)
	return m, nil
}

// sshToVM opens SSH connection to the selected VM
func (m *Model) sshToVM() (app.Tab, tea.Cmd) {
	vm := m.selectedVM()
	if vm == nil {
		return m, nil
	}
	if vm.IP == "" || vm.IP == "pending" {
		m.actionMessage = "Cannot SSH: VM has no IP address"
		return m, nil
	}
	// TODO: Implement SSH opening (requires exec)
	m.actionMessage = fmt.Sprintf("SSH to %s: ssh ubuntu@%s", vm.Name, vm.IP)
	return m, nil
}

// showDetails shows detailed info for the selected VM
func (m *Model) showDetails() (app.Tab, tea.Cmd) {
	vm := m.selectedVM()
	if vm == nil {
		return m, nil
	}
	// TODO: Implement details view
	m.actionMessage = fmt.Sprintf("Details for: %s", vm.Name)
	return m, nil
}

// Focus sets focus on this tab
func (m *Model) Focus() tea.Cmd {
	m.BaseTab.Focus()
	return tea.Batch(
		m.spinner.Tick,
		m.loadVMs,
		m.scheduleRefresh(),
	)
}

// Blur removes focus from this tab
func (m *Model) Blur() {
	m.BaseTab.Blur()
	m.autoRefresh = false
}

// SetSize sets the tab dimensions
func (m *Model) SetSize(width, height int) {
	m.BaseTab.SetSize(width, height)
	// Reserve space for header and status bar
	tableHeight := height - 4
	if tableHeight < 5 {
		tableHeight = 5
	}
	m.table.SetWidth(width)
	m.table.SetHeight(tableHeight)
}

// KeyBindings returns the key bindings for this tab
func (m *Model) KeyBindings() []string {
	return []string{
		"[s] start",
		"[S] stop",
		"[d] delete",
		"[c] console",
		"[x] ssh",
		"[r] refresh",
		"[Enter] details",
	}
}

// HasFocusedInput returns false as this tab has no text inputs
func (m *Model) HasFocusedInput() bool {
	return false
}
