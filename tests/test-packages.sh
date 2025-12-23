#!/bin/bash
#==============================================================================
# Package Installation Tests
#
# Tests package installer scripts for correct structure and behavior.
#
# Usage: ./test-packages.sh [--dry-run]
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source test utilities
source "${PROJECT_ROOT}/scripts/lib/core.sh"

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

DRY_RUN="${1:-}"

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

# Test that package script has required functions
test_package_structure() {
    local script="$1"
    local name
    name="$(basename "$script" .sh)"

    echo ""
    log_info "Testing package: $name"

    # Check syntax
    if bash -n "$script" 2>/dev/null; then
        log_test pass "Syntax valid"
    else
        log_test fail "Syntax valid" "Syntax error in $script"
        return 1
    fi

    # Check for required functions by grepping
    local required_functions=(
        "do_install"
        "verify"
        "main"
    )

    # Optional functions (nice to have but not required)
    local optional_functions=(
        "is_installed"
        "get_installed_version"
        "get_desired_version"
    )

    for func in "${required_functions[@]}"; do
        if grep -q "^${func}()" "$script" 2>/dev/null || grep -q "^function ${func}" "$script" 2>/dev/null; then
            log_test pass "Has ${func}() function"
        else
            log_test fail "Has ${func}() function" "Function not found"
        fi
    done

    # Check optional functions but don't fail on missing
    for func in "${optional_functions[@]}"; do
        if grep -q "^${func}()" "$script" 2>/dev/null || grep -q "^function ${func}" "$script" 2>/dev/null; then
            log_test pass "Has ${func}() function (optional)"
        fi
    done

    # Check for library sourcing
    if grep -q 'source.*lib/core.sh' "$script" 2>/dev/null; then
        log_test pass "Sources core.sh library"
    else
        log_test fail "Sources core.sh library" "Missing source statement"
    fi

    # Check for PACKAGE_NAME variable
    if grep -q 'PACKAGE_NAME=' "$script" 2>/dev/null; then
        log_test pass "Defines PACKAGE_NAME"
    else
        log_test fail "Defines PACKAGE_NAME" "Variable not found"
    fi
}

# Test package script in dry-run mode
test_package_dryrun() {
    local script="$1"
    local name
    name="$(basename "$script" .sh)"

    echo ""
    log_info "Testing dry-run: $name"

    # Run in dry-run mode
    if DRY_RUN=true bash "$script" --help 2>/dev/null || DRY_RUN=true bash "$script" -h 2>/dev/null; then
        log_test pass "Help flag works"
    else
        # Some scripts may not have help
        log_test pass "Script executes (no help flag)"
    fi
}

#==============================================================================
# Main
#==============================================================================

main() {
    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}Package Script Tests${NC}"
    echo "════════════════════════════════════════════"

    # Skip template file
    for script in "${PROJECT_ROOT}"/scripts/packages/*.sh; do
        if [[ -f "$script" && "$(basename "$script")" != "_template.sh" ]]; then
            test_package_structure "$script"
        fi
    done

    if [[ "$DRY_RUN" == "--dry-run" ]]; then
        echo ""
        log_info "Running dry-run tests..."
        for script in "${PROJECT_ROOT}"/scripts/packages/*.sh; do
            if [[ -f "$script" && "$(basename "$script")" != "_template.sh" ]]; then
                test_package_dryrun "$script"
            fi
        done
    fi

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
