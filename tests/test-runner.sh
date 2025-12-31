#!/bin/bash
#==============================================================================
# Test Runner
#
# Orchestrates all tests for the cloud-init setup scripts.
#
# Usage: ./test-runner.sh [--verbose] [--filter PATTERN]
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Options
VERBOSE=false
FILTER=""

#==============================================================================
# Functions
#==============================================================================

log_info()    { echo -e "${BLUE}→ $1${NC}"; }
log_success() { echo -e "${GREEN}✓ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
log_error()   { echo -e "${RED}✗ $1${NC}" >&2; }

log_test() {
    local status="$1"
    local name="$2"
    local msg="${3:-}"

    : $((TESTS_RUN++))

    case "$status" in
        pass)
            : $((TESTS_PASSED++))
            echo -e "  ${GREEN}✓${NC} $name"
            ;;
        fail)
            : $((TESTS_FAILED++))
            echo -e "  ${RED}✗${NC} $name"
            if [[ -n "$msg" ]]; then
                echo -e "    ${RED}$msg${NC}"
            fi
            ;;
        skip)
            : $((TESTS_SKIPPED++))
            echo -e "  ${YELLOW}○${NC} $name (skipped)"
            ;;
    esac
}

run_test_file() {
    local test_file="$1"
    local test_name
    test_name="$(basename "$test_file" .sh)"

    if [[ -n "$FILTER" && ! "$test_name" =~ $FILTER ]]; then
        return 0
    fi

    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}Running: $test_name${NC}"
    echo "════════════════════════════════════════════"

    if [[ ! -x "$test_file" ]]; then
        chmod +x "$test_file"
    fi

    if "$test_file"; then
        log_success "Test file passed: $test_name"
    else
        log_error "Test file failed: $test_name"
        : $((TESTS_FAILED++))
    fi
}

#==============================================================================
# Built-in Tests
#==============================================================================

test_library_files_exist() {
    echo ""
    log_info "Testing library files exist..."

    local libs=(
        "scripts/lib/core.sh"
        "scripts/lib/version.sh"
        "scripts/lib/lock.sh"
        "scripts/lib/backup.sh"
        "scripts/lib/health.sh"
        "scripts/lib/dryrun.sh"
    )

    for lib in "${libs[@]}"; do
        if [[ -f "${PROJECT_ROOT}/$lib" ]]; then
            log_test pass "$lib exists"
        else
            log_test fail "$lib exists" "File not found"
        fi
    done
}

