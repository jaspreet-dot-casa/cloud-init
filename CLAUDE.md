# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a declarative Ubuntu server configuration system using **cloud-init** for automated setup.

**Approach:** Shell script-based configuration with cloud-init orchestration for bare-metal USB boot and VM provisioning.

The project automates installation of development tools (git, docker, lazygit, neovim), modern CLI utilities (ripgrep, fd, bat, fzf, zoxide), shell configuration (zsh + Oh-My-Zsh + Starship), and Tailscale VPN with SSH support.

## Common Commands

```bash
# Go CLI (ucli)
make build-cli                    # Build ucli binary to bin/ucli
make run-cli                      # Build and run interactive TUI
make test-cli                     # Run Go tests
make install-cli                  # Install to GOPATH/bin
./bin/ucli generate               # Interactive configuration wizard
./bin/ucli packages               # List available packages

# Cloud-init installation
./cloud-init/generate.sh          # Generate cloud-init.yaml from template
make update                       # Run idempotent package updates
make update-dry                   # Preview changes without applying
make verify-cloud                 # Verify cloud-init installation

# Testing
make test-syntax                  # Bash syntax validation (fast)
make test-multipass               # Multipass VM test for cloud-init
make test-multipass-keep          # Keep VM for debugging
make shellcheck                   # ShellCheck linting
go test ./...                     # Run all Go tests
```

## Architecture

```
cloud-init/
├── cmd/ucli/              # Go CLI entry point
│   ├── main.go            # Cobra commands (generate, packages, validate, build)
│   └── main_test.go       # CLI tests
│
├── pkg/                   # Go packages
│   ├── config/            # Config file generation (config.go, writer.go)
│   ├── packages/          # Package discovery from scripts/packages/ (discovery.go)
│   └── tui/               # Interactive TUI with charmbracelet/huh (form.go, styles.go)
│
├── config/                # Configuration files
│   └── tailscale.conf     # Tailscale configuration
│
├── scripts/
│   ├── lib/               # Shared libraries (core.sh, health.sh, backup.sh)
│   ├── packages/          # Per-package installers (lazygit.sh, starship.sh, etc.)
│   ├── shared/            # Shared scripts (configure-git.sh, configure-zsh.sh)
│   ├── cloud-init/        # Cloud-init orchestrators (install-all.sh)
│   └── local-remote/      # Post-login auth scripts (Tailscale, Git SSH)
│
├── cloud-init/
│   ├── cloud-init.template.yaml  # Template with ${VARIABLE} placeholders
│   ├── secrets.env.template      # Secrets template (copy to secrets.env)
│   └── generate.sh               # Generates cloud-init.yaml via envsubst
│
├── tests/multipass/       # Multipass VM integration tests
├── docs/implementation/   # 9-phase implementation documentation
├── bin/                   # Built binaries (gitignored)
├── go.mod                 # Go module definition
└── config.env             # Main configuration (git, packages, Tailscale)
```

## Key Patterns

### Go CLI
- **Package discovery**: `pkg/packages/` scans `scripts/packages/*.sh` parsing PACKAGE_NAME and comments
- **TUI forms**: Uses `charmbracelet/huh` for interactive forms, `lipgloss` for styling
- **Multi-step wizard flow**:
  1. SSH Key Source (GitHub/Local/Manual)
  2. SSH Key Selection (multi-select if GitHub has multiple keys)
  3. Git Configuration (auto-fill from GitHub profile, always shown)
  4. Host Details (username, hostname, display name defaults to git name)
  5. Package Selection, Optional Services, Output Mode
- **GitHub integration**: Fetches SSH keys from `github.com/<user>.keys`, profile from GitHub API
- **Config generation**: `pkg/config/writer.go` generates shell-sourceable env files
- **Cobra commands**: CLI structure follows `rootCmd` → subcommands pattern

### Shell Scripts
- **Template-based config**: `secrets.env.template` → `secrets.env`, variables substituted via `envsubst`
- **Per-package scripts**: Each tool has `scripts/packages/<tool>.sh` with install/update/verify actions
- **Idempotent operations**: All scripts safe to run multiple times
- **POSIX shell in cloud-init**: Use `[ ]` not `[[ ]]`, pipe instead of `<<<` (cloud-init uses /bin/sh)
- **PATH for user binaries**: `~/.local/bin` must be in PATH for starship, zoxide detection

## Configuration

- **config.env** - Package enables, git settings, Tailscale options
- **cloud-init/secrets.env** - Credentials (SSH keys, auth tokens) - gitignored
- **GITHUB_USER env var** - Set during generate.sh to import SSH keys from GitHub

## Testing Cloud-Init Changes

```bash
# 1. Edit cloud-init/cloud-init.template.yaml
# 2. Regenerate and commit
./cloud-init/generate.sh
git add -A && git commit -m "description" && git push

# 3. Test in Multipass VM
make test-multipass              # Full test with cleanup
make test-multipass-keep         # Keep VM for debugging
multipass shell <vm-name>        # SSH into kept VM
```

## Important Files

### Go CLI
- `cmd/ucli/main.go` - CLI entry point with cobra commands
- `pkg/packages/discovery.go` - Discovers packages from scripts/packages/*.sh
- `pkg/tui/form.go` - Interactive TUI form with charmbracelet/huh
- `pkg/config/writer.go` - Generates config.env and secrets.env files

### Shell Scripts
- `scripts/lib/core.sh` - Logging, colors, utility functions used everywhere
- `scripts/cloud-init/install-all.sh` - Main cloud-init installation orchestrator
- `scripts/local-remote-login` - Post-login auth script (Tailscale + Git SSH)
- `cloud-init/cloud-init.template.yaml` - Cloud-init configuration template
- `config/tailscale.conf` - Tailscale configuration

# Notes from user: Please do not ignore any of them
- In Makefile, only use `@` for echo statements, not for any other commands.
