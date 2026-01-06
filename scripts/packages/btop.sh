#!/bin/bash
#==============================================================================
# btop Installer
#
# Resource monitor that shows usage and stats
# https://github.com/aristocratos/btop
#
# Installs via Homebrew for latest version
#
# Usage: ./btop.sh [install|update|verify|version]
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

PACKAGE_NAME="btop"

# shellcheck source=scripts/lib/brew.sh
source "${SCRIPT_DIR}/../lib/brew.sh"

is_installed() { command_exists btop; }

get_installed_version() {
    if is_installed; then
        btop --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
    fi
}

do_install() {
    log_info "Installing btop via Homebrew..."

    if is_dry_run; then
        echo "[DRY-RUN] Would run: brew install btop"
        return 0
    fi

    if ! command_exists brew; then
        log_error "Homebrew not installed. Please install homebrew first."
        return 1
    fi

    brew install btop

    log_success "btop installed"
}

do_update() {
    log_info "Updating btop via Homebrew..."

    if is_dry_run; then
        echo "[DRY-RUN] Would run: brew upgrade btop"
        return 0
    fi

    if ! command_exists brew; then
        log_error "Homebrew not installed. Please install homebrew first."
        return 1
    fi

    brew upgrade btop || {
        log_info "btop is already up-to-date"
    }

    log_success "btop updated"
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
    # btop doesn't need shell config
    log_debug "btop doesn't require shell configuration"
}

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

    # shellcheck source=config.env.template
    [[ -f "${PROJECT_ROOT}/config.env" ]] && source "${PROJECT_ROOT}/config.env"
    [[ "${PACKAGE_BTOP_ENABLED:-true}" != "true" ]] && { log_info "btop disabled"; return 0; }

    case "${action}" in
        install)
            if is_installed; then
                log_success "btop already installed: v$(get_installed_version)"
            else
                do_install
            fi
            create_shell_config
            verify
            ;;
        update)
            if is_installed; then
                do_update
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
