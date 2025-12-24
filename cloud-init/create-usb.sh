#!/bin/bash
#==============================================================================
# Cloud-Init USB Creator
#
# Creates a bootable USB drive with cloud-init configuration for NoCloud
# datasource. Works on both macOS and Linux.
#
# Usage: ./create-usb.sh [device]
#
# The USB drive will contain:
#   - meta-data: Instance metadata
#   - user-data: Cloud-init configuration (from cloud-init.yaml)
#
# For bare-metal installation:
#   1. Boot from Ubuntu Server ISO
#   2. Insert this USB during installation
#   3. Ubuntu will auto-detect and apply cloud-init config
#==============================================================================

set -e
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Files
CLOUD_INIT_YAML="${SCRIPT_DIR}/cloud-init.yaml"
META_DATA_FILE=""
USER_DATA_FILE=""

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
# Platform Detection
#==============================================================================

detect_platform() {
    case "$(uname -s)" in
        Darwin)
            echo "macos"
            ;;
        Linux)
            echo "linux"
            ;;
        *)
            log_error "Unsupported platform: $(uname -s)"
            exit 1
            ;;
    esac
}

#==============================================================================
# Device Selection
#==============================================================================

list_devices_macos() {
    echo ""
    echo "Available devices:"
    diskutil list external | grep -E "^/dev/disk" || true
    echo ""
}

list_devices_linux() {
    echo ""
    echo "Available devices:"
    lsblk -d -o NAME,SIZE,TYPE,MOUNTPOINT | grep -E "^sd|^nvme" || true
    echo ""
}

select_device() {
    local platform="$1"
    local device="${2:-}"

    if [[ -n "${device}" ]]; then
        echo "${device}"
        return
    fi

    echo ""
    log_warning "No device specified. Please select a USB device."

    if [[ "${platform}" == "macos" ]]; then
        list_devices_macos
        read -rp "Enter device (e.g., /dev/disk2): " device
    else
        list_devices_linux
        read -rp "Enter device (e.g., /dev/sdb): " device
    fi

    echo "${device}"
}

#==============================================================================
# Safety Checks
#==============================================================================

confirm_device() {
    local device="$1"
    local platform="$2"

    echo ""
    log_warning "THIS WILL ERASE ALL DATA ON: ${device}"
    echo ""

    # Show device info
    if [[ "${platform}" == "macos" ]]; then
        diskutil info "${device}" 2>/dev/null | grep -E "Device|Media Name|Total Size" || true
    else
        lsblk "${device}" 2>/dev/null || true
    fi

    echo ""
    read -rp "Are you sure? Type 'yes' to continue: " confirm

    if [[ "${confirm}" != "yes" ]]; then
        log_info "Cancelled"
        exit 0
    fi
}

#==============================================================================
# USB Creation
#==============================================================================

create_usb_macos() {
    local device="$1"

    log_info "Unmounting device..."
    diskutil unmountDisk "${device}" || true

    log_info "Formatting as FAT32 with label 'cidata'..."
    diskutil eraseDisk FAT32 cidata MBRFormat "${device}"

    # Get the partition
    local partition="${device}s1"

    log_info "Mounting partition..."
    diskutil mount "${partition}"

    local mount_point="/Volumes/cidata"

    log_info "Copying cloud-init files..."
    create_cloud_init_files "${mount_point}"

    log_info "Unmounting..."
    diskutil unmount "${mount_point}"

    log_success "USB created successfully!"
}

create_usb_linux() {
    local device="$1"

    # Unmount if mounted
    log_info "Unmounting device..."
    umount "${device}"* 2>/dev/null || true

    log_info "Creating partition table..."
    sudo parted -s "${device}" mklabel msdos
    sudo parted -s "${device}" mkpart primary fat32 1MiB 100%

    # Get partition name
    local partition="${device}1"
    if [[ "${device}" == *"nvme"* ]]; then
        partition="${device}p1"
    fi

    log_info "Formatting as FAT32 with label 'cidata'..."
    sudo mkfs.vfat -n cidata "${partition}"

    log_info "Mounting partition..."
    local mount_point="/tmp/cidata-$$"
    mkdir -p "${mount_point}"
    sudo mount "${partition}" "${mount_point}"

    log_info "Copying cloud-init files..."
    create_cloud_init_files "${mount_point}"

    log_info "Unmounting..."
    sudo umount "${mount_point}"
    rmdir "${mount_point}"

    log_success "USB created successfully!"
}

