#!/bin/bash
#==============================================================================
# Tailscale Authentication Setup
#
# Authenticates with Tailscale and enables SSH access.
#
# Usage: ./setup-tailscale.sh
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

setup_tailscale() {
    log_section "Tailscale Authentication"

    if ! command -v tailscale &>/dev/null; then
        log_warning "Tailscale is not installed. Skipping."
        return 0
    fi

    # Check if already authenticated
    local status
    status=$(tailscale status 2>&1 || true)

    if [[ "$status" != *"Logged out"* ]] && [[ "$status" != *"not logged in"* ]] && [[ "$status" != *"NeedsLogin"* ]]; then
        log_success "Tailscale already authenticated"
        echo ""
        echo "Current status:"
        tailscale status 2>/dev/null | head -5 || true
        return 0
    fi

    log_info "Starting Tailscale authentication..."
    echo ""
    echo "This will open a browser or display a URL to authenticate."
    echo ""

    # Start tailscale with SSH enabled
    sudo tailscale up --ssh --advertise-exit-node

    log_success "Tailscale authenticated"
    echo ""
    tailscale status 2>/dev/null | head -5 || true
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    setup_tailscale "$@"
fi
