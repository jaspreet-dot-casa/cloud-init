// Package doctor provides dependency checking and fixing for ucli.
package doctor

// CheckStatus represents the status of a dependency check.
type CheckStatus int

const (
	// StatusOK indicates the dependency is installed and working.
	StatusOK CheckStatus = iota
	// StatusMissing indicates the dependency is not installed.
	StatusMissing
	// StatusError indicates an error occurred during the check.
	StatusError
	// StatusWarning indicates the dependency has issues but may work.
	StatusWarning
)

// String returns the string representation of the status.
func (s CheckStatus) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusMissing:
		return "missing"
	case StatusError:
		return "error"
	case StatusWarning:
		return "warning"
	default:
		return "unknown"
	}
}

// Check represents a single dependency check result.
type Check struct {
	ID          string      // Unique identifier, e.g., "terraform", "multipass"
	Name        string      // Display name
	Description string      // What this tool does
	Status      CheckStatus // Current status
	Message     string      // Status message (version info, error, etc.)
	FixCommand  *FixCommand // How to fix if missing (nil if not fixable)
}

// FixCommand describes how to fix a missing dependency.
type FixCommand struct {
	Description string // Human-readable description of what the fix does
	Command     string // Shell command to run
	Sudo        bool   // Whether the command requires sudo
	Platform    string // Target platform: "darwin", "linux", or "" for both
}

// CheckGroup represents a group of related dependency checks.
type CheckGroup struct {
	ID          string  // Unique identifier, e.g., "terraform", "multipass", "iso"
	Name        string  // Display name
	Description string  // What this group is for
	Platform    string  // Target platform: "darwin", "linux", or "" for both
	Checks      []Check // Individual checks in this group
}

// GroupID constants for check groups.
const (
	GroupTerraform = "terraform"
	GroupMultipass = "multipass"
	GroupISO       = "iso"
	GroupTerminal  = "terminal"
)

// CheckID constants for individual checks.
const (
	IDTerraform  = "terraform"
	IDLibvirt    = "libvirt"
	IDVirsh      = "virsh"
	IDQemuKVM    = "qemu-kvm"
	IDCloudImage = "cloud-image"
	IDMultipass  = "multipass"
	IDXorriso    = "xorriso"
	IDGhostty    = "ghostty"
)
