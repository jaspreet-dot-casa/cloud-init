package doctor

import (
	"errors"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFixer(t *testing.T) {
	fixer := NewFixer()
	assert.NotNil(t, fixer)
	assert.NotNil(t, fixer.executor)
}

func TestNewFixerWithExecutor(t *testing.T) {
	mockExec := &MockExecutor{}
	fixer := NewFixerWithExecutor(mockExec)
	assert.NotNil(t, fixer)
	assert.Equal(t, mockExec, fixer.executor)
}

func TestFixer_RunFix_Success(t *testing.T) {
	mockExec := &MockExecutor{
		CombinedOutputFunc: func(name string, args ...string) ([]byte, error) {
			assert.Equal(t, "sh", name)
			assert.Equal(t, []string{"-c", "echo hello"}, args)
			return []byte("hello\n"), nil
		},
	}

	fixer := NewFixerWithExecutor(mockExec)
	fix := &FixCommand{
		Command:     "echo hello",
		Description: "Test command",
	}

	err := fixer.RunFix(fix)
	assert.NoError(t, err)
}

func TestFixer_RunFix_Failure(t *testing.T) {
	mockExec := &MockExecutor{
		CombinedOutputFunc: func(name string, args ...string) ([]byte, error) {
			return []byte("command not found"), errors.New("exit status 127")
		},
	}

	fixer := NewFixerWithExecutor(mockExec)
	fix := &FixCommand{
		Command:     "nonexistent-command",
		Description: "Test command",
	}

	err := fixer.RunFix(fix)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fix failed")
	assert.Contains(t, err.Error(), "command not found")
}

func TestFixer_RunFix_NilFix(t *testing.T) {
	fixer := NewFixer()

	err := fixer.RunFix(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no fix command available")
}

func TestFixer_CopyToClipboard_NilFix(t *testing.T) {
	fixer := NewFixer()

	err := fixer.CopyToClipboard(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no fix command available")
}

func TestFixer_CopyToClipboard_Success(t *testing.T) {
	// This test only runs on darwin where pbcopy is available
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping clipboard test on non-darwin platform")
	}

	fixer := NewFixer()
	fix := &FixCommand{
		Command:     "test command",
		Description: "Test",
	}

	// This will actually use the system clipboard
	err := fixer.CopyToClipboard(fix)
	assert.NoError(t, err)
}

func TestGetFixCommand_AllPlatforms(t *testing.T) {
	tests := []struct {
		toolID      string
		platform    string
		expectNil   bool
		expectSudo  bool
		containsCmd string
	}{
		// Terraform
		{IDTerraform, PlatformDarwin, false, false, "brew install terraform"},
		{IDTerraform, PlatformLinux, false, true, "apt"},
		{IDTerraform, "windows", true, false, ""},

		// Multipass
		{IDMultipass, PlatformDarwin, false, false, "brew install --cask multipass"},
		{IDMultipass, PlatformLinux, false, true, "snap install multipass"},

		// Xorriso
		{IDXorriso, PlatformDarwin, false, false, "brew install xorriso"},
		{IDXorriso, PlatformLinux, false, true, "apt install"},

		// Libvirt (Linux only)
		{IDLibvirt, PlatformDarwin, true, false, ""},
		{IDLibvirt, PlatformLinux, false, true, "libvirt"},

		// Virsh (Linux only)
		{IDVirsh, PlatformDarwin, true, false, ""},
		{IDVirsh, PlatformLinux, false, true, "libvirt-clients"},

		// QEMU/KVM (Linux only)
		{IDQemuKVM, PlatformDarwin, true, false, ""},
		{IDQemuKVM, PlatformLinux, false, true, "qemu-kvm"},

		// Unknown tool
		{"unknown-tool", PlatformDarwin, true, false, ""},
		{"unknown-tool", PlatformLinux, true, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.toolID+"_"+tt.platform, func(t *testing.T) {
			fix := GetFixCommand(tt.toolID, tt.platform)

			if tt.expectNil {
				assert.Nil(t, fix)
			} else {
				assert.NotNil(t, fix)
				assert.Equal(t, tt.expectSudo, fix.Sudo)
				assert.Contains(t, fix.Command, tt.containsCmd)
				assert.NotEmpty(t, fix.Description)
				assert.Equal(t, tt.platform, fix.Platform)
			}
		})
	}
}

func TestFixCommand_Properties(t *testing.T) {
	fix := GetFixCommand(IDTerraform, PlatformDarwin)

	assert.NotNil(t, fix)
	assert.Equal(t, "Install via Homebrew", fix.Description)
	assert.Equal(t, "brew install terraform", fix.Command)
	assert.False(t, fix.Sudo)
	assert.Equal(t, PlatformDarwin, fix.Platform)
}

func TestFixCommand_LinuxSudo(t *testing.T) {
	fix := GetFixCommand(IDLibvirt, PlatformLinux)

	assert.NotNil(t, fix)
	assert.True(t, fix.Sudo)
	assert.Contains(t, fix.Command, "sudo")
}
