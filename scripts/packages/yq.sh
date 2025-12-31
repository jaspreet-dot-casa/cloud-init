#!/bin/bash
#==============================================================================
# yq Installer
#
# A portable command-line YAML processor
# https://github.com/mikefarah/yq
#
# Uses GitHub releases (downloads latest)
#
# Usage: ./yq.sh [install|update|verify|version]
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

PACKAGE_NAME="yq"
INSTALL_PATH="/usr/local/bin/yq"

is_installed() { command_exists yq; }

get_installed_version() {
    if is_installed; then
        yq --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
    fi
}

do_install() {
    log_info "Installing yq..."

    if is_dry_run; then
        echo "[DRY-RUN] Would download and install yq"
        return 0
    fi

    local arch
    arch=$(uname -m)
    # yq uses amd64 and arm64
    case "${arch}" in
        x86_64) arch="amd64" ;;
        aarch64) arch="arm64" ;;
    esac

    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf '${tmp_dir}'" EXIT

    # Download latest binary directly
    local url="https://github.com/mikefarah/yq/releases/latest/download/yq_linux_${arch}"

    curl -fsSL "${url}" -o "${tmp_dir}/yq"
    chmod +x "${tmp_dir}/yq"
    sudo install -m 755 "${tmp_dir}/yq" "${INSTALL_PATH}"

    log_success "yq installed"
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
    # yq doesn't need shell config
    log_debug "yq doesn't require shell configuration"
}

main() {
    parse_dry_run_flag "$@"
    local action="${1:-install}"

    # shellcheck source=config.env.template
    [[ -f "${PROJECT_ROOT}/config.env" ]] && source "${PROJECT_ROOT}/config.env"
    [[ "${PACKAGE_YQ_ENABLED:-true}" != "true" ]] && { log_info "yq disabled"; return 0; }

    case "${action}" in
        install|update)
            if is_installed && [[ "${action}" == "install" ]]; then
                log_success "yq already installed: v$(get_installed_version)"
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
