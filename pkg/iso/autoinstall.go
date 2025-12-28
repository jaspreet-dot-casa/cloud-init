package iso

import (
	"fmt"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"gopkg.in/yaml.v3"
)

// AutoInstallConfig represents Ubuntu autoinstall configuration.
// See: https://canonical-subiquity.readthedocs-hosted.com/en/latest/reference/autoinstall-reference.html
type AutoInstallConfig struct {
	Version       int                    `yaml:"version"`
	Locale        string                 `yaml:"locale,omitempty"`
	Keyboard      *KeyboardConfig        `yaml:"keyboard,omitempty"`
	Identity      IdentityConfig         `yaml:"identity"`
	SSH           SSHConfig              `yaml:"ssh"`
	Storage       StorageConfig          `yaml:"storage,omitempty"`
	Packages      []string               `yaml:"packages,omitempty"`
	Snaps         []SnapConfig           `yaml:"snaps,omitempty"`
	LateCommands  []string               `yaml:"late-commands,omitempty"`
	Timezone      string                 `yaml:"timezone,omitempty"`
	UserData      map[string]interface{} `yaml:"user-data,omitempty"`
}

// KeyboardConfig represents keyboard layout configuration.
type KeyboardConfig struct {
	Layout  string `yaml:"layout"`
	Variant string `yaml:"variant,omitempty"`
}

// IdentityConfig represents user identity configuration.
type IdentityConfig struct {
	Hostname string `yaml:"hostname"`
	Username string `yaml:"username"`
	// Password is optional when SSH keys are provided
	Password string `yaml:"password,omitempty"`
}

// SSHConfig represents SSH server configuration.
type SSHConfig struct {
	InstallServer  bool     `yaml:"install-server"`
	AuthorizedKeys []string `yaml:"authorized-keys,omitempty"`
	AllowPW        bool     `yaml:"allow-pw"`
}

// StorageConfig represents disk storage configuration.
type StorageConfig struct {
	Layout StorageLayoutConfig `yaml:"layout"`
}

// StorageLayoutConfig represents the storage layout type.
type StorageLayoutConfig struct {
	Name string `yaml:"name"`
}

// SnapConfig represents a snap package to install.
type SnapConfig struct {
	Name    string `yaml:"name"`
	Channel string `yaml:"channel,omitempty"`
	Classic bool   `yaml:"classic,omitempty"`
}

// CloudConfigWrapper wraps autoinstall config in cloud-config format.
type CloudConfigWrapper struct {
	AutoInstall AutoInstallConfig `yaml:"autoinstall"`
}

// AutoInstallGenerator generates Ubuntu autoinstall configuration.
type AutoInstallGenerator struct {
	projectRoot string
}

// NewAutoInstallGenerator creates a new generator.
func NewAutoInstallGenerator(projectRoot string) *AutoInstallGenerator {
	return &AutoInstallGenerator{projectRoot: projectRoot}
}

// Generate creates autoinstall user-data from FullConfig.
func (g *AutoInstallGenerator) Generate(cfg *config.FullConfig, opts *ISOOptions) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if opts == nil {
		return nil, fmt.Errorf("options is nil")
	}

	autoinstall := AutoInstallConfig{
		Version:  1,
		Locale:   opts.Locale,
		Timezone: opts.Timezone,
		Keyboard: &KeyboardConfig{
			Layout: "us",
		},
		Identity: IdentityConfig{
			Hostname: cfg.Hostname,
			Username: cfg.Username,
		},
		SSH: SSHConfig{
			InstallServer:  true,
			AuthorizedKeys: cfg.SSHPublicKeys,
			AllowPW:        false,
		},
		Storage: StorageConfig{
			Layout: StorageLayoutConfig{
				Name: string(opts.StorageLayout),
			},
		},
	}

	// Add base packages
	autoinstall.Packages = g.buildPackageList(cfg)

	// Add late-commands for post-install setup
	autoinstall.LateCommands = g.buildLateCommands(cfg, opts)

	// Wrap in cloud-config format
	wrapper := CloudConfigWrapper{
		AutoInstall: autoinstall,
	}

	content, err := yaml.Marshal(wrapper)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal autoinstall config: %w", err)
	}

	// Prepend cloud-config header
	return append([]byte("#cloud-config\n"), content...), nil
}

// buildPackageList returns the list of packages to install.
func (g *AutoInstallGenerator) buildPackageList(cfg *config.FullConfig) []string {
	// Base packages always installed
	packages := []string{
		"curl",
		"wget",
		"git",
		"zsh",
		"tree",
		"jq",
		"htop",
		"unzip",
		"build-essential",
		"ca-certificates",
		"gnupg",
		"apt-transport-https",
	}

	return packages
}

// buildLateCommands returns post-install commands to run.
func (g *AutoInstallGenerator) buildLateCommands(cfg *config.FullConfig, opts *ISOOptions) []string {
	commands := []string{}
	username := cfg.Username

	// Set timezone
	commands = append(commands,
		fmt.Sprintf("curtin in-target -- timedatectl set-timezone %s", opts.Timezone),
	)

	// Install Docker if enabled
	if cfg.DockerEnabled {
		commands = append(commands,
			"curtin in-target -- bash -c 'curl -fsSL https://get.docker.com | sh'",
			fmt.Sprintf("curtin in-target -- usermod -aG docker %s", username),
			"curtin in-target -- systemctl enable docker",
		)
	}

	// Install GitHub CLI
	commands = append(commands,
		"curtin in-target -- bash -c 'curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/etc/apt/keyrings/githubcli-archive-keyring.gpg'",
		"curtin in-target -- chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg",
		"curtin in-target -- bash -c 'echo \"deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main\" > /etc/apt/sources.list.d/github-cli.list'",
		"curtin in-target -- apt-get update",
		"curtin in-target -- apt-get install -y gh",
	)

	// Install Tailscale
	commands = append(commands,
		"curtin in-target -- bash -c 'curl -fsSL https://tailscale.com/install.sh | sh'",
		"curtin in-target -- systemctl enable tailscaled",
	)

	// Set default shell to zsh
	commands = append(commands,
		fmt.Sprintf("curtin in-target -- chsh -s /bin/zsh %s", username),
	)

	// Create user directories
	commands = append(commands,
		fmt.Sprintf("curtin in-target -- mkdir -p /home/%s/.config /home/%s/.local/bin", username, username),
		fmt.Sprintf("curtin in-target -- chown -R %s:%s /home/%s/.config /home/%s/.local", username, username, username, username),
	)

	// Configure git if email and name provided
	if cfg.Email != "" && cfg.FullName != "" {
		commands = append(commands,
			fmt.Sprintf("curtin in-target -- sudo -u %s git config --global user.email '%s'", username, cfg.Email),
			fmt.Sprintf("curtin in-target -- sudo -u %s git config --global user.name '%s'", username, cfg.FullName),
			fmt.Sprintf("curtin in-target -- sudo -u %s git config --global init.defaultBranch main", username),
		)
	}

	return commands
}

// GenerateMetaData creates the meta-data file content (can be empty).
func (g *AutoInstallGenerator) GenerateMetaData() []byte {
	// meta-data can be empty but must exist for cloud-init nocloud datasource
	return []byte("")
}
