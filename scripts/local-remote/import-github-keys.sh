#!/bin/bash
#==============================================================================
# Import SSH Authorized Keys from GitHub
#
# Fetches your public SSH keys from GitHub and adds them to authorized_keys.
# This allows you to SSH into this machine using keys already on your GitHub.
#
# Usage: ./import-github-keys.sh [github-username]
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

import_github_keys() {
    log_section "Import SSH Keys from GitHub"

    local ssh_dir="$HOME/.ssh"
    local auth_keys_file="$ssh_dir/authorized_keys"

    # Get GitHub username from argument or detect/ask
    local github_username="${1:-}"

    # Try to get GitHub username from gh CLI if not provided
    if [[ -z "$github_username" ]]; then
        if command -v gh &>/dev/null && gh auth status &>/dev/null 2>&1; then
            github_username=$(gh api user -q .login 2>/dev/null || echo "")
        fi
    fi

    # If still not found, ask the user
    if [[ -z "$github_username" ]]; then
        echo -n "Enter your GitHub username: "
        read -r github_username
    fi

    if [[ -z "$github_username" ]]; then
        log_warning "No GitHub username provided. Skipping key import."
        return 0
    fi

    log_info "Fetching SSH keys for GitHub user: $github_username"

    # Fetch keys from GitHub
    local github_keys_url="https://github.com/${github_username}.keys"
    local keys
    keys=$(curl -fsSL "$github_keys_url" 2>/dev/null || echo "")

    if [[ -z "$keys" ]]; then
        log_warning "No SSH keys found for $github_username (or user not found)"
        return 0
    fi

    # Count keys
    local key_count
    key_count=$(echo "$keys" | wc -l | tr -d ' ')
    log_info "Found $key_count SSH key(s)"

    # Ensure .ssh directory exists
    mkdir -p "$ssh_dir"
    chmod 700 "$ssh_dir"

    # Create authorized_keys if it doesn't exist
    touch "$auth_keys_file"
    chmod 600 "$auth_keys_file"

    # Add keys that aren't already present
    local added=0
    while IFS= read -r key; do
        [[ -z "$key" ]] && continue

        # Check if key already exists
        if grep -qF "$key" "$auth_keys_file" 2>/dev/null; then
            log_info "Key already exists (skipping): ${key:0:40}..."
        else
            echo "$key" >> "$auth_keys_file"
            ((added++))
            log_success "Added key: ${key:0:40}..."
        fi
    done <<< "$keys"

    if [[ $added -gt 0 ]]; then
        log_success "Added $added new SSH key(s) to authorized_keys"
    else
        log_info "All keys already present in authorized_keys"
    fi

    echo ""
    echo "You can now SSH to this machine using your GitHub SSH keys."
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    import_github_keys "$@"
fi
