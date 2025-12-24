#!/bin/bash
set -e
set -u
set -o pipefail

#==============================================================================
# Tailscale Setup Script
#
# This script sets up Tailscale daemon and configures it based on config file.
# Features:
#   - Binary detection (system installation)
#   - Daemon installation via systemd
#   - Idempotent (safe to run multiple times)
#   - Automatic authentication flow
#   - Config file-based setup
#   - Docker-aware (skips daemon setup in containers)
#
# Location: scripts/shared/tailscale.sh
# Used by: Cloud-init post-installation setup
#==============================================================================

#==============================================================================
# Script Location and Project Root
#==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Calculate project root from scripts/shared/
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Config file location
CONFIG_FILE="${PROJECT_ROOT}/config/tailscale.conf"

#==============================================================================
# Source shared libraries if available
#==============================================================================

if [[ -f "${SCRIPT_DIR}/../lib/core.sh" ]]; then
    # shellcheck source=scripts/lib/core.sh
    source "${SCRIPT_DIR}/../lib/core.sh"

    # Use library functions
    echo_error() { log_error "$1"; }
    echo_success() { log_success "$1"; }
    echo_info() { log_info "$1"; }
    echo_warning() { log_warning "$1"; }
    echo_section() { log_section "$1"; }
else
    # Fallback to inline color definitions
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color

    echo_error() {
        echo -e "${RED}✗ $1${NC}" >&2
    }

    echo_success() {
        echo -e "${GREEN}✓ $1${NC}"
    }

    echo_info() {
        echo -e "${BLUE}→ $1${NC}"
    }

    echo_warning() {
        echo -e "${YELLOW}⚠ $1${NC}"
    }

    echo_section() {
        echo ""
        echo "════════════════════════════════════════════"
        echo -e "${BLUE}$1${NC}"
        echo "════════════════════════════════════════════"
    }
fi

#==============================================================================
# Global Variables
#==============================================================================

SKIP_CONFIRMATION=false
TAILSCALE_BIN=""
TAILSCALED_BIN=""

#==============================================================================
# Argument Parsing
#==============================================================================

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -y|--yes)
                SKIP_CONFIRMATION=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                echo_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

show_help() {
    cat << EOF
Tailscale Setup Script

Usage: $0 [OPTIONS]

Options:
    -y, --yes       Skip confirmation prompts
    -h, --help      Show this help message

Description:
    Sets up Tailscale daemon and configures it based on config file.

    This script will:
    1. Detect Tailscale binaries (system installation)
    2. Install and start tailscaled daemon (systemd)
    3. Configure Tailscale based on config/tailscale.conf
    4. Authenticate with Tailscale (interactive or automatic)

    The script is idempotent and safe to run multiple times.

Config File:
    ${CONFIG_FILE}

EOF
}

#==============================================================================
# Docker Detection
#==============================================================================

is_docker() {
    [ -f /.dockerenv ]
}

#==============================================================================
# Binary Detection
#==============================================================================

detect_binaries() {
    echo_section "Detecting Tailscale Binaries"

    # Check for system tailscale
    if command -v tailscale &> /dev/null; then
        TAILSCALE_BIN="$(command -v tailscale)"
        TAILSCALED_BIN="$(command -v tailscaled 2>/dev/null || echo "")"
        echo_success "Found Tailscale"
        echo_info "Location: ${TAILSCALE_BIN}"
    else
        echo_error "Tailscale not found"
        echo_error "Install via package manager or run cloud-init setup"
        exit 1
    fi

    # Verify tailscaled exists
    if [ -z "${TAILSCALED_BIN}" ] || [ ! -f "${TAILSCALED_BIN}" ]; then
        echo_error "tailscaled daemon binary not found"
        exit 1
    fi

    # Show versions
    local version
    version=$("${TAILSCALE_BIN}" version 2>/dev/null | head -n1 || echo "unknown")
    echo_info "Tailscale version: ${version}"
}

#==============================================================================
# Config File Loading
#==============================================================================

