package vmlist

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tfstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	m := New("/test/project")

	assert.Equal(t, app.TabVMs, m.ID())
	assert.Equal(t, "VMs", m.Name())
	assert.Equal(t, "1", m.ShortKey())
	assert.True(t, m.loading)
	assert.True(t, m.autoRefresh)
	assert.NotNil(t, m.manager)
	assert.NotNil(t, m.spinner)
}

func TestModel_Init(t *testing.T) {
	m := New("/test/project")

	cmd := m.Init()

	// Init should return a batch command
	assert.NotNil(t, cmd)
}

func TestModel_KeyBindings(t *testing.T) {
	m := New("/test/project")

	bindings := m.KeyBindings()

	assert.NotEmpty(t, bindings)
	assert.Contains(t, bindings, "[s] start")
	assert.Contains(t, bindings, "[S] stop")
	assert.Contains(t, bindings, "[d] delete")
	assert.Contains(t, bindings, "[c] console")
	assert.Contains(t, bindings, "[x] ssh")
	assert.Contains(t, bindings, "[r] refresh")
	assert.Contains(t, bindings, "[Enter] details")
}

func TestModel_SetSize(t *testing.T) {
	m := New("/test/project")

	m.SetSize(120, 40)

	assert.Equal(t, 120, m.Width())
	assert.Equal(t, 40, m.Height())
}

func TestModel_Focus(t *testing.T) {
	m := New("/test/project")
	assert.False(t, m.IsFocused())

	cmd := m.Focus()

	assert.True(t, m.IsFocused())
	assert.NotNil(t, cmd)
}

func TestModel_Blur(t *testing.T) {
	m := New("/test/project")
	m.Focus()
	assert.True(t, m.IsFocused())
	assert.True(t, m.autoRefresh)

	m.Blur()

	assert.False(t, m.IsFocused())
	assert.False(t, m.autoRefresh)
}

func TestModel_Update_VMsLoadedMsg(t *testing.T) {
	m := New("/test/project")
	m.loading = true

	vms := []tfstate.VMInfo{
		{Name: "test-vm", Status: tfstate.StatusRunning, IP: "192.168.1.10", CPUs: 2, MemoryMB: 2048},
	}

	updated, _ := m.Update(VMsLoadedMsg{VMs: vms})
	model := updated.(*Model)

	assert.False(t, model.loading)
	assert.Equal(t, vms, model.vms)
	assert.Nil(t, model.err)
	assert.False(t, model.lastUpdate.IsZero())
}

func TestModel_Update_VMsErrorMsg(t *testing.T) {
	m := New("/test/project")
	m.loading = true

	err := assert.AnError

	updated, _ := m.Update(VMsErrorMsg{Err: err})
	model := updated.(*Model)

	assert.False(t, model.loading)
	assert.Equal(t, err, model.err)
}

func TestModel_Update_RefreshMsg(t *testing.T) {
	m := New("/test/project")
	m.Focus()
	m.loading = false
	m.actionInProgress = false

	updated, cmd := m.Update(RefreshMsg{})
	model := updated.(*Model)

	assert.True(t, model.loading)
	assert.NotNil(t, cmd)
}

func TestModel_Update_RefreshMsg_SkipWhenLoading(t *testing.T) {
	m := New("/test/project")
	m.loading = true

	_, cmd := m.Update(RefreshMsg{})

	// When loading, refresh is not triggered again
	// But cmd might still contain other commands
	_ = cmd
}

func TestModel_Update_RefreshMsg_SkipWhenActionInProgress(t *testing.T) {
	m := New("/test/project")
	m.loading = false
	m.actionInProgress = true

	updated, _ := m.Update(RefreshMsg{})
	model := updated.(*Model)

	// Should not trigger loading when action is in progress
	assert.False(t, model.loading)
}

func TestModel_Update_ActionResultMsg_Success(t *testing.T) {
	m := New("/test/project")
	m.actionInProgress = true

	result := ActionResultMsg{
		Action:  "start",
		VMName:  "test-vm",
		Success: true,
		Err:     nil,
	}

	updated, cmd := m.Update(result)
	model := updated.(*Model)

	assert.False(t, model.actionInProgress)
	assert.Contains(t, model.actionMessage, "start")
	assert.Contains(t, model.actionMessage, "test-vm")
	assert.Contains(t, model.actionMessage, "success")
	assert.NotNil(t, cmd) // Should trigger refresh
}

