#!/bin/bash
#==============================================================================
# Configuration Tests
#
# Tests cloud-init template structure.
#
# Usage: ./test-config.sh
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

test_cloud_init_template() {
    echo ""
    log_info "Testing cloud-init template..."

    local template="${PROJECT_ROOT}/cloud-init/cloud-init.template.yaml"

    if [[ ! -f "$template" ]]; then
        log_test fail "cloud-init.template.yaml exists" "File not found"
        return 1
    fi

    log_test pass "cloud-init.template.yaml exists"

    # Check for required sections
    local sections=(
        "users:"
        "hostname:"
        "packages:"
        "write_files:"
        "runcmd:"
    )

    for section in "${sections[@]}"; do
        if grep -q "^${section}" "$template" 2>/dev/null; then
            log_test pass "Has $section section"
        else
            log_test fail "Has $section section" "Section not found"
        fi
    done

    # Check for template variables
    if grep -q '\${USERNAME}' "$template" 2>/dev/null; then
        log_test pass "Uses \${USERNAME} variable"
    else
        log_test fail "Uses \${USERNAME} variable" "Variable not found"
    fi

    if grep -q '\${HOSTNAME}' "$template" 2>/dev/null; then
        log_test pass "Uses \${HOSTNAME} variable"
    else
        log_test fail "Uses \${HOSTNAME} variable" "Variable not found"
    fi

    # Check for package disable exports
    if grep -q '\${DISABLED_PACKAGE_EXPORTS}' "$template" 2>/dev/null; then
        log_test pass "Uses \${DISABLED_PACKAGE_EXPORTS} variable"
    else
        log_test fail "Uses \${DISABLED_PACKAGE_EXPORTS} variable" "Variable not found"
    fi
}

test_fixture_configs() {
    echo ""
    log_info "Testing fixture configurations..."

    local fixtures_dir="${SCRIPT_DIR}/fixtures"

    if [[ ! -d "$fixtures_dir" ]]; then
        log_test fail "Fixtures directory exists" "Directory not found"
        return 1
    fi

    log_test pass "Fixtures directory exists"

    for fixture in "$fixtures_dir"/*.env; do
        if [[ -f "$fixture" ]]; then
            local name
            name="$(basename "$fixture")"

            if bash -n "$fixture" 2>/dev/null; then
                log_test pass "$name syntax valid"
            else
                log_test fail "$name syntax valid" "Syntax error"
            fi

            # Test sourcing
            if (source "$fixture" 2>/dev/null); then
                log_test pass "$name sources without error"
            else
                log_test fail "$name sources without error" "Source error"
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
    echo -e "${BLUE}Configuration Tests${NC}"
    echo "════════════════════════════════════════════"

    test_cloud_init_template
    test_fixture_configs

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
