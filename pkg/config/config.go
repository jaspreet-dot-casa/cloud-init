// Package config handles configuration for cloud-init generation.
package config

// FullConfig represents all configuration needed for cloud-init.
type FullConfig struct {
	// User configuration
	Username      string
	Hostname      string
	SSHPublicKeys []string // Multiple SSH keys supported
	FullName      string   // Git commit name
	Email         string   // Git commit email
	MachineName   string   // Display name for the machine user

	// Package configuration
	EnabledPackages  []string
	DisabledPackages []string // Packages not selected (for DISABLED_PACKAGE_EXPORTS)

	// Git configuration
	GitDefaultBranch       string
	GitPushAutoSetupRemote bool
	GitPullRebase          bool
	GitPager               string
	GitURLRewriteGithub    bool

	// Tailscale configuration
	TailscaleAuthKey        string
	TailscaleSSHEnabled     bool
	TailscaleExitNode       bool
	TailscaleSSHCheckMode   bool
	TailscaleSSHCheckPeriod string

	// Docker configuration
	DockerEnabled     bool
	DockerAddToGroup  bool
	DockerStartOnBoot bool

	// Optional integrations
	GithubUser string
	GithubPAT  string

	// Repository configuration
	RepoURL    string
	RepoBranch string
}

// NewFullConfig creates a new FullConfig with sensible defaults.
func NewFullConfig() *FullConfig {
	return &FullConfig{
		// Git defaults
		GitDefaultBranch:       "main",
		GitPushAutoSetupRemote: true,
		GitPullRebase:          true,
		GitPager:               "delta",
		GitURLRewriteGithub:    true,

		// Tailscale defaults
		TailscaleSSHEnabled:     true,
		TailscaleExitNode:       true,
		TailscaleSSHCheckMode:   true,
		TailscaleSSHCheckPeriod: "12h",

		// Docker defaults
		DockerEnabled:     true,
		DockerAddToGroup:  true,
		DockerStartOnBoot: true,

		// Repository defaults
		RepoBranch: "main",
	}
}

// CalculateDisabledPackages returns packages in allPackages that are not in enabledPackages.
func CalculateDisabledPackages(allPackages, enabledPackages []string) []string {
	enabledSet := make(map[string]bool)
	for _, pkg := range enabledPackages {
		enabledSet[pkg] = true
	}

	var disabled []string
	for _, pkg := range allPackages {
		if !enabledSet[pkg] {
			disabled = append(disabled, pkg)
		}
	}
	return disabled
}

// ContainsPackage checks if a package name is in the list.
func ContainsPackage(packages []string, name string) bool {
	for _, pkg := range packages {
		if pkg == name {
			return true
		}
	}
	return false
}