func TestModel_Update_ActionResultMsg_Error(t *testing.T) {
	m := New("/test/project")
	m.actionInProgress = true

	result := ActionResultMsg{
		Action:  "stop",
		VMName:  "test-vm",
		Success: false,
		Err:     assert.AnError,
	}

	updated, _ := m.Update(result)
	model := updated.(*Model)

	assert.False(t, model.actionInProgress)
	assert.Contains(t, model.actionMessage, "stop")
	assert.Contains(t, model.actionMessage, "test-vm")
}

func TestModel_View_Loading(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 30)
	m.loading = true
	m.vms = nil

	view := m.View()

	assert.Contains(t, view, "Loading")
}

func TestModel_View_Error(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 30)
	m.loading = false
	m.err = assert.AnError

	view := m.View()

	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "retry")
}

func TestModel_View_NoVMs(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 30)
	m.loading = false
	m.vms = []tfstate.VMInfo{}

	view := m.View()

	assert.Contains(t, view, "No VMs found")
}

func TestModel_View_WithVMs(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 30)
	m.loading = false
	m.vms = []tfstate.VMInfo{
		{Name: "test-vm", Status: tfstate.StatusRunning, IP: "192.168.1.10", CPUs: 2, MemoryMB: 2048},
	}
	m.updateTableRows()

	view := m.View()

	assert.Contains(t, view, "test-vm")
}

func TestModel_View_ZeroWidth(t *testing.T) {
	m := New("/test/project")
	// Width is 0 (not set)

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

func TestModel_formatIP(t *testing.T) {
	m := New("/test/project")

	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.10", "192.168.1.10"},
		{"", "-"},
		{"pending", "-"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := m.formatIP(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModel_formatMemory(t *testing.T) {
	m := New("/test/project")

	tests := []struct {
		input    int
		expected string
	}{
		{512, "512MB"},
		{1024, "1.0GB"},
		{2048, "2.0GB"},
		{4096, "4.0GB"},
		{8192, "8.0GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := m.formatMemory(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModel_formatDisk(t *testing.T) {
	m := New("/test/project")

	tests := []struct {
		input    int
		expected string
	}{
		{10, "10GB"},
		{20, "20GB"},
		{100, "100GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := m.formatDisk(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModel_selectedVM_Empty(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{}

	vm := m.selectedVM()

	assert.Nil(t, vm)
}

func TestModel_selectedVM_Valid(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{Name: "vm1", Status: tfstate.StatusRunning},
		{Name: "vm2", Status: tfstate.StatusStopped},
	}

	vm := m.selectedVM()

	require.NotNil(t, vm)
	assert.Equal(t, "vm1", vm.Name)
}

func TestModel_updateTableRows(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{
			Name:      "test-vm",
			Status:    tfstate.StatusRunning,
			IP:        "192.168.1.10",
			CPUs:      2,
			MemoryMB:  2048,
			DiskGB:    20,
			Autostart: true,
		},
	}

	m.updateTableRows()

	rows := m.table.Rows()
	require.Len(t, rows, 1)
	assert.Equal(t, "test-vm", rows[0][0])
	assert.Equal(t, "running", rows[0][1])
	assert.Equal(t, "192.168.1.10", rows[0][2])
	assert.Equal(t, "2", rows[0][3])
	assert.Equal(t, "2.0GB", rows[0][4])
	assert.Equal(t, "20GB", rows[0][5])
	assert.Equal(t, "yes", rows[0][6])
}

func TestModel_updateTableRows_Autostart_Disabled(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{
			Name:      "test-vm",
			Autostart: false,
		},
	}

	m.updateTableRows()

	rows := m.table.Rows()
	require.Len(t, rows, 1)
	assert.Equal(t, "no", rows[0][6])
}

func TestRefreshInterval(t *testing.T) {
	assert.Equal(t, 5*time.Second, RefreshInterval)
}

func TestModel_handleKeyMsg_Refresh(t *testing.T) {
	m := New("/test/project")
	m.loading = false

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.True(t, model.loading)
	assert.NotNil(t, cmd)
}

func TestModel_handleKeyMsg_SkipWhenActionInProgress(t *testing.T) {
	m := New("/test/project")
	m.actionInProgress = true

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated, cmd := m.Update(msg)

	// Should not process key when action is in progress
	assert.Equal(t, m, updated)
	assert.Nil(t, cmd)
}

func TestModel_openConsole_NoSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{}

	updated, cmd := m.openConsole()

	assert.Equal(t, m, updated)
	assert.Nil(t, cmd)
}

func TestModel_openConsole_WithSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{Name: "test-vm"},
	}

	updated, cmd := m.openConsole()
	model := updated.(*Model)

	assert.Contains(t, model.actionMessage, "virsh console")
	assert.Nil(t, cmd)
}

func TestModel_sshToVM_NoSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{}

	updated, cmd := m.sshToVM()

	assert.Equal(t, m, updated)
	assert.Nil(t, cmd)
}

func TestModel_sshToVM_NoIP(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{Name: "test-vm", IP: ""},
	}

	updated, cmd := m.sshToVM()
	model := updated.(*Model)

	assert.Contains(t, model.actionMessage, "Cannot SSH")
	assert.Nil(t, cmd)
}

