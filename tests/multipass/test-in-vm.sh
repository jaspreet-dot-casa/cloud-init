#!/bin/bash
#==============================================================================
# In-VM Verification Script for Multipass Testing
#
# This script runs INSIDE the Multipass VM after cloud-init completes.
# It performs comprehensive verification of the installation.
#
# Output: JSON results written to /tmp/test-results.json
#
# Usage: This script is embedded in cloud-init and runs automatically.
#        For manual testing: sudo -u testuser /opt/local-remote/test-in-vm.sh
#==============================================================================

set -u
set -o pipefail

# Configuration
RESULTS_FILE="/tmp/test-results.json"
MARKER_FILE="/tmp/cloud-init-test-complete"

# Colors (for terminal output)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test tracking
declare -a TESTS
PASSED=0
FAILED=0
SKIPPED=0

#==============================================================================
# Helper Functions
#==============================================================================

log_info()    { echo -e "${BLUE}→ $1${NC}"; }
log_success() { echo -e "${GREEN}✓ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
log_error()   { echo -e "${RED}✗ $1${NC}"; }

# Record a test result
# Usage: record_test "name" "pass|fail|skip" "message"
record_test() {
    local name="$1"
    local status="$2"
    local message="${3:-}"

    # Escape quotes in message for JSON
    message="${message//\"/\\\"}"
    message="${message//$'\n'/\\n}"

    TESTS+=("{\"name\":\"$name\",\"status\":\"$status\",\"message\":\"$message\"}")

    case "$status" in
        pass)
            ((PASSED++))
            log_success "$name: $message"
            ;;
        fail)
            ((FAILED++))
            log_error "$name: $message"
            ;;
        skip)
            ((SKIPPED++))
            log_warning "$name: skipped - $message"
            ;;
    esac
}

#==============================================================================
# Test Functions
#==============================================================================

# Test that essential packages are installed
test_packages() {
    log_info "Testing package installations..."

    # Essential packages that MUST be present
    local essential_packages=(
        git
        gh
        docker
        zsh
        curl
        wget
        jq
    )

    # Optional packages (nice to have)
    local optional_packages=(
        lazygit
        lazydocker
        nvim
        tmux
        zellij
        fzf
        zoxide
        rg
        fd
        bat
        delta
        starship
        btop
        yq
        tailscale
    )

    # Test essential packages
    for pkg in "${essential_packages[@]}"; do
        if command -v "$pkg" &>/dev/null; then
            local version
            version=$("$pkg" --version 2>&1 | head -1 | cut -c1-50)
            record_test "package:$pkg" "pass" "$version"
        else
            record_test "package:$pkg" "fail" "not found in PATH"
        fi
    done

    # Test optional packages
    for pkg in "${optional_packages[@]}"; do
        if command -v "$pkg" &>/dev/null; then
            local version
            version=$("$pkg" --version 2>&1 | head -1 | cut -c1-50)
            record_test "package:$pkg" "pass" "$version"
        else
            record_test "package:$pkg" "skip" "not installed"
        fi
    done
}

# Test Git configuration
test_git_config() {
    log_info "Testing Git configuration..."

    local required_configs=(
        "user.name"
        "user.email"
        "init.defaultBranch"
    )

    local optional_configs=(
        "core.pager"
        "push.autoSetupRemote"
        "pull.rebase"
    )

    for cfg in "${required_configs[@]}"; do
        local value
        value=$(git config --global "$cfg" 2>/dev/null)
        if [[ -n "$value" ]]; then
            record_test "git:$cfg" "pass" "$value"
        else
            record_test "git:$cfg" "fail" "not set"
        fi
    done

    for cfg in "${optional_configs[@]}"; do
        local value
        value=$(git config --global "$cfg" 2>/dev/null)
        if [[ -n "$value" ]]; then
            record_test "git:$cfg" "pass" "$value"
        else
            record_test "git:$cfg" "skip" "not configured"
        fi
    done
}

# Test shell configuration
test_shell_config() {
    log_info "Testing shell configuration..."

    # Check default shell is zsh
    if [[ "$SHELL" == *"zsh"* ]]; then
        record_test "shell:default" "pass" "zsh"
    else
        record_test "shell:default" "fail" "expected zsh, got $SHELL"
    fi

    # Check Oh-My-Zsh installation
    if [[ -d "$HOME/.oh-my-zsh" ]]; then
        record_test "shell:oh-my-zsh" "pass" "installed"
    else
        record_test "shell:oh-my-zsh" "skip" "not installed"
    fi

    # Check .zshrc exists
    if [[ -f "$HOME/.zshrc" ]]; then
        record_test "shell:zshrc" "pass" "exists"
    else
        record_test "shell:zshrc" "fail" "missing"
    fi

    # Check starship config
    if [[ -f "$HOME/.config/starship.toml" ]]; then
        record_test "shell:starship-config" "pass" "configured"
    elif command -v starship &>/dev/null; then
        record_test "shell:starship-config" "skip" "starship installed but not configured"
    else
        record_test "shell:starship-config" "skip" "starship not installed"
    fi
}

