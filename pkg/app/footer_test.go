package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalBindings(t *testing.T) {
	bindings := GlobalBindings()

	assert.NotEmpty(t, bindings)
	// Should contain common bindings
	found := false
	for _, b := range bindings {
		if b == "[q] quit" {
			found = true
			break
		}
	}
	assert.True(t, found, "should contain quit binding")
}

func TestRenderFooter(t *testing.T) {
	tabBindings := []string{"[s] start", "[S] stop"}

	footer := renderFooter(tabBindings, 100)

	// Should contain tab bindings
	assert.Contains(t, footer, "start")
	assert.Contains(t, footer, "stop")

	// Should contain global bindings
	assert.Contains(t, footer, "quit")
}

func TestRenderFooter_NoTabBindings(t *testing.T) {
	footer := renderFooter(nil, 100)

	// Should still contain global bindings
	assert.Contains(t, footer, "quit")
	assert.Contains(t, footer, "help")
}

func TestFormatBinding(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "standard binding",
			input:    "[q] quit",
			contains: []string{"q", "quit"},
		},
		{
			name:     "multi-char key",
			input:    "[Tab] next",
			contains: []string{"Tab", "next"},
		},
		{
			name:     "no brackets",
			input:    "plain text",
			contains: []string{"plain text"},
		},
		{
			name:     "empty",
			input:    "",
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBinding(tt.input)
			for _, c := range tt.contains {
				assert.Contains(t, result, c)
			}
		})
	}
}

func TestRenderKeyBindings(t *testing.T) {
	bindings := []string{"[s] start", "[S] stop", "[d] delete"}

	result := RenderKeyBindings(bindings)

	assert.Contains(t, result, "start")
	assert.Contains(t, result, "stop")
	assert.Contains(t, result, "delete")
}

func TestRenderKeyBindings_Empty(t *testing.T) {
	result := RenderKeyBindings(nil)
	assert.Equal(t, "", result)
}

func TestFormatBindingHelp(t *testing.T) {
	help := BindingHelp{Key: "s", Description: "start"}

	result := FormatBindingHelp(help)

	assert.Contains(t, result, "s")
	assert.Contains(t, result, "start")
}

func TestFormatBinding_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing close bracket", "[key action"},
		{"only brackets", "[]"},
		{"short string", "["},
		{"bracket at end", "text["},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := formatBinding(tt.input)
			assert.NotEmpty(t, result)
		})
	}
}
