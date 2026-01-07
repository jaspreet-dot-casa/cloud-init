package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	// MaxConfigNameLength is the maximum length for a config name.
	MaxConfigNameLength = 50
	// MinConfigNameLength is the minimum length for a config name.
	MinConfigNameLength = 1
)

// validConfigNamePattern matches alphanumeric, hyphens, underscores, and spaces.
var validConfigNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-_\s]*$`)

// multipleSpacesPattern matches one or more consecutive whitespace characters.
var multipleSpacesPattern = regexp.MustCompile(`\s+`)

// ValidateConfigName validates a configuration name.
// Returns an error if the name is invalid.
func ValidateConfigName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return fmt.Errorf("config name cannot be empty")
	}

	if utf8.RuneCountInString(name) > MaxConfigNameLength {
		return fmt.Errorf("config name cannot exceed %d characters", MaxConfigNameLength)
	}

	// Check for path traversal attempts before regex check
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("config name contains invalid characters")
	}

	if !validConfigNamePattern.MatchString(name) {
		return fmt.Errorf("config name can only contain letters, numbers, hyphens, underscores, and spaces")
	}

	return nil
}

// SanitizeConfigName cleans up a config name for safe use.
func SanitizeConfigName(name string) string {
	name = strings.TrimSpace(name)

	// Replace multiple spaces with single space
	name = multipleSpacesPattern.ReplaceAllString(name, " ")

	// Truncate if too long
	if utf8.RuneCountInString(name) > MaxConfigNameLength {
		runes := []rune(name)
		name = string(runes[:MaxConfigNameLength])
	}

	return name
}
