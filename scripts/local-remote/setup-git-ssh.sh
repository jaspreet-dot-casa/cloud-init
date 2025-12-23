#!/bin/bash
#==============================================================================
# Git SSH Key Setup
#
# Generates an SSH key and optionally adds it to GitHub.
#
# Usage: ./setup-git-ssh.sh
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

setup_git_ssh() {
    log_section "Git SSH Key Setup"

    local ssh_dir="$HOME/.ssh"
    local key_file="$ssh_dir/id_ed25519"
    local pub_key_file="${key_file}.pub"

    # Check if key already exists
    if [[ -f "$key_file" ]]; then
        log_success "SSH key already exists: $key_file"
        echo ""
        echo "Public key:"
        cat "$pub_key_file"
        echo ""
    else
        log_info "Generating new SSH key..."

        # Get email for key
        local email
        email=$(git config --global user.email 2>/dev/null || echo "")

        if [[ -z "$email" ]]; then
            echo -n "Enter your email for the SSH key: "
            read -r email
        fi

        mkdir -p "$ssh_dir"
        chmod 700 "$ssh_dir"

        ssh-keygen -t ed25519 -C "$email" -f "$key_file" -N ""

        log_success "SSH key generated"
        echo ""
        echo "Public key:"
        cat "$pub_key_file"
        echo ""
    fi

    # Ensure SSH agent is running and key is added
    eval "$(ssh-agent -s)" &>/dev/null || true
    ssh-add "$key_file" 2>/dev/null || true

    # Check if gh is installed for GitHub auth
    if command -v gh &>/dev/null; then
        setup_github_ssh "$pub_key_file"
    else
        log_warning "GitHub CLI (gh) not installed. Skipping GitHub SSH setup."
        echo ""
        echo "To add this key to GitHub manually:"
        echo "  1. Go to https://github.com/settings/keys"
        echo "  2. Click 'New SSH key'"
        echo "  3. Paste the public key shown above"
    fi
}

setup_github_ssh() {
    local pub_key_file="$1"

    log_section "GitHub Authentication"

    # Check if already authenticated with gh
    if gh auth status &>/dev/null; then
        log_success "Already authenticated with GitHub CLI"

        # Check if SSH key is already added
        local key_fingerprint
        key_fingerprint=$(ssh-keygen -lf "$pub_key_file" | awk '{print $2}')

        if gh ssh-key list 2>/dev/null | grep -q "$key_fingerprint"; then
            log_success "SSH key already added to GitHub"
            return 0
        fi
    else
        log_info "Authenticating with GitHub..."
        echo ""
        echo "This will open a browser to authenticate with GitHub."
        echo ""

        gh auth login --web --git-protocol ssh

        log_success "GitHub authenticated"
    fi

    # Add SSH key to GitHub
    log_info "Adding SSH key to GitHub..."

    local hostname
    hostname=$(hostname)
    local key_title="local-remote@${hostname}"

    if gh ssh-key add "$pub_key_file" --title "$key_title" 2>/dev/null; then
        log_success "SSH key added to GitHub as '$key_title'"
    else
        log_warning "Could not add SSH key (may already exist)"
    fi

    # Test SSH connection
    log_info "Testing GitHub SSH connection..."
    if ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
        log_success "GitHub SSH connection working"
    else
        log_warning "GitHub SSH test returned unexpected result (this may be OK)"
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    setup_git_ssh "$@"
fi