test_library_syntax() {
    echo ""
    log_info "Testing library syntax..."

    for lib in "${PROJECT_ROOT}"/scripts/lib/*.sh; do
        local name
        name="$(basename "$lib")"
        if bash -n "$lib" 2>/dev/null; then
            log_test pass "$name syntax valid"
        else
            log_test fail "$name syntax valid" "Syntax error"
        fi
    done
}

test_library_sourcing() {
    echo ""
    log_info "Testing library sourcing..."

    for lib in "${PROJECT_ROOT}"/scripts/lib/*.sh; do
        local name
        name="$(basename "$lib")"
        if (source "$lib" 2>/dev/null); then
            log_test pass "$name sources without error"
        else
            log_test fail "$name sources without error" "Source error"
        fi
    done
}

test_package_scripts_exist() {
    echo ""
    log_info "Testing package scripts exist..."

    local packages=(
        "scripts/packages/apt.sh"
        "scripts/packages/docker.sh"
        "scripts/packages/github-cli.sh"
        "scripts/packages/lazygit.sh"
        "scripts/packages/starship.sh"
    )

    for pkg in "${packages[@]}"; do
        if [[ -f "${PROJECT_ROOT}/$pkg" ]]; then
            log_test pass "$pkg exists"
        else
            log_test fail "$pkg exists" "File not found"
        fi
    done
}

test_shared_scripts_exist() {
    echo ""
    log_info "Testing shared scripts exist..."

    local scripts=(
        "scripts/shared/tailscale.sh"
        "scripts/shared/configure-git.sh"
        "scripts/shared/configure-zsh.sh"
    )

    for script in "${scripts[@]}"; do
        if [[ -f "${PROJECT_ROOT}/$script" ]]; then
            log_test pass "$script exists"
        else
            log_test fail "$script exists" "File not found"
        fi
    done
}

test_cloud_init_files_exist() {
    echo ""
    log_info "Testing cloud-init files exist..."

    local files=(
        "cloud-init/cloud-init.template.yaml"
        "cloud-init/create-usb.sh"
        "cloud-init/Makefile"
    )

    for file in "${files[@]}"; do
        if [[ -f "${PROJECT_ROOT}/$file" ]]; then
            log_test pass "$file exists"
        else
            log_test fail "$file exists" "File not found"
        fi
    done
}

test_terraform_files_exist() {
    echo ""
    log_info "Testing Terraform files exist..."

    local files=(
        "terraform/main.tf"
        "terraform/variables.tf"
        "terraform/outputs.tf"
    )

    for file in "${files[@]}"; do
        if [[ -f "${PROJECT_ROOT}/$file" ]]; then
            log_test pass "$file exists"
        else
            log_test fail "$file exists" "File not found"
        fi
    done
}

test_core_library_functions() {
    echo ""
    log_info "Testing core library functions..."

    source "${PROJECT_ROOT}/scripts/lib/core.sh"

    # Test command_exists
    if command_exists bash; then
        log_test pass "command_exists detects bash"
    else
        log_test fail "command_exists detects bash"
    fi

    if ! command_exists nonexistent_command_12345; then
        log_test pass "command_exists returns false for missing command"
    else
        log_test fail "command_exists returns false for missing command"
    fi

    # Test require_root (should fail as non-root)
    if ! require_root 2>/dev/null; then
        log_test pass "require_root fails when not root"
    else
        log_test skip "require_root (running as root)"
    fi
}

test_version_library_functions() {
    echo ""
    log_info "Testing version library functions..."

    source "${PROJECT_ROOT}/scripts/lib/core.sh"
    source "${PROJECT_ROOT}/scripts/lib/version.sh"

    # Test version_lt (less than)
    if version_lt "1.0.0" "2.0.0"; then
        log_test pass "version_lt: 1.0.0 < 2.0.0"
    else
        log_test fail "version_lt: 1.0.0 < 2.0.0"
    fi

    # Test version_gt (greater than)
    if version_gt "2.0.0" "1.0.0"; then
        log_test pass "version_gt: 2.0.0 > 1.0.0"
    else
        log_test fail "version_gt: 2.0.0 > 1.0.0"
    fi

    # Test version_eq (equal)
    if version_eq "1.0.0" "1.0.0"; then
        log_test pass "version_eq: 1.0.0 = 1.0.0"
    else
        log_test fail "version_eq: 1.0.0 = 1.0.0"
    fi
}

test_dryrun_library() {
    echo ""
    log_info "Testing dry-run library..."

    source "${PROJECT_ROOT}/scripts/lib/core.sh"
    source "${PROJECT_ROOT}/scripts/lib/dryrun.sh"

    # Test is_dry_run
    DRY_RUN=true
    if is_dry_run; then
        log_test pass "is_dry_run returns true when DRY_RUN=true"
    else
        log_test fail "is_dry_run returns true when DRY_RUN=true"
    fi

    DRY_RUN=false
    if ! is_dry_run; then
        log_test pass "is_dry_run returns false when DRY_RUN=false"
    else
        log_test fail "is_dry_run returns false when DRY_RUN=false"
    fi
}

#==============================================================================
# Main
#==============================================================================

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --filter|-f)
                FILTER="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: $0 [--verbose] [--filter PATTERN]"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}Cloud-Init Test Suite${NC}"
    echo "════════════════════════════════════════════"
    echo ""
    echo "Project root: ${PROJECT_ROOT}"
    echo ""

    # Run built-in tests
    test_library_files_exist
    test_library_syntax
    test_library_sourcing
    test_package_scripts_exist
    test_shared_scripts_exist
    test_cloud_init_files_exist
    test_terraform_files_exist
    test_core_library_functions
    test_version_library_functions
    test_dryrun_library

    # Run external test files
    for test_file in "${SCRIPT_DIR}"/test-*.sh; do
        if [[ -f "$test_file" && "$test_file" != "${SCRIPT_DIR}/test-runner.sh" ]]; then
            run_test_file "$test_file"
        fi
    done

    # Summary
    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}Test Summary${NC}"
    echo "════════════════════════════════════════════"
    echo ""
    echo -e "  Total:   ${TESTS_RUN}"
    echo -e "  ${GREEN}Passed:  ${TESTS_PASSED}${NC}"
    echo -e "  ${RED}Failed:  ${TESTS_FAILED}${NC}"
    echo -e "  ${YELLOW}Skipped: ${TESTS_SKIPPED}${NC}"
    echo ""

    if [[ $TESTS_FAILED -gt 0 ]]; then
        log_error "Some tests failed!"
        exit 1
    else
        log_success "All tests passed!"
        exit 0
    fi
}

main "$@"
