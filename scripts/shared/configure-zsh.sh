#!/bin/bash
set -e
set -u
set -o pipefail

#==============================================================================
# Zsh Configuration Script
#
# Configures Zsh and Oh-My-Zsh based on config.env settings.
# Used by cloud-init setup for Zsh configuration.
#
# Location: scripts/shared/configure-zsh.sh
#==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source shared libraries
source "${SCRIPT_DIR}/../lib/core.sh"
source "${SCRIPT_DIR}/../lib/dryrun.sh"

#==============================================================================
# Load Configuration
#==============================================================================

load_zsh_config() {
    # Load from config.env if available
    if [[ -f "${PROJECT_ROOT}/config.env" ]]; then
        log_debug "Loading configuration from config.env"
        # shellcheck source=/dev/null
        source "${PROJECT_ROOT}/config.env"
    fi

    # Set defaults
    ZSH_THEME="${ZSH_THEME:-robbyrussell}"
    ZSH_PLUGINS="${ZSH_PLUGINS:-git}"
}

#==============================================================================
# Oh-My-Zsh Installation
#==============================================================================

install_oh_my_zsh() {
    local omz_dir="${HOME}/.oh-my-zsh"

    if [[ -d "$omz_dir" ]]; then
        log_success "Oh-My-Zsh already installed"
        return 0
    fi

    log_info "Installing Oh-My-Zsh..."

    if is_dry_run; then
        echo "[DRY-RUN] Would install Oh-My-Zsh to $omz_dir"
        return 0
    fi

    # Install without running zsh or changing shell
    sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended

    log_success "Oh-My-Zsh installed"
}

#==============================================================================
# Zsh Configuration
#==============================================================================

configure_zshrc() {
    log_section "Configuring Zsh"

    load_zsh_config

    local zshrc="${HOME}/.zshrc"

    # Check if .zshrc exists
    if [[ ! -f "$zshrc" ]]; then
        log_warning ".zshrc not found, Oh-My-Zsh may need to be installed first"
        return 1
    fi

    # Update theme
    log_info "Setting theme: ${ZSH_THEME}"
    if ! is_dry_run; then
        sed -i.bak "s/^ZSH_THEME=.*/ZSH_THEME=\"${ZSH_THEME}\"/" "$zshrc"
        rm -f "${zshrc}.bak"
    else
        echo "[DRY-RUN] Would set ZSH_THEME=\"${ZSH_THEME}\" in $zshrc"
    fi

    # Update plugins
    log_info "Setting plugins: ${ZSH_PLUGINS}"
    local plugins_line="plugins=(${ZSH_PLUGINS})"
    if ! is_dry_run; then
        sed -i.bak "s/^plugins=.*/plugins=(${ZSH_PLUGINS})/" "$zshrc"
        rm -f "${zshrc}.bak"
    else
        echo "[DRY-RUN] Would set plugins=(${ZSH_PLUGINS}) in $zshrc"
    fi

    # Add custom config sourcing if not present
    local custom_config_line="# Source custom shell config"
    local source_line='[[ -f ~/.zsh_custom_config ]] && source ~/.zsh_custom_config'

    if ! grep -q "zsh_custom_config" "$zshrc" 2>/dev/null; then
        log_info "Adding custom config sourcing to .zshrc"
        if ! is_dry_run; then
            {
                echo ""
                echo "$custom_config_line"
                echo "$source_line"
            } >> "$zshrc"
        else
            echo "[DRY-RUN] Would append custom config sourcing to $zshrc"
        fi
    else
        log_debug "Custom config sourcing already present"
    fi

    log_success "Zsh configuration updated"
}

#==============================================================================
# Verification
#==============================================================================

verify_zsh_config() {
    log_section "Verifying Zsh Configuration"

    local errors=0

    # Check Oh-My-Zsh
    if [[ -d "${HOME}/.oh-my-zsh" ]]; then
        log_success "Oh-My-Zsh: installed"
    else
        log_error "Oh-My-Zsh: not installed"
        ((errors++))
    fi

    # Check .zshrc
    if [[ -f "${HOME}/.zshrc" ]]; then
        log_success ".zshrc: exists"

        # Check theme
        local theme
        theme=$(grep "^ZSH_THEME=" "${HOME}/.zshrc" 2>/dev/null | cut -d'"' -f2 || echo "")
        if [[ -n "$theme" ]]; then
            log_success "Theme: $theme"
        else
            log_warning "Theme not set"
        fi

        # Check plugins
        local plugins
        plugins=$(grep "^plugins=" "${HOME}/.zshrc" 2>/dev/null || echo "")
        if [[ -n "$plugins" ]]; then
            log_success "Plugins: configured"
        else
            log_warning "Plugins not configured"
        fi

        # Check custom config sourcing
        if grep -q "zsh_custom_config" "${HOME}/.zshrc" 2>/dev/null; then
            log_success "Custom config sourcing: enabled"
        else
            log_warning "Custom config sourcing: not enabled"
        fi
    else
        log_error ".zshrc: not found"
        ((errors++))
    fi

    return $errors
}

#==============================================================================
# Main
#==============================================================================

main() {
    parse_dry_run_flag "$@"

    local install_omz=false

    # Parse additional arguments
    for arg in "$@"; do
        case "$arg" in
            --install-omz)
                install_omz=true
                ;;
        esac
    done

    if [[ "$install_omz" == "true" ]]; then
        install_oh_my_zsh
    fi

    configure_zshrc

    if is_dry_run; then
        print_dry_run_summary
    else
        verify_zsh_config
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
