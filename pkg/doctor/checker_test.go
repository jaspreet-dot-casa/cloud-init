package doctor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockExecutor is a mock command executor for testing.
type MockExecutor struct {
	LookPathFunc       func(file string) (string, error)
	RunFunc            func(name string, args ...string) (string, error)
	CombinedOutputFunc func(name string, args ...string) ([]byte, error)
	FileExistsFunc     func(path string) bool
}

func (m *MockExecutor) LookPath(file string) (string, error) {
	if m.LookPathFunc != nil {
		return m.LookPathFunc(file)
	}
	return "/usr/bin/" + file, nil
}

func (m *MockExecutor) Run(name string, args ...string) (string, error) {
	if m.RunFunc != nil {
		return m.RunFunc(name, args...)
	}
	return "1.0.0", nil
}

func (m *MockExecutor) CombinedOutput(name string, args ...string) ([]byte, error) {
	if m.CombinedOutputFunc != nil {
		return m.CombinedOutputFunc(name, args...)
	}
	return []byte("success"), nil
}

func (m *MockExecutor) FileExists(path string) bool {
	if m.FileExistsFunc != nil {
		return m.FileExistsFunc(path)
	}
	return true
}

func TestCheckTerraform_Installed(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			if file == "terraform" {
				return "/usr/local/bin/terraform", nil
			}
			return "", errors.New("not found")
		},
		RunFunc: func(name string, args ...string) (string, error) {
			return "Terraform v1.5.7\non linux_amd64", nil
		},
	}

	check := CheckTerraform(exec)

	assert.Equal(t, IDTerraform, check.ID)
	assert.Equal(t, "Terraform", check.Name)
	assert.Equal(t, StatusOK, check.Status)
	assert.Equal(t, "1.5.7", check.Message)
}

func TestCheckTerraform_NotInstalled(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			return "", errors.New("not found")
		},
	}

	check := CheckTerraform(exec)

	assert.Equal(t, StatusMissing, check.Status)
	assert.Equal(t, "not installed", check.Message)
	assert.NotNil(t, check.FixCommand)
}

func TestCheckMultipass_Installed(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			if file == "multipass" {
				return "/usr/local/bin/multipass", nil
			}
			return "", errors.New("not found")
		},
		RunFunc: func(name string, args ...string) (string, error) {
			return "multipass   1.12.2+mac\nmultipassd  1.12.2+mac", nil
		},
	}

	check := CheckMultipass(exec)

	assert.Equal(t, StatusOK, check.Status)
	assert.Equal(t, "1.12.2", check.Message)
}

func TestCheckXorriso_Installed(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			if file == "xorriso" {
				return "/usr/bin/xorriso", nil
			}
			return "", errors.New("not found")
		},
		RunFunc: func(name string, args ...string) (string, error) {
			return "xorriso 1.5.4 : RockRidge filesystem manipulator", nil
		},
	}

	check := CheckXorriso(exec)

	assert.Equal(t, StatusOK, check.Status)
	assert.Equal(t, "1.5.4", check.Message)
}

func TestCheckCloudImage_Exists(t *testing.T) {
	exec := &MockExecutor{
		FileExistsFunc: func(path string) bool {
			return path == "/var/lib/libvirt/images/test.img"
		},
	}

	check := CheckCloudImage(exec, "/var/lib/libvirt/images/test.img")

	assert.Equal(t, StatusOK, check.Status)
	assert.Contains(t, check.Message, "/var/lib/libvirt/images/test.img")
}

func TestCheckCloudImage_Missing(t *testing.T) {
	exec := &MockExecutor{
		FileExistsFunc: func(path string) bool {
			return false
		},
	}

	check := CheckCloudImage(exec, "/var/lib/libvirt/images/missing.img")

	assert.Equal(t, StatusMissing, check.Status)
	assert.Contains(t, check.Message, "no image")
}

