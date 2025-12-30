package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
)

// RunForm executes the interactive TUI form and returns the result.
// If opts is nil, all questions are shown (for ucli generate compatibility).
func RunForm(registry *packages.Registry, opts *FormOptions) (*FormResult, error) {
	if opts == nil {
		opts = &FormOptions{}
	}
	result := &FormResult{
		OutputMode: OutputConfigOnly,
	}

	// =========================================================================
	// Step 1: SSH Keys + GitHub Username (unified form)
	// =========================================================================

	// Discover local SSH keys
	localKeys := getLocalSSHKeys()
	var selectedLocalKeys []string
	var githubUsername string

	// Build form fields
	var sshFields []huh.Field

	if len(localKeys) > 0 {
		// Create options for local keys (all selected by default)
		localKeyOptions := make([]huh.Option[string], len(localKeys))
		for i, key := range localKeys {
			label := fmt.Sprintf("~/.ssh/%s (%s)", filepath.Base(key.Path), key.Fingerprint)
			localKeyOptions[i] = huh.NewOption(label, key.Content).Selected(true)
		}

		sshFields = append(sshFields,
			huh.NewMultiSelect[string]().
				Title("Local SSH Keys").
				Description("Select keys from your machine").
				Options(localKeyOptions...).
				Value(&selectedLocalKeys),
		)
	} else {
		// No local keys found - show a note
		sshFields = append(sshFields,
			huh.NewNote().
				Title("No Local SSH Keys Found").
				Description("No SSH keys found in ~/.ssh/\nEnter a GitHub username below to fetch keys, or you'll be asked to enter one manually."),
		)
	}

	// GitHub username input (optional)
	sshFields = append(sshFields,
		huh.NewInput().
			Title("GitHub Username").
			Description("Fetch SSH keys and profile info (optional, press Enter to skip)").
			Placeholder("your-github-username").
			Value(&githubUsername),
	)

	// Run the SSH form
	sshForm := huh.NewForm(
		huh.NewGroup(sshFields...).
			Title("SSH & Git Configuration").
			Description("Configure SSH keys and Git identity"),
	).WithTheme(Theme())

	if err := sshForm.Run(); err != nil {
		return nil, fmt.Errorf("form cancelled: %w", err)
	}

	// Add selected local keys to result
	result.User.SSHPublicKeys = append(result.User.SSHPublicKeys, selectedLocalKeys...)

	// =========================================================================
	// Step 2: Fetch from GitHub if username provided
	// =========================================================================

	var githubProfile *GitHubProfile
	var githubSSHKeys []string

	if githubUsername != "" {
		result.Optional.GithubUser = githubUsername

		// Fetch SSH keys from GitHub
		fmt.Printf("\n%s Fetching data from GitHub...\n", InfoStyle.Render("⟳"))

		keys, err := fetchGitHubSSHKeys(githubUsername)
		if err != nil {
			fmt.Printf("%s Failed to fetch SSH keys: %v\n", WarningStyle.Render("⚠"), err)
		} else if len(keys) == 0 {
			fmt.Printf("%s No SSH keys found for user '%s'\n", WarningStyle.Render("⚠"), githubUsername)
		} else {
			githubSSHKeys = keys
			fmt.Printf("%s Found %d SSH key(s) from GitHub\n", SuccessStyle.Render("✓"), len(keys))
		}

		// Fetch profile
		profile, err := fetchGitHubProfile(githubUsername)
		if err != nil {
			fmt.Printf("%s Could not fetch profile: %v\n", WarningStyle.Render("⚠"), err)
		} else {
			githubProfile = profile
			if profile.Name != "" {
				fmt.Printf("%s Found name: %s\n", SuccessStyle.Render("✓"), profile.Name)
			}
			if profile.Email != "" {
				fmt.Printf("%s Found email: %s\n", SuccessStyle.Render("✓"), profile.Email)
			} else if profile.NoReplyEmail() != "" {
				fmt.Printf("%s Using noreply email: %s\n", SuccessStyle.Render("✓"), profile.NoReplyEmail())
			}
		}
		fmt.Println()
	}

	// =========================================================================
	// Step 3: Git Details + GitHub SSH Keys (with pre-filled values)
	// =========================================================================

	// Pre-fill from GitHub profile if available
	if githubProfile != nil {
		if githubProfile.Name != "" {
			result.User.FullName = githubProfile.Name
		}
		// Use BestEmail which returns public email if available, otherwise noreply
		if email := githubProfile.BestEmail(); email != "" {
			result.User.Email = email
		}
	}

	// Build git form fields
	var gitFields []huh.Field

	gitFields = append(gitFields,
		huh.NewInput().
			Title("Git Name").
			Description("Your name for Git commits").
			Placeholder("Your Name").
			Value(&result.User.FullName).
			Validate(validateRequired("Git Name")),

		huh.NewInput().
			Title("Git Email").
			Description("Your email for Git commits").
			Placeholder("you@example.com").
			Value(&result.User.Email).
			Validate(validateEmail),
	)

	// Add GitHub SSH keys if any were fetched
	var selectedGitHubKeys []string
	if len(githubSSHKeys) > 0 {
		githubKeyOptions := make([]huh.Option[string], len(githubSSHKeys))
		for i, key := range githubSSHKeys {
			displayKey := key
			if len(key) > 50 {
				displayKey = key[:47] + "..."
			}
			githubKeyOptions[i] = huh.NewOption(displayKey, key).Selected(true)
		}

		gitFields = append(gitFields,
			huh.NewMultiSelect[string]().
				Title("GitHub SSH Keys").
				Description(fmt.Sprintf("Additional keys from GitHub user '%s'", githubUsername)).
				Options(githubKeyOptions...).
				Value(&selectedGitHubKeys),
		)
	}

	// Run the git form
	gitForm := huh.NewForm(
		huh.NewGroup(gitFields...).
			Title("Git Configuration").
			Description("Configure your Git identity"),
	).WithTheme(Theme())

	if err := gitForm.Run(); err != nil {
		return nil, fmt.Errorf("form cancelled: %w", err)
	}

	// Add selected GitHub keys to result
	result.User.SSHPublicKeys = append(result.User.SSHPublicKeys, selectedGitHubKeys...)

	// =========================================================================
	// Step 4: Build and run the main form (Host Details, Packages, Optional, Output)
	// =========================================================================
	form := buildForm(result, registry, opts)

	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("form cancelled or failed: %w", err)
	}

	// =========================================================================
	// Step 5: ISO Options (if OutputBootableISO selected)
	// =========================================================================

	if result.OutputMode == OutputBootableISO {
		// Set defaults
		result.ISO.StorageLayout = "lvm"

		isoForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Source Ubuntu ISO").
					Description("Path to Ubuntu Server ISO file (version is detected from ISO)").
					Placeholder("/path/to/ubuntu-24.04-live-server-amd64.iso").
					Value(&result.ISO.SourcePath).
					Validate(validateISOPath),

				huh.NewSelect[string]().
					Title("Storage Layout").
					Description("Disk partitioning scheme for installation").
					Options(
						huh.NewOption("LVM - Flexible partitions, snapshots, easy resizing", "lvm"),
						huh.NewOption("Direct - Simple partitions, no overhead, full disk access", "direct"),
						huh.NewOption("ZFS - Advanced filesystem, built-in snapshots, compression", "zfs"),
					).
					Value(&result.ISO.StorageLayout),
			).Title("Bootable ISO Options"),
		).WithTheme(Theme())

		if err := isoForm.Run(); err != nil {
			return nil, fmt.Errorf("form cancelled: %w", err)
		}

		// Expand home directory in ISO path (validator only validates, doesn't modify)
		if strings.HasPrefix(result.ISO.SourcePath, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				result.ISO.SourcePath = filepath.Join(home, result.ISO.SourcePath[2:])
			}
		}

		// Detect Ubuntu version from ISO filename (e.g., ubuntu-24.04-live-server-amd64.iso)
		if result.ISO.UbuntuVersion == "" {
			result.ISO.UbuntuVersion = detectUbuntuVersion(result.ISO.SourcePath)
		}
	}

	// =========================================================================
	// Step 6: Post-form processing
	// =========================================================================

	// Handle manual SSH key entry if needed
	if len(result.User.SSHPublicKeys) == 0 {
		var manualSSHKey string
		sshKeyForm := huh.NewForm(
			huh.NewGroup(
				huh.NewText().
					Title("SSH Public Key").
					Description("Your SSH public key for passwordless login").
					Placeholder("ssh-ed25519 AAAA...").
					Value(&manualSSHKey).
					Validate(validateSSHKey).
					Lines(3).
					CharLimit(1000),
			).Title("SSH Key"),
		).WithTheme(Theme())

		if err := sshKeyForm.Run(); err != nil {
			return nil, fmt.Errorf("form cancelled: %w", err)
		}

		if manualSSHKey != "" {
			result.User.SSHPublicKeys = []string{manualSSHKey}
		}
	}

	return result, nil
}
