#!/bin/bash
#==============================================================================
# Version Library - Version comparison and management
#
# Usage: source "${SCRIPT_DIR}/lib/version.sh"
#==============================================================================

# Prevent double-sourcing
if [[ -n "${_LIB_VERSION_SOURCED:-}" ]]; then
    return 0
fi
_LIB_VERSION_SOURCED=1

# Source core library if not already sourced
SCRIPT_DIR_VERSION="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR_VERSION}/core.sh"

#==============================================================================
# Version Parsing
#==============================================================================

# Extract semantic version from a string (e.g., "v1.2.3" -> "1.2.3")
extract_version() {
    local input="$1"
    # Remove 'v' prefix if present, extract X.Y.Z pattern
    echo "${input}" | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1
}

# Split version into components
# Usage: read -ra parts <<< "$(split_version "1.2.3")"
split_version() {
    local version="$1"
    echo "${version//./ }"
}

#==============================================================================
# Version Comparison
#==============================================================================

# Compare two semantic versions
# Returns: 0 if equal, 1 if v1 > v2, 2 if v1 < v2
# Usage: version_compare "1.2.3" "1.2.4"
version_compare() {
    local v1="$1"
    local v2="$2"

    # Handle empty versions
    if [[ -z "${v1}" && -z "${v2}" ]]; then
        return 0
    elif [[ -z "${v1}" ]]; then
        return 2
    elif [[ -z "${v2}" ]]; then
        return 1
    fi

    # Extract just the version numbers
    v1=$(extract_version "${v1}")
    v2=$(extract_version "${v2}")

    # If versions are identical strings, they're equal
    if [[ "${v1}" == "${v2}" ]]; then
        return 0
    fi

    # Split into components
    local IFS='.'
    read -ra parts1 <<< "${v1}"
    read -ra parts2 <<< "${v2}"

    # Compare each component
    local max_parts=${#parts1[@]}
    if [[ ${#parts2[@]} -gt ${max_parts} ]]; then
        max_parts=${#parts2[@]}
    fi

    for ((i = 0; i < max_parts; i++)); do
        local p1=${parts1[i]:-0}
        local p2=${parts2[i]:-0}

        if [[ ${p1} -gt ${p2} ]]; then
            return 1
        elif [[ ${p1} -lt ${p2} ]]; then
            return 2
        fi
    done

    return 0
}

# Check if version1 is greater than version2
# Usage: if version_gt "1.2.3" "1.2.2"; then echo "newer"; fi
version_gt() {
    local result
    version_compare "$1" "$2"
    result=$?
    [[ ${result} -eq 1 ]]
}

# Check if version1 is greater than or equal to version2
version_gte() {
    local result
    version_compare "$1" "$2"
    result=$?
    [[ ${result} -eq 0 || ${result} -eq 1 ]]
}

# Check if version1 is less than version2
version_lt() {
    local result
    version_compare "$1" "$2"
    result=$?
    [[ ${result} -eq 2 ]]
}

# Check if version1 equals version2
version_eq() {
    local result
    version_compare "$1" "$2"
    result=$?
    [[ ${result} -eq 0 ]]
}

#==============================================================================
# Update Detection
#==============================================================================

# Check if an update is needed (current < desired)
# Usage: if needs_update "1.2.2" "1.2.3"; then install; fi
needs_update() {
    local current="$1"
    local desired="$2"

    # If desired is "latest", we need to fetch the actual version
    if [[ "${desired}" == "latest" ]]; then
        log_debug "Desired version is 'latest', needs_update returns true"
        return 0
    fi

    # If current is empty/unknown, definitely need update
    if [[ -z "${current}" || "${current}" == "unknown" ]]; then
        return 0
    fi

    version_lt "${current}" "${desired}"
}

#==============================================================================
# GitHub Release Helpers
#==============================================================================

# Cache directory for API responses
GITHUB_API_CACHE_DIR="${HOME}/.cache/local-remote/github-api"

# Get latest version from GitHub releases
# Usage: get_github_latest_version "jesseduffield/lazygit"
get_github_latest_version() {
    local repo="$1"
    local cache_file="${GITHUB_API_CACHE_DIR}/${repo//\//_}_latest.txt"
    local cache_max_age=3600  # 1 hour

    ensure_dir "${GITHUB_API_CACHE_DIR}"

    # Check cache first
    if [[ -f "${cache_file}" ]]; then
        local cache_age
        cache_age=$(($(date +%s) - $(stat -c %Y "${cache_file}" 2>/dev/null || stat -f %m "${cache_file}" 2>/dev/null)))
        if [[ ${cache_age} -lt ${cache_max_age} ]]; then
            log_debug "Using cached version for ${repo} (age: ${cache_age}s)"
            cat "${cache_file}"
            return 0
        fi
    fi

    # Fetch from GitHub API
    log_debug "Fetching latest version for ${repo} from GitHub API"
    local response
    response=$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" 2>/dev/null)

    if [[ -z "${response}" ]]; then
        log_warning "Failed to fetch latest version for ${repo}"
        # Return cached version if available
        if [[ -f "${cache_file}" ]]; then
            cat "${cache_file}"
            return 0
        fi
        return 1
    fi

    local version
    version=$(echo "${response}" | grep '"tag_name":' | sed -E 's/.*"tag_name": *"v?([^"]+)".*/\1/')

    if [[ -n "${version}" ]]; then
        echo "${version}" > "${cache_file}"
        echo "${version}"
        return 0
    fi

    return 1
}

# Clear GitHub API cache
clear_github_cache() {
    if [[ -d "${GITHUB_API_CACHE_DIR}" ]]; then
        rm -rf "${GITHUB_API_CACHE_DIR}"
        log_debug "Cleared GitHub API cache"
    fi
}

#==============================================================================
# Version Resolution
#==============================================================================

# Resolve a version string to an actual version number
# Handles "latest" by fetching from GitHub
# Usage: resolved=$(resolve_version "latest" "jesseduffield/lazygit")
resolve_version() {
    local version="$1"
    local github_repo="$2"

    if [[ "${version}" == "latest" ]]; then
        if [[ -n "${github_repo}" ]]; then
            get_github_latest_version "${github_repo}"
        else
            log_error "Cannot resolve 'latest' without GitHub repo"
            echo "latest"
        fi
    else
        echo "${version}"
    fi
}
