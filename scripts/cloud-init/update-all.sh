#!/bin/bash
#==============================================================================
# Cloud-Init Master Update Script
#
# Updates all packages and configurations idempotently.
# Safe to run multiple times - only applies necessary changes.
#
# Usage: ./update-all.sh [--dry-run] [--verify-only]
#
# Environment Variables:
#   DRY_RUN=true    Preview changes without applying
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
LOG_DIR="${HOME}/.config/ucli/logs"
LOG_FILE=""
VERIFY_ONLY=false

# Track updates and errors
declare -a UPDATED=()
declare -a ERRORS=()

#==============================================================================
# Logging
#==============================================================================

setup_logging() {
    ensure_dir "${LOG_DIR}"
    LOG_FILE="${LOG_DIR}/update-$(date +%Y%m%d_%H%M%S).log"
    exec > >(tee -a "${LOG_FILE}") 2>&1
    log_info "Log file: ${LOG_FILE}"
}

#==============================================================================
# Error Handling
#==============================================================================

record_error() {
    ERRORS+=("[$1] $2")
    log_error "[$1] $2"
}

record_update() {
    UPDATED+=("$1")
    log_success "Updated: $1"
}

#==============================================================================
# Update Functions
#==============================================================================

update_apt_packages() {
    log_section "Updating APT Packages"

    if [[ -f "${PACKAGES_DIR}/apt.sh" ]]; then
        bash "${PACKAGES_DIR}/apt.sh" update || record_error "apt" "Failed to update"
    fi
}

update_binary_packages() {
    log_section "Updating Binary Packages"

    local packages=(
        "yq"
        "github-cli"
        "lazygit"
        "lazydocker"
        "starship"
        "delta"
        "zellij"
        "zoxide"
        "btop"
    )

    for pkg in "${packages[@]}"; do
        local script="${PACKAGES_DIR}/${pkg}.sh"
        if [[ -f "${script}" ]]; then
            log_info "Checking ${pkg}..."
            if bash "${script}" update 2>&1 | grep -q "Updating\|Installing"; then
                record_update "${pkg}"
            fi
        fi
    done
}

regenerate_configs() {
    log_section "Regenerating Configurations"

    # Regenerate Git config
    if [[ -f "${SHARED_DIR}/configure-git.sh" ]]; then
        log_info "Updating Git configuration..."
        bash "${SHARED_DIR}/configure-git.sh" || record_error "git" "Failed to configure"
    fi

    # Regenerate Zsh config
    if [[ -f "${SHARED_DIR}/configure-zsh.sh" ]]; then
        log_info "Updating Zsh configuration..."
        bash "${SHARED_DIR}/configure-zsh.sh" || record_error "zsh" "Failed to configure"
    fi

    # Regenerate shell config
    if [[ -f "${SHARED_DIR}/generate-shell-config.sh" ]]; then
        log_info "Regenerating shell configuration..."
        bash "${SHARED_DIR}/generate-shell-config.sh" || record_error "shell-config" "Failed to generate"
    fi
}

verify_all() {
    log_section "Verifying Installation"

    reset_health_checks

    # Verify all packages
    for script in "${PACKAGES_DIR}"/*.sh; do
        [[ -f "${script}" && "$(basename "${script}")" != "_template.sh" ]] || continue
        local pkg
        pkg=$(basename "${script}" .sh)
        log_debug "Verifying ${pkg}..."
        bash "${script}" verify 2>/dev/null || true
    done

    print_health_summary
    return $?
}

#==============================================================================
# Summary
#==============================================================================

print_summary() {
    echo ""
    log_section "Update Summary"

    if [[ ${#UPDATED[@]} -gt 0 ]]; then
        echo "Updated packages:"
        for pkg in "${UPDATED[@]}"; do
            echo "  - ${pkg}"
        done
    else
        echo "No packages needed updating"
    fi

    if [[ ${#ERRORS[@]} -gt 0 ]]; then
        echo ""
        echo "Errors:"
        for error in "${ERRORS[@]}"; do
            echo "  - ${error}"
        done
    fi

    echo ""
    if [[ ${#ERRORS[@]} -eq 0 ]]; then
        log_success "Update completed successfully"
    else
        log_warning "Update completed with ${#ERRORS[@]} error(s)"
    fi
}

#==============================================================================
# Main
#==============================================================================

main() {
    local start_time
    start_time=$(date +%s)

    # Parse arguments
    parse_dry_run_flag "$@"

    for arg in "$@"; do
        case "${arg}" in
            --verify-only)
                VERIFY_ONLY=true
                ;;
            --help|-h)
                echo "Usage: $0 [--dry-run] [--verify-only]"
                echo ""
                echo "Options:"
                echo "  --dry-run, -n     Preview changes without applying"
                echo "  --verify-only     Only run verification, no updates"
                echo ""
                exit 0
                ;;
        esac
    done

    log_section "Cloud-Init Update"

    if [[ "${VERIFY_ONLY}" == "true" ]]; then
        log_info "Running verification only..."
        verify_all
        exit $?
    fi

    if is_dry_run; then
        log_warning "DRY-RUN MODE: No changes will be made"
    else
        setup_logging
        backup_before_changes
    fi

    # Run update steps
    update_apt_packages
    update_binary_packages
    regenerate_configs

    # Verify
    verify_all

    # Print summary
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    print_summary
    log_info "Duration: ${duration} seconds"

    if ! is_dry_run && [[ -n "${LOG_FILE}" ]]; then
        log_info "Log file: ${LOG_FILE}"
    fi

    if is_dry_run; then
        print_dry_run_summary
    fi

    # Return error count as exit code (0 = success)
    return ${#ERRORS[@]}
}

[[ "${BASH_SOURCE[0]}" == "${0}" ]] && main "$@"