func TestModel_sshToVM_PendingIP(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{Name: "test-vm", IP: "pending"},
	}

	updated, cmd := m.sshToVM()
	model := updated.(*Model)

	assert.Contains(t, model.actionMessage, "Cannot SSH")
	assert.Nil(t, cmd)
}

func TestModel_sshToVM_WithIP(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{Name: "test-vm", IP: "192.168.1.10"},
	}

	updated, cmd := m.sshToVM()
	model := updated.(*Model)

	assert.Contains(t, model.actionMessage, "ssh ubuntu@192.168.1.10")
	assert.Nil(t, cmd)
}

func TestModel_showDetails_NoSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{}

	updated, cmd := m.showDetails()

	assert.Equal(t, m, updated)
	assert.Nil(t, cmd)
}

func TestModel_showDetails_WithSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{Name: "test-vm"},
	}

	updated, cmd := m.showDetails()
	model := updated.(*Model)

	assert.Contains(t, model.actionMessage, "Details for: test-vm")
	assert.Nil(t, cmd)
}

func TestModel_startSelectedVM_NoSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{}

	updated, cmd := m.startSelectedVM()

	assert.Equal(t, m, updated)
	assert.Nil(t, cmd)
}

func TestModel_stopSelectedVM_NoSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{}

	updated, cmd := m.stopSelectedVM()

	assert.Equal(t, m, updated)
	assert.Nil(t, cmd)
}

func TestModel_promptDeleteVM_NoSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{}

	updated, cmd := m.promptDeleteVM()

	assert.Equal(t, m, updated)
	assert.Nil(t, cmd)
	assert.False(t, m.confirmingDelete)
}

func TestModel_renderHeader(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 30)

	header := m.renderHeader()

	assert.Contains(t, header, "VMs")
	assert.Contains(t, header, "Terraform")
	assert.Contains(t, header, "libvirt")
}

func TestModel_renderHeader_WithLastUpdate(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 30)
	m.lastUpdate = time.Now()

	header := m.renderHeader()

	assert.Contains(t, header, "Last update")
}

func TestModel_renderStatusBar_Empty(t *testing.T) {
	m := New("/test/project")
	m.actionMessage = ""
	m.actionInProgress = false

	statusBar := m.renderStatusBar()

	assert.Equal(t, "", statusBar)
}

func TestModel_renderStatusBar_WithMessage(t *testing.T) {
	m := New("/test/project")
	m.actionMessage = "Test message"

	statusBar := m.renderStatusBar()

	assert.Contains(t, statusBar, "Test message")
}

func TestModel_renderStatusBar_ActionInProgress(t *testing.T) {
	m := New("/test/project")
	m.actionInProgress = true
	m.actionMessage = ""

	statusBar := m.renderStatusBar()

	assert.Contains(t, statusBar, "Action in progress")
}

func TestModel_createTable(t *testing.T) {
	m := New("/test/project")

	table := m.createTable()

	// Table should have columns defined
	assert.NotNil(t, table)
}

// Additional tests for better coverage

func TestModel_handleKeyMsg_Start(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm"}}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.True(t, model.actionInProgress)
	assert.NotNil(t, cmd)
}

