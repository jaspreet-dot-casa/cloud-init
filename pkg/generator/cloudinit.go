// Package generator provides cloud-init.yaml generation from configuration.
package generator

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
)

// TemplateVars holds all variables for template substitution.
type TemplateVars struct {
	USERNAME           string
	HOSTNAME           string
	SSH_PUBLIC_KEY     string
	SSH_PUBLIC_KEYS    string // YAML formatted list of keys
	USER_NAME          string
	USER_EMAIL         string
	MACHINE_USER_NAME  string
	TAILSCALE_AUTH_KEY string
	GITHUB_USER        string
	GITHUB_PAT         string
	REPO_URL           string
	REPO_BRANCH        string

	// Package configuration
	DISABLED_PACKAGE_EXPORTS string // Shell export statements for disabled packages
}

// Generator generates cloud-init.yaml files.
type Generator struct {
	ProjectRoot string
}

// NewGenerator creates a new Generator.
func NewGenerator(projectRoot string) *Generator {
	return &Generator{ProjectRoot: projectRoot}
}

// Generate reads the template and generates cloud-init.yaml.
func (g *Generator) Generate(cfg *config.FullConfig, templatePath, outputPath string) error {
	// Read template
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Create template vars from config
	vars := configToVars(cfg)

	// Substitute variables
	output := substituteVars(string(templateContent), vars)

	// Write output
	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// configToVars converts FullConfig to TemplateVars.
func configToVars(cfg *config.FullConfig) *TemplateVars {
	vars := &TemplateVars{
		USERNAME:           cfg.Username,
		HOSTNAME:           cfg.Hostname,
		USER_NAME:          cfg.FullName,
		USER_EMAIL:         cfg.Email,
		TAILSCALE_AUTH_KEY: cfg.TailscaleAuthKey,
		GITHUB_USER:        cfg.GithubUser,
		GITHUB_PAT:         cfg.GithubPAT,
		REPO_URL:           cfg.RepoURL,
		REPO_BRANCH:        cfg.RepoBranch,
	}

	// Set machine name (fallback to full name)
	vars.MACHINE_USER_NAME = cfg.MachineName
	if vars.MACHINE_USER_NAME == "" {
		vars.MACHINE_USER_NAME = cfg.FullName
	}

	// Set SSH key (use first key for backward compatibility)
	if len(cfg.SSHPublicKeys) > 0 {
		vars.SSH_PUBLIC_KEY = cfg.SSHPublicKeys[0]

		// Build YAML formatted list of keys.
		// Note: The 6-space indentation matches the cloud-init.template.yaml structure
		// where ssh_authorized_keys is nested under users. If the template structure
		// changes, this indentation may need to be updated.
		var yamlKeys strings.Builder
		for _, key := range cfg.SSHPublicKeys {
			yamlKeys.WriteString(fmt.Sprintf("      - %s\n", key))
		}
		vars.SSH_PUBLIC_KEYS = strings.TrimSuffix(yamlKeys.String(), "\n")
	}

	// Set repo defaults
	if vars.REPO_BRANCH == "" {
		vars.REPO_BRANCH = "main"
	}
	if vars.REPO_URL == "" {
		vars.REPO_URL = "https://github.com/jaspreet-dot-casa/cloud-init.git"
	}

	// Build disabled package exports
	vars.DISABLED_PACKAGE_EXPORTS = buildDisabledPackageExports(cfg.DisabledPackages)

	return vars
}

// buildDisabledPackageExports generates shell export statements for disabled packages.
// Each disabled package gets an export like: export PACKAGE_LAZYGIT_ENABLED=false
func buildDisabledPackageExports(disabledPackages []string) string {
	if len(disabledPackages) == 0 {
		return "# All packages enabled"
	}

	var exports strings.Builder
	exports.WriteString("# Disabled packages\n")
	for _, pkg := range disabledPackages {
		// Convert package name to env var format: lazygit -> PACKAGE_LAZYGIT_ENABLED
		envVar := "PACKAGE_" + strings.ToUpper(strings.ReplaceAll(pkg, "-", "_")) + "_ENABLED"
		exports.WriteString(fmt.Sprintf("export %s=false\n", envVar))
	}
	return strings.TrimSuffix(exports.String(), "\n")
}

// substituteVars replaces ${VARIABLE} placeholders with values.
func substituteVars(template string, vars *TemplateVars) string {
	// Map of variable names to values
	varMap := map[string]string{
		"USERNAME":                 vars.USERNAME,
		"HOSTNAME":                 vars.HOSTNAME,
		"SSH_PUBLIC_KEY":           vars.SSH_PUBLIC_KEY,
		"SSH_PUBLIC_KEYS":          vars.SSH_PUBLIC_KEYS,
		"USER_NAME":                vars.USER_NAME,
		"USER_EMAIL":               vars.USER_EMAIL,
		"MACHINE_USER_NAME":        vars.MACHINE_USER_NAME,
		"TAILSCALE_AUTH_KEY":       vars.TAILSCALE_AUTH_KEY,
		"GITHUB_USER":              vars.GITHUB_USER,
		"GITHUB_PAT":               vars.GITHUB_PAT,
		"REPO_URL":                 vars.REPO_URL,
		"REPO_BRANCH":              vars.REPO_BRANCH,
		"DISABLED_PACKAGE_EXPORTS": vars.DISABLED_PACKAGE_EXPORTS,
	}

	result := template

	// Replace ${VARIABLE} format
	for name, value := range varMap {
		placeholder := "${" + name + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Also replace $VARIABLE format (without braces) for inline usage
	// Use word boundary matching to avoid partial replacements
	for name, value := range varMap {
		// Match $VARIABLE only when followed by non-word character or end
		re := regexp.MustCompile(`\$` + name + `\b`)
		result = re.ReplaceAllString(result, value)
	}

	return result
}

// ValidateTemplate checks if a template file exists and is readable.
func ValidateTemplate(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("template file not found: %s", path)
	}
	if err != nil {
		return fmt.Errorf("failed to stat template: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("template path is a directory: %s", path)
	}
	return nil
}
