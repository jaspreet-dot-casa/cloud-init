#!/bin/bash
#==============================================================================
# Cloud-Init Master Installation Script
#
# Installs all packages and configurations from scratch.
# This is the main entry point for cloud-init based installations.
#
# Usage: ./install-all.sh [--dry-run] [-y|--yes] [--gh=USERNAME]
#
# Options:
#   --gh=USERNAME         Fetch SSH public keys from GitHub username
#
# Environment Variables:
#   DRY_RUN=true          Preview changes without applying
#   TAILSCALE_AUTH_KEY    Tailscale authentication key (optional)
#   GITHUB_PAT            GitHub Personal Access Token (optional)
#   GITHUB_USER           GitHub username for SSH key import (alternative to --gh)
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source shared libraries
# shellcheck source=scripts/lib/core.sh
source "${SCRIPT_DIR}/../lib/core.sh"
# shellcheck source=scripts/lib/dryrun.sh
source "${SCRIPT_DIR}/../lib/dryrun.sh"
# shellcheck source=scripts/lib/backup.sh
source "${SCRIPT_DIR}/../lib/backup.sh"
# shellcheck source=scripts/lib/health.sh
source "${SCRIPT_DIR}/../lib/health.sh"
# shellcheck source=scripts/lib/lock.sh
source "${SCRIPT_DIR}/../lib/lock.sh"

#==============================================================================
# Configuration
#==============================================================================

PACKAGES_DIR="${SCRIPT_DIR}/../packages"
SHARED_DIR="${SCRIPT_DIR}/../shared"
LOG_DIR="${HOME}/.local-remote/logs"
LOG_FILE=""

# Ensure ~/.local/bin is in PATH (for user-installed binaries like starship, zoxide)
[[ -d "${HOME}/.local/bin" ]] && export PATH="${HOME}/.local/bin:${PATH}"

# Track errors
declare -a ERRORS=()

#==============================================================================
# Logging Setup
#==============================================================================

setup_logging() {
    ensure_dir "${LOG_DIR}"
    LOG_FILE="${LOG_DIR}/install-$(date +%Y%m%d_%H%M%S).log"

    # Redirect output to both console and log file
    exec > >(tee -a "${LOG_FILE}") 2>&1

    log_info "Log file: ${LOG_FILE}"
}

#==============================================================================
# Error Handling
#==============================================================================

record_error() {
    local component="$1"
    local message="$2"
    ERRORS+=("[${component}] ${message}")
    log_error "[${component}] ${message}"
}

