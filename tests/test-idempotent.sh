#!/bin/bash
#==============================================================================
# Idempotency Tests
#
# Tests that scripts can be run multiple times without side effects.
#
# Usage: ./test-idempotent.sh
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source test utilities
source "${PROJECT_ROOT}/scripts/lib/core.sh"
source "${PROJECT_ROOT}/scripts/lib/dryrun.sh"

# Always run in dry-run mode for these tests
export DRY_RUN=true

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

#==============================================================================
# Test Functions
#==============================================================================

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
    esac
}


test_library_multiple_source() {
    echo ""
    log_info "Testing library multiple sourcing..."

    # Libraries should be safe to source multiple times
    local libs=(
        "scripts/lib/core.sh"
        "scripts/lib/version.sh"
        "scripts/lib/lock.sh"
        "scripts/lib/dryrun.sh"
    )

    for lib in "${libs[@]}"; do
        local lib_path="${PROJECT_ROOT}/$lib"
        if [[ -f "$lib_path" ]]; then
            # Source twice in a subshell
            if (source "$lib_path" && source "$lib_path" 2>/dev/null); then
                log_test pass "$(basename "$lib") safe to source multiple times"
            else
                log_test fail "$(basename "$lib") safe to source multiple times" "Error on re-source"
            fi
        fi
    done
}


test_package_help_idempotent() {
    echo ""
    log_info "Testing package script --help idempotency..."

    for script in "${PROJECT_ROOT}"/scripts/packages/*.sh; do
        if [[ -f "$script" && "$(basename "$script")" != "_template.sh" ]]; then
            local name
            name="$(basename "$script" .sh)"

            # Run --help twice
            local help1 help2
            help1=$(bash "$script" --help 2>&1 || true)
            help2=$(bash "$script" --help 2>&1 || true)

            if [[ "$help1" == "$help2" ]]; then
                log_test pass "$name --help is idempotent"
            else
                log_test fail "$name --help is idempotent" "Help output changed"
            fi
        fi
    done
}

#==============================================================================
# Main
#==============================================================================

main() {
    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}Idempotency Tests${NC}"
    echo "════════════════════════════════════════════"
    echo ""
    echo "All tests run in DRY_RUN=true mode"

    test_library_multiple_source
    test_package_help_idempotent

    # Summary
    echo ""
    echo "════════════════════════════════════════════"
    echo "Results: ${TESTS_PASSED}/${TESTS_RUN} passed"
    echo "════════════════════════════════════════════"

    if [[ $TESTS_FAILED -gt 0 ]]; then
        exit 1
    fi
}

main "$@"
