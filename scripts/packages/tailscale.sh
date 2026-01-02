#!/bin/bash
#==============================================================================
# Tailscale Installer
#
# Zero-config VPN with built-in SSH and 2FA
# https://tailscale.com
#
# Installs via official apt repository
#
# Usage: ./tailscale.sh [install|update|verify|version]
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# shellcheck source=scripts/lib/core.sh
source "${SCRIPT_DIR}/../lib/core.sh"
# shellcheck source=scripts/lib/health.sh
source "${SCRIPT_DIR}/../lib/health.sh"
# shellcheck source=scripts/lib/dryrun.sh
source "${SCRIPT_DIR}/../lib/dryrun.sh"

PACKAGE_NAME="tailscale"

is_installed() { command_exists tailscale; }

get_installed_version() {
    if is_installed; then
        tailscale version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
    fi
}

do_install() {
    log_info "Installing Tailscale..."

    if is_dry_run; then
        echo "[DRY-RUN] Would install Tailscale via apt repository"
        return 0
    fi

    # Detect Ubuntu/Debian codename dynamically
    local distro_codename
    if [[ -f /etc/os-release ]]; then
        distro_codename=$(. /etc/os-release && echo "${VERSION_CODENAME}")
    else
        distro_codename=$(lsb_release -cs 2>/dev/null || echo "noble")
    fi

    # Add Tailscale's GPG key and repository
    curl -fsSL "https://pkgs.tailscale.com/stable/ubuntu/${distro_codename}.noarmor.gpg" | sudo tee /usr/share/keyrings/tailscale-archive-keyring.gpg >/dev/null
    curl -fsSL "https://pkgs.tailscale.com/stable/ubuntu/${distro_codename}.tailscale-keyring.list" | sudo tee /etc/apt/sources.list.d/tailscale.list

    # Install Tailscale
    sudo apt-get update -qq
    sudo apt-get install -y tailscale

    # Enable and start the daemon
    sudo systemctl enable --now tailscaled

    log_success "Tailscale installed"
    log_info "Run 'sudo tailscale up --ssh' to authenticate"
}

verify() {
    if ! is_installed; then
        health_fail "${PACKAGE_NAME}" "not installed"
        return 1
    fi
    health_pass "${PACKAGE_NAME}" "v$(get_installed_version)"
    return 0
}

create_shell_config() {
    # Tailscale doesn't need shell config
    log_debug "Tailscale doesn't require shell configuration"
}

main() {
    parse_dry_run_flag "$@"
    local action="${1:-install}"

    # shellcheck source=config.env.template
    [[ -f "${PROJECT_ROOT}/config.env" ]] && source "${PROJECT_ROOT}/config.env"
    [[ "${PACKAGE_TAILSCALE_ENABLED:-true}" != "true" ]] && { log_info "Tailscale disabled"; return 0; }

    case "${action}" in
        install|update)
            if is_installed && [[ "${action}" == "install" ]]; then
                log_success "Tailscale already installed: v$(get_installed_version)"
            else
                do_install
            fi
            create_shell_config
            verify
            ;;
        verify) verify ;;
        version)
            if is_installed; then
                get_installed_version
            else
                echo "not installed"
                return 1
            fi
            ;;
        *) echo "Usage: $0 [install|update|verify|version] [--dry-run]"; exit 1 ;;
    esac
}

[[ "${BASH_SOURCE[0]}" == "${0}" ]] && main "$@"
