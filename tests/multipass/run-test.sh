#!/bin/bash
#==============================================================================
# Multipass Integration Test Runner
#
# This script orchestrates cloud-init integration testing using Multipass VMs.
# It launches a VM, waits for cloud-init to complete, retrieves test results,
# and cleans up.
#
# Usage:
#   ./run-test.sh                    # Run test and cleanup
#   ./run-test.sh --keep             # Keep VM for debugging
#   ./run-test.sh --timeout 600      # Custom timeout (seconds)
#   ./run-test.sh --name my-vm       # Custom VM name
#
# Requirements:
#   - Multipass installed (brew install multipass)
#   - cloud-init/secrets.env or tests/fixtures/secrets-test.env
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

#==============================================================================
# Configuration
#==============================================================================

VM_NAME_PREFIX="cloud-init-test"
VM_NAME=""
DEFAULT_TIMEOUT=900  # 15 minutes
TIMEOUT=$DEFAULT_TIMEOUT
KEEP_VM=false
VERBOSE=false

# Cloud-init files
CLOUD_INIT_DIR="${PROJECT_ROOT}/cloud-init"
SECRETS_FILE="${CLOUD_INIT_DIR}/secrets.env"
TEST_SECRETS_FILE="${PROJECT_ROOT}/tests/fixtures/secrets-test.env"
GENERATED_YAML="${CLOUD_INIT_DIR}/cloud-init.yaml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m'

#==============================================================================
# Logging
#==============================================================================

log_info()    { echo -e "${BLUE}→ $1${NC}"; }
log_success() { echo -e "${GREEN}✓ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
log_error()   { echo -e "${RED}✗ $1${NC}" >&2; }
log_debug()   { [[ "$VERBOSE" == "true" ]] && echo -e "${MAGENTA}[DEBUG] $1${NC}" || true; }

log_section() {
    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}$1${NC}"
    echo "════════════════════════════════════════════"
}

#==============================================================================
# Helper Functions
#==============================================================================

check_multipass_installed() {
    if ! command -v multipass &>/dev/null; then
        log_error "Multipass is not installed."
        echo ""
        echo "Install with:"
        echo "  brew install multipass"
        echo ""
        exit 1
    fi
    log_success "Multipass installed: $(multipass version | head -1)"
}

generate_vm_name() {
    local timestamp
    timestamp=$(date +%Y%m%d-%H%M%S)
    VM_NAME="${VM_NAME_PREFIX}-${timestamp}"
    log_debug "Generated VM name: $VM_NAME"
}

setup_secrets() {
    # Use test secrets if real secrets don't exist
    if [[ ! -f "$SECRETS_FILE" ]]; then
        if [[ -f "$TEST_SECRETS_FILE" ]]; then
            log_info "Using test secrets from $TEST_SECRETS_FILE"
            cp "$TEST_SECRETS_FILE" "$SECRETS_FILE"
        else
            log_error "No secrets file found. Create $SECRETS_FILE or $TEST_SECRETS_FILE"
            exit 1
        fi
    else
        log_info "Using existing secrets from $SECRETS_FILE"
    fi
}

generate_cloud_init() {
    log_info "Generating cloud-init.yaml..."

    # Run the generator script
    if ! bash "${CLOUD_INIT_DIR}/generate.sh" >/dev/null 2>&1; then
        log_error "Failed to generate cloud-init.yaml"
        exit 1
    fi

    if [[ ! -f "$GENERATED_YAML" ]]; then
        log_error "cloud-init.yaml was not generated"
        exit 1
    fi

    log_success "Generated cloud-init.yaml"
}

launch_vm() {
    log_info "Launching Multipass VM: $VM_NAME"
    log_info "This may take several minutes..."

    # Launch with cloud-init
    # Using Ubuntu 24.04 LTS (noble) for better compatibility
    # Use --timeout to give cloud-init more time during launch (in seconds)
    if ! multipass launch \
        --name "$VM_NAME" \
        --cpus 2 \
        --memory 4G \
        --disk 20G \
        --timeout 900 \
        --cloud-init "$GENERATED_YAML" \
        24.04; then
        log_error "Failed to launch VM"
        # Don't cleanup here - let the trap handle it based on KEEP_VM flag
        return 1
    fi

    log_success "VM launched successfully"
}

wait_for_cloud_init() {
    log_info "Waiting for cloud-init to complete (timeout: ${TIMEOUT}s)..."

    local elapsed=0
    local poll_interval=10
    local status=""

    while [[ $elapsed -lt $TIMEOUT ]]; do
        # Check cloud-init status
        status=$(multipass exec "$VM_NAME" -- cloud-init status 2>/dev/null || echo "pending")

        if [[ "$status" == *"done"* ]]; then
            echo ""
            log_success "Cloud-init completed successfully"
            return 0
        elif [[ "$status" == *"error"* ]]; then
            echo ""
            log_error "Cloud-init encountered an error"
            log_info "Retrieving cloud-init logs..."
            multipass exec "$VM_NAME" -- sudo cat /var/log/cloud-init-output.log 2>/dev/null | tail -50 || true
            return 1
        fi

        # Show progress
        echo -n "."
        sleep $poll_interval
        elapsed=$((elapsed + poll_interval))
    done

    echo ""
    log_error "Timeout waiting for cloud-init (${TIMEOUT}s)"
    log_info "Retrieving partial cloud-init logs..."
    multipass exec "$VM_NAME" -- sudo cat /var/log/cloud-init-output.log 2>/dev/null | tail -50 || true
    return 2
}