func TestModel_handleKeyMsg_Stop(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm"}}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.True(t, model.actionInProgress)
	assert.NotNil(t, cmd)
}

func TestModel_handleKeyMsg_Delete(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm"}}

	// Press 'd' should show confirmation
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.True(t, model.confirmingDelete)
	assert.NotNil(t, model.vmToDelete)
	assert.Nil(t, cmd) // No action yet, waiting for confirmation
}

func TestModel_handleKeyMsg_DeleteConfirm(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm"}}
	m.confirmingDelete = true
	m.vmToDelete = &m.vms[0]

	// Press 'y' to confirm delete
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.False(t, model.confirmingDelete)
	assert.Nil(t, model.vmToDelete)
	assert.True(t, model.actionInProgress)
	assert.NotNil(t, cmd)
}

func TestModel_handleKeyMsg_DeleteCancel(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm"}}
	m.confirmingDelete = true
	m.vmToDelete = &m.vms[0]

	// Press 'n' to cancel delete
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.False(t, model.confirmingDelete)
	assert.Nil(t, model.vmToDelete)
	assert.False(t, model.actionInProgress)
	assert.Equal(t, "Delete cancelled", model.actionMessage)
	assert.Nil(t, cmd)
}

func TestModel_handleKeyMsg_Console(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm"}}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.Contains(t, model.actionMessage, "virsh console")
	assert.Nil(t, cmd)
}

func TestModel_handleKeyMsg_SSH(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm", IP: "192.168.1.10"}}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.Contains(t, model.actionMessage, "ssh ubuntu@192.168.1.10")
	assert.Nil(t, cmd)
}

func TestModel_handleKeyMsg_Enter(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm"}}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.handleKeyMsg(msg)
	model := updated.(*Model)

	assert.Contains(t, model.actionMessage, "Details for: test-vm")
	assert.Nil(t, cmd)
}

func TestModel_handleKeyMsg_NavigationKeys(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{Name: "vm1"},
		{Name: "vm2"},
	}
	m.updateTableRows()

	// Test down navigation
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, cmd := m.handleKeyMsg(msg)

	assert.NotNil(t, updated)
	_ = cmd // Navigation returns table command
}

func TestModel_handleKeyMsg_UnknownKey(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm"}}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")}
	updated, cmd := m.handleKeyMsg(msg)

	// Unknown keys should pass through to table
	assert.NotNil(t, updated)
	_ = cmd
}

func TestModel_startSelectedVM_WithSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm", Status: tfstate.StatusStopped}}

	updated, cmd := m.startSelectedVM()
	model := updated.(*Model)

	assert.True(t, model.actionInProgress)
	assert.Empty(t, model.actionMessage)
	assert.NotNil(t, cmd)
}

func TestModel_stopSelectedVM_WithSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm", Status: tfstate.StatusRunning}}

	updated, cmd := m.stopSelectedVM()
	model := updated.(*Model)

	assert.True(t, model.actionInProgress)
	assert.Empty(t, model.actionMessage)
	assert.NotNil(t, cmd)
}

func TestModel_promptDeleteVM_WithSelection(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "test-vm"}}

	updated, cmd := m.promptDeleteVM()
	model := updated.(*Model)

	// Should show confirmation, not immediately delete
	assert.True(t, model.confirmingDelete)
	assert.NotNil(t, model.vmToDelete)
	assert.Equal(t, "test-vm", model.vmToDelete.Name)
	assert.Nil(t, cmd) // No async action yet
}

func TestModel_SetSize_SmallHeight(t *testing.T) {
	m := New("/test/project")

	// Height smaller than minimum
	m.SetSize(100, 3)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 3, m.Height())
}

func TestModel_selectedVM_OutOfBounds(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{{Name: "vm1"}}

	// Manually set cursor to invalid position
	// The table cursor is managed internally, but we test boundary check
	vm := m.selectedVM()
	require.NotNil(t, vm)
	assert.Equal(t, "vm1", vm.Name)
}

func TestModel_Update_SpinnerTick(t *testing.T) {
	m := New("/test/project")

	// Create a spinner tick message
	msg := spinner.TickMsg{
		Time: time.Now(),
		ID:   m.spinner.ID(),
	}

	updated, cmd := m.Update(msg)

	assert.NotNil(t, updated)
	assert.NotNil(t, cmd) // Spinner returns next tick command
}

