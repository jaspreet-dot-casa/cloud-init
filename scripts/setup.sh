#!/bin/bash
set -e  # Exit on error
set -u  # Exit on undefined variable

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_success() { echo -e "${GREEN}✓${NC} $1"; }
echo_error() { echo -e "${RED}✗${NC} $1"; }
echo_info() { echo -e "${YELLOW}➜${NC} $1"; }

echo "════════════════════════════════════════════"
echo "  Remote Ubuntu Server Setup with Cloud-Init"
echo "════════════════════════════════════════════"
echo ""

# 1. Check prerequisites
echo_info "Checking prerequisites..."

# Check if running on Ubuntu
if [ ! -f /etc/os-release ]; then
    echo_error "Cannot detect OS. This script is for Ubuntu."
    exit 1
fi

# shellcheck disable=SC1091 source=/etc/os-release
. /etc/os-release
# shellcheck disable=SC2154
if [ "${ID}" != "ubuntu" ]; then
    # shellcheck disable=SC2154
    echo_error "This script is designed for Ubuntu. Detected: ${ID}"
    exit 1
fi
# shellcheck disable=SC2154
echo_success "Running on Ubuntu ${VERSION_ID}"

# Check sudo access
if ! sudo -v; then
    echo_error "This script requires sudo access"
    exit 1
fi
echo_success "Sudo access confirmed"

# Check curl is installed
if ! command -v curl &> /dev/null; then
    echo_info "Installing curl..."
    sudo apt-get update
    sudo apt-get install -y curl
fi
echo_success "curl is available"

# 2. Install Docker via official repository
echo ""
echo_info "Checking Docker installation..."
if command -v docker &> /dev/null; then
    echo_success "Docker already installed, skipping..."
else
    echo_info "Installing Docker (official repository)..."
    
    # Remove old versions
    sudo apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true
    
    # Setup repository
    sudo apt-get update
    sudo apt-get install -y ca-certificates curl gnupg
    sudo install -m 0755 -d /etc/apt/keyrings
    
    # Add Docker's official GPG key
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    sudo chmod a+r /etc/apt/keyrings/docker.gpg
    
    # Add repository
    # shellcheck disable=SC1091,SC2154
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
      $(. /etc/os-release && echo "${VERSION_CODENAME}") stable" | \
      sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    # Install Docker Engine
    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    echo_success "Docker installed successfully"
fi

# 5. Add user to docker group
echo ""
echo_info "Configuring Docker permissions..."
CURRENT_USER="${USER:-$(whoami)}"
if groups "${CURRENT_USER}" | grep -q docker; then
    echo_success "User already in docker group"
else
    sudo usermod -aG docker "${CURRENT_USER}"
    echo_success "User added to docker group"
fi

# 3. Configure Git
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo ""
echo_info "Configuring Git..."
bash "${SCRIPT_DIR}/shared/configure-git.sh"

# Done!
echo ""
echo "════════════════════════════════════════════"
echo_success "Setup completed successfully!"
echo "════════════════════════════════════════════"
echo ""
echo "Next steps:"
echo "  1. Run: bash scripts/cloud-init/install-all.sh"
echo "  2. Log out and back in (for docker group + shell changes)"
echo "  3. Run: make verify-cloud"
echo ""