print_error_summary() {
    if [[ ${#ERRORS[@]} -eq 0 ]]; then
        return 0
    fi

    echo ""
    log_section "Error Summary"
    for error in "${ERRORS[@]}"; do
        echo "  - ${error}"
    done
    echo ""
    log_error "${#ERRORS[@]} error(s) occurred during installation"
    return ${#ERRORS[@]}
}

#==============================================================================
# SSH Key Setup
#==============================================================================

setup_github_ssh_keys() {
    local github_user="${1:-}"

    # If no user provided, prompt for it
    if [[ -z "${github_user}" ]]; then
        if [[ "${SKIP_CONFIRMATION:-false}" == "true" ]]; then
            log_warning "No GitHub username provided, skipping SSH key import"
            return 0
        fi

        echo ""
        read -rp "Enter GitHub username to import SSH keys (or press Enter to skip): " github_user

        if [[ -z "${github_user}" ]]; then
            log_info "Skipping SSH key import"
            return 0
        fi
    fi

    log_section "Importing SSH Keys from GitHub"
    log_info "Fetching keys for: ${github_user}"

    if is_dry_run; then
        echo "[DRY-RUN] Would fetch SSH keys from https://github.com/${github_user}.keys"
        echo "[DRY-RUN] Would add keys to ~/.ssh/authorized_keys"
        return 0
    fi

    # Fetch keys from GitHub
    local keys_url="https://github.com/${github_user}.keys"
    local keys
    keys=$(curl -fsSL "${keys_url}" 2>/dev/null) || {
        log_error "Failed to fetch keys from ${keys_url}"
        log_error "Check if the username is correct and has public SSH keys"
        return 1
    }

    if [[ -z "${keys}" ]]; then
        log_warning "No SSH keys found for ${github_user}"
        return 0
    fi

    # Count keys
    local key_count
    key_count=$(echo "${keys}" | wc -l)
    log_info "Found ${key_count} SSH key(s)"

    # Setup .ssh directory
    local ssh_dir="${HOME}/.ssh"
    local auth_keys="${ssh_dir}/authorized_keys"

    mkdir -p "${ssh_dir}"
    chmod 700 "${ssh_dir}"
    touch "${auth_keys}"
    chmod 600 "${auth_keys}"

    # Add keys if not already present
    local added=0
    while IFS= read -r key; do
        [[ -z "${key}" ]] && continue

        # Check if key already exists
        if grep -qF "${key}" "${auth_keys}" 2>/dev/null; then
            log_debug "Key already exists, skipping"
        else
            echo "${key}" >> "${auth_keys}"
            added=$((added + 1))
        fi
    done <<< "${keys}"

    if [[ ${added} -gt 0 ]]; then
        log_success "Added ${added} new SSH key(s) to authorized_keys"
    else
        log_info "All keys already present in authorized_keys"
    fi
}

#==============================================================================
# Installation Steps
#==============================================================================

install_apt_packages() {
    log_section "Installing APT Packages"

    if [[ -f "${PACKAGES_DIR}/apt.sh" ]]; then
        bash "${PACKAGES_DIR}/apt.sh" install || record_error "apt" "Failed to install APT packages"
    else
        log_warning "apt.sh not found, skipping"
    fi
}

install_docker() {
    log_section "Installing Docker"

    if [[ -f "${PACKAGES_DIR}/docker.sh" ]]; then
        bash "${PACKAGES_DIR}/docker.sh" install || record_error "docker" "Failed to install Docker"
    else
        log_warning "docker.sh not found, skipping"
    fi
}

install_github_cli() {
    log_section "Installing GitHub CLI"

    if [[ -f "${PACKAGES_DIR}/github-cli.sh" ]]; then
        bash "${PACKAGES_DIR}/github-cli.sh" install || record_error "github-cli" "Failed to install GitHub CLI"
    else
        log_warning "github-cli.sh not found, skipping"
    fi
}

install_binary_packages() {
    log_section "Installing Binary Packages"

    local packages=(
        "yq"
        "lazygit"
        "lazydocker"
        "starship"
        "delta"
        "zellij"
        "zoxide"
        "btop"
        "mise"
        "tailscale"
        "podman"
    )

    for pkg in "${packages[@]}"; do
        local script="${PACKAGES_DIR}/${pkg}.sh"
        if [[ -f "${script}" ]]; then
            log_info "Installing ${pkg}..."
            bash "${script}" install || record_error "${pkg}" "Failed to install"
        else
            log_warning "${pkg}.sh not found, skipping"
        fi
    done
}

configure_git() {
    log_section "Configuring Git"

    if [[ -f "${SHARED_DIR}/configure-git.sh" ]]; then
        bash "${SHARED_DIR}/configure-git.sh" || record_error "git" "Failed to configure Git"
    else
        log_warning "configure-git.sh not found, skipping"
    fi
}

configure_zsh() {
    log_section "Configuring Zsh"

    if [[ -f "${SHARED_DIR}/configure-zsh.sh" ]]; then
        bash "${SHARED_DIR}/configure-zsh.sh" --install-omz || record_error "zsh" "Failed to configure Zsh"
    else
        log_warning "configure-zsh.sh not found, skipping"
    fi
}

switch_shell_to_zsh() {
    log_section "Switching Default Shell to Zsh"

    if ! command_exists zsh; then
        log_warning "zsh not installed, skipping shell switch"
        return 0
    fi

    local current_shell
    current_shell=$(getent passwd "${USER}" | cut -d: -f7)
    local zsh_path
    zsh_path=$(command -v zsh)

    if [[ "${current_shell}" == "${zsh_path}" ]]; then
        log_success "Shell already set to zsh"
        return 0
    fi

    log_info "Changing shell from ${current_shell} to ${zsh_path}..."

    if is_dry_run; then
        echo "[DRY-RUN] Would run: chsh -s ${zsh_path}"
        return 0
    fi

    # Use sudo if available, otherwise try without
    if command_exists sudo; then
        sudo chsh -s "${zsh_path}" "${USER}"
    else
        chsh -s "${zsh_path}"
    fi

    log_success "Default shell changed to zsh"
}

generate_shell_config() {
    log_section "Generating Shell Configuration"

    if [[ -f "${SHARED_DIR}/generate-shell-config.sh" ]]; then
        bash "${SHARED_DIR}/generate-shell-config.sh" || record_error "shell-config" "Failed to generate shell config"
    else
        log_warning "generate-shell-config.sh not found, skipping"
    fi
}

setup_tailscale() {
    log_section "Setting Up Tailscale"

    # Skip tailscale setup during cloud-init - authentication is handled by local-remote-login
    # Cloud-init already installs and starts tailscaled daemon
    if [[ "${CLOUD_INIT:-false}" == "true" ]]; then
        log_info "Skipping Tailscale setup during cloud-init"
        log_info "Run 'local-remote-login' after first login to authenticate"
        return 0
    fi

    if [[ -f "${SHARED_DIR}/tailscale.sh" ]]; then
        local args=""
        # Pass through confirmation skip if set
        [[ "${SKIP_CONFIRMATION:-false}" == "true" ]] && args="-y"

        bash "${SHARED_DIR}/tailscale.sh" ${args} || record_error "tailscale" "Failed to setup Tailscale"
    else
        log_warning "tailscale.sh not found, skipping"
    fi
}

setup_github() {
    log_section "Setting Up GitHub CLI Authentication"

    if [[ -f "${SCRIPT_DIR}/setup-github.sh" ]]; then
        bash "${SCRIPT_DIR}/setup-github.sh" || record_error "github" "Failed to setup GitHub"
    else
        log_info "setup-github.sh not found, skipping GitHub auth setup"
    fi
}

#==============================================================================
# Main
#==============================================================================

main() {
    local start_time
    start_time=$(date +%s)

    # GitHub user for SSH key import
    local github_user="${GITHUB_USER:-}"

    # Parse arguments
    parse_dry_run_flag "$@"

    for arg in "$@"; do
        case "${arg}" in
            -y|--yes)
                export SKIP_CONFIRMATION=true
                ;;
            --gh=*)
                github_user="${arg#--gh=}"
                ;;
            --help|-h)
                echo "Usage: $0 [--dry-run] [-y|--yes] [--gh=USERNAME]"
                echo ""
                echo "Options:"
                echo "  --dry-run, -n    Preview changes without applying"
                echo "  -y, --yes        Skip confirmation prompts"
                echo "  --gh=USERNAME    Import SSH keys from GitHub username"
                echo ""
                echo "Environment Variables:"
                echo "  DRY_RUN=true          Enable dry-run mode"
                echo "  GITHUB_USER           GitHub username for SSH key import"
                echo "  TAILSCALE_AUTH_KEY    Tailscale auth key"
                echo "  GITHUB_PAT            GitHub Personal Access Token"
                exit 0
                ;;
        esac
    done

    log_section "Cloud-Init Installation"
    log_info "Starting installation..."

    if is_dry_run; then
        log_warning "DRY-RUN MODE: No changes will be made"
    else
        setup_logging
    fi

    # Setup SSH keys from GitHub (first step for remote access)
    setup_github_ssh_keys "${github_user}"

    # Create backup before making changes
    if ! is_dry_run; then
        backup_before_changes
    fi

    # Run installation steps
    install_apt_packages
    install_docker
    install_github_cli
    install_binary_packages
    configure_git
    configure_zsh
    switch_shell_to_zsh
    generate_shell_config
    setup_tailscale

    # Optional: Setup GitHub if PAT is available
    if [[ -n "${GITHUB_PAT:-}" ]]; then
        setup_github
    fi

    # Print summary
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo ""
    log_section "Installation Complete"
    log_info "Duration: ${duration} seconds"

    if ! is_dry_run; then
        log_info "Log file: ${LOG_FILE}"
    fi

    # Run health checks
    reset_health_checks
    log_section "Running Health Checks"

    for script in "${PACKAGES_DIR}"/*.sh; do
        [[ -f "${script}" && "$(basename "${script}")" != "_template.sh" ]] || continue
        bash "${script}" verify 2>/dev/null || true
    done

    print_health_summary

    # Print errors if any
    print_error_summary || true

    if is_dry_run; then
        print_dry_run_summary
    fi

    # Final message
    if [[ ${#ERRORS[@]} -eq 0 ]]; then
        echo ""
        log_success "Installation completed successfully!"
        echo ""
        echo "Next steps:"
        echo "  1. Log out and back in for shell changes to take effect"
        echo "  2. Run 'make verify-cloud' to verify installation"
        echo "  3. Configure Tailscale ACLs in the admin console"
    else
        echo ""
        log_warning "Installation completed with ${#ERRORS[@]} error(s)"
        echo "Check the log file for details: ${LOG_FILE}"
    fi
}

[[ "${BASH_SOURCE[0]}" == "${0}" ]] && main "$@"
