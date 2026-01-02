package doctor

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Platform constants.
const (
	PlatformDarwin = "darwin"
	PlatformLinux  = "linux"
)

// fixCommands defines platform-specific fix commands for each tool.
var fixCommands = map[string]map[string]*FixCommand{
	IDTerraform: {
		PlatformDarwin: {
			Description: "Install via Homebrew",
			Command:     "brew install terraform",
			Sudo:        false,
			Platform:    PlatformDarwin,
		},
		PlatformLinux: {
			Description: "Install via apt (HashiCorp repository)",
			Command:     "wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg && echo \"deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main\" | sudo tee /etc/apt/sources.list.d/hashicorp.list && sudo apt update && sudo apt install terraform",
			Sudo:        true,
			Platform:    PlatformLinux,
		},
	},
	IDMultipass: {
		PlatformDarwin: {
			Description: "Install via Homebrew",
			Command:     "brew install --cask multipass",
			Sudo:        false,
			Platform:    PlatformDarwin,
		},
		PlatformLinux: {
			Description: "Install via snap",
			Command:     "sudo snap install multipass",
			Sudo:        true,
			Platform:    PlatformLinux,
		},
	},
	IDXorriso: {
		PlatformDarwin: {
			Description: "Install via Homebrew",
			Command:     "brew install xorriso",
			Sudo:        false,
			Platform:    PlatformDarwin,
		},
		PlatformLinux: {
			Description: "Install via apt",
			Command:     "sudo apt install -y xorriso",
			Sudo:        true,
			Platform:    PlatformLinux,
		},
	},
	IDLibvirt: {
		PlatformLinux: {
			Description: "Install libvirt and start service",
			Command:     "sudo apt install -y qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils && sudo systemctl enable --now libvirtd && sudo usermod -aG libvirt $USER",
			Sudo:        true,
			Platform:    PlatformLinux,
		},
	},
	IDVirsh: {
		PlatformLinux: {
			Description: "Install libvirt clients",
			Command:     "sudo apt install -y libvirt-clients",
			Sudo:        true,
			Platform:    PlatformLinux,
		},
	},
	IDQemuKVM: {
		PlatformLinux: {
			Description: "Install QEMU/KVM",
			Command:     "sudo apt install -y qemu-kvm qemu-utils",
			Sudo:        true,
			Platform:    PlatformLinux,
		},
	},
	IDGhostty: {
		PlatformDarwin: {
			Description: "Install via Homebrew",
			Command:     "brew install --cask ghostty",
			Sudo:        false,
			Platform:    PlatformDarwin,
		},
		PlatformLinux: {
			Description: "Install via snap (recommended)",
			Command:     "sudo snap install ghostty --classic",
			Sudo:        true,
			Platform:    PlatformLinux,
		},
	},
}

// GetFixCommand returns the fix command for a tool on the given platform.
func GetFixCommand(toolID, platform string) *FixCommand {
	toolFixes, ok := fixCommands[toolID]
	if !ok {
		return nil
	}

	fix, ok := toolFixes[platform]
	if !ok {
		return nil
	}

	return fix
}

// Fixer provides functionality to run fix commands.
type Fixer struct {
	executor CommandExecutor
}

// NewFixer creates a new Fixer.
func NewFixer() *Fixer {
	return &Fixer{
		executor: &RealExecutor{},
	}
}

// NewFixerWithExecutor creates a new Fixer with a custom executor.
func NewFixerWithExecutor(exec CommandExecutor) *Fixer {
	return &Fixer{
		executor: exec,
	}
}

// RunFix executes a fix command.
func (f *Fixer) RunFix(fix *FixCommand) error {
	if fix == nil {
		return fmt.Errorf("no fix command available")
	}

	// Run the command through shell using the executor
	output, err := f.executor.CombinedOutput("sh", "-c", fix.Command)
	if err != nil {
		return fmt.Errorf("fix failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// CopyToClipboard copies the fix command to clipboard.
func (f *Fixer) CopyToClipboard(fix *FixCommand) error {
	if fix == nil {
		return fmt.Errorf("no fix command available")
	}

	return copyToClipboard(fix.Command)
}

// copyToClipboard copies text to the system clipboard.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, fall back to xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip or xsel)")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start clipboard command: %w", err)
	}

	if _, err := stdin.Write([]byte(text)); err != nil {
		return fmt.Errorf("failed to write to clipboard: %w", err)
	}

	if err := stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("clipboard command failed: %w", err)
	}

	return nil
}