# Test system services
test_services() {
    log_info "Testing services..."

    # Docker daemon
    if systemctl is-active docker &>/dev/null; then
        record_test "service:docker" "pass" "running"
    else
        record_test "service:docker" "fail" "not running"
    fi

    # Check if current user is in docker group
    if groups 2>/dev/null | grep -q docker; then
        record_test "service:docker-group" "pass" "user in docker group"
    else
        record_test "service:docker-group" "fail" "user not in docker group"
    fi

    # Docker socket accessible
    if docker info &>/dev/null; then
        record_test "service:docker-socket" "pass" "accessible"
    else
        record_test "service:docker-socket" "skip" "may require newgrp docker"
    fi

    # Tailscale daemon
    if systemctl is-active tailscaled &>/dev/null; then
        record_test "service:tailscaled" "pass" "running"
    else
        record_test "service:tailscaled" "skip" "not running (may not be configured)"
    fi
}

# Test directory structure
test_directories() {
    log_info "Testing directory structure..."

    local required_dirs=(
        "$HOME/.config"
    )

    local optional_dirs=(
        "$HOME/.local-remote"
        "$HOME/.local/bin"
        "$HOME/.config/shell"
    )

    for dir in "${required_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            record_test "dir:$dir" "pass" "exists"
        else
            record_test "dir:$dir" "fail" "missing"
        fi
    done

    for dir in "${optional_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            record_test "dir:$dir" "pass" "exists"
        else
            record_test "dir:$dir" "skip" "not created"
        fi
    done
}

# Test user setup
test_user() {
    log_info "Testing user setup..."

    # Check sudo access
    if sudo -n true 2>/dev/null; then
        record_test "user:sudo" "pass" "passwordless sudo works"
    else
        record_test "user:sudo" "skip" "passwordless sudo not configured"
    fi

    # Check SSH authorized_keys
    if [[ -f "$HOME/.ssh/authorized_keys" ]]; then
        local key_count
        key_count=$(wc -l < "$HOME/.ssh/authorized_keys")
        record_test "user:ssh-keys" "pass" "$key_count key(s) configured"
    else
        record_test "user:ssh-keys" "fail" "no authorized_keys file"
    fi
}

#==============================================================================
# Results Output
#==============================================================================

write_results() {
    log_info "Writing test results to $RESULTS_FILE..."

    # Build JSON array of tests
    local tests_json=""
    local first=true
    for test in "${TESTS[@]}"; do
        if [[ "$first" == "true" ]]; then
            tests_json="$test"
            first=false
        else
            tests_json="$tests_json,$test"
        fi
    done

    # Get system info
    local os_info
    os_info=$(lsb_release -ds 2>/dev/null || cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)
    local kernel
    kernel=$(uname -r)
    local arch
    arch=$(uname -m)

    # Write JSON
    cat > "$RESULTS_FILE" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "hostname": "$(hostname)",
  "summary": {
    "total": $((PASSED + FAILED + SKIPPED)),
    "passed": $PASSED,
    "failed": $FAILED,
    "skipped": $SKIPPED
  },
  "system_info": {
    "os": "$os_info",
    "kernel": "$kernel",
    "arch": "$arch"
  },
  "tests": [$tests_json]
}
EOF

    log_success "Results written to $RESULTS_FILE"
}

print_summary() {
    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}Test Summary${NC}"
    echo "════════════════════════════════════════════"
    echo ""
    echo -e "  Total:   $((PASSED + FAILED + SKIPPED))"
    echo -e "  ${GREEN}Passed:  $PASSED${NC}"
    echo -e "  ${RED}Failed:  $FAILED${NC}"
    echo -e "  ${YELLOW}Skipped: $SKIPPED${NC}"
    echo ""

    if [[ $FAILED -gt 0 ]]; then
        log_error "Some tests failed!"
    else
        log_success "All required tests passed!"
    fi
}

#==============================================================================
# Main
#==============================================================================

main() {
    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}Cloud-Init In-VM Verification Tests${NC}"
    echo "════════════════════════════════════════════"
    echo ""
    echo "Running as: $(whoami)@$(hostname)"
    echo "Date: $(date)"
    echo ""

    # Run all test suites
    test_packages
    test_git_config
    test_shell_config
    test_services
    test_directories
    test_user

    # Write results
    write_results
    print_summary

    # Create completion marker
    touch "$MARKER_FILE"

    # Exit with failure if any required tests failed
    [[ $FAILED -eq 0 ]]
}

main "$@"
