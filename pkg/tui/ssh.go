package tui

import (
	"os"
	"path/filepath"
	"strings"
)

// SSHKeyInfo holds information about an SSH key.
type SSHKeyInfo struct {
	Path        string // Full path: ~/.ssh/id_ed25519.pub
	Type        string // Key type: ed25519, rsa, ecdsa
	Content     string // Full key content
	Fingerprint string // Short display: ssh-ed25519 AAAA...xyz
}

// getLocalSSHKeys returns all available SSH public keys from ~/.ssh/
func getLocalSSHKeys() []SSHKeyInfo {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	keyFiles := []struct {
		name    string
		keyType string
	}{
		{"id_ed25519.pub", "ed25519"},
		{"id_rsa.pub", "rsa"},
		{"id_ecdsa.pub", "ecdsa"},
	}

	var keys []SSHKeyInfo
	for _, kf := range keyFiles {
		path := filepath.Join(home, ".ssh", kf.name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}

		// Create fingerprint (truncated display)
		fingerprint := content
		if len(content) > 50 {
			fingerprint = content[:47] + "..."
		}

		keys = append(keys, SSHKeyInfo{
			Path:        path,
			Type:        kf.keyType,
			Content:     content,
			Fingerprint: fingerprint,
		})
	}

	return keys
}

// getDefaultSSHKey tries to read the user's default SSH public key.
func getDefaultSSHKey() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Try common key types
	keyFiles := []string{
		home + "/.ssh/id_ed25519.pub",
		home + "/.ssh/id_rsa.pub",
		home + "/.ssh/id_ecdsa.pub",
	}

	for _, keyFile := range keyFiles {
		data, err := os.ReadFile(keyFile)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	return ""
}
