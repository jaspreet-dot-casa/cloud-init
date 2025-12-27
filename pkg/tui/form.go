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

	// Step 1: Get SSH key source preference
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

	// Step 2: Handle SSH key based on source
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
			// Let user select which key to use if multiple
			if len(keys) == 1 {
				result.User.SSHPublicKey = keys[0]
				fmt.Printf("\n%s Fetched SSH key from GitHub\n", SuccessStyle.Render("✓"))
			} else {
				// Multiple keys - let user choose
				selectedKey := ""
				keyOptions := make([]huh.Option[string], len(keys))
				for i, key := range keys {
					// Truncate key for display
					displayKey := key
					if len(key) > 60 {
						displayKey = key[:57] + "..."
					}
					keyOptions[i] = huh.NewOption(displayKey, key)
				}

				keySelectForm := huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[string]().
							Title("Select SSH Key").
							Description(fmt.Sprintf("Found %d keys for %s", len(keys), githubUsername)).
							Options(keyOptions...).
							Value(&selectedKey),
					).Title("Choose SSH Key"),
				).WithTheme(Theme())

				if err := keySelectForm.Run(); err != nil {
					return nil, fmt.Errorf("form cancelled: %w", err)
				}
				result.User.SSHPublicKey = selectedKey
				fmt.Printf("\n%s Selected SSH key from GitHub\n", SuccessStyle.Render("✓"))
			}
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

		// Let user choose to use GitHub name or enter manually
		if githubProfile != nil && githubProfile.Name != "" {
			useGitHubName := ""
			nameForm := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Full Name").
						Description("Use name from GitHub or enter your own?").
						Options(
							huh.NewOption(fmt.Sprintf("Use \"%s\" from GitHub", githubProfile.Name), githubProfile.Name),
							huh.NewOption("Enter manually", ""),
						).
						Value(&useGitHubName),
				).Title("Git Configuration"),
			).WithTheme(Theme())

			if err := nameForm.Run(); err != nil {
				return nil, fmt.Errorf("form cancelled: %w", err)
			}

			if useGitHubName != "" {
				result.User.FullName = useGitHubName
			}
		}

		// Let user choose to use GitHub email or enter manually
		if githubProfile != nil && githubProfile.Email != "" {
			useGitHubEmail := ""
			emailForm := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Email").
						Description("Use email from GitHub or enter your own?").
						Options(
							huh.NewOption(fmt.Sprintf("Use \"%s\" from GitHub", githubProfile.Email), githubProfile.Email),
							huh.NewOption("Enter manually", ""),
						).
						Value(&useGitHubEmail),
				).Title("Git Configuration"),
			).WithTheme(Theme())

			if err := emailForm.Run(); err != nil {
				return nil, fmt.Errorf("form cancelled: %w", err)
			}

			if useGitHubEmail != "" {
				result.User.Email = useGitHubEmail
			}
		}

	case SSHKeyFromLocal:
		localKey := getDefaultSSHKey()
		if localKey != "" {
			result.User.SSHPublicKey = localKey
			fmt.Printf("\n%s Found local SSH key\n\n", SuccessStyle.Render("✓"))
		} else {
			fmt.Printf("\n%s No local SSH key found in ~/.ssh/\n", WarningStyle.Render("⚠"))
			fmt.Println("You'll need to enter it manually...")
			fmt.Println()
		}

	case SSHKeyManual:
		// Will be handled in the main form
	}

	// Step 3: Build and run the main form
	form := buildForm(result, registry, sshKeySource)

	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("form cancelled or failed: %w", err)
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

// buildForm creates the complete multi-step form.
func buildForm(result *FormResult, registry *packages.Registry, sshKeySource SSHKeySource) *huh.Form {
	// Build package options with all enabled by default
	packageOptions := buildPackageOptions(registry)

	// Build user config fields
	userFields := []huh.Field{
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

	// Only show SSH key input if we don't have one yet or if manual entry was chosen
	if result.User.SSHPublicKey == "" || sshKeySource == SSHKeyManual {
		userFields = append(userFields,
			huh.NewText().
				Title("SSH Public Key").
				Description("Your SSH public key for passwordless login").
				Placeholder("ssh-ed25519 AAAA...").
				Value(&result.User.SSHPublicKey).
				Validate(validateSSHKey).
				Lines(3).
				CharLimit(1000),
		)
	}

	// Only show Full Name input if not already set from GitHub
	if result.User.FullName == "" {
		userFields = append(userFields,
			huh.NewInput().
				Title("Full Name").
				Description("Your name for Git commits").
				Placeholder("Your Name").
				Value(&result.User.FullName).
				Validate(validateRequired("Full Name")),
		)
	}

	// Only show Email input if not already set from GitHub
	if result.User.Email == "" {
		userFields = append(userFields,
			huh.NewInput().
				Title("Email").
				Description("Your email for Git commits").
				Placeholder("you@example.com").
				Value(&result.User.Email).
				Validate(validateEmail),
		)
	}

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
		// Group 1: User Configuration
		huh.NewGroup(userFields...).
			Title("User Configuration").
			Description("Configure your user account"),

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
