#!/bin/bash
#==============================================================================
# btop Installer
#
# Resource monitor that shows usage and stats
# https://github.com/aristocratos/btop
#
# Uses apt on Ubuntu 22.04+ (available in universe repo)
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

is_installed() { command_exists btop; }

get_installed_version() {
    if is_installed; then
        btop --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
    fi
}

do_install() {
    log_info "Installing btop via apt..."

    if is_dry_run; then
        echo "[DRY-RUN] Would run: sudo apt-get install -y btop"
        return 0
    fi

    sudo apt-get update -qq
    sudo apt-get install -y btop

    log_success "btop installed"
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
    local action="${1:-install}"

    # shellcheck source=config.env.template
    [[ -f "${PROJECT_ROOT}/config.env" ]] && source "${PROJECT_ROOT}/config.env"
    [[ "${PACKAGE_BTOP_ENABLED:-true}" != "true" ]] && { log_info "btop disabled"; return 0; }

    case "${action}" in
        install|update)
            if is_installed && [[ "${action}" == "install" ]]; then
                log_success "btop already installed: v$(get_installed_version)"
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
