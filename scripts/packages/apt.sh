#!/bin/bash
#==============================================================================
# APT Package Installer
#
# Installs system packages via apt-get based on config.env
#
# Usage: ./apt.sh [install|update|verify]
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source shared libraries
source "${SCRIPT_DIR}/../lib/core.sh"
source "${SCRIPT_DIR}/../lib/dryrun.sh"
source "${SCRIPT_DIR}/../lib/health.sh"

#==============================================================================
# Configuration
#==============================================================================

PACKAGE_NAME="apt"

# Default packages if not specified in config.env
DEFAULT_APT_PACKAGES="curl wget git zsh tree jq htop unzip build-essential"

#==============================================================================
# Functions
#==============================================================================

# Load package list from config
get_packages() {
    if [[ -f "${PROJECT_ROOT}/config.env" ]]; then
        # shellcheck source=/dev/null
        source "${PROJECT_ROOT}/config.env"
    fi
    echo "${APT_PACKAGES:-$DEFAULT_APT_PACKAGES}"
}

# Update apt cache
update_cache() {
    log_info "Updating apt cache..."
    apt_or_print update -qq
}

# Install packages
do_install() {
    local packages
    packages=$(get_packages)

    log_section "Installing APT Packages"
    log_info "Packages: $packages"

    update_cache

    log_info "Installing packages..."
    # shellcheck disable=SC2086
    apt_or_print install -y -qq $packages

    log_success "APT packages installed"
}

# Upgrade installed packages
do_upgrade() {
    log_section "Upgrading APT Packages"

    update_cache

    log_info "Upgrading packages..."
    apt_or_print upgrade -y -qq

    log_success "APT packages upgraded"
}

# Verify packages are installed
verify() {
    local packages
    packages=$(get_packages)

    log_section "Verifying APT Packages"

    local missing=()
    for pkg in $packages; do
        if dpkg -l "$pkg" &>/dev/null; then
            health_pass "apt:$pkg" "installed"
        else
            health_fail "apt:$pkg" "not installed"
            missing+=("$pkg")
        fi
    done

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing packages: ${missing[*]}"
        return 1
    fi

    return 0
}

#==============================================================================
# Main
#==============================================================================

main() {
    parse_dry_run_flag "$@"

    local action="${1:-install}"

    case "$action" in
        install)
            do_install
            verify
            ;;
        update|upgrade)
            do_upgrade
            verify
            ;;
        verify)
            verify
            ;;
        *)
            echo "Usage: $0 [install|update|verify] [--dry-run]"
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
