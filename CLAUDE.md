# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a declarative Ubuntu server configuration system using **cloud-init** for automated setup.

**Primary Use Case:** Create VMs in Ubuntu via **Terraform + libvirt** (first-class citizen).
**Secondary:** Multipass VMs for local testing, bootable ISOs for bare-metal installs.

The project automates installation of development tools (git, docker, lazygit, neovim), modern CLI utilities (ripgrep, fd, bat, fzf, zoxide), shell configuration (zsh + Oh-My-Zsh + Starship), and Tailscale VPN with SSH support.

## Common Commands

```bash
# Go CLI (ucli)
make build-cli                    # Build ucli binary to bin/ucli
make run-cli                      # Build and run interactive TUI
make test-cli                     # Run Go tests
make install-cli                  # Install to GOPATH/bin
./bin/ucli create                 # Interactive configuration and deployment
./bin/ucli build                  # Interactive cloud-init.yaml generation (no deploy)
./bin/ucli packages               # List available packages

# Package updates (on installed systems)
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
│   ├── main.go            # Cobra root command
│   ├── commands.go        # Command definitions (create, build, packages)
│   ├── create.go          # Create command (TUI + deploy)
│   ├── build.go           # Build command (TUI + cloud-init.yaml only)
│   └── packages.go        # Packages list command
│
├── pkg/                   # Go packages
│   ├── config/            # FullConfig struct and FormResult conversion
│   ├── create/            # Interactive create workflow (app.go, target.go, forms)
│   ├── deploy/            # Deployment abstraction
│   │   ├── deployer.go    # Deployer interface and options
│   │   ├── progress.go    # Progress events and stages
│   │   ├── multipass/     # Multipass VM deployer
│   │   ├── terraform/     # Terraform/libvirt deployer (primary)
│   │   └── usb/           # USB/ISO deployer
│   ├── tfstate/           # Terraform state and VM lifecycle management
│   │   ├── state.go       # Manager for reading terraform state
│   │   └── virsh.go       # VirshClient for fast VM operations
│   ├── generator/         # Cloud-init YAML generation
│   ├── iso/               # Bootable ISO generation
│   ├── packages/          # Package discovery from scripts/packages/
│   ├── project/           # Project root detection
│   └── tui/               # Interactive TUI forms (charmbracelet/huh)
│
├── terraform/             # Terraform libvirt configuration
│   ├── main.tf            # VM provisioning with libvirt provider
│   ├── variables.tf       # Input variables (vm_name, cpus, memory, etc.)
│   ├── outputs.tf         # VM outputs (ip, console command)
│   └── README.md          # Terraform setup guide
│
├── config/                # Configuration files
│   └── tailscale.conf     # Tailscale configuration
│
├── scripts/
│   ├── lib/               # Shared libraries (core.sh, health.sh, backup.sh)
│   ├── packages/          # Per-package installers (lazygit.sh, starship.sh, etc.)
│   ├── shared/            # Shared scripts (configure-git.sh, configure-zsh.sh)
│   └── cloud-init/        # Cloud-init orchestrators (install-all.sh)
│
├── cloud-init/
│   └── cloud-init.template.yaml  # Template with ${VARIABLE} placeholders
│
├── tests/multipass/       # Multipass VM integration tests
├── docs/implementation/   # Implementation documentation
├── bin/                   # Built binaries (gitignored)
├── output/                # Generated ISOs (gitignored)
└── go.mod                 # Go module definition
```

## Key Patterns

### Go CLI
- **Package discovery**: `pkg/packages/` scans `scripts/packages/*.sh` parsing PACKAGE_NAME and comments
- **TUI forms**: Uses `charmbracelet/huh` for interactive forms, `lipgloss` for styling
- **Multi-step wizard flow**:
  1. Target Selection (Terraform/Multipass/USB/SSH) - for `create` command
  2. Target-specific options (VM specs, ISO source, etc.)
  3. SSH Key Source (GitHub/Local/Manual)
  4. Git Configuration (auto-fill from GitHub profile)
  5. Host Details, Package Selection, Optional Services
  6. Review and Confirm → Deploy (or generate cloud-init.yaml for `build`)
- **GitHub integration**: Fetches SSH keys from `github.com/<user>.keys`, profile from GitHub API
- **Direct generation**: All config values embedded directly in cloud-init.yaml (no intermediate files)
- **Package disables**: Disabled packages exported as `PACKAGE_*_ENABLED=false` in bootstrap.sh
- **Cobra commands**: CLI structure follows `rootCmd` → subcommands pattern

### Deployment Abstraction
- **Deployer interface**: `pkg/deploy/deployer.go` defines `Deployer` interface with Validate/Deploy/Cleanup
- **Target types**: `TargetTerraform` (primary), `TargetMultipass`, `TargetUSB`, `TargetSSH`
- **Progress stages**: Validating → CloudInit → Preparing → Planning → Confirming → Applying → Verifying → Complete
- **Terraform deployer** (`pkg/deploy/terraform/`):
  - Generates `terraform.tfvars` from options
  - Runs `terraform init/plan/apply`
  - Requires user confirmation before apply (unless AutoApprove)
  - Parses terraform outputs for VM IP
- **Options structs**: `TerraformOptions`, `MultipassOptions`, `USBOptions` with sensible defaults

### Terraform State Management (`pkg/tfstate/`)
- **Manager**: Reads terraform state and outputs to get VM information
- **VirshClient**: Direct virsh commands for fast start/stop operations (no terraform needed)
- **VMInfo struct**: Name, Status, IP, CPUs, MemoryMB, DiskGB, Autostart
- **VMStatus types**: `StatusRunning`, `StatusStopped`, `StatusPaused`, `StatusShutoff`, `StatusCrashed`, `StatusUnknown`
- **Lifecycle ops**: StartVM, StopVM, ForceStopVM, DeleteVM (via terraform destroy)
- **State checks**: IsInitialized(), HasState() to verify terraform setup

### Shell Scripts
- **Per-package scripts**: Each tool has `scripts/packages/<tool>.sh` with install/update/verify actions
- **Package opt-out pattern**: Scripts use `${PACKAGE_*_ENABLED:-true}` (defaults to enabled)
- **Idempotent operations**: All scripts safe to run multiple times
- **POSIX shell in cloud-init**: Use `[ ]` not `[[ ]]`, pipe instead of `<<<` (cloud-init uses /bin/sh)
- **PATH for user binaries**: `~/.local/bin` must be in PATH for starship, zoxide detection

## Testing Cloud-Init Changes

```bash
# 1. Edit cloud-init/cloud-init.template.yaml
# 2. Test in Multipass VM
make test-multipass              # Full test with cleanup
make test-multipass-keep         # Keep VM for debugging
multipass shell <vm-name>        # SSH into kept VM
```

## Important Files

### Go CLI
- `cmd/ucli/main.go` - CLI entry point with cobra commands
- `pkg/packages/discovery.go` - Discovers packages from scripts/packages/*.sh
- `pkg/tui/form.go` - Interactive TUI form with charmbracelet/huh
- `pkg/generator/cloudinit.go` - Generates cloud-init.yaml from FullConfig

### Shell Scripts
- `scripts/lib/core.sh` - Logging, colors, utility functions used everywhere
- `scripts/cloud-init/install-all.sh` - Main cloud-init installation orchestrator
- `cloud-init/cloud-init.template.yaml` - Cloud-init configuration template
- `config/tailscale.conf` - Tailscale configuration

# Notes from user: Please do not ignore any of them
- In Makefile, only use `@` for echo statements, not for any other commands.