retrieve_results() {
    log_info "Retrieving test results..."

    local results_file="/tmp/test-results.json"
    local local_results="${SCRIPT_DIR}/test-results.json"

    # Check if test results exist
    if ! multipass exec "$VM_NAME" -- test -f "$results_file" 2>/dev/null; then
        log_warning "Test results file not found. Running tests manually..."

        # Try to run the test script manually
        if multipass exec "$VM_NAME" -- test -f /opt/local-remote/test-in-vm.sh 2>/dev/null; then
            multipass exec "$VM_NAME" -- sudo -u testuser /opt/local-remote/test-in-vm.sh || true
        else
            log_error "Test script not found in VM"
            return 1
        fi
    fi

    # Retrieve results
    if multipass exec "$VM_NAME" -- cat "$results_file" 2>/dev/null > "$local_results"; then
        log_success "Results saved to $local_results"
        return 0
    else
        log_error "Failed to retrieve test results"
        return 1
    fi
}

display_results() {
    local results_file="${SCRIPT_DIR}/test-results.json"

    if [[ ! -f "$results_file" ]]; then
        log_error "No results file to display"
        return 1
    fi

    log_section "Test Results"

    # Parse and display results
    if command -v jq &>/dev/null; then
        local total passed failed skipped
        total=$(jq -r '.summary.total' "$results_file")
        passed=$(jq -r '.summary.passed' "$results_file")
        failed=$(jq -r '.summary.failed' "$results_file")
        skipped=$(jq -r '.summary.skipped' "$results_file")

        echo ""
        echo "  Total:   $total"
        echo -e "  ${GREEN}Passed:  $passed${NC}"
        echo -e "  ${RED}Failed:  $failed${NC}"
        echo -e "  ${YELLOW}Skipped: $skipped${NC}"
        echo ""

        # Show failed tests
        if [[ "$failed" -gt 0 ]]; then
            log_error "Failed tests:"
            jq -r '.tests[] | select(.status == "fail") | "  - \(.name): \(.message)"' "$results_file"
            echo ""
        fi

        # Return failure if any tests failed
        [[ "$failed" -eq 0 ]]
    else
        # No jq, just show the raw JSON
        cat "$results_file"
        echo ""
        grep -q '"failed": 0' "$results_file"
    fi
}

cleanup() {
    # Check if VM exists first
    local vm_exists=false
    if [[ -n "$VM_NAME" ]] && multipass list 2>/dev/null | grep -q "$VM_NAME"; then
        vm_exists=true
    fi

    if [[ "$KEEP_VM" == "true" ]] && [[ "$vm_exists" == "true" ]]; then
        log_warning "Keeping VM for debugging: $VM_NAME"
        echo ""
        echo "To inspect the VM:"
        echo "  multipass shell $VM_NAME"
        echo ""
        echo "To view cloud-init logs:"
        echo "  multipass exec $VM_NAME -- sudo cat /var/log/cloud-init-output.log"
        echo ""
        echo "To view test results:"
        echo "  multipass exec $VM_NAME -- cat /tmp/test-results.json"
        echo ""
        echo "To cleanup when done:"
        echo "  multipass delete $VM_NAME && multipass purge"
        echo ""
        return 0
    fi

    if [[ "$vm_exists" == "true" ]]; then
        log_info "Cleaning up VM: $VM_NAME"
        multipass delete "$VM_NAME" 2>/dev/null || true
        multipass purge 2>/dev/null || true
        log_success "Cleanup complete"
    fi

    # Clean up generated secrets if we copied test secrets
    if [[ -f "$SECRETS_FILE" ]] && cmp -s "$SECRETS_FILE" "$TEST_SECRETS_FILE" 2>/dev/null; then
        rm -f "$SECRETS_FILE"
        log_debug "Removed test secrets"
    fi
}

show_help() {
    echo "Multipass Cloud-Init Integration Test Runner"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --keep            Keep VM after test for debugging"
    echo "  --timeout SECS    Timeout in seconds (default: $DEFAULT_TIMEOUT)"
    echo "  --name NAME       Custom VM name (default: auto-generated)"
    echo "  --verbose, -v     Verbose output"
    echo "  --help, -h        Show this help"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run test and cleanup"
    echo "  $0 --keep             # Keep VM for debugging"
    echo "  $0 --timeout 600      # 10 minute timeout"
    echo ""
}

#==============================================================================
# Main
#==============================================================================

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --keep)
                KEEP_VM=true
                shift
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --name)
                VM_NAME="$2"
                shift 2
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Generate VM name if not provided
    if [[ -z "$VM_NAME" ]]; then
        generate_vm_name
    fi

    # Set up trap for cleanup
    trap cleanup EXIT

    log_section "Multipass Cloud-Init Integration Test"
    echo ""
    echo "VM Name: $VM_NAME"
    echo "Timeout: ${TIMEOUT}s"
    echo "Keep VM: $KEEP_VM"
    echo ""

    # Run the test
    check_multipass_installed
    setup_secrets
    generate_cloud_init
    launch_vm
    wait_for_cloud_init
    retrieve_results
    display_results

    local exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        log_section "Test Passed"
    else
        log_section "Test Failed"
    fi

    exit $exit_code
}

main "$@"
