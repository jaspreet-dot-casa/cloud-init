package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Reader handles reading configuration files.
type Reader struct {
	ProjectRoot string
}

// NewReader creates a new config reader.
func NewReader(projectRoot string) *Reader {
	return &Reader{ProjectRoot: projectRoot}
}

// ReadAll reads both config.env and secrets.env and returns a FullConfig.
func (r *Reader) ReadAll() (*FullConfig, error) {
	secretsPath := filepath.Join(r.ProjectRoot, "cloud-init", "secrets.env")
	configPath := filepath.Join(r.ProjectRoot, "config.env")

	secretsEnv, err := parseEnvFile(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets.env: %w", err)
	}

	configEnv, err := parseEnvFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config.env: %w", err)
	}

	cfg := &FullConfig{}

	// Parse secrets.env
	cfg.Username = secretsEnv["USERNAME"]
	cfg.Hostname = secretsEnv["HOSTNAME"]
	cfg.FullName = secretsEnv["USER_NAME"]
	cfg.Email = secretsEnv["USER_EMAIL"]
	cfg.MachineName = secretsEnv["MACHINE_USER_NAME"]
	cfg.TailscaleAuthKey = secretsEnv["TAILSCALE_AUTH_KEY"]
	cfg.GithubUser = secretsEnv["GITHUB_USER"]
	cfg.GithubPAT = secretsEnv["GITHUB_PAT"]
	cfg.RepoURL = secretsEnv["REPO_URL"]
	cfg.RepoBranch = secretsEnv["REPO_BRANCH"]

	// Parse SSH keys
	if sshKey := secretsEnv["SSH_PUBLIC_KEY"]; sshKey != "" {
		cfg.SSHPublicKeys = []string{sshKey}
	}
	// If SSH_KEY_COUNT > 1, we only have the first key from secrets.env
	// The YAML format is in SSH_PUBLIC_KEYS_YAML but we store just the first for backward compat

	// Parse config.env
	if name := configEnv["USER_NAME"]; name != "" {
		cfg.FullName = name
	}
	if email := configEnv["USER_EMAIL"]; email != "" {
		cfg.Email = email
	}

	cfg.GitDefaultBranch = getStringOrDefault(configEnv, "GIT_DEFAULT_BRANCH", "main")
	cfg.GitPushAutoSetupRemote = getBoolOrDefault(configEnv, "GIT_PUSH_AUTO_SETUP_REMOTE", true)
	cfg.GitPullRebase = getBoolOrDefault(configEnv, "GIT_PULL_REBASE", true)
	cfg.GitPager = getStringOrDefault(configEnv, "GIT_PAGER", "delta")
	cfg.GitURLRewriteGithub = getBoolOrDefault(configEnv, "GIT_URL_REWRITE_GITHUB", true)

	cfg.TailscaleSSHEnabled = getBoolOrDefault(configEnv, "TAILSCALE_SSH_ENABLED", true)
	cfg.TailscaleExitNode = getBoolOrDefault(configEnv, "TAILSCALE_EXIT_NODE_ADVERTISE", true)
	cfg.TailscaleSSHCheckMode = getBoolOrDefault(configEnv, "TAILSCALE_SSH_CHECK_MODE", true)
	cfg.TailscaleSSHCheckPeriod = getStringOrDefault(configEnv, "TAILSCALE_SSH_CHECK_PERIOD", "12h")

	cfg.DockerEnabled = getBoolOrDefault(configEnv, "DOCKER_ENABLED", true)
	cfg.DockerAddToGroup = getBoolOrDefault(configEnv, "DOCKER_ADD_TO_GROUP", true)
	cfg.DockerStartOnBoot = getBoolOrDefault(configEnv, "DOCKER_START_ON_BOOT", true)

	// Parse enabled packages
	cfg.EnabledPackages = parseEnabledPackages(configEnv)

	// Set repo defaults
	if cfg.RepoBranch == "" {
		cfg.RepoBranch = "main"
	}

	return cfg, nil
}

// ReadSecretsEnv reads only the secrets.env file.
func (r *Reader) ReadSecretsEnv() (map[string]string, error) {
	secretsPath := filepath.Join(r.ProjectRoot, "cloud-init", "secrets.env")
	return parseEnvFile(secretsPath)
}

// ReadConfigEnv reads only the config.env file.
func (r *Reader) ReadConfigEnv() (map[string]string, error) {
	configPath := filepath.Join(r.ProjectRoot, "config.env")
	return parseEnvFile(configPath)
}

// parseEnvFile parses a shell-style env file and returns key-value pairs.
func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envVars := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE or KEY="VALUE"
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		envVars[key] = value
	}

	return envVars, scanner.Err()
}

// parseEnabledPackages extracts enabled packages from config env vars.
func parseEnabledPackages(envVars map[string]string) []string {
	var packages []string
	for key, value := range envVars {
		if strings.HasPrefix(key, "PACKAGE_") && strings.HasSuffix(key, "_ENABLED") {
			if strings.ToLower(value) == "true" {
				// Extract package name: PACKAGE_LAZYGIT_ENABLED -> lazygit
				name := strings.TrimPrefix(key, "PACKAGE_")
				name = strings.TrimSuffix(name, "_ENABLED")
				name = strings.ToLower(strings.ReplaceAll(name, "_", "-"))
				packages = append(packages, name)
			}
		}
	}
	return packages
}

// getStringOrDefault returns the value for key or a default if not present.
func getStringOrDefault(envVars map[string]string, key, defaultValue string) string {
	if value, exists := envVars[key]; exists && value != "" {
		return value
	}
	return defaultValue
}

// getBoolOrDefault returns the boolean value for key or a default if not present.
func getBoolOrDefault(envVars map[string]string, key string, defaultValue bool) bool {
	if value, exists := envVars[key]; exists {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return defaultValue
}
