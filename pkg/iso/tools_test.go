package iso

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewToolChain(t *testing.T) {
	tc := NewToolChain()

	assert.NotNil(t, tc)
	assert.Equal(t, runtime.GOOS, tc.Platform)
	assert.Empty(t, tc.XorrisoPath)
}

func TestToolChain_Available(t *testing.T) {
	t.Run("not available when path empty", func(t *testing.T) {
		tc := &ToolChain{}
		assert.False(t, tc.Available())
	})

	t.Run("available when path set", func(t *testing.T) {
		tc := &ToolChain{XorrisoPath: "/usr/bin/xorriso"}
		assert.True(t, tc.Available())
	})
}

func TestToolChain_InstallInstructions(t *testing.T) {
	tests := []struct {
		platform string
		contains string
	}{
		{"darwin", "brew install xorriso"},
		{"linux", "apt install xorriso"},
		{"windows", "install xorriso"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			tc := &ToolChain{Platform: tt.platform}
			instructions := tc.InstallInstructions()
			assert.Contains(t, instructions, tt.contains)
		})
	}
}

func TestToolChain_Detect(t *testing.T) {
	tc := NewToolChain()
	err := tc.Detect()

	// This test depends on whether xorriso is installed
	// If installed, it should succeed; if not, it should fail with a clear message
	if err != nil {
		assert.Contains(t, err.Error(), "xorriso")
		assert.False(t, tc.Available())
	} else {
		assert.True(t, tc.Available())
		assert.NotEmpty(t, tc.XorrisoPath)
	}
}

func TestToolChain_XorrisoVersion(t *testing.T) {
	t.Run("error when not detected", func(t *testing.T) {
		tc := &ToolChain{}
		_, err := tc.XorrisoVersion()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not detected")
	})

	t.Run("returns version when available", func(t *testing.T) {
		tc := NewToolChain()
		if err := tc.Detect(); err != nil {
			t.Skip("xorriso not installed")
		}

		version, err := tc.XorrisoVersion()
		assert.NoError(t, err)
		assert.Contains(t, version, "xorriso")
	})
}
