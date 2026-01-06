#!/bin/bash
#==============================================================================
# Ripgrep Installer
#
# A line-oriented search tool that recursively searches directories
# https://github.com/BurntSushi/ripgrep
#
# Installs via Homebrew for latest version
#
# Usage: ./ripgrep.sh [install|update|verify|version]
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

PACKAGE_NAME="ripgrep"

# shellcheck source=scripts/lib/brew.sh
source "${SCRIPT_DIR}/../lib/brew.sh"

is_installed() { command_exists rg; }

get_installed_version() {
    if is_installed; then
        rg --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
    fi
}

do_install() {
    log_info "Installing ripgrep via Homebrew..."

    if is_dry_run; then
        echo "[DRY-RUN] Would run: brew install ripgrep"
        return 0
    fi

    if ! command_exists brew; then
        log_error "Homebrew not installed. Please install homebrew first."
        return 1
    fi

    brew install ripgrep

    log_success "ripgrep installed"
}

verify() {
    if ! is_installed; then
        health_fail "${PACKAGE_NAME}" "not installed"
        return 1
    fi
    health_pass "${PACKAGE_NAME}" "v$(get_installed_version)"
    return 0
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
    [[ "${PACKAGE_RIPGREP_ENABLED:-true}" != "true" ]] && { log_info "ripgrep disabled"; return 0; }

    case "${action}" in
        install|update)
            if is_installed && [[ "${action}" == "install" ]]; then
                log_success "ripgrep already installed: v$(get_installed_version)"
            else
                do_install
            fi
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
