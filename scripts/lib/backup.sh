#!/bin/bash
#==============================================================================
# Backup Library - Configuration backup and rollback
#
# Usage: source "${SCRIPT_DIR}/lib/backup.sh"
#
# Backup structure:
#   ~/.local-remote/backups/
#     2024-12-22T10:30:00/
#       shell/
#         35-lazygit.sh
#         70-starship.sh
#       gitconfig
#       zshrc
#     2024-12-22T09:15:00/
#       ...
#==============================================================================

# Prevent double-sourcing
if [[ -n "${_LIB_BACKUP_SOURCED:-}" ]]; then
    return 0
fi
_LIB_BACKUP_SOURCED=1

# Source core library if not already sourced
SCRIPT_DIR_BACKUP="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR_BACKUP}/core.sh"

#==============================================================================
# Backup Configuration
#==============================================================================

BACKUP_ROOT="${HOME}/.local-remote/backups"
MAX_BACKUPS=5  # Keep last N backups

# Files/directories to backup before config changes
BACKUP_PATHS=(
    "${HOME}/.config/shell"
    "${HOME}/.gitconfig"
    "${HOME}/.zshrc"
    "${HOME}/.zsh_custom_config"
)

#==============================================================================
# Backup Operations
#==============================================================================

# Create a new backup with timestamp
# Usage: backup_id=$(create_backup)
create_backup() {
    local timestamp
    timestamp=$(date +%Y-%m-%dT%H:%M:%S)
    local backup_dir="${BACKUP_ROOT}/${timestamp}"

    ensure_dir "${backup_dir}"

    log_info "Creating backup: ${timestamp}"

    local backed_up=0
    for path in "${BACKUP_PATHS[@]}"; do
        if [[ -e "${path}" ]]; then
            local relative_path="${path#"${HOME}"/}"
            local dest_dir
            dest_dir="${backup_dir}/$(dirname "${relative_path}")"
            ensure_dir "${dest_dir}"

            if [[ -d "${path}" ]]; then
                cp -r "${path}" "${dest_dir}/"
            else
                cp "${path}" "${dest_dir}/"
            fi
            ((backed_up++)) || true
            log_debug "Backed up: ${relative_path}"
        fi
    done

    if [[ ${backed_up} -eq 0 ]]; then
        log_debug "No files to backup (first install)"
        rmdir "${backup_dir}" 2>/dev/null
        return 0  # Not an error - just nothing to backup
    fi

    # Create manifest
    cat > "${backup_dir}/manifest.txt" << EOF
Backup created: ${timestamp}
Files backed up: ${backed_up}
Paths:
$(for p in "${BACKUP_PATHS[@]}"; do [[ -e "${p}" ]] && echo "  - ${p#"${HOME}"/}"; done)
EOF

    log_success "Backup created: ${timestamp} (${backed_up} files)"

    # Cleanup old backups
    cleanup_old_backups

    echo "${timestamp}"
}

# Cleanup old backups, keeping only MAX_BACKUPS
cleanup_old_backups() {
    if [[ ! -d "${BACKUP_ROOT}" ]]; then
        return 0
    fi

    local count
    count=$(find "${BACKUP_ROOT}" -mindepth 1 -maxdepth 1 -type d | wc -l)

    if [[ ${count} -le ${MAX_BACKUPS} ]]; then
        return 0
    fi

    log_debug "Cleaning up old backups (keeping ${MAX_BACKUPS})"

    # Get list of backups sorted by name (timestamp), oldest first
    local to_delete=$((count - MAX_BACKUPS))
    find "${BACKUP_ROOT}" -mindepth 1 -maxdepth 1 -type d | sort | head -n "${to_delete}" | while read -r dir; do
        log_debug "Removing old backup: $(basename "${dir}")"
        rm -rf "${dir}"
    done
}

# List available backups
# Usage: list_backups
# shellcheck disable=SC2154
list_backups() {
    if [[ ! -d "${BACKUP_ROOT}" ]]; then
        log_info "No backups found"
        return 0
    fi

    log_section "Available Backups"

    local count=0
    for backup_dir in $(find "${BACKUP_ROOT}" -mindepth 1 -maxdepth 1 -type d | sort -r); do
        local timestamp
        timestamp=$(basename "${backup_dir}")
        local file_count
        file_count=$(find "${backup_dir}" -type f ! -name 'manifest.txt' | wc -l)

        if [[ ${count} -eq 0 ]]; then
            echo -e "  ${GREEN}[latest]${NC} ${timestamp} (${file_count} files)"
        else
            echo "  [${count}]      ${timestamp} (${file_count} files)"
        fi
        ((count++)) || true
    done

    if [[ ${count} -eq 0 ]]; then
        log_info "No backups found"
    fi
}