create_cloud_init_files() {
    local mount_point="$1"

    # Create meta-data
    log_info "Creating meta-data..."
    # Extract hostname from cloud-init.yaml or use default
    local instance_id
    instance_id="iid-local-$(date +%Y%m%d%H%M%S)"

    local hostname
    if [[ -f "${CLOUD_INIT_YAML}" ]]; then
        hostname=$(grep -E "^hostname:" "${CLOUD_INIT_YAML}" | awk '{print $2}' || echo "ubuntu-server")
    else
        hostname="ubuntu-server"
    fi

    cat > "${mount_point}/meta-data" << EOF
instance-id: ${instance_id}
local-hostname: ${hostname}
EOF

    # Copy user-data (cloud-init.yaml)
    log_info "Copying user-data..."
    if [[ -f "${CLOUD_INIT_YAML}" ]]; then
        cp "${CLOUD_INIT_YAML}" "${mount_point}/user-data"
    else
        log_error "cloud-init.yaml not found. Run ./generate.sh first."
        exit 1
    fi

    # Create empty network-config (use DHCP)
    log_info "Creating network-config..."
    cat > "${mount_point}/network-config" << EOF
version: 2
ethernets:
  id0:
    match:
      driver: "*"
    dhcp4: true
EOF

    log_success "Cloud-init files created"
}

#==============================================================================
# Main
#==============================================================================

main() {
    local device="${1:-}"

    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${BLUE}Cloud-Init USB Creator${NC}"
    echo "════════════════════════════════════════════"
    echo ""

    # Check for cloud-init.yaml
    if [[ ! -f "${CLOUD_INIT_YAML}" ]]; then
        log_error "cloud-init.yaml not found"
        log_info "Run ./generate.sh first to create it"
        exit 1
    fi

    # Parse arguments
    for arg in "$@"; do
        case "${arg}" in
            --help|-h)
                echo "Usage: $0 [device]"
                echo ""
                echo "Creates a bootable USB with cloud-init configuration."
                echo ""
                echo "Arguments:"
                echo "  device    Target device (e.g., /dev/disk2 on macOS, /dev/sdb on Linux)"
                echo ""
                echo "Examples:"
                echo "  $0 /dev/disk2      # macOS"
                echo "  $0 /dev/sdb        # Linux"
                exit 0
                ;;
            --dry-run|-n)
                log_info "Dry-run mode - would create USB with:"
                echo "  meta-data: instance metadata"
                echo "  user-data: $(wc -l < "${CLOUD_INIT_YAML}") lines from cloud-init.yaml"
                echo "  network-config: DHCP configuration"
                exit 0
                ;;
        esac
    done

    local platform
    platform=$(detect_platform)
    log_info "Platform: ${platform}"

    device=$(select_device "${platform}" "${device}")

    if [[ -z "${device}" ]]; then
        log_error "No device specified"
        exit 1
    fi

    # Validate device exists
    if [[ ! -e "${device}" ]]; then
        log_error "Device not found: ${device}"
        exit 1
    fi

    confirm_device "${device}" "${platform}"

    if [[ "${platform}" == "macos" ]]; then
        create_usb_macos "${device}"
    else
        create_usb_linux "${device}"
    fi

    echo ""
    echo "════════════════════════════════════════════"
    echo -e "${GREEN}USB Ready!${NC}"
    echo "════════════════════════════════════════════"
    echo ""
    echo "To use:"
    echo "  1. Boot target machine from Ubuntu Server ISO"
    echo "  2. Insert this USB during installation"
    echo "  3. Cloud-init will auto-detect and apply configuration"
    echo ""
    echo "The USB contains:"
    echo "  - meta-data: Instance identification"
    echo "  - user-data: Your cloud-init configuration"
    echo "  - network-config: DHCP network setup"
    echo ""
}

main "$@"
