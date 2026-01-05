#!/bin/bash
#==============================================================================
# Docker Installer
#
# Installs Docker Engine using the official Docker repository.
#
# Usage: ./docker.sh [install|update|verify]
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
# shellcheck source=scripts/lib/health.sh
source "${SCRIPT_DIR}/../lib/health.sh"
# shellcheck source=scripts/lib/lock.sh
source "${SCRIPT_DIR}/../lib/lock.sh"

#==============================================================================
# Configuration
#==============================================================================

PACKAGE_NAME="docker"

#==============================================================================
# Functions
#==============================================================================

# Check if Docker is installed
is_installed() {
    command_exists docker
}

# Get installed Docker version
get_installed_version() {
    if is_installed; then
        docker --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
    fi
}

# Load config
load_config() {
    if [[ -f "${PROJECT_ROOT}/config.env" ]]; then
        # shellcheck source=/dev/null
        source "${PROJECT_ROOT}/config.env"
    fi
}

# Check if Docker repository is configured
is_repo_configured() {
    [[ -f /etc/apt/sources.list.d/docker.list ]]
}

# Add Docker repository
add_docker_repo() {
    log_info "Adding Docker repository..."

    # Install prerequisites
    apt_or_print update -qq
    apt_or_print install -y -qq ca-certificates curl gnupg

    # Add Docker's official GPG key
    if ! is_dry_run; then
        sudo install -m 0755 -d /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        sudo chmod a+r /etc/apt/keyrings/docker.gpg
    else
        echo "[DRY-RUN] Would add Docker GPG key"
    fi

    # Add repository
    if ! is_dry_run; then
        # shellcheck source=/dev/null disable=SC2154
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "${VERSION_CODENAME}") stable" | \
            sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    else
        echo "[DRY-RUN] Would add Docker repository"
    fi

    apt_or_print update -qq

    log_success "Docker repository added"
}

# Install Docker
do_install() {
    log_section "Installing Docker"

    load_config

    # Skip if Docker already installed
    if is_installed; then
        local version
        version=$(get_installed_version)
        log_success "Docker already installed: v${version}"
        return 0
    fi

    # Add repository if not configured
    if ! is_repo_configured; then
        add_docker_repo
    fi

    # Install Docker packages
    log_info "Installing Docker packages..."
    apt_or_print install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

    # Add user to docker group if configured
    if [[ "${DOCKER_ADD_TO_GROUP:-true}" == "true" ]]; then
        log_info "Adding ${USER} to docker group..."
        if ! is_dry_run; then
            sudo usermod -aG docker "${USER}" || true
        else
            echo "[DRY-RUN] Would add ${USER} to docker group"
        fi
        log_success "User added to docker group (logout required)"
    fi

    # Enable Docker on boot if configured
    if [[ "${DOCKER_START_ON_BOOT:-true}" == "true" ]]; then
        log_info "Enabling Docker on boot..."
        systemctl_or_print enable docker
        systemctl_or_print enable containerd
    fi

    # Update lock file
    local version
    version=$(get_installed_version)
    if [[ -n "${version}" ]]; then
        update_lock "${PACKAGE_NAME}" "${version}"
    fi

    log_success "Docker installed successfully"
}

# Verify installation
verify() {
    log_section "Verifying Docker Installation"

    # Check docker command
    if is_installed; then
        local version
        version=$(get_installed_version)
        health_pass "docker" "v${version}"
    else
        health_fail "docker" "not installed"
        return 1
    fi

    # Check docker group
    if groups | grep -q docker; then
        health_pass "docker-group" "user in group"
    else
        health_warn "docker-group" "user not in docker group (logout/login required)"
    fi

    # Check service (skip in Docker)
    if [[ ! -f /.dockerenv ]]; then
        if systemctl is-active --quiet docker 2>/dev/null; then
            health_pass "docker-service" "running"
        else
            health_warn "docker-service" "not running"
        fi
    fi

    return 0
}

# Create shell config
create_shell_config() {
    local config_dir="${HOME}/.config/shell"
    local config_file="${config_dir}/40-docker.sh"

    mkdir -p "${config_dir}" 2>/dev/null || true

    local content="# Docker configuration
# Generated by docker.sh

# Docker aliases
alias d='docker'
alias dc='docker compose'
alias dps='docker ps'
alias dimg='docker images'
alias dlogs='docker logs -f'
alias dexec='docker exec -it'
"
    write_or_print "${config_file}" "${content}"
}

#==============================================================================
# Main
#==============================================================================

main() {
    parse_dry_run_flag "$@"

    # Extract action from args, skipping flags
    local action="install"
    for arg in "$@"; do
        case "${arg}" in
            --dry-run|-n) ;;  # Skip flags
            *) action="${arg}"; break ;;
        esac
    done

    # Check if enabled
    load_config
    if [[ "${DOCKER_ENABLED:-true}" != "true" ]]; then
        log_info "Docker is disabled in config"
        return 0
    fi

    case "${action}" in
        install)
            do_install
            create_shell_config
            verify
            ;;
        update)
            log_info "Updating Docker via apt..."
            apt_or_print update -qq
            apt_or_print upgrade -y -qq docker-ce docker-ce-cli containerd.io
            verify
            ;;
        verify)
            verify
            ;;
        version)
            if is_installed; then
                get_installed_version
            else
                echo "not installed"
                return 1
            fi
            ;;
        *)
            echo "Usage: $0 [install|update|verify|version] [--dry-run]"
            exit 1
            ;;
    esac

    if is_dry_run; then
        print_dry_run_summary
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
