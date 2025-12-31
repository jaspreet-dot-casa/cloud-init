// Package config handles configuration file generation for cloud-init.
package config

import (
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

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

// NewFullConfigFromFormResult creates a FullConfig from the TUI form result.
func NewFullConfigFromFormResult(result *tui.FormResult) *FullConfig {
	cfg := &FullConfig{
		// User configuration
		Username:      result.User.Username,
		Hostname:      result.User.Hostname,
		SSHPublicKeys: result.User.SSHPublicKeys,
		FullName:      result.User.FullName,
		Email:         result.User.Email,
		MachineName:   result.User.MachineName,

		// Package configuration
		EnabledPackages:  result.SelectedPackages,
		DisabledPackages: calculateDisabledPackages(result.AllPackages, result.SelectedPackages),

		// Git defaults
		GitDefaultBranch:       "main",
		GitPushAutoSetupRemote: true,
		GitPullRebase:          true,
		GitPager:               "delta",
		GitURLRewriteGithub:    true,

		// Tailscale defaults
		TailscaleAuthKey:        result.Optional.TailscaleKey,
		TailscaleSSHEnabled:     true,
		TailscaleExitNode:       true,
		TailscaleSSHCheckMode:   true,
		TailscaleSSHCheckPeriod: "12h",

		// Docker defaults
		DockerEnabled:     true,
		DockerAddToGroup:  true,
		DockerStartOnBoot: true,

		// Optional integrations
		GithubUser: result.Optional.GithubUser,
		GithubPAT:  result.Optional.GithubPAT,

		// Repository defaults
		RepoBranch: "main",
	}

	// Check if docker is in enabled packages
	cfg.DockerEnabled = containsPackage(result.SelectedPackages, "docker")

	return cfg
}

// calculateDisabledPackages returns packages in allPackages that are not in enabledPackages.
func calculateDisabledPackages(allPackages, enabledPackages []string) []string {
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

// containsPackage checks if a package name is in the list.
func containsPackage(packages []string, name string) bool {
	for _, pkg := range packages {
		if pkg == name {
			return true
		}
	}
	return false
}
