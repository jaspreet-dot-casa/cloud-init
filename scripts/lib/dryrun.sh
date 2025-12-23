#!/bin/bash
#==============================================================================
# Dry-Run Library - Preview changes without applying them
#
# Usage: source "${SCRIPT_DIR}/lib/dryrun.sh"
#
# Set DRY_RUN=true to enable dry-run mode:
#   DRY_RUN=true ./install-all.sh
#==============================================================================

# Prevent double-sourcing
if [[ -n "${_LIB_DRYRUN_SOURCED:-}" ]]; then
    return 0
fi
_LIB_DRYRUN_SOURCED=1

# Source core library if not already sourced
SCRIPT_DIR_DRYRUN="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR_DRYRUN}/core.sh"

#==============================================================================
# Dry-Run State
#==============================================================================

# Global dry-run flag (set via environment or command line)
DRY_RUN="${DRY_RUN:-false}"

# Check if dry-run mode is enabled
is_dry_run() {
    [[ "$DRY_RUN" == "true" ]]
}

# Print dry-run prefix for messages
dry_run_prefix() {
    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} "
    fi
}

#==============================================================================
# Dry-Run Wrappers
#==============================================================================

# Execute command or print what would be done
# Usage: run_or_print command arg1 arg2
run_or_print() {
    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would execute: $*"
        return 0
    else
        "$@"
    fi
}

# Execute command with sudo or print
# Usage: sudo_or_print command arg1 arg2
sudo_or_print() {
    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would execute (sudo): $*"
        return 0
    else
        sudo "$@"
    fi
}

# Download file or print
# Usage: download_or_print URL destination
download_or_print() {
    local url="$1"
    local dest="$2"

    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would download: $url"
        echo -e "${MAGENTA}[DRY-RUN]${NC}     -> $dest"
        return 0
    else
        curl -fsSL "$url" -o "$dest"
    fi
}

# Install binary or print
# Usage: install_or_print source destination [mode]
install_or_print() {
    local src="$1"
    local dest="$2"
    local mode="${3:-755}"

    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would install: $src -> $dest (mode: $mode)"
        return 0
    else
        sudo install -m "$mode" "$src" "$dest"
    fi
}

# Create directory or print
# Usage: mkdir_or_print directory
mkdir_or_print() {
    local dir="$1"

    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would create directory: $dir"
        return 0
    else
        mkdir -p "$dir"
    fi
}

# Write file or print
# Usage: write_or_print destination content
# Note: Use heredoc for multi-line content
write_or_print() {
    local dest="$1"
    local content="$2"

    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would write to: $dest"
        if [[ "${VERBOSE:-false}" == "true" ]]; then
            echo -e "${MAGENTA}[DRY-RUN]${NC} Content preview:"
            echo "$content" | head -5 | sed 's/^/    /'
            local lines
            lines=$(echo "$content" | wc -l)
            if [[ $lines -gt 5 ]]; then
                echo "    ... ($((lines - 5)) more lines)"
            fi
        fi
        return 0
    else
        local dir
        dir=$(dirname "$dest")
        [[ -d "$dir" ]] || mkdir -p "$dir"
        echo "$content" > "$dest"
    fi
}

# Append to file or print
# Usage: append_or_print destination content
append_or_print() {
    local dest="$1"
    local content="$2"

    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would append to: $dest"
        return 0
    else
        echo "$content" >> "$dest"
    fi
}

# Remove file/directory or print
# Usage: rm_or_print path
rm_or_print() {
    local path="$1"

    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would remove: $path"
        return 0
    else
        rm -rf "$path"
    fi
}

# Symlink or print
# Usage: ln_or_print source destination
ln_or_print() {
    local src="$1"
    local dest="$2"

    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would symlink: $src -> $dest"
        return 0
    else
        ln -sf "$src" "$dest"
    fi
}

# Copy file or print
# Usage: cp_or_print source destination
cp_or_print() {
    local src="$1"
    local dest="$2"

    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would copy: $src -> $dest"
        return 0
    else
        cp -r "$src" "$dest"
    fi
}

#==============================================================================
# Package-Specific Wrappers
#==============================================================================

# Run apt-get or print
# Usage: apt_or_print install package1 package2
apt_or_print() {
    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would run: apt-get $*"
        return 0
    else
        sudo apt-get "$@"
    fi
}

# Run systemctl or print
# Usage: systemctl_or_print enable service
systemctl_or_print() {
    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would run: systemctl $*"
        return 0
    else
        sudo systemctl "$@"
    fi
}

# Run git config or print
# Usage: git_config_or_print --global user.name "Name"
git_config_or_print() {
    if is_dry_run; then
        echo -e "${MAGENTA}[DRY-RUN]${NC} Would run: git config $*"
        return 0
    else
        git config "$@"
    fi
}

#==============================================================================
# Dry-Run Summary
#==============================================================================

# Counter for dry-run actions (use simple assignment for bash 3.2 compatibility)
DRY_RUN_ACTION_COUNT="${DRY_RUN_ACTION_COUNT:-0}"

# Print dry-run summary
print_dry_run_summary() {
    if is_dry_run; then
        echo ""
        log_section "Dry-Run Summary"
        echo -e "${MAGENTA}No changes were made.${NC}"
        echo "To apply these changes, run without DRY_RUN=true"
        echo ""
    fi
}

#==============================================================================
# Command Line Parsing Helper
#==============================================================================

# Parse --dry-run flag from command line
# Usage: parse_dry_run_flag "$@"
parse_dry_run_flag() {
    for arg in "$@"; do
        case "$arg" in
            --dry-run|-n)
                DRY_RUN=true
                log_info "Dry-run mode enabled"
                ;;
        esac
    done
}
