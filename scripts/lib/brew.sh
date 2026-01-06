#!/bin/bash
#==============================================================================
# Homebrew Library - Detection and initialization functions
#
# Usage: source "${SCRIPT_DIR}/../lib/brew.sh"
#==============================================================================

# Prevent double-sourcing
if [[ -n "${_LIB_BREW_SOURCED:-}" ]]; then
    return 0
fi
_LIB_BREW_SOURCED=1

#==============================================================================
# Homebrew Detection
#==============================================================================

# Detect Homebrew prefix dynamically across Linux and macOS
# Returns the path to Homebrew prefix, or empty string if not found
get_brew_prefix() {
    # If brew is already in PATH, use it directly
    if command -v brew &>/dev/null; then
        brew --prefix
        return 0
    fi

    # Check common Homebrew locations
    local prefixes=(
        "/home/linuxbrew/.linuxbrew"  # Linux (Linuxbrew)
        "/opt/homebrew"                # macOS Apple Silicon
        "/usr/local"                   # macOS Intel
    )

    for prefix in "${prefixes[@]}"; do
        if [[ -x "${prefix}/bin/brew" ]]; then
            echo "${prefix}"
            return 0
        fi
    done

    # Not found
    return 1
}

# Initialize Homebrew environment (add to PATH, set env vars)
# Call this before checking for brew-installed commands
init_brew_env() {
    local prefix
    prefix=$(get_brew_prefix) || return 1

    # Only initialize if brew exists at this prefix
    if [[ -x "${prefix}/bin/brew" ]]; then
        eval "$("${prefix}/bin/brew" shellenv)"
        return 0
    fi

    return 1
}

# Check if Homebrew is installed and available
ensure_brew_installed() {
    if ! command -v brew &>/dev/null; then
        # Try to initialize brew env first
        if ! init_brew_env; then
            return 1
        fi
    fi

    # Double-check brew is now available
    command -v brew &>/dev/null
}

# Global BREW_PREFIX variable - set once when this library is sourced
# This is used by scripts that need the prefix for constructing paths
BREW_PREFIX=""
if _prefix=$(get_brew_prefix 2>/dev/null); then
    BREW_PREFIX="${_prefix}"
    # Auto-initialize brew environment when library is sourced
    [[ -x "${BREW_PREFIX}/bin/brew" ]] && eval "$("${BREW_PREFIX}/bin/brew" shellenv)"
fi
unset _prefix