# Get the most recent backup directory
# Usage: latest=$(get_latest_backup)
get_latest_backup() {
    if [[ ! -d "${BACKUP_ROOT}" ]]; then
        return 1
    fi

    find "${BACKUP_ROOT}" -mindepth 1 -maxdepth 1 -type d | sort -r | head -1
}

# Restore from a backup
# Usage: restore_backup [timestamp]
# If no timestamp provided, restores from latest
restore_backup() {
    local timestamp="${1:-}"
    local backup_dir

    if [[ -z "${timestamp}" ]]; then
        backup_dir=$(get_latest_backup)
        if [[ -z "${backup_dir}" ]]; then
            log_error "No backups available to restore"
            return 1
        fi
        timestamp=$(basename "${backup_dir}")
    else
        backup_dir="${BACKUP_ROOT}/${timestamp}"
    fi

    if [[ ! -d "${backup_dir}" ]]; then
        log_error "Backup not found: ${timestamp}"
        return 1
    fi

    log_info "Restoring from backup: ${timestamp}"

    local restored=0
    for path in "${BACKUP_PATHS[@]}"; do
        local relative_path="${path#"${HOME}"/}"
        local source="${backup_dir}/${relative_path}"

        if [[ -e "${source}" ]]; then
            local dest_dir
            dest_dir=$(dirname "${path}")
            ensure_dir "${dest_dir}"

            # Remove existing before restore
            if [[ -e "${path}" ]]; then
                rm -rf "${path}"
            fi

            if [[ -d "${source}" ]]; then
                cp -r "${source}" "${dest_dir}/"
            else
                cp "${source}" "${path}"
            fi
            ((restored++)) || true
            log_debug "Restored: ${relative_path}"
        fi
    done

    log_success "Restored ${restored} files from backup: ${timestamp}"
    return 0
}

# Show diff between current config and backup
# Usage: diff_backup [timestamp]
diff_backup() {
    local timestamp="${1:-}"
    local backup_dir

    if [[ -z "${timestamp}" ]]; then
        backup_dir=$(get_latest_backup)
        if [[ -z "${backup_dir}" ]]; then
            log_error "No backups available"
            return 1
        fi
        timestamp=$(basename "${backup_dir}")
    else
        backup_dir="${BACKUP_ROOT}/${timestamp}"
    fi

    if [[ ! -d "${backup_dir}" ]]; then
        log_error "Backup not found: ${timestamp}"
        return 1
    fi

    log_section "Diff: Current vs Backup (${timestamp})"

    for path in "${BACKUP_PATHS[@]}"; do
        local relative_path="${path#"${HOME}"/}"
        local backup_path="${backup_dir}/${relative_path}"

        if [[ -f "${path}" && -f "${backup_path}" ]]; then
            echo "--- ${relative_path} ---"
            diff -u "${backup_path}" "${path}" || true
            echo ""
        elif [[ -d "${path}" && -d "${backup_path}" ]]; then
            echo "--- ${relative_path}/ ---"
            diff -rq "${backup_path}" "${path}" || true
            echo ""
        fi
    done
}

#==============================================================================
# Pre-operation Hooks
#==============================================================================

# Backup before making changes (idempotent - only creates if changes pending)
# Usage: backup_before_changes
backup_before_changes() {
    local last_backup
    last_backup=$(get_latest_backup) || true

    # If no backup exists, create one
    if [[ -z "${last_backup}" ]]; then
        create_backup
        return 0
    fi

    # If last backup is less than 1 hour old, skip
    local backup_time
    backup_time=$(basename "${last_backup}")
    local backup_epoch
    backup_epoch=$(date -d "${backup_time}" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%S" "${backup_time}" +%s 2>/dev/null)
    local now_epoch
    now_epoch=$(date +%s)
    local age=$((now_epoch - backup_epoch))

    if [[ ${age} -lt 3600 ]]; then
        log_debug "Recent backup exists (${age}s ago), skipping"
        return 0
    fi

    create_backup
}
