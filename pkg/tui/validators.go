package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// validateRequired returns a validator that ensures a field is not empty.
func validateRequired(field string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
}

// validateHostname validates a hostname according to RFC 1123.
func validateHostname(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("hostname is required")
	}

	// RFC 1123 hostname validation
	hostnameRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	if !hostnameRegex.MatchString(strings.ToLower(s)) {
		return fmt.Errorf("invalid hostname: must be alphanumeric with optional hyphens")
	}

	return nil
}

// validateSSHKey validates an SSH public key format.
func validateSSHKey(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("SSH public key is required")
	}

	// Basic SSH key validation
	validPrefixes := []string{"ssh-rsa", "ssh-ed25519", "ssh-ecdsa", "ecdsa-sha2"}
	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(s, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return fmt.Errorf("invalid SSH key: must start with ssh-rsa, ssh-ed25519, or ssh-ecdsa")
	}

	return nil
}

// validateEmail validates an email address format.
func validateEmail(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("email is required")
	}

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(s) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// validateISOPath validates that a path points to a valid ISO file.
func validateISOPath(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("source ISO path is required")
	}

	// Expand home directory if path starts with ~/
	if strings.HasPrefix(s, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			s = filepath.Join(home, s[2:])
		}
	}

	// Check file exists
	info, err := os.Stat(s)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", s)
	}
	if err != nil {
		return fmt.Errorf("cannot access file: %v", err)
	}

	// Check it's a file, not a directory
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	// Check file extension
	if !strings.HasSuffix(strings.ToLower(s), ".iso") {
		return fmt.Errorf("file must have .iso extension")
	}

	return nil
}

// detectUbuntuVersion extracts the Ubuntu version from an ISO filename.
// Supports patterns like: ubuntu-24.04-live-server-amd64.iso, ubuntu-22.04.3-live-server-amd64.iso
func detectUbuntuVersion(isoPath string) string {
	filename := filepath.Base(isoPath)

	// Try to match version pattern (e.g., 24.04, 22.04.3)
	versionRegex := regexp.MustCompile(`(\d{2}\.\d{2}(?:\.\d+)?)`)
	if matches := versionRegex.FindStringSubmatch(filename); len(matches) > 1 {
		// Return just major.minor (e.g., 24.04 from 22.04.3)
		version := matches[1]
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1]
		}
		return version
	}

	// Default to 24.04 if detection fails
	return "24.04"
}
