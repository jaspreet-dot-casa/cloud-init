#!/bin/bash
#==============================================================================
# bat Installer
#
# A cat clone with syntax highlighting and Git integration
# https://github.com/sharkdp/bat
#
# Uses apt on Ubuntu (package name: bat, binary: batcat)
# Creates symlink for 'bat' command
#
# Usage: ./bat.sh [install|update|verify|version]
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

PACKAGE_NAME="bat"

# bat is installed as 'batcat' on Ubuntu due to name conflict
is_installed() { command_exists batcat || command_exists bat; }

get_installed_version() {
    if command_exists bat; then
        bat --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
    elif command_exists batcat; then
        batcat --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
    fi
}

do_install() {
    log_info "Installing bat via apt..."

    apt_or_print update -qq
    apt_or_print install -y bat

    # Create symlink for 'bat' command (Ubuntu installs as 'batcat')
    if command_exists batcat && ! command_exists bat; then
        mkdir -p "${HOME}/.local/bin"
        if is_dry_run; then
            echo "[DRY-RUN] Would create symlink: ln -sf batcat ~/.local/bin/bat"
        else
            ln -sf "$(command -v batcat)" "${HOME}/.local/bin/bat"
        fi
        log_info "Created symlink: bat -> batcat"
    fi

    log_success "bat installed"
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
    # bat doesn't need shell config beyond the symlink
    log_debug "bat doesn't require shell configuration"
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
    [[ "${PACKAGE_BAT_ENABLED:-true}" != "true" ]] && { log_info "bat disabled"; return 0; }

    case "${action}" in
        install|update)
            if is_installed && [[ "${action}" == "install" ]]; then
                log_success "bat already installed: v$(get_installed_version)"
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