func TestChecker_CheckGroup(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			if file == "multipass" {
				return "/usr/local/bin/multipass", nil
			}
			return "", errors.New("not found")
		},
		RunFunc: func(name string, args ...string) (string, error) {
			return "multipass 1.12.2", nil
		},
	}

	checker := NewCheckerWithExecutor(exec)
	group := checker.CheckGroup(GroupMultipass)

	assert.Equal(t, GroupMultipass, group.ID)
	assert.Equal(t, "Multipass", group.Name)
	require.Len(t, group.Checks, 1)
	assert.Equal(t, StatusOK, group.Checks[0].Status)
}

func TestChecker_GetSummary(t *testing.T) {
	groups := []CheckGroup{
		{
			ID: GroupMultipass,
			Checks: []Check{
				{ID: "test1", Status: StatusOK},
				{ID: "test2", Status: StatusMissing},
				{ID: "test3", Status: StatusWarning},
			},
		},
	}

	checker := NewChecker()
	summary := checker.GetSummary(groups)

	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 1, summary.OK)
	assert.Equal(t, 1, summary.Missing)
	assert.Equal(t, 1, summary.Warnings)
	assert.Equal(t, 0, summary.Errors)
}

func TestChecker_HasIssues(t *testing.T) {
	tests := []struct {
		name     string
		groups   []CheckGroup
		expected bool
	}{
		{
			name: "no issues",
			groups: []CheckGroup{
				{Checks: []Check{{Status: StatusOK}, {Status: StatusOK}}},
			},
			expected: false,
		},
		{
			name: "has missing",
			groups: []CheckGroup{
				{Checks: []Check{{Status: StatusOK}, {Status: StatusMissing}}},
			},
			expected: true,
		},
		{
			name: "has error",
			groups: []CheckGroup{
				{Checks: []Check{{Status: StatusOK}, {Status: StatusError}}},
			},
			expected: true,
		},
		{
			name: "warning only",
			groups: []CheckGroup{
				{Checks: []Check{{Status: StatusOK}, {Status: StatusWarning}}},
			},
			expected: false,
		},
	}

	checker := NewChecker()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.HasIssues(tt.groups)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFixCommand(t *testing.T) {
	tests := []struct {
		toolID   string
		platform string
		wantNil  bool
	}{
		{IDTerraform, PlatformDarwin, false},
		{IDTerraform, PlatformLinux, false},
		{IDMultipass, PlatformDarwin, false},
		{IDMultipass, PlatformLinux, false},
		{IDXorriso, PlatformDarwin, false},
		{IDXorriso, PlatformLinux, false},
		{IDLibvirt, PlatformDarwin, true},  // Not available on macOS
		{IDLibvirt, PlatformLinux, false},
		{"unknown", PlatformDarwin, true},
	}

	for _, tt := range tests {
		t.Run(tt.toolID+"_"+tt.platform, func(t *testing.T) {
			fix := GetFixCommand(tt.toolID, tt.platform)
			if tt.wantNil {
				assert.Nil(t, fix)
			} else {
				assert.NotNil(t, fix)
				assert.NotEmpty(t, fix.Command)
				assert.NotEmpty(t, fix.Description)
			}
		})
	}
}

func TestCheckStatus_String(t *testing.T) {
	assert.Equal(t, "ok", StatusOK.String())
	assert.Equal(t, "missing", StatusMissing.String())
	assert.Equal(t, "error", StatusError.String())
	assert.Equal(t, "warning", StatusWarning.String())
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		output   string
		expected string
	}{
		{"Terraform v1.5.7", "1.5.7"},
		{"version 2.3.4", "2.3.4"},
		{"tool 1.2.3-beta", "1.2.3-beta"},
		{"no version here", ""},
	}

	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			result := extractVersion(tt.output, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// MockEnvGetter is a mock environment variable getter for testing.
type MockEnvGetter struct {
	Vars map[string]string
}

func (m *MockEnvGetter) Getenv(key string) string {
	if m.Vars == nil {
		return ""
	}
	return m.Vars[key]
}

func TestCheckGhostty_RunningInGhostty_TermProgram(t *testing.T) {
	exec := &MockExecutor{}
	env := &MockEnvGetter{
		Vars: map[string]string{
			"TERM_PROGRAM": "ghostty",
			"TERM":         "xterm-256color",
		},
	}

	check := CheckGhostty(exec, env)

	assert.Equal(t, IDGhostty, check.ID)
	assert.Equal(t, "Ghostty", check.Name)
	assert.Equal(t, StatusOK, check.Status)
	assert.Equal(t, "running in Ghostty", check.Message)
}

func TestCheckGhostty_RunningInGhostty_TermVar(t *testing.T) {
	exec := &MockExecutor{}
	env := &MockEnvGetter{
		Vars: map[string]string{
			"TERM_PROGRAM": "",
			"TERM":         "ghostty",
		},
	}

	check := CheckGhostty(exec, env)

	assert.Equal(t, StatusOK, check.Status)
	assert.Equal(t, "running in Ghostty", check.Message)
}

func TestCheckGhostty_RunningInGhostty_TermVarXterm(t *testing.T) {
	exec := &MockExecutor{}
	env := &MockEnvGetter{
		Vars: map[string]string{
			"TERM_PROGRAM": "",
			"TERM":         "xterm-ghostty",
		},
	}

	check := CheckGhostty(exec, env)

	assert.Equal(t, StatusOK, check.Status)
	assert.Equal(t, "running in Ghostty", check.Message)
}

func TestCheckGhostty_InstalledButNotCurrentTerminal(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			if file == "ghostty" {
				return "/usr/local/bin/ghostty", nil
			}
			return "", errors.New("not found")
		},
	}
	env := &MockEnvGetter{
		Vars: map[string]string{
			"TERM_PROGRAM": "iTerm.app",
			"TERM":         "xterm-256color",
		},
	}

	check := CheckGhostty(exec, env)

	assert.Equal(t, StatusWarning, check.Status)
	assert.Equal(t, "installed (not current terminal)", check.Message)
}

