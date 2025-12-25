// Package config handles configuration file generation for cloud-init.
package config

import (
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// FullConfig represents all configuration needed for cloud-init.
type FullConfig struct {
	// User configuration
	Username     string
	Hostname     string
	SSHPublicKey string
	FullName     string
	Email        string

	// Package configuration
	EnabledPackages []string

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
		Username:     result.User.Username,
		Hostname:     result.User.Hostname,
		SSHPublicKey: result.User.SSHPublicKey,
		FullName:     result.User.FullName,
		Email:        result.User.Email,

		// Package configuration
		EnabledPackages: result.SelectedPackages,

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

// containsPackage checks if a package name is in the list.
func containsPackage(packages []string, name string) bool {
	for _, pkg := range packages {
		if pkg == name {
			return true
		}
	}
	return false
}
