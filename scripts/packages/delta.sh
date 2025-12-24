#!/bin/bash
#==============================================================================
# Delta Installer
#
# A syntax-highlighting pager for git, diff, and grep output
# https://github.com/dandavison/delta
#
# Uses GitHub releases (no official installer script)
#
# Usage: ./delta.sh [install|update|verify|version]
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# shellcheck source=../lib/core.sh disable=SC1091
source "${SCRIPT_DIR}/../lib/core.sh"
# shellcheck source=../lib/health.sh disable=SC1091
source "${SCRIPT_DIR}/../lib/health.sh"
# shellcheck source=../lib/dryrun.sh disable=SC1091
source "${SCRIPT_DIR}/../lib/dryrun.sh"

PACKAGE_NAME="delta"
INSTALL_PATH="/usr/local/bin/delta"

is_installed() { command_exists delta; }

get_installed_version() {
    if is_installed; then
        delta --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
    fi
}

get_latest_version() {
    curl -fsSL "https://api.github.com/repos/dandavison/delta/releases/latest" 2>/dev/null | \
        grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
}

do_install() {
    log_info "Installing delta..."

    if is_dry_run; then
        echo "[DRY-RUN] Would download and install delta"
        return 0
    fi

    local arch
    arch=$(uname -m)
    # delta uses x86_64 and aarch64
    [[ "${arch}" == "arm64" ]] && arch="aarch64"

    local version
    version=$(get_latest_version)
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "${tmp_dir}"' EXIT

    local url="https://github.com/dandavison/delta/releases/download/${version}/delta-${version}-${arch}-unknown-linux-gnu.tar.gz"

    curl -fsSL "${url}" -o "${tmp_dir}/delta.tar.gz"
    tar -xzf "${tmp_dir}/delta.tar.gz" -C "${tmp_dir}"
    sudo install -m 755 "${tmp_dir}/delta-${version}-${arch}-unknown-linux-gnu/delta" "${INSTALL_PATH}"

    log_success "delta installed"
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
    # Delta is configured via gitconfig, not shell
    log_debug "delta is configured via gitconfig"
}

main() {
    parse_dry_run_flag "$@"
    local action="${1:-install}"

    # shellcheck source=../../config.env disable=SC1091
    [[ -f "${PROJECT_ROOT}/config.env" ]] && source "${PROJECT_ROOT}/config.env"
    [[ "${PACKAGE_DELTA_ENABLED:-true}" != "true" ]] && { log_info "delta disabled"; return 0; }

    case "${action}" in
        install|update)
            if is_installed && [[ "${action}" == "install" ]]; then
                log_success "delta already installed: v$(get_installed_version)"
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
