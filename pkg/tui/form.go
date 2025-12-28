package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
)

// SSHKeySource represents where to get the SSH key from.
type SSHKeySource string

const (
	SSHKeyFromGitHub SSHKeySource = "github"
	SSHKeyFromLocal  SSHKeySource = "local"
	SSHKeyManual     SSHKeySource = "manual"
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
	Username      string
	Hostname      string
	SSHPublicKeys []string // Multiple SSH keys supported
	FullName      string   // Git commit name
	Email         string   // Git commit email
	MachineName   string   // Display name for the machine user (can differ from git name)
}

// OptionalConfig holds optional service configuration.
type OptionalConfig struct {
	GithubUser   string
	TailscaleKey string
	GithubPAT    string
}

// ISOConfig holds ISO generation configuration.
type ISOConfig struct {
	SourcePath    string // Path to source Ubuntu ISO
	UbuntuVersion string // "22.04" or "24.04"
	StorageLayout string // "lvm", "direct", or "zfs"
}

// FormResult holds all collected user input.
type FormResult struct {
	User             UserConfig
	SelectedPackages []string
	Optional         OptionalConfig
	OutputMode       OutputMode
	ISO              ISOConfig // ISO options (if OutputBootableISO)
}

// RunForm executes the interactive TUI form and returns the result.
func RunForm(registry *packages.Registry) (*FormResult, error) {
	result := &FormResult{
		OutputMode: OutputConfigOnly,
	}

	// =========================================================================
	// Step 1: SSH Key Source Selection
	// =========================================================================
	sshKeySource := SSHKeyFromLocal
	githubUsername := ""

	sshSourceForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[SSHKeySource]().
				Title("SSH Key Source").
				Description("Where should we get your SSH public key from?").
				Options(
					huh.NewOption("Fetch from GitHub username", SSHKeyFromGitHub),
					huh.NewOption("Use local machine key (~/.ssh/id_*.pub)", SSHKeyFromLocal),
					huh.NewOption("Enter manually", SSHKeyManual),
				).
				Value(&sshKeySource),
		).Title("SSH Key Configuration"),
	).WithTheme(Theme())

	if err := sshSourceForm.Run(); err != nil {
		return nil, fmt.Errorf("form cancelled: %w", err)
	}

	// Track what we fetched from GitHub for later form customization
	var githubProfile *GitHubProfile

	// =========================================================================
	// Step 2: SSH Key Details (based on source)
	// =========================================================================
	switch sshKeySource {
	case SSHKeyFromGitHub:
		// Ask for GitHub username
		githubForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("GitHub Username").
					Description("We'll fetch your SSH keys and profile info").
					Placeholder("your-github-username").
					Value(&githubUsername).
					Validate(validateRequired("GitHub username")),
			).Title("GitHub Integration"),
		).WithTheme(Theme())

		if err := githubForm.Run(); err != nil {
			return nil, fmt.Errorf("form cancelled: %w", err)
		}

		// Fetch SSH keys from GitHub
		keys, err := fetchGitHubSSHKeys(githubUsername)
		if err != nil {
			fmt.Printf("\n%s Failed to fetch SSH keys: %v\n", ErrorStyle.Render("✗"), err)
			fmt.Println("Falling back to manual entry...")
			fmt.Println()
		} else if len(keys) == 0 {
			fmt.Printf("\n%s No SSH keys found for user '%s'\n", WarningStyle.Render("⚠"), githubUsername)
			fmt.Println("Falling back to manual entry...")
			fmt.Println()
		} else {
			// Multi-select SSH keys (all selected by default)
			keyOptions := make([]huh.Option[string], len(keys))
			for i, key := range keys {
				// Truncate key for display
				displayKey := key
				if len(key) > 60 {
					displayKey = key[:57] + "..."
				}
				keyOptions[i] = huh.NewOption(displayKey, key).Selected(true) // All selected by default
			}

			keySelectForm := huh.NewForm(
				huh.NewGroup(
					huh.NewMultiSelect[string]().
						Title("Select SSH Keys").
						Description(fmt.Sprintf("Found %d keys for %s (all selected by default)", len(keys), githubUsername)).
						Options(keyOptions...).
						Value(&result.User.SSHPublicKeys),
				).Title("Choose SSH Keys"),
			).WithTheme(Theme())

			if err := keySelectForm.Run(); err != nil {
				return nil, fmt.Errorf("form cancelled: %w", err)
			}
			fmt.Printf("\n%s Selected %d SSH key(s) from GitHub\n", SuccessStyle.Render("✓"), len(result.User.SSHPublicKeys))

			// Store GitHub username for later use
			result.Optional.GithubUser = githubUsername
		}

		// Fetch GitHub profile for name/email
		profile, err := fetchGitHubProfile(githubUsername)
		if err != nil {
			fmt.Printf("%s Could not fetch profile info: %v\n\n", WarningStyle.Render("⚠"), err)
		} else {
			githubProfile = profile
			if profile.Name != "" || profile.Email != "" {
				fmt.Printf("%s Fetched profile from GitHub\n\n", SuccessStyle.Render("✓"))
			} else {
				fmt.Printf("%s Profile fetched but name/email are private\n\n", WarningStyle.Render("⚠"))
			}
		}

	case SSHKeyFromLocal:
		localKey := getDefaultSSHKey()
		if localKey != "" {
			result.User.SSHPublicKeys = []string{localKey}
			fmt.Printf("\n%s Found local SSH key\n\n", SuccessStyle.Render("✓"))
		} else {
			fmt.Printf("\n%s No local SSH key found in ~/.ssh/\n", WarningStyle.Render("⚠"))
			fmt.Println("You'll need to enter it manually...")
			fmt.Println()
		}

	case SSHKeyManual:
		// Will be handled in the host details form
	}

	// =========================================================================
	// Step 3: Git Configuration (ALWAYS shown)
	// =========================================================================

	// Git Name - offer choice if we have GitHub profile, otherwise simple input
	if githubProfile != nil && githubProfile.Name != "" {
		// Bind directly to result.User.FullName - empty string means "Enter manually"
		result.User.FullName = githubProfile.Name // Default to GitHub name (first option selected)

		gitNameForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Git Name").
					Description("Name for Git commits").
					Options(
						huh.NewOption(fmt.Sprintf("Use \"%s\" from GitHub", githubProfile.Name), githubProfile.Name),
						huh.NewOption("Enter manually", ""),
					).
					Value(&result.User.FullName),
			).Title("Git Configuration"),
		).WithTheme(Theme())

		if err := gitNameForm.Run(); err != nil {
			return nil, fmt.Errorf("form cancelled: %w", err)
		}

		// If user chose "Enter manually", ask for input
		if result.User.FullName == "" {
			manualNameForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Git Name").
						Description("Enter your name for Git commits").
						Placeholder("Your Name").
						Value(&result.User.FullName).
						Validate(validateRequired("Git Name")),
				).Title("Git Configuration"),
			).WithTheme(Theme())

			if err := manualNameForm.Run(); err != nil {
				return nil, fmt.Errorf("form cancelled: %w", err)
			}
		}
	} else {
		// No GitHub profile - simple input
		gitNameForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Git Name").
					Description("Your name for Git commits").
					Placeholder("Your Name").
					Value(&result.User.FullName).
					Validate(validateRequired("Git Name")),
			).Title("Git Configuration"),
		).WithTheme(Theme())

		if err := gitNameForm.Run(); err != nil {
			return nil, fmt.Errorf("form cancelled: %w", err)
		}
	}

	// Git Email - offer choice if we have GitHub profile, otherwise simple input
	if githubProfile != nil && githubProfile.Email != "" {
		// Bind directly to result.User.Email - empty string means "Enter manually"
		result.User.Email = githubProfile.Email // Default to GitHub email (first option selected)

		gitEmailForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Git Email").
					Description("Email for Git commits").
					Options(
						huh.NewOption(fmt.Sprintf("Use \"%s\" from GitHub", githubProfile.Email), githubProfile.Email),
						huh.NewOption("Enter manually", ""),
					).
					Value(&result.User.Email),
			).Title("Git Configuration"),
		).WithTheme(Theme())

		if err := gitEmailForm.Run(); err != nil {
			return nil, fmt.Errorf("form cancelled: %w", err)
		}

		// If user chose "Enter manually", ask for input
		if result.User.Email == "" {
			manualEmailForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Git Email").
						Description("Enter your email for Git commits").
						Placeholder("you@example.com").
						Value(&result.User.Email).
						Validate(validateEmail),
				).Title("Git Configuration"),
			).WithTheme(Theme())

			if err := manualEmailForm.Run(); err != nil {
				return nil, fmt.Errorf("form cancelled: %w", err)
			}
		}
	} else {
		// No GitHub profile - simple input
		gitEmailForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Git Email").
					Description("Your email for Git commits").
					Placeholder("you@example.com").
					Value(&result.User.Email).
					Validate(validateEmail),
			).Title("Git Configuration"),
		).WithTheme(Theme())

		if err := gitEmailForm.Run(); err != nil {
			return nil, fmt.Errorf("form cancelled: %w", err)
		}
	}

	// =========================================================================
	// Step 4: Build and run the main form (Host Details, Packages, Optional, Output)
	// =========================================================================
	form := buildForm(result, registry)

	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("form cancelled or failed: %w", err)
	}

	// =========================================================================
	// Step 5: ISO Options (if OutputBootableISO selected)
	// =========================================================================

	if result.OutputMode == OutputBootableISO {
		// Set defaults
		result.ISO.UbuntuVersion = "24.04"
		result.ISO.StorageLayout = "lvm"

		isoForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Source Ubuntu ISO").
					Description("Path to Ubuntu Server ISO file").
					Placeholder("/path/to/ubuntu-24.04-live-server-amd64.iso").
					Value(&result.ISO.SourcePath).
					Validate(validateISOPath),

				huh.NewSelect[string]().
					Title("Ubuntu Version").
					Description("Version of the source ISO").
					Options(
						huh.NewOption("Ubuntu 24.04 LTS", "24.04"),
						huh.NewOption("Ubuntu 22.04 LTS", "22.04"),
					).
					Value(&result.ISO.UbuntuVersion),

				huh.NewSelect[string]().
					Title("Storage Layout").
					Description("Disk partitioning scheme for installation").
					Options(
						huh.NewOption("LVM (recommended)", "lvm"),
						huh.NewOption("Direct (no LVM)", "direct"),
						huh.NewOption("ZFS (experimental)", "zfs"),
					).
					Value(&result.ISO.StorageLayout),
			).Title("Bootable ISO Options"),
		).WithTheme(Theme())

		if err := isoForm.Run(); err != nil {
			return nil, fmt.Errorf("form cancelled: %w", err)
		}
	}

	// =========================================================================
	// Step 6: Post-form processing
	// =========================================================================

	// Handle "Your Name" manual entry if user chose "Enter different name"
	if result.User.MachineName == "" {
		machineNameForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Your Name").
					Description("Enter your display name for this machine").
					Placeholder("Your Name").
					Value(&result.User.MachineName).
					Validate(validateRequired("Your Name")),
			).Title("Host Details"),
		).WithTheme(Theme())

		if err := machineNameForm.Run(); err != nil {
			return nil, fmt.Errorf("form cancelled: %w", err)
		}
	}

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

