#!/bin/bash
set -e
set -u

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo_success() { echo -e "${GREEN}✓${NC} $1"; }
echo_info() { echo -e "${YELLOW}➜${NC} $1"; }
echo_section() {
    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}$1${NC}"
    echo "════════════════════════════════════════════"
}

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Pass through arguments to sub-scripts
ARGS="${*}"

# Find zsh (system or installed)
ZSH_PATH="$(which zsh 2>/dev/null || echo "/usr/bin/zsh")"

if [ ! -f "${ZSH_PATH}" ]; then
    echo "Error: zsh not found"
    echo "Please install zsh first (included in cloud-init setup)"
    exit 1
fi

# Add zsh to valid shells if not present
if ! grep -q "${ZSH_PATH}" /etc/shells; then
    echo_info "Adding ${ZSH_PATH} to /etc/shells..."
    echo "${ZSH_PATH}" | sudo tee -a /etc/shells
fi

# Change default shell
if [ "${SHELL}" != "${ZSH_PATH}" ]; then
    echo_info "Changing default shell to zsh..."
    chsh -s "${ZSH_PATH}"
    echo_success "Default shell changed to zsh"
else
    echo_success "Default shell is already zsh"
fi

# Tailscale Setup
echo_section "Setting up Tailscale..."
TAILSCALE_SCRIPT="${PROJECT_ROOT}/scripts/shared/tailscale.sh"
if [ -f "${TAILSCALE_SCRIPT}" ]; then
    # shellcheck source=/dev/null
    # shellcheck disable=SC2086
    bash "${TAILSCALE_SCRIPT}" ${ARGS}
else
    echo_info "Tailscale script not found - skipping"
fi

# Final summary
echo ""
echo "════════════════════════════════════════════"
echo_success "Post-install configuration complete!"
echo "════════════════════════════════════════════"
echo ""
echo "Please log out and back in for changes to take effect:"
echo "  • Docker group membership"
echo "  • Zsh as default shell"
echo "  • All environment variables"
echo ""
