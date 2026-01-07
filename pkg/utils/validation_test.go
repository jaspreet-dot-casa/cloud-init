package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConfigName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid simple name",
			input:     "my-config",
			wantError: false,
		},
		{
			name:      "valid with spaces",
			input:     "my config name",
			wantError: false,
		},
		{
			name:      "valid with underscores",
			input:     "my_config_name",
			wantError: false,
		},
		{
			name:      "valid with numbers",
			input:     "config123",
			wantError: false,
		},
		{
			name:      "valid mixed",
			input:     "My Config-Name_123",
			wantError: false,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "only whitespace",
			input:     "   ",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "too long",
			input:     strings.Repeat("a", 51),
			wantError: true,
			errorMsg:  "cannot exceed",
		},
		{
			name:      "path traversal dots",
			input:     "../etc/passwd",
			wantError: true,
			errorMsg:  "invalid characters",
		},
		{
			name:      "path traversal slash",
			input:     "config/name",
			wantError: true,
			errorMsg:  "invalid characters",
		},
		{
			name:      "path traversal backslash",
			input:     "config\\name",
			wantError: true,
			errorMsg:  "invalid characters",
		},
		{
			name:      "special characters",
			input:     "config<script>",
			wantError: true,
			errorMsg:  "can only contain",
		},
		{
			name:      "starts with hyphen",
			input:     "-config",
			wantError: true,
			errorMsg:  "can only contain",
		},
		{
			name:      "starts with number is valid",
			input:     "1config",
			wantError: false,
		},
		{
			name:      "quotes",
			input:     `"config"`,
			wantError: true,
			errorMsg:  "can only contain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigName(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeConfigName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trim whitespace",
			input:    "  my config  ",
			expected: "my config",
		},
		{
			name:     "collapse multiple spaces",
			input:    "my    config    name",
			expected: "my config name",
		},
		{
			name:     "truncate long name",
			input:    strings.Repeat("a", 60),
			expected: strings.Repeat("a", 50),
		},
		{
			name:     "normal name unchanged",
			input:    "my-config",
			expected: "my-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeConfigName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
