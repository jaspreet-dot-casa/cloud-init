package doctor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockExecutor is a mock command executor for testing.
type MockExecutor struct {
	LookPathFunc   func(file string) (string, error)
	RunFunc        func(name string, args ...string) (string, error)
	FileExistsFunc func(path string) bool
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