load_config() {
    echo_section "Loading Configuration"

    if [ ! -f "${CONFIG_FILE}" ]; then
        echo_error "Config file not found: ${CONFIG_FILE}"
        exit 1
    fi

    echo_info "Loading: ${CONFIG_FILE}"

    # Source config file
    # shellcheck source=/dev/null
    source "${CONFIG_FILE}"

    # Validate required variables
    if [ -z "${TAILSCALE_SSH_ENABLED:-}" ]; then
        echo_error "TAILSCALE_SSH_ENABLED not set in config"
        exit 1
    fi

    if [ -z "${TAILSCALE_ADVERTISE_EXIT_NODE:-}" ]; then
        echo_error "TAILSCALE_ADVERTISE_EXIT_NODE not set in config"
        exit 1
    fi

    # Show loaded config
    echo_success "Configuration loaded:"
    echo "  SSH Enabled: ${TAILSCALE_SSH_ENABLED}"
    echo "  Exit Node: ${TAILSCALE_ADVERTISE_EXIT_NODE}"
    echo "  SSH Check Mode: ${TAILSCALE_SSH_CHECK_MODE:-false}"
    echo "  Check Period: ${TAILSCALE_SSH_CHECK_PERIOD:-12h}"
    echo "  Additional Flags: ${TAILSCALE_ADDITIONAL_FLAGS:-none}"
}

#==============================================================================
# Daemon Management
#==============================================================================

install_daemon() {
    echo_section "Setting Up Tailscale Daemon"

    # Check if daemon is already running
    if systemctl is-active --quiet tailscaled 2>/dev/null; then
        echo_success "Daemon already running"
        return 0
    fi

    # Check if service exists
    if systemctl list-unit-files tailscaled.service &> /dev/null 2>&1; then
        echo_info "Service exists, starting..."
        sudo systemctl start tailscaled
        echo_success "Daemon started"
    else
        echo_info "Installing systemd service..."

        # Install systemd service
        if ! sudo "${TAILSCALED_BIN}" install-systemd 2>&1; then
            echo_error "Failed to install systemd service"
            exit 1
        fi

        # Enable and start service
        sudo systemctl enable --now tailscaled

        # Wait for daemon to be ready
        echo_info "Waiting for daemon to initialize..."
        sleep 2

        # Verify it's running
        if systemctl is-active --quiet tailscaled; then
            echo_success "Daemon installed and running"
        else
            echo_error "Daemon failed to start"
            echo_info "Check logs: sudo journalctl -u tailscaled -n 50"
            exit 1
        fi
    fi
}

#==============================================================================
# Tailscale Command Builder
#==============================================================================

build_tailscale_command() {
    local -a cmd=("${TAILSCALE_BIN}" "up")

    # Add SSH flag
    if [ "${TAILSCALE_SSH_ENABLED}" = "true" ]; then
        cmd+=("--ssh")
    fi

    # Add exit node advertising
    if [ "${TAILSCALE_ADVERTISE_EXIT_NODE}" = "true" ]; then
        cmd+=("--advertise-exit-node")
    fi

    # Add additional flags (word splitting intentional here)
    if [ -n "${TAILSCALE_ADDITIONAL_FLAGS:-}" ]; then
        # shellcheck disable=SC2206
        cmd+=(${TAILSCALE_ADDITIONAL_FLAGS})
    fi

    # Print array elements one per line for caller to parse
    printf '%s\n' "${cmd[@]}"
}

#==============================================================================
# Authentication Check
#==============================================================================

is_authenticated() {
    "${TAILSCALE_BIN}" status &> /dev/null 2>&1
}

#==============================================================================
# Confirmation Prompts
#==============================================================================

confirm_action() {
    local message="$1"

    if [ "${SKIP_CONFIRMATION}" = "true" ]; then
        return 0
    fi

    echo ""
    echo -e "${YELLOW:-}${message}${NC:-}"
    read -rp "Continue? [y/N]: " response

    case "${response}" in
        [yY][eE][sS]|[yY])
            return 0
            ;;
        *)
            echo_info "Cancelled by user"
            exit 0
            ;;
    esac
}

#==============================================================================
# Authentication Flow
#==============================================================================

