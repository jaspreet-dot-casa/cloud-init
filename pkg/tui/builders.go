package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
)

// buildForm creates the complete multi-step form for Host Details, Packages, Optional Services, and Output.
// Note: SSH keys and Git configuration are handled before this in RunForm.
func buildForm(result *FormResult, registry *packages.Registry, opts *FormOptions) *huh.Form {
	// Build package options with all enabled by default
	packageOptions := buildPackageOptions(registry)

	// Build host details fields
	var hostFields []huh.Field

	// "Your Name" (MachineName) - pre-fill with Git name if available
	if result.User.FullName != "" {
		result.User.MachineName = result.User.FullName // Default to git name
	}

	hostFields = append(hostFields,
		huh.NewInput().
			Title("Your Name").
			Description("Display name for this machine's user account").
			Placeholder("Your Name").
			Value(&result.User.MachineName).
			Validate(validateRequired("Your Name")),
	)

	// Username and Hostname
	hostFields = append(hostFields,
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
	)

	// Note: SSH key manual entry is handled in post-form processing in RunForm

	// Build optional services fields - skip GitHub username if already set from SSH key fetch
	optionalFields := []huh.Field{}

	// Note: GitHub username for SSH key import is now asked in the SSH step

	optionalFields = append(optionalFields,
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
	)

	// Build form groups
	groups := []*huh.Group{
		// Group 1: Host Details
		huh.NewGroup(hostFields...).
			Title("Host Details").
			Description("Configure your machine"),

		// Group 2: Package Selection
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select Packages").
				Description("Choose packages to install (all enabled by default)").
				Options(packageOptions...).
				Value(&result.SelectedPackages),
		).Title("Package Selection").Description("Select packages to install"),

		// Group 3: Optional Services
		huh.NewGroup(optionalFields...).
			Title("Optional Services").
			Description("Configure optional integrations"),
	}

	// Group 4: Output Mode (only shown in generate command, not create)
	if !opts.SkipOutputMode {
		groups = append(groups,
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
		)
	}

	return huh.NewForm(groups...).
		WithTheme(Theme()).
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
