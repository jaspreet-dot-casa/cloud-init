#!/bin/bash
#==============================================================================
# Cloud-Init YAML Generator
#
# Generates cloud-init.yaml from template by substituting variables
# from secrets.env and config.env.
#
# Usage: ./generate.sh [--dry-run] [--validate]
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Files
TEMPLATE_FILE="${SCRIPT_DIR}/cloud-init.template.yaml"
SECRETS_FILE="${SCRIPT_DIR}/secrets.env"
CONFIG_FILE="${PROJECT_ROOT}/config.env"
OUTPUT_FILE="${SCRIPT_DIR}/cloud-init.yaml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}→ $1${NC}"; }
log_success() { echo -e "${GREEN}✓ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
log_error()   { echo -e "${RED}✗ $1${NC}" >&2; }

#==============================================================================
# Functions
#==============================================================================

check_dependencies() {
    if ! command -v envsubst &>/dev/null; then
        log_error "envsubst not found. Install with: apt-get install gettext-base"
        exit 1
    fi
}

check_secrets() {
    if [[ ! -f "${SECRETS_FILE}" ]]; then
        log_error "Secrets file not found: ${SECRETS_FILE}"
        log_info "Copy secrets.env.template to secrets.env and fill in your values:"
        log_info "  cp secrets.env.template secrets.env"
        exit 1
    fi
}

validate_secrets() {
    log_info "Validating secrets..."

    # Source secrets
    # shellcheck source=/dev/null
    source "${SECRETS_FILE}"

    local errors=0

    # Required fields
    if [[ -z "${USERNAME:-}" ]]; then
        log_error "USERNAME is required"
        ((errors++))
    fi

    if [[ -z "${HOSTNAME:-}" ]]; then
        log_error "HOSTNAME is required"
        ((errors++))
    fi

    if [[ -z "${SSH_PUBLIC_KEY:-}" || "${SSH_PUBLIC_KEY}" == *"your-email"* ]]; then
        log_error "SSH_PUBLIC_KEY is required (and must be your actual key)"
        ((errors++))
    fi

    if [[ -z "${USER_NAME:-}" ]]; then
        log_error "USER_NAME is required"
        ((errors++))
    fi

    if [[ -z "${USER_EMAIL:-}" ]]; then
        log_error "USER_EMAIL is required"
        ((errors++))
    fi

    if [[ ${errors} -gt 0 ]]; then
        log_error "${errors} validation error(s). Edit secrets.env and try again."
        exit 1
    fi

    log_success "Secrets validated"
}

generate_yaml() {
    log_info "Generating cloud-init.yaml..."

    # Source secrets and config
    set -a  # Export all variables
    # shellcheck source=/dev/null
    source "${SECRETS_FILE}"

    if [[ -f "${CONFIG_FILE}" ]]; then
        # shellcheck source=/dev/null
        source "${CONFIG_FILE}"
    fi

    # Set defaults for optional values
    TAILSCALE_AUTH_KEY="${TAILSCALE_AUTH_KEY:-}"
    GITHUB_PAT="${GITHUB_PAT:-}"
    GITHUB_USER="${GITHUB_USER:-}"
    REPO_URL="${REPO_URL:-https://github.com/jaspreet-dot-casa/cloud-init.git}"
    REPO_BRANCH="${REPO_BRANCH:-main}"

    set +a

    # Only substitute these specific template variables
    # This prevents envsubst from replacing local shell variables in embedded scripts
    local TEMPLATE_VARS
    # shellcheck disable=SC2016
    TEMPLATE_VARS='$USERNAME $HOSTNAME $SSH_PUBLIC_KEY $USER_NAME $USER_EMAIL $REPO_URL $REPO_BRANCH $TAILSCALE_AUTH_KEY $GITHUB_PAT $GITHUB_USER'

    # Generate output
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        log_info "[DRY-RUN] Would generate: ${OUTPUT_FILE}"
        envsubst "${TEMPLATE_VARS}" < "${TEMPLATE_FILE}"
    else
        envsubst "${TEMPLATE_VARS}" < "${TEMPLATE_FILE}" > "${OUTPUT_FILE}"
        log_success "Generated: ${OUTPUT_FILE}"
    fi
}

validate_yaml() {
    log_info "Validating generated YAML..."

    local file="${1:-${OUTPUT_FILE}}"

    if [[ ! -f "${file}" ]]; then
        log_error "File not found: ${file}"
        return 1
    fi

    # Check for unsubstituted template variables (only our known template vars)
    # Don't flag bash variables like ${BLUE}, ${NC}, etc.
    local template_vars='USERNAME|HOSTNAME|SSH_PUBLIC_KEY|USER_NAME|USER_EMAIL|REPO_URL|REPO_BRANCH|TAILSCALE_AUTH_KEY|GITHUB_PAT|GITHUB_USER'
    if grep -qE "\\\$\{(${template_vars})\}" "${file}"; then
        log_warning "Found unsubstituted template variables in output:"
        grep -oE "\\\$\{(${template_vars})\}" "${file}" | sort -u | while read -r var; do
            echo "  - ${var}"
        done
        return 1
    fi

    # Check YAML syntax if yq is available
    if command -v yq &>/dev/null; then
        if yq eval '.' "${file}" >/dev/null 2>&1; then
            log_success "YAML syntax is valid"
        else
            log_error "YAML syntax error"
            yq eval '.' "${file}" 2>&1 || true
            return 1
        fi
    else
        log_info "yq not installed, skipping YAML validation"
    fi

    log_success "Validation passed"
}

show_summary() {
    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${GREEN}Cloud-Init Configuration Generated${NC}"
    echo "════════════════════════════════════════════"
    echo ""
    echo "Output file: ${OUTPUT_FILE}"
    echo ""
    echo "Configuration:"

    # shellcheck source=/dev/null
    source "${SECRETS_FILE}"
    echo "  Hostname: ${HOSTNAME}"
    echo "  Username: ${USERNAME}"
    echo "  Tailscale: ${TAILSCALE_AUTH_KEY:+configured}${TAILSCALE_AUTH_KEY:-not configured}"
    echo ""
    echo "Next steps:"
    echo "  1. For cloud VM: Use cloud-init.yaml as user-data"
    echo "  2. For bare-metal: Run ./create-usb.sh"
    echo "  3. For libvirt: Run 'cd ../terraform && terraform apply'"
    echo ""
}

#==============================================================================
# Main
#==============================================================================

main() {
    local validate_only=false

    for arg in "$@"; do
        case "${arg}" in
            --dry-run|-n)
                export DRY_RUN=true
                ;;
            --validate)
                validate_only=true
                ;;
            --help|-h)
                echo "Usage: $0 [--dry-run] [--validate]"
                echo ""
                echo "Options:"
                echo "  --dry-run, -n    Preview output without writing file"
                echo "  --validate       Validate existing cloud-init.yaml"
                exit 0
                ;;
        esac
    done

    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}Cloud-Init Generator${NC}"
    echo "════════════════════════════════════════════"
    echo ""

    check_dependencies

    if [[ "${validate_only}" == "true" ]]; then
        validate_yaml
        exit $?
    fi

    check_secrets
    validate_secrets
    generate_yaml

    if [[ "${DRY_RUN:-false}" != "true" ]]; then
        validate_yaml
        show_summary
    fi
}

main "$@"
