#!/bin/bash
#==============================================================================
# Core Library - Colors, logging, and common functions
#
# Usage: source "${SCRIPT_DIR}/lib/core.sh"
#==============================================================================

# Prevent double-sourcing
if [[ -n "${_LIB_CORE_SOURCED:-}" ]]; then
    return 0
fi
_LIB_CORE_SOURCED=1

#==============================================================================
# Color Definitions
#==============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m' # No Color

#==============================================================================
# Logging Functions
#==============================================================================

log_info() {
    echo -e "${BLUE}→ $1${NC}" >&2
}

log_success() {
    echo -e "${GREEN}✓ $1${NC}" >&2
}

log_warning() {
    echo -e "${YELLOW}⚠ $1${NC}" >&2
}

log_error() {
    echo -e "${RED}✗ $1${NC}" >&2
}

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo -e "${CYAN}[DEBUG] $1${NC}" >&2
    fi
}

log_section() {
    echo "" >&2
    echo -e "${BLUE}════════════════════════════════════════════${NC}" >&2
    echo -e "${BLUE}${BOLD}$1${NC}" >&2
    echo -e "${BLUE}════════════════════════════════════════════${NC}" >&2
}

#==============================================================================
# Common Utility Functions
#==============================================================================

# Check if running as root
is_root() {
    [[ ${EUID} -eq 0 ]]
}

# Check if running in Docker container
is_docker() {
    [[ -f /.dockerenv ]] || grep -q docker /proc/1/cgroup 2>/dev/null
}

# Check if a command exists
command_exists() {
    command -v "$1" &>/dev/null
}

# Get the project root directory (assumes scripts are in scripts/lib/)
# Uses subshells to avoid changing the caller's working directory
get_project_root() {
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[1]}")" && pwd)"
    # Navigate up from scripts/lib/ or scripts/ to project root
    if [[ "${script_dir}" == */scripts/lib ]]; then
        (cd "${script_dir}/../.." && pwd)
    elif [[ "${script_dir}" == */scripts/* ]]; then
        (cd "${script_dir}/../.." && pwd)
    elif [[ "${script_dir}" == */scripts ]]; then
        (cd "${script_dir}/.." && pwd)
    else
        (cd "${script_dir}" && pwd)
    fi
}

# Ensure we're running on a supported OS
check_os() {
    if [[ ! -f /etc/os-release ]]; then
        log_error "Cannot detect OS. /etc/os-release not found."
        return 1
    fi

    # shellcheck disable=SC1091,SC2034
    source /etc/os-release

    # shellcheck disable=SC2154
    if [[ "${ID}" != "ubuntu" ]]; then
        log_warning "This script is designed for Ubuntu. Detected: ${ID}"
        return 1
    fi

    # shellcheck disable=SC2154
    log_debug "OS: ${PRETTY_NAME}"
    return 0
}

# Get system architecture in a standardized format
get_arch() {
    local arch
    arch=$(uname -m)
    case "${arch}" in
        x86_64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l)
            echo "armv7"
            ;;
        *)
            echo "${arch}"
            ;;
    esac
}

# Get architecture in GitHub release format (often different)
get_github_arch() {
    local arch
    arch=$(uname -m)
    case "${arch}" in
        x86_64)
            echo "x86_64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            echo "${arch}"
            ;;
    esac
}

#==============================================================================
# Error Handling
#==============================================================================

# Array to collect errors for summary
declare -a COLLECTED_ERRORS=()

# Add an error to the collection
collect_error() {
    local component="$1"
    local message="$2"
    COLLECTED_ERRORS+=("[${component}] ${message}")
    log_error "[${component}] ${message}"
}

# Print error summary and return exit code
print_error_summary() {
    local count=${#COLLECTED_ERRORS[@]}

    if [[ ${count} -eq 0 ]]; then
        log_success "All operations completed successfully"
        return 0
    fi

    echo "" >&2
    log_section "Error Summary"
    for error in "${COLLECTED_ERRORS[@]}"; do
        echo -e "  ${RED}${error}${NC}" >&2
    done
    echo "" >&2
    log_error "${count} error(s) occurred"
    return "${count}"
}

#==============================================================================
# Configuration Loading
#==============================================================================

# Load config.env from project root
load_config() {
    local project_root
    project_root="$(get_project_root)"
    local config_file="${project_root}/config.env"

    if [[ -f "${config_file}" ]]; then
        log_debug "Loading configuration from ${config_file}"
        # shellcheck disable=SC1090
        source "${config_file}"
        return 0
    else
        log_debug "No config.env found at ${config_file}"
        return 1
    fi
}

#==============================================================================
# File Operations
#==============================================================================

# Safely create a directory
ensure_dir() {
    local dir="$1"
    if [[ ! -d "${dir}" ]]; then
        mkdir -p "${dir}"
        log_debug "Created directory: ${dir}"
    fi
}

# Check if file exists and is readable
file_readable() {
    [[ -f "$1" && -r "$1" ]]
}

# Backup a file before modifying
backup_file() {
    local file="$1"
    local backup_dir="${2:-${HOME}/.local-remote/backups}"
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)

    if [[ -f "${file}" ]]; then
        ensure_dir "${backup_dir}"
        local filename
        filename=$(basename "${file}")
        cp "${file}" "${backup_dir}/${filename}.${timestamp}.bak"
        log_debug "Backed up ${file} to ${backup_dir}/${filename}.${timestamp}.bak"
    fi
}