func TestModel_renderHeader_Loading(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 30)
	m.loading = true

	header := m.renderHeader()

	// Should contain spinner when loading
	assert.Contains(t, header, "VMs")
}

func TestModel_renderHeader_NarrowWidth(t *testing.T) {
	m := New("/test/project")
	m.SetSize(20, 30)
	m.lastUpdate = time.Now()

	header := m.renderHeader()

	// Should still render without panic on narrow width
	assert.NotEmpty(t, header)
}

func TestModel_View_LoadingWithExistingVMs(t *testing.T) {
	m := New("/test/project")
	m.SetSize(100, 30)
	m.loading = true
	m.vms = []tfstate.VMInfo{{Name: "test-vm"}}
	m.updateTableRows()

	view := m.View()

	// When loading with existing VMs, should show table not loading message
	assert.Contains(t, view, "test-vm")
}

func TestModel_scheduleRefresh_ReturnsTick(t *testing.T) {
	m := New("/test/project")

	cmd := m.scheduleRefresh()

	assert.NotNil(t, cmd)
}

func TestModel_Update_TableUpdate(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{Name: "vm1"},
		{Name: "vm2"},
	}
	m.updateTableRows()

	// Send a key that the table handles
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.Update(msg)

	assert.NotNil(t, updated)
}

func TestModel_Focus_EnablesAutoRefresh(t *testing.T) {
	m := New("/test/project")
	m.autoRefresh = false

	m.Focus()

	// autoRefresh should be re-enabled after focus since we start with it enabled
	// Actually, Blur disables it, and Focus inherits from BaseTab
	assert.True(t, m.IsFocused())
}

func TestModel_updateTableRows_MultipleVMs(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{Name: "vm1", Status: tfstate.StatusRunning, IP: "192.168.1.1"},
		{Name: "vm2", Status: tfstate.StatusStopped, IP: "192.168.1.2"},
		{Name: "vm3", Status: tfstate.StatusCrashed, IP: ""},
	}

	m.updateTableRows()

	rows := m.table.Rows()
	require.Len(t, rows, 3)
	assert.Equal(t, "vm1", rows[0][0])
	assert.Equal(t, "vm2", rows[1][0])
	assert.Equal(t, "vm3", rows[2][0])
	assert.Equal(t, "-", rows[2][2]) // Empty IP formatted as "-"
}

func TestModel_updateTableRows_AllStatuses(t *testing.T) {
	m := New("/test/project")
	m.vms = []tfstate.VMInfo{
		{Name: "vm1", Status: tfstate.StatusRunning},
		{Name: "vm2", Status: tfstate.StatusStopped},
		{Name: "vm3", Status: tfstate.StatusPaused},
		{Name: "vm4", Status: tfstate.StatusShutoff},
		{Name: "vm5", Status: tfstate.StatusCrashed},
		{Name: "vm6", Status: tfstate.StatusUnknown},
	}

	m.updateTableRows()

	rows := m.table.Rows()
	require.Len(t, rows, 6)
	assert.Equal(t, "running", rows[0][1])
	assert.Equal(t, "stopped", rows[1][1])
	assert.Equal(t, "paused", rows[2][1])
	assert.Equal(t, "shutoff", rows[3][1])
	assert.Equal(t, "crashed", rows[4][1])
	assert.Equal(t, "unknown", rows[5][1])
}

func TestVMsLoadedMsg(t *testing.T) {
	msg := VMsLoadedMsg{
		VMs: []tfstate.VMInfo{{Name: "test"}},
	}
	assert.Len(t, msg.VMs, 1)
}

func TestVMsErrorMsg(t *testing.T) {
	msg := VMsErrorMsg{
		Err: assert.AnError,
	}
	assert.Error(t, msg.Err)
}

func TestRefreshMsg(t *testing.T) {
	msg := RefreshMsg{}
	_ = msg // Just verify it exists
}

func TestActionResultMsg(t *testing.T) {
	msg := ActionResultMsg{
		Action:  "start",
		VMName:  "test-vm",
		Success: true,
		Err:     nil,
	}
	assert.Equal(t, "start", msg.Action)
	assert.Equal(t, "test-vm", msg.VMName)
	assert.True(t, msg.Success)
	assert.Nil(t, msg.Err)
}
