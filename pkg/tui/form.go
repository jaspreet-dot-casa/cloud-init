package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

// FormOptions configures the behavior of RunForm.
type FormOptions struct {
	// SkipOutputMode skips the output mode question (used by ucli create
	// where target is selected first).
	SkipOutputMode bool
}

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

// GitHubProfile holds public profile information from GitHub.
type GitHubProfile struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Login string `json:"login"`
}

// NoReplyEmail returns the GitHub noreply email address for this user.
// Format: {id}+{username}@users.noreply.github.com
func (p *GitHubProfile) NoReplyEmail() string {
	if p.ID == 0 || p.Login == "" {
		return ""
	}
	return fmt.Sprintf("%d+%s@users.noreply.github.com", p.ID, p.Login)
}

// BestEmail returns the public email if available, otherwise the noreply email.
func (p *GitHubProfile) BestEmail() string {
	if p.Email != "" {
		return p.Email
	}
	return p.NoReplyEmail()
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

// SSHKeyInfo holds information about an SSH key.
type SSHKeyInfo struct {
	Path        string // Full path: ~/.ssh/id_ed25519.pub
	Type        string // Key type: ed25519, rsa, ecdsa
	Content     string // Full key content
	Fingerprint string // Short display: ssh-ed25519 AAAA...xyz
}

// getLocalSSHKeys returns all available SSH public keys from ~/.ssh/
func getLocalSSHKeys() []SSHKeyInfo {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	keyFiles := []struct {
		name    string
		keyType string
	}{
		{"id_ed25519.pub", "ed25519"},
		{"id_rsa.pub", "rsa"},
		{"id_ecdsa.pub", "ecdsa"},
	}

	var keys []SSHKeyInfo
	for _, kf := range keyFiles {
		path := filepath.Join(home, ".ssh", kf.name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}

		// Create fingerprint (truncated display)
		fingerprint := content
		if len(content) > 50 {
			fingerprint = content[:47] + "..."
		}

		keys = append(keys, SSHKeyInfo{
			Path:        path,
			Type:        kf.keyType,
			Content:     content,
			Fingerprint: fingerprint,
		})
	}

	return keys
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
