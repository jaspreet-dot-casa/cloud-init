#!/bin/bash
set -e
set -u
set -o pipefail

#==============================================================================
# Git Configuration Script
#
# Configures Git based on config.env settings.
# Used by cloud-init setup for Git configuration.
#
# Location: scripts/shared/configure-git.sh
#==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source shared libraries
source "${SCRIPT_DIR}/../lib/core.sh"
source "${SCRIPT_DIR}/../lib/dryrun.sh"

#==============================================================================
# Load Configuration
#==============================================================================

load_git_config() {
    # Load from config.env if available
    if [[ -f "${PROJECT_ROOT}/config.env" ]]; then
        log_debug "Loading configuration from config.env"
        # shellcheck source=/dev/null
        source "${PROJECT_ROOT}/config.env"
    else
        log_warning "config.env not found, using defaults"
    fi

    # Set defaults if not configured
    USER_NAME="${USER_NAME:-$(whoami)}"
    USER_EMAIL="${USER_EMAIL:-$(whoami)@$(hostname)}"
    GIT_DEFAULT_BRANCH="${GIT_DEFAULT_BRANCH:-main}"
    GIT_PUSH_AUTO_SETUP_REMOTE="${GIT_PUSH_AUTO_SETUP_REMOTE:-true}"
    GIT_PULL_REBASE="${GIT_PULL_REBASE:-true}"
    GIT_PAGER="${GIT_PAGER:-less}"
    GIT_URL_REWRITE_GITHUB="${GIT_URL_REWRITE_GITHUB:-false}"
}

#==============================================================================
# Git Configuration
#==============================================================================

configure_git() {
    log_section "Configuring Git"

    load_git_config

    log_info "Setting user configuration..."
    git_config_or_print --global user.name "${USER_NAME}"
    git_config_or_print --global user.email "${USER_EMAIL}"
    log_success "User: ${USER_NAME} <${USER_EMAIL}>"

    log_info "Setting branch defaults..."
    git_config_or_print --global init.defaultBranch "${GIT_DEFAULT_BRANCH}"
    log_success "Default branch: ${GIT_DEFAULT_BRANCH}"

    log_info "Setting push configuration..."
    if [[ "${GIT_PUSH_AUTO_SETUP_REMOTE}" == "true" ]]; then
        git_config_or_print --global push.autoSetupRemote true
        log_success "Auto setup remote: enabled"
    fi

    log_info "Setting pull configuration..."
    if [[ "${GIT_PULL_REBASE}" == "true" ]]; then
        git_config_or_print --global pull.rebase true
        log_success "Pull rebase: enabled"
    fi

    # Configure pager (delta if available)
    if [[ "${GIT_PAGER}" == "delta" ]]; then
        if command_exists delta; then
            log_info "Configuring delta as git pager..."
            git_config_or_print --global core.pager delta
            git_config_or_print --global interactive.diffFilter "delta --color-only"
            git_config_or_print --global delta.navigate true
            git_config_or_print --global delta.light false
            git_config_or_print --global delta.line-numbers true
            git_config_or_print --global merge.conflictstyle diff3
            git_config_or_print --global diff.colorMoved default
            log_success "Delta pager: configured"
        else
            log_warning "Delta not installed, using default pager"
        fi
    fi

    # GitHub URL rewrite (SSH instead of HTTPS)
    if [[ "${GIT_URL_REWRITE_GITHUB}" == "true" ]]; then
        log_info "Configuring GitHub URL rewrite (SSH)..."
        git_config_or_print --global url."git@github.com:".insteadOf "https://github.com/"
        log_success "GitHub URL rewrite: enabled"
    fi

    log_success "Git configuration complete"
}

#==============================================================================
# Verification
#==============================================================================

verify_git_config() {
    log_section "Verifying Git Configuration"

    local errors=0

    # Check user name
    local name
    name=$(git config --global user.name 2>/dev/null || echo "")
    if [[ -n "$name" ]]; then
        log_success "user.name: $name"
    else
        log_error "user.name not set"
        ((errors++)) || true
    fi

    # Check email
    local email
    email=$(git config --global user.email 2>/dev/null || echo "")
    if [[ -n "$email" ]]; then
        log_success "user.email: $email"
    else
        log_error "user.email not set"
        ((errors++)) || true
    fi

    # Check default branch
    local branch
    branch=$(git config --global init.defaultBranch 2>/dev/null || echo "")
    if [[ -n "$branch" ]]; then
        log_success "init.defaultBranch: $branch"
    else
        log_warning "init.defaultBranch not set (will use git default)"
    fi

    return $errors
}

#==============================================================================
# Main
#==============================================================================

main() {
    parse_dry_run_flag "$@"

    configure_git

    if is_dry_run; then
        print_dry_run_summary
    else
        verify_git_config
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