func TestCheckGhostty_NotInstalled(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			return "", errors.New("not found")
		},
	}
	env := &MockEnvGetter{
		Vars: map[string]string{
			"TERM_PROGRAM": "Terminal.app",
			"TERM":         "xterm-256color",
		},
	}

	check := CheckGhostty(exec, env)

	assert.Equal(t, StatusMissing, check.Status)
	assert.Equal(t, "not installed", check.Message)
	assert.NotNil(t, check.FixCommand)
}

func TestCheckGhostty_EmptyEnvVars(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			return "", errors.New("not found")
		},
	}
	env := &MockEnvGetter{
		Vars: map[string]string{},
	}

	check := CheckGhostty(exec, env)

	assert.Equal(t, StatusMissing, check.Status)
	assert.Equal(t, "not installed", check.Message)
}

func TestCheckGhostty_CaseInsensitive(t *testing.T) {
	exec := &MockExecutor{}
	env := &MockEnvGetter{
		Vars: map[string]string{
			"TERM_PROGRAM": "",
			"TERM":         "GHOSTTY",
		},
	}

	check := CheckGhostty(exec, env)

	assert.Equal(t, StatusOK, check.Status)
	assert.Equal(t, "running in Ghostty", check.Message)
}

func TestGetFixCommand_Ghostty(t *testing.T) {
	tests := []struct {
		platform string
		wantNil  bool
	}{
		{PlatformDarwin, false},
		{PlatformLinux, false},
	}

	for _, tt := range tests {
		t.Run("ghostty_"+tt.platform, func(t *testing.T) {
			fix := GetFixCommand(IDGhostty, tt.platform)
			if tt.wantNil {
				assert.Nil(t, fix)
			} else {
				require.NotNil(t, fix)
				assert.NotEmpty(t, fix.Command)
				assert.NotEmpty(t, fix.Description)
				assert.Equal(t, tt.platform, fix.Platform)
			}
		})
	}
}

func TestChecker_CheckGroup_Terminal(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			return "", errors.New("not found")
		},
	}

	checker := NewCheckerWithExecutor(exec)
	group := checker.CheckGroup(GroupTerminal)

	assert.Equal(t, GroupTerminal, group.ID)
	assert.Equal(t, "Terminal", group.Name)
	require.Len(t, group.Checks, 1)
	assert.Equal(t, IDGhostty, group.Checks[0].ID)
}

