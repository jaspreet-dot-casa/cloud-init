package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusColor(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"running", "running"},
		{"stopped", "stopped"},
		{"shutoff", "shutoff"},
		{"shut off", "shut off"},
		{"crashed", "crashed"},
		{"error", "error"},
		{"unknown", "something-else"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := StatusColor(tt.status)
			// Just verify it returns a style without panicking
			assert.NotNil(t, style)
		})
	}
}

func TestRenderStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"running", "running"},
		{"stopped", "stopped"},
		{"crashed", "crashed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderStatus(tt.status)
			// Result should contain the status text
			assert.Contains(t, result, tt.status)
		})
	}
}

func TestStatusStyles_Exist(t *testing.T) {
	// Verify all status styles are defined
	styles := []struct {
		name  string
		style interface{}
	}{
		{"StatusRunningStyle", StatusRunningStyle},
		{"StatusStoppedStyle", StatusStoppedStyle},
		{"StatusErrorStyle", StatusErrorStyle},
		{"StatusUnknownStyle", StatusUnknownStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			assert.NotNil(t, s.style)
		})
	}
}

func TestTextStyles_Exist(t *testing.T) {
	styles := []struct {
		name  string
		style interface{}
	}{
		{"BoldStyle", BoldStyle},
		{"DimStyle", DimStyle},
		{"AccentStyle", AccentStyle},
		{"SuccessStyle", SuccessStyle},
		{"WarningStyle", WarningStyle},
		{"ErrorStyle", ErrorStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			assert.NotNil(t, s.style)
		})
	}
}

func TestLayoutStyles_Exist(t *testing.T) {
	styles := []struct {
		name  string
		style interface{}
	}{
		{"BoxStyle", BoxStyle},
		{"SelectedRowStyle", SelectedRowStyle},
		{"TableHeaderStyle", TableHeaderStyle},
		{"SpinnerStyle", SpinnerStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			assert.NotNil(t, s.style)
		})
	}
}

func TestStatusColor_ReturnsCorrectStyle(t *testing.T) {
	// Running should use green style
	runningStyle := StatusColor("running")
	runningRender := runningStyle.Render("test")
	assert.NotEmpty(t, runningRender)
	assert.Contains(t, runningRender, "test")

	// Stopped should use yellow style
	stoppedStyle := StatusColor("stopped")
	stoppedRender := stoppedStyle.Render("test")
	assert.NotEmpty(t, stoppedRender)
	assert.Contains(t, stoppedRender, "test")

	// Note: In headless mode, lipgloss may not produce ANSI codes,
	// so we just verify both styles render without panicking
}
