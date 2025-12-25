package tui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
)

// OutputMode represents what the CLI should generate.
type OutputMode string

const (
	OutputConfigOnly  OutputMode = "config"     // Just config.env + secrets.env
	OutputCloudInit   OutputMode = "cloudinit"  // + cloud-init.yaml
	OutputBootableISO OutputMode = "bootable"   // + bootable ISO for bare metal
	OutputSeedISO     OutputMode = "seed"       // + seed ISO for libvirt (future)
)

// UserConfig holds user configuration inputs.
type UserConfig struct {
	Username     string
	Hostname     string
	SSHPublicKey string
	FullName     string
	Email        string
}

// OptionalConfig holds optional service configuration.
type OptionalConfig struct {
	GithubUser   string
	TailscaleKey string
	GithubPAT    string
}

// FormResult holds all collected user input.
type FormResult struct {
	User             UserConfig
	SelectedPackages []string
	Optional         OptionalConfig
	OutputMode       OutputMode
	ISOPath          string // Path to selected ISO (if OutputBootableISO)
}

// RunForm executes the interactive TUI form and returns the result.
func RunForm(registry *packages.Registry) (*FormResult, error) {
	result := &FormResult{
		OutputMode: OutputConfigOnly,
	}

	// Build the form
	form := buildForm(result, registry)

	// Run the form
	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("form cancelled or failed: %w", err)
	}

	return result, nil
}

// buildForm creates the complete multi-step form.
func buildForm(result *FormResult, registry *packages.Registry) *huh.Form {
	// Build package options with all enabled by default
	packageOptions := buildPackageOptions(registry)

	return huh.NewForm(
		// Group 1: User Configuration
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Description("System username for the new account").
				Placeholder("ubuntu").
				Value(&result.User.Username).
				Validate(validateRequired("Username")),

			huh.NewInput().
				Title("Hostname").
				Description("Machine hostname").
				Placeholder("ubuntu-server").
				Value(&result.User.Hostname).
				Validate(validateHostname),

			huh.NewText().
				Title("SSH Public Key").
				Description("Your SSH public key for passwordless login").
				Placeholder("ssh-ed25519 AAAA...").
				Value(&result.User.SSHPublicKey).
				Validate(validateSSHKey).
				Lines(3).
				CharLimit(1000),

			huh.NewInput().
				Title("Full Name").
				Description("Your name for Git commits").
				Placeholder("Your Name").
				Value(&result.User.FullName).
				Validate(validateRequired("Full Name")),

			huh.NewInput().
				Title("Email").
				Description("Your email for Git commits").
				Placeholder("you@example.com").
				Value(&result.User.Email).
				Validate(validateEmail),
		).Title("User Configuration").Description("Configure your user account"),

		// Group 2: Package Selection
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select Packages").
				Description("Choose packages to install (all enabled by default)").
				Options(packageOptions...).
				Value(&result.SelectedPackages),
		).Title("Package Selection").Description("Select packages to install"),

		// Group 3: Optional Services
		huh.NewGroup(
			huh.NewInput().
				Title("GitHub Username").
				Description("For importing SSH authorized keys (optional)").
				Placeholder("your-github-username").
				Value(&result.Optional.GithubUser),

			huh.NewInput().
				Title("Tailscale Auth Key").
				Description("For automatic Tailscale setup (optional)").
				Placeholder("tskey-auth-...").
				Value(&result.Optional.TailscaleKey).
				EchoMode(huh.EchoModePassword),

			huh.NewInput().
				Title("GitHub Personal Access Token").
				Description("For gh CLI authentication (optional)").
				Placeholder("ghp_...").
				Value(&result.Optional.GithubPAT).
				EchoMode(huh.EchoModePassword),
		).Title("Optional Services").Description("Configure optional integrations"),

		// Group 4: Output Mode
		huh.NewGroup(
			huh.NewSelect[OutputMode]().
				Title("Output Format").
				Description("What should be generated?").
				Options(
					huh.NewOption("Config files only (config.env + secrets.env)", OutputConfigOnly),
					huh.NewOption("Config + cloud-init.yaml", OutputCloudInit),
					huh.NewOption("Bootable ISO for bare metal installation", OutputBootableISO),
				).
				Value(&result.OutputMode),
		).Title("Output Mode").Description("Choose what to generate"),
	).WithTheme(Theme()).
		WithShowHelp(true).
		WithShowErrors(true)
}

// buildPackageOptions creates huh options from the package registry.
func buildPackageOptions(registry *packages.Registry) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(registry.Packages))

	for _, category := range registry.Categories() {
		for _, pkg := range registry.ByCategory[category] {
			label := pkg.Name
			if pkg.Description != "" {
				label = fmt.Sprintf("%s - %s", pkg.Name, pkg.Description)
			}
			opt := huh.NewOption(label, pkg.Name).Selected(true) // All selected by default
			options = append(options, opt)
		}
	}

	return options
}

// getDefaultSSHKey tries to read the user's default SSH public key.
func getDefaultSSHKey() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Try common key types
	keyFiles := []string{
		home + "/.ssh/id_ed25519.pub",
		home + "/.ssh/id_rsa.pub",
		home + "/.ssh/id_ecdsa.pub",
	}

	for _, keyFile := range keyFiles {
		data, err := os.ReadFile(keyFile)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	return ""
}

// Validators

func validateRequired(field string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
}

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
