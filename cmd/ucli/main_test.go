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
	assert.Contains(t, output, "packages")
	// create and build commands have been removed
	assert.NotContains(t, output, "create")
	assert.NotContains(t, output, "build")
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

func TestPackagesCmd(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"packages"})

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
			name:    "packages help",
			args:    []string{"packages", "--help"},
			expects: []string{"packages", "cloud-init"},
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