func TestNewCheckerWithEnv(t *testing.T) {
	exec := &MockExecutor{}
	env := &MockEnvGetter{
		Vars: map[string]string{
			"TERM_PROGRAM": "ghostty",
		},
	}

	checker := NewCheckerWithEnv(exec, env)
	check := checker.GetCheck(IDGhostty)

	assert.Equal(t, StatusOK, check.Status)
	assert.Equal(t, "running in Ghostty", check.Message)
}

func TestGetAllGroupIDs_IncludesTerminal(t *testing.T) {
	groupIDs := GetAllGroupIDs()

	assert.Contains(t, groupIDs, GroupTerminal)
	assert.Contains(t, groupIDs, GroupTerraform)
	assert.Contains(t, groupIDs, GroupMultipass)
	assert.Contains(t, groupIDs, GroupISO)
}

func TestGetGroupDefinition_Terminal(t *testing.T) {
	def, ok := GetGroupDefinition(GroupTerminal)

	require.True(t, ok)
	assert.Equal(t, "Terminal", def.Name)
	assert.Contains(t, def.CheckIDs, IDGhostty)
	assert.Equal(t, "", def.Platform) // Works on both platforms
}

func TestCheckGhostty_TermProgramPriority(t *testing.T) {
	// When TERM_PROGRAM is set to ghostty, it should be OK
	// even if TERM is something else
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			return "", errors.New("not found")
		},
	}
	env := &MockEnvGetter{
		Vars: map[string]string{
			"TERM_PROGRAM": "ghostty",
			"TERM":         "xterm-256color",
		},
	}

	check := CheckGhostty(exec, env)

	assert.Equal(t, StatusOK, check.Status)
	assert.Equal(t, "running in Ghostty", check.Message)
}

func TestCheckGhostty_NilEnvVarsMap(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			return "", errors.New("not found")
		},
	}
	// nil Vars map should use empty strings and result in "not installed"
	env := &MockEnvGetter{Vars: nil}

	check := CheckGhostty(exec, env)

	assert.Equal(t, StatusMissing, check.Status)
}

func TestRealEnvGetter(t *testing.T) {
	getter := &RealEnvGetter{}

	// This should return empty string for a non-existent env var
	result := getter.Getenv("SOME_DEFINITELY_NOT_SET_VAR_12345")
	assert.Equal(t, "", result)
}

func TestChecker_CheckAllAsync_IncludesTerminal(t *testing.T) {
	exec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			return "", errors.New("not found")
		},
	}

	checker := NewCheckerWithExecutor(exec)
	groups := checker.CheckAllAsync()

	// Find the Terminal group
	var terminalGroup *CheckGroup
	for i := range groups {
		if groups[i].ID == GroupTerminal {
			terminalGroup = &groups[i]
			break
		}
	}

	require.NotNil(t, terminalGroup, "Terminal group should be included")
	assert.Equal(t, "Terminal", terminalGroup.Name)
	require.Len(t, terminalGroup.Checks, 1)
	assert.Equal(t, IDGhostty, terminalGroup.Checks[0].ID)
}

func TestCheckGhostty_Description(t *testing.T) {
	exec := &MockExecutor{}
	env := &MockEnvGetter{
		Vars: map[string]string{
			"TERM_PROGRAM": "ghostty",
		},
	}

	check := CheckGhostty(exec, env)

	assert.Equal(t, "Modern GPU-accelerated terminal", check.Description)
}

func TestGhosttyFixCommand_DarwinUsesBrewCask(t *testing.T) {
	fix := GetFixCommand(IDGhostty, PlatformDarwin)

	require.NotNil(t, fix)
	assert.Contains(t, fix.Command, "brew install --cask ghostty")
	assert.False(t, fix.Sudo)
}

func TestGhosttyFixCommand_LinuxUsesPPA(t *testing.T) {
	fix := GetFixCommand(IDGhostty, PlatformLinux)

	require.NotNil(t, fix)
	assert.Contains(t, fix.Command, "ppa:mkasberg/ghostty-ubuntu")
	assert.True(t, fix.Sudo)
}
