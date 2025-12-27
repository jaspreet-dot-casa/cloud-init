package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCmd(t *testing.T) {
	rootCmd := newRootCmd()

	assert.Equal(t, "ucli", rootCmd.Use)
	assert.Equal(t, "Ubuntu Cloud-Init CLI Tool", rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)
}

func TestRootCmdHelp(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"--help"})

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "ucli")
	assert.Contains(t, output, "generate")
	assert.Contains(t, output, "packages")
	assert.Contains(t, output, "validate")
	assert.Contains(t, output, "build")
}

func TestRootCmdVersion(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"--version"})

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "ucli version")
}

func TestGenerateCmd(t *testing.T) {
	// Skip this test as generate command requires an interactive TTY
	// The TUI forms are tested separately in pkg/tui/form_test.go
	t.Skip("generate command requires interactive TTY")
}

func TestPackagesCmd(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"packages"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestValidateCmd(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"validate"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestBuildCmd(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"build"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestSubcommandHelp(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		expects []string
	}{
		{
			name:    "generate help",
			args:    []string{"generate", "--help"},
			expects: []string{"TUI", "cloud-init"},
		},
		{
			name:    "packages help",
			args:    []string{"packages", "--help"},
			expects: []string{"packages", "cloud-init"},
		},
		{
			name:    "validate help",
			args:    []string{"validate", "--help"},
			expects: []string{"config.env", "secrets.env"},
		},
		{
			name:    "build help",
			args:    []string{"build", "--help"},
			expects: []string{"Non-interactive", "config.env"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := newRootCmd()
			rootCmd.SetArgs(tt.args)

			var buf bytes.Buffer
			rootCmd.SetOut(&buf)

			err := rootCmd.Execute()
			require.NoError(t, err)

			output := buf.String()
			for _, expect := range tt.expects {
				assert.Contains(t, output, expect)
			}
		})
	}
}