// GitHubProfile holds public profile information from GitHub.
type GitHubProfile struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Login string `json:"login"`
}

// fetchGitHubSSHKeys fetches public SSH keys from GitHub for a given username.
func fetchGitHubSSHKeys(username string) ([]string, error) {
	url := fmt.Sprintf("https://github.com/%s.keys", username)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("GitHub user '%s' not found", username)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse keys (one per line)
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	keys := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			keys = append(keys, line)
		}
	}

	return keys, nil
}

// fetchGitHubProfile fetches public profile information from GitHub API.
func fetchGitHubProfile(username string) (*GitHubProfile, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", username)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("GitHub user '%s' not found", username)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var profile GitHubProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	return &profile, nil
}

// buildForm creates the complete multi-step form for Host Details, Packages, Optional Services, and Output.
// Note: SSH keys and Git configuration are handled before this in RunForm.
func buildForm(result *FormResult, registry *packages.Registry) *huh.Form {
	// Build package options with all enabled by default
	packageOptions := buildPackageOptions(registry)

	// Build host details fields
	hostFields := []huh.Field{
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
	}

	// "Your Name" (MachineName) - offer to use Git name or enter different
	// Default to git name (FullName was set in Step 3)
	if result.User.FullName != "" {
		result.User.MachineName = result.User.FullName // Default to git name
		hostFields = append(hostFields,
			huh.NewSelect[string]().
				Title("Your Name").
				Description("Display name for this machine's user account").
				Options(
					huh.NewOption(fmt.Sprintf("Use \"%s\" (same as Git)", result.User.FullName), result.User.FullName),
					huh.NewOption("Enter different name", ""),
				).
				Value(&result.User.MachineName),
		)
	} else {
		// Fallback if git name somehow wasn't set
		hostFields = append(hostFields,
			huh.NewInput().
				Title("Your Name").
				Description("Display name for this machine's user account").
				Placeholder("Your Name").
				Value(&result.User.MachineName).
				Validate(validateRequired("Your Name")),
		)
	}

	// Note: SSH key manual entry is handled in post-form processing in RunForm

	// Build optional services fields - skip GitHub username if already set from SSH key fetch
	optionalFields := []huh.Field{}

	if result.Optional.GithubUser == "" {
		optionalFields = append(optionalFields,
			huh.NewInput().
				Title("GitHub Username").
				Description("For importing SSH authorized keys (optional)").
				Placeholder("your-github-username").
				Value(&result.Optional.GithubUser),
		)
	}

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

	return huh.NewForm(
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

func validateISOPath(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("source ISO path is required")
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
