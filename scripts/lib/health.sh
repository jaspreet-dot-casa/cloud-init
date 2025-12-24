#!/bin/bash
#==============================================================================
# Health Check Library - Verify package installations
#
# Usage: source "${SCRIPT_DIR}/lib/health.sh"
#==============================================================================

# Prevent double-sourcing
if [[ -n "${_LIB_HEALTH_SOURCED:-}" ]]; then
    return 0
fi
_LIB_HEALTH_SOURCED=1

# Source core library if not already sourced
SCRIPT_DIR_HEALTH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR_HEALTH}/core.sh"

#==============================================================================
# Health Check Results
#==============================================================================

# Arrays to track health check results
declare -a HEALTH_PASSED=()
declare -a HEALTH_FAILED=()
declare -a HEALTH_WARNINGS=()

# Reset health check state
reset_health_checks() {
    HEALTH_PASSED=()
    HEALTH_FAILED=()
    HEALTH_WARNINGS=()
}

#==============================================================================
# Health Check Registration
#==============================================================================

# Register a passed health check
# Usage: health_pass "lazygit" "version 0.40.2"
health_pass() {
    local component="$1"
    local message="${2:-OK}"
    HEALTH_PASSED+=("${component}: ${message}")
    log_success "[HEALTH] ${component}: ${message}"
}

# Register a failed health check
# Usage: health_fail "lazygit" "not found in PATH"
health_fail() {
    local component="$1"
    local message="${2:-FAILED}"
    HEALTH_FAILED+=("${component}: ${message}")
    log_error "[HEALTH] ${component}: ${message}"
}

# Register a warning (non-critical issue)
# Usage: health_warn "lazygit" "version mismatch"
health_warn() {
    local component="$1"
    local message="${2:-WARNING}"
    HEALTH_WARNINGS+=("${component}: ${message}")
    log_warning "[HEALTH] ${component}: ${message}"
}

#==============================================================================
# Common Health Checks
#==============================================================================

# Check if a command exists and is executable
# Usage: check_command "lazygit" && health_pass "lazygit"
check_command() {
    local cmd="$1"
    local component="${2:-${cmd}}"

    if command_exists "${cmd}"; then
        return 0
    else
        health_fail "${component}" "command '${cmd}' not found"
        return 1
    fi
}

# Check command exists and get version
# Usage: check_command_version "lazygit" "--version" "lazygit"
check_command_version() {
    local cmd="$1"
    local version_flag="${2:---version}"
    local component="${3:-${cmd}}"

    if ! command_exists "${cmd}"; then
        health_fail "${component}" "not installed"
        return 1
    fi

    local version
    version=$("${cmd}" "${version_flag}" 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)

    if [[ -n "${version}" ]]; then
        health_pass "${component}" "v${version}"
        return 0
    else
        health_warn "${component}" "installed but version unknown"
        return 0
    fi
}

# Check if a file exists
# Usage: check_file_exists "$HOME/.config/starship.toml" "starship config"
check_file_exists() {
    local file="$1"
    local component="${2:-${file}}"

    if [[ -f "${file}" ]]; then
        health_pass "${component}" "exists"
        return 0
    else
        health_fail "${component}" "file not found: ${file}"
        return 1
    fi
}

# Check if a directory exists
# Usage: check_dir_exists "$HOME/.config/shell" "shell config dir"
check_dir_exists() {
    local dir="$1"
    local component="${2:-${dir}}"

    if [[ -d "${dir}" ]]; then
        health_pass "${component}" "exists"
        return 0
    else
        health_fail "${component}" "directory not found: ${dir}"
        return 1
    fi
}

# Check if a service is running (systemd)
# Usage: check_service_running "tailscaled" "Tailscale daemon"
check_service_running() {
    local service="$1"
    local component="${2:-${service}}"

    if ! command_exists systemctl; then
        health_warn "${component}" "systemctl not available"
        return 1
    fi

    if systemctl is-active --quiet "${service}" 2>/dev/null; then
        health_pass "${component}" "running"
        return 0
    else
        health_fail "${component}" "not running"
        return 1
    fi
}

# Check if user is in a group
# Usage: check_user_in_group "docker" "Docker group"
check_user_in_group() {
    local group="$1"
    local component="${2:-${group} group}"

    if groups | grep -q "\b${group}\b"; then
        health_pass "${component}" "user in group"
        return 0
    else
        health_warn "${component}" "user not in ${group} group"
        return 1
    fi
}

#==============================================================================
# Health Summary
#==============================================================================

# Print health check summary
# Usage: print_health_summary
# Returns: Number of failures
print_health_summary() {
    # shellcheck disable=SC2154
    local passed=${#HEALTH_PASSED[@]}
    local failed=${#HEALTH_FAILED[@]}
    local warnings=${#HEALTH_WARNINGS[@]}

    echo ""
    log_section "Health Check Summary"

    # shellcheck disable=SC2154
    echo -e "${GREEN}Passed:${NC}   ${passed}"
    # shellcheck disable=SC2154
    echo -e "${YELLOW}Warnings:${NC} ${warnings}"
    # shellcheck disable=SC2154
    echo -e "${RED}Failed:${NC}   ${failed}"

    if [[ ${warnings} -gt 0 ]]; then
        echo ""
        echo -e "${YELLOW}Warnings:${NC}"
        for warn in "${HEALTH_WARNINGS[@]}"; do
            echo "  - ${warn}"
        done
    fi

    if [[ ${failed} -gt 0 ]]; then
        echo ""
        echo -e "${RED}Failures:${NC}"
        for fail in "${HEALTH_FAILED[@]}"; do
            echo "  - ${fail}"
        done
    fi

    echo ""
    if [[ ${failed} -eq 0 ]]; then
        log_success "All health checks passed!"
    else
        log_error "${failed} health check(s) failed"
    fi

    return "${failed}"
}

# Quick overall health status
# Usage: if health_ok; then echo "healthy"; fi
health_ok() {
    [[ ${#HEALTH_FAILED[@]} -eq 0 ]]
}

#==============================================================================
# Structured Health Check Runner
#==============================================================================

# Run a health check function and capture result
# Usage: run_health_check "lazygit" verify_lazygit
run_health_check() {
    local name="$1"
    local check_fn="$2"

    log_debug "Running health check: ${name}"

    if "${check_fn}" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Run all health checks for installed packages
# Usage: run_all_health_checks
run_all_health_checks() {
    reset_health_checks

    log_section "Running Health Checks"

    local project_root
    project_root="$(get_project_root)"
    local packages_dir="${project_root}/scripts/packages"

    if [[ ! -d "${packages_dir}" ]]; then
        log_warning "No packages directory found"
        return 0
    fi

    # Run verify function from each package script
    for script in "${packages_dir}"/*.sh; do
        [[ -f "${script}" ]] || continue

        local package_name
        package_name=$(basename "${script}" .sh)

        # Source the script and run its verify function if it exists
        # shellcheck disable=SC1090
        source "${script}"
        if declare -f verify &>/dev/null; then
            verify
            # Unset to avoid conflicts with next script
            unset -f verify
        else
            log_debug "No verify function in ${package_name}"
        fi
    done

    print_health_summary
}
