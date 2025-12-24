#!/bin/bash
#==============================================================================
# GitHub CLI Setup Script
#
# Authenticates with GitHub CLI and sets up SSH keys.
#
# Usage: ./setup-github.sh [--dry-run]
#
# Environment Variables:
#   GITHUB_PAT    GitHub Personal Access Token (required for non-interactive)
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# shellcheck source=scripts/lib/core.sh
source "${SCRIPT_DIR}/../lib/core.sh"
# shellcheck source=scripts/lib/dryrun.sh
source "${SCRIPT_DIR}/../lib/dryrun.sh"
# shellcheck source=scripts/lib/health.sh
source "${SCRIPT_DIR}/../lib/health.sh"

#==============================================================================
# Configuration
#==============================================================================

SSH_KEY_PATH="${HOME}/.ssh/id_ed25519"
SSH_KEY_TYPE="ed25519"

#==============================================================================
# Functions
#==============================================================================

is_gh_installed() {
    command_exists gh
}

is_authenticated() {
    gh auth status &>/dev/null 2>&1
}

authenticate_with_pat() {
    local pat="${GITHUB_PAT:-}"

    if [[ -z "${pat}" ]]; then
        log_warning "GITHUB_PAT not set, skipping automatic authentication"
        log_info "To authenticate manually, run: gh auth login"
        return 1
    fi

    log_info "Authenticating with GitHub using PAT..."

    if is_dry_run; then
        echo "[DRY-RUN] Would authenticate with GitHub using PAT"
        return 0
    fi

    echo "${pat}" | gh auth login --with-token

    log_success "Authenticated with GitHub"
}

authenticate_interactive() {
    log_info "Starting interactive GitHub authentication..."

    if is_dry_run; then
        echo "[DRY-RUN] Would start interactive authentication"
        return 0
    fi

    gh auth login

    log_success "Authenticated with GitHub"
}

generate_ssh_key() {
    if [[ -f "${SSH_KEY_PATH}" ]]; then
        log_success "SSH key already exists: ${SSH_KEY_PATH}"
        return 0
    fi

    log_info "Generating SSH key..."

    # Load config for email
    if [[ -f "${PROJECT_ROOT}/config.env" ]]; then
        # shellcheck source=/dev/null
        source "${PROJECT_ROOT}/config.env"
    fi
    local email="${USER_EMAIL:-$(whoami)@$(hostname)}"

    if is_dry_run; then
        echo "[DRY-RUN] Would generate SSH key: ${SSH_KEY_PATH}"
        return 0
    fi

    ssh-keygen -t "${SSH_KEY_TYPE}" -C "${email}" -f "${SSH_KEY_PATH}" -N ""

    log_success "SSH key generated: ${SSH_KEY_PATH}"
}

upload_ssh_key() {
    if ! is_authenticated; then
        log_warning "Not authenticated with GitHub, cannot upload SSH key"
        return 1
    fi

    if [[ ! -f "${SSH_KEY_PATH}.pub" ]]; then
        log_warning "SSH public key not found: ${SSH_KEY_PATH}.pub"
        return 1
    fi

    log_info "Uploading SSH key to GitHub..."

    local hostname
    hostname=$(hostname)
    local key_title
    key_title="cloud-init-${hostname}-$(date +%Y%m%d)"

    if is_dry_run; then
        echo "[DRY-RUN] Would upload SSH key: ${key_title}"
        return 0
    fi

    # Check if key already exists
    if gh ssh-key list 2>/dev/null | grep -q "$(cat "${SSH_KEY_PATH}.pub" | cut -d' ' -f2)"; then
        log_success "SSH key already uploaded to GitHub"
        return 0
    fi

    gh ssh-key add "${SSH_KEY_PATH}.pub" --title "${key_title}"

    log_success "SSH key uploaded to GitHub"
}

setup_ssh_agent() {
    log_info "Configuring SSH agent..."

    if is_dry_run; then
        echo "[DRY-RUN] Would configure SSH agent"
        return 0
    fi

    # Start ssh-agent if not running
    if [[ -z "${SSH_AUTH_SOCK:-}" ]]; then
        eval "$(ssh-agent -s)" >/dev/null
    fi

    # Add key to agent
    ssh-add "${SSH_KEY_PATH}" 2>/dev/null || true

    log_success "SSH agent configured"
}

verify() {
    log_section "Verifying GitHub Setup"

    # Check gh installed
    if is_gh_installed; then
        health_pass "gh-cli" "installed"
    else
        health_fail "gh-cli" "not installed"
        return 1
    fi

    # Check authentication
    if is_authenticated; then
        local user
        user=$(gh api user --jq '.login' 2>/dev/null || echo "unknown")
        health_pass "gh-auth" "authenticated as ${user}"
    else
        health_fail "gh-auth" "not authenticated"
    fi

    # Check SSH key
    if [[ -f "${SSH_KEY_PATH}" ]]; then
        health_pass "ssh-key" "exists"
    else
        health_warn "ssh-key" "not generated"
    fi

    print_health_summary
}

#==============================================================================
# Main
#==============================================================================

main() {
    parse_dry_run_flag "$@"

    log_section "GitHub Setup"

    # Check if gh is installed
    if ! is_gh_installed; then
        log_error "GitHub CLI (gh) is not installed"
        log_info "Run: bash scripts/packages/github-cli.sh install"
        exit 1
    fi

    # Authenticate
    if is_authenticated; then
        local user
        user=$(gh api user --jq '.login' 2>/dev/null || echo "unknown")
        log_success "Already authenticated as: ${user}"
    else
        if [[ -n "${GITHUB_PAT:-}" ]]; then
            authenticate_with_pat
        else
            log_info "For automatic authentication, set GITHUB_PAT environment variable"
            log_info "Skipping authentication (run 'gh auth login' manually)"
        fi
    fi

    # Generate and upload SSH key
    if is_authenticated; then
        generate_ssh_key
        upload_ssh_key
        setup_ssh_agent
    fi

    # Verify
    verify

    if is_dry_run; then
        print_dry_run_summary
    fi

    # Show next steps
    if is_authenticated; then
        echo ""
        log_section "Next Steps"
        echo "Test SSH connection: ssh -T git@github.com"
        echo "Clone a repo: git clone git@github.com:user/repo.git"
    fi
}

[[ "${BASH_SOURCE[0]}" == "${0}" ]] && main "$@"
