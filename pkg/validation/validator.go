// Package validation provides configuration file validation for ucli.
package validation

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Severity represents the severity of a validation issue.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue represents a validation issue found in a config file.
type Issue struct {
	File     string   `json:"file"`
	Field    string   `json:"field,omitempty"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
}

// Result holds all validation results.
type Result struct {
	Issues []Issue `json:"issues"`
}

// HasErrors returns true if there are any error-level issues.
func (r *Result) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of error-level issues.
func (r *Result) ErrorCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warning-level issues.
func (r *Result) WarningCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarning {
			count++
		}
	}
	return count
}

// Validator validates configuration files.
type Validator struct {
	ProjectRoot string
}

// NewValidator creates a new Validator.
func NewValidator(projectRoot string) *Validator {
	return &Validator{ProjectRoot: projectRoot}
}

// ValidateAll validates all configuration files and returns the result.
func (v *Validator) ValidateAll() *Result {
	result := &Result{Issues: []Issue{}}

	// Validate secrets.env
	secretsPath := filepath.Join(v.ProjectRoot, "cloud-init", "secrets.env")
	secretsIssues := v.ValidateSecretsEnv(secretsPath)
	result.Issues = append(result.Issues, secretsIssues...)

	// Validate config.env
	configPath := filepath.Join(v.ProjectRoot, "config.env")
	configIssues := v.ValidateConfigEnv(configPath)
	result.Issues = append(result.Issues, configIssues...)

	return result
}

// ValidateSecretsEnv validates the secrets.env file.
func (v *Validator) ValidateSecretsEnv(path string) []Issue {
	issues := []Issue{}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		issues = append(issues, Issue{
			File:     path,
			Message:  "secrets.env file not found",
			Severity: SeverityError,
		})
		return issues
	}

	// Parse the env file
	envVars, err := parseEnvFile(path)
	if err != nil {
		issues = append(issues, Issue{
			File:     path,
			Message:  fmt.Sprintf("failed to parse file: %v", err),
			Severity: SeverityError,
		})
		return issues
	}

	// Required fields
	requiredFields := []string{"USERNAME", "HOSTNAME", "SSH_PUBLIC_KEY", "USER_NAME", "USER_EMAIL"}
	for _, field := range requiredFields {
		value, exists := envVars[field]
		if !exists || strings.TrimSpace(value) == "" {
			issues = append(issues, Issue{
				File:     path,
				Field:    field,
				Message:  fmt.Sprintf("%s is required", field),
				Severity: SeverityError,
			})
		}
	}

	// Validate SSH key format
	if sshKey, exists := envVars["SSH_PUBLIC_KEY"]; exists && sshKey != "" {
		if err := validateSSHKey(sshKey); err != nil {
			issues = append(issues, Issue{
				File:     path,
				Field:    "SSH_PUBLIC_KEY",
				Message:  err.Error(),
				Severity: SeverityError,
			})
		}
	}

	// Validate hostname format
	if hostname, exists := envVars["HOSTNAME"]; exists && hostname != "" {
		if err := validateHostname(hostname); err != nil {
			issues = append(issues, Issue{
				File:     path,
				Field:    "HOSTNAME",
				Message:  err.Error(),
				Severity: SeverityError,
			})
		}
	}

	// Validate email format
	if email, exists := envVars["USER_EMAIL"]; exists && email != "" {
		if err := validateEmail(email); err != nil {
			issues = append(issues, Issue{
				File:     path,
				Field:    "USER_EMAIL",
				Message:  err.Error(),
				Severity: SeverityError,
			})
		}
	}

	// Validate SSH_KEY_COUNT if present
	if keyCount, exists := envVars["SSH_KEY_COUNT"]; exists && keyCount != "" {
		if _, err := strconv.Atoi(keyCount); err != nil {
			issues = append(issues, Issue{
				File:     path,
				Field:    "SSH_KEY_COUNT",
				Message:  "SSH_KEY_COUNT must be a valid integer",
				Severity: SeverityWarning,
			})
		}
	}

	return issues
}

// ValidateConfigEnv validates the config.env file.
func (v *Validator) ValidateConfigEnv(path string) []Issue {
	issues := []Issue{}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		issues = append(issues, Issue{
			File:     path,
			Message:  "config.env file not found",
			Severity: SeverityError,
		})
		return issues
	}

	// Parse the env file
	envVars, err := parseEnvFile(path)
	if err != nil {
		issues = append(issues, Issue{
			File:     path,
			Message:  fmt.Sprintf("failed to parse file: %v", err),
			Severity: SeverityError,
		})
		return issues
	}

	// Validate boolean fields
	boolFields := []string{
		"GIT_PUSH_AUTO_SETUP_REMOTE",
		"GIT_PULL_REBASE",
		"GIT_URL_REWRITE_GITHUB",
		"TAILSCALE_SSH_ENABLED",
		"TAILSCALE_EXIT_NODE_ADVERTISE",
		"TAILSCALE_SSH_CHECK_MODE",
		"DOCKER_ENABLED",
		"DOCKER_ADD_TO_GROUP",
		"DOCKER_START_ON_BOOT",
	}

	for _, field := range boolFields {
		if value, exists := envVars[field]; exists {
			if !isValidBool(value) {
				issues = append(issues, Issue{
					File:     path,
					Field:    field,
					Message:  fmt.Sprintf("%s must be 'true' or 'false', got '%s'", field, value),
					Severity: SeverityError,
				})
			}
		}
	}

	// Validate PACKAGE_*_ENABLED fields
	for key, value := range envVars {
		if strings.HasPrefix(key, "PACKAGE_") && strings.HasSuffix(key, "_ENABLED") {
			if !isValidBool(value) {
				issues = append(issues, Issue{
					File:     path,
					Field:    key,
					Message:  fmt.Sprintf("%s must be 'true' or 'false', got '%s'", key, value),
					Severity: SeverityError,
				})
			}
		}
	}

	// Cross-reference validation: if TAILSCALE_SSH_ENABLED is true, check tailscale package
	if sshEnabled, exists := envVars["TAILSCALE_SSH_ENABLED"]; exists && sshEnabled == "true" {
		if !hasPackageEnabled(envVars, "tailscale") {
			issues = append(issues, Issue{
				File:     path,
				Field:    "TAILSCALE_SSH_ENABLED",
				Message:  "TAILSCALE_SSH_ENABLED is true but PACKAGE_TAILSCALE_ENABLED is not set",
				Severity: SeverityWarning,
			})
		}
	}

	return issues
}

// parseEnvFile parses a shell-style env file and returns key-value pairs.
func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envVars := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE or KEY="VALUE"
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		envVars[key] = value
	}

	return envVars, scanner.Err()
}

// validateHostname validates hostname format (RFC 1123).
func validateHostname(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("hostname is required")
	}

	// RFC 1123 hostname validation
	hostnameRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	if !hostnameRegex.MatchString(strings.ToLower(s)) {
		return fmt.Errorf("invalid hostname: must be alphanumeric with optional hyphens, no leading/trailing hyphens")
	}

	return nil
}

// validateSSHKey validates SSH public key format.
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
		return fmt.Errorf("invalid SSH key format: must start with ssh-rsa, ssh-ed25519, or ecdsa-sha2")
	}

	return nil
}

// validateEmail validates email format.
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

// isValidBool checks if a string is a valid boolean value.
func isValidBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "false"
}

// hasPackageEnabled checks if a package is enabled in the env vars.
func hasPackageEnabled(envVars map[string]string, packageName string) bool {
	key := "PACKAGE_" + strings.ToUpper(strings.ReplaceAll(packageName, "-", "_")) + "_ENABLED"
	value, exists := envVars[key]
	return exists && strings.ToLower(value) == "true"
}