authenticate_tailscale() {
    echo_section "Configuring Tailscale"

    # Build command
    local -a cmd_array
    mapfile -t cmd_array < <(build_tailscale_command)

    echo_info "Configuration command:"
    echo "  ${cmd_array[*]}"
    echo ""

    # Check if already authenticated
    if is_authenticated; then
        echo_success "Already authenticated with Tailscale"

        # Show current status
        echo ""
        echo_info "Current status:"
        "${TAILSCALE_BIN}" status | head -n 5 || true
        echo ""

        confirm_action "Re-apply configuration? This may trigger re-authentication."

        echo_info "Applying configuration..."
        if ! sudo "${cmd_array[@]}"; then
            echo_error "Failed to apply configuration"
            exit 1
        fi

        echo_success "Configuration applied"
    else
        echo_info "Authentication required"
        echo ""
        echo_info "This will:"
        echo "  1. Open your browser for authentication (or print a URL)"
        echo "  2. Connect to your Tailscale network"
        echo "  3. Apply the configuration from tailscale.conf"
        echo ""

        confirm_action "Authenticate with Tailscale now?"

        echo ""
        echo_info "Starting authentication..."
        echo_warning "If browser doesn't open, copy the URL that appears below"
        echo ""

        if ! sudo "${cmd_array[@]}"; then
            echo_error "Authentication failed"
            exit 1
        fi

        echo ""
        echo_success "Authentication complete"
    fi
}

#==============================================================================
# Verification
#==============================================================================

verify_setup() {
    echo_section "Verifying Setup"

    # Check daemon status
    if systemctl is-active --quiet tailscaled; then
        echo_success "Daemon is running"
    else
        echo_error "Daemon is not running"
        return 1
    fi

    # Check authentication
    if is_authenticated; then
        echo_success "Authenticated with Tailscale"
    else
        echo_warning "Not authenticated (may need to complete auth flow)"
        return 1
    fi

    # Show status
    echo ""
    echo_info "Tailscale Status:"
    echo "----------------------------------------"
    "${TAILSCALE_BIN}" status || true
    echo "----------------------------------------"
}

#==============================================================================
# Next Steps
#==============================================================================

show_next_steps() {
    echo_section "Next Steps"

    cat << EOF

${GREEN:-}✓ Tailscale setup complete!${NC:-}

${BLUE:-}Important Next Steps:${NC:-}

${YELLOW:-}1. Configure Tailscale ACLs for SSH${NC:-}
   Tailscale SSH requires ACL configuration in the admin console.

   Go to: https://login.tailscale.com/admin/acls

   Add SSH rules (example):

   {
     "ssh": [
       {
         "action": "check",
         "src": ["autogroup:member"],
         "dst": ["autogroup:self"],
         "users": ["autogroup:nonroot", "root"],
         "checkPeriod": "${TAILSCALE_SSH_CHECK_PERIOD:-12h}"
       }
     ]
   }

${YELLOW:-}2. Enable Exit Node (if desired)${NC:-}
   Your server is advertising as an exit node, but needs approval.

   Go to: https://login.tailscale.com/admin/machines
   → Click on this machine
   → Edit route settings
   → Enable "Use as exit node"

${YELLOW:-}3. Test SSH Connection${NC:-}
   From another device on your Tailscale network:

   ssh ${USER}@\$(tailscale ip -4)

   Or use the machine name from the admin console.

${BLUE:-}Useful Commands:${NC:-}
   tailscale status    - Show connection status
   tailscale ip        - Show Tailscale IPs
   tailscale ping HOST - Test connectivity
   tailscale ssh HOST  - SSH to another Tailscale machine

${BLUE:-}Troubleshooting:${NC:-}
   Daemon logs: sudo journalctl -u tailscaled -f
   Check status: systemctl status tailscaled
   Re-run setup: make post-install

EOF
}

#==============================================================================
# Main Function
#==============================================================================

main() {
    echo_section "Tailscale Setup"

    # Parse arguments
    parse_arguments "$@"

    # Skip daemon setup if in Docker
    if is_docker; then
        echo_info "Running in Docker - skipping daemon setup"
        echo_success "Tailscale CLI is available"
        echo_info "Daemon setup requires system networking (run on host)"
        exit 0
    fi

    # Execute setup steps
    detect_binaries
    load_config
    install_daemon
    authenticate_tailscale

    # Verify everything is working
    if verify_setup; then
        show_next_steps
        exit 0
    else
        echo_error "Setup verification failed"
        echo_info "Check logs: sudo journalctl -u tailscaled -n 50"
        exit 1
    fi
}

# Run main function
main "$@"
