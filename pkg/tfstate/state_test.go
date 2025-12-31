package tfstate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	m := NewManager("/test/terraform")

	assert.NotNil(t, m)
	assert.Equal(t, "/test/terraform", m.workDir)
	assert.Equal(t, "qemu:///system", m.libvirtURI)
}

func TestManager_SetLibvirtURI(t *testing.T) {
	m := NewManager("/test")

	m.SetLibvirtURI("qemu+ssh://user@host/system")
	assert.Equal(t, "qemu+ssh://user@host/system", m.libvirtURI)
}

func TestManager_SetVerbose(t *testing.T) {
	m := NewManager("/test")
	assert.False(t, m.verbose)

	m.SetVerbose(true)
	assert.True(t, m.verbose)
}

func TestManager_WorkDir(t *testing.T) {
	m := NewManager("/path/to/terraform")
	assert.Equal(t, "/path/to/terraform", m.WorkDir())
}

func TestManager_ConsoleCommand(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		vmName   string
		expected string
	}{
		{
			name:     "default URI",
			uri:      "qemu:///system",
			vmName:   "test-vm",
			expected: "virsh console test-vm",
		},
		{
			name:     "empty URI",
			uri:      "",
			vmName:   "my-vm",
			expected: "virsh console my-vm",
		},
		{
			name:     "remote URI",
			uri:      "qemu+ssh://root@server/system",
			vmName:   "remote-vm",
			expected: "virsh -c qemu+ssh://root@server/system console remote-vm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager("/test")
			m.SetLibvirtURI(tt.uri)

			result := m.ConsoleCommand(tt.vmName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManager_SSHCommand(t *testing.T) {
	m := NewManager("/test")

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "valid IP",
			ip:       "192.168.122.10",
			expected: "ssh ubuntu@192.168.122.10",
		},
		{
			name:     "empty IP",
			ip:       "",
			expected: "",
		},
		{
			name:     "pending IP",
			ip:       "pending",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.SSHCommand(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManager_IsInitialized(t *testing.T) {
	t.Run("not initialized", func(t *testing.T) {
		tmpDir := t.TempDir()
		m := NewManager(tmpDir)

		assert.False(t, m.IsInitialized())
	})

	t.Run("initialized", func(t *testing.T) {
		tmpDir := t.TempDir()
		tfDir := filepath.Join(tmpDir, ".terraform")
		err := os.MkdirAll(tfDir, 0755)
		require.NoError(t, err)

		m := NewManager(tmpDir)
		assert.True(t, m.IsInitialized())
	})
}

func TestManager_HasState(t *testing.T) {
	t.Run("no state file", func(t *testing.T) {
		tmpDir := t.TempDir()
		m := NewManager(tmpDir)

		assert.False(t, m.HasState())
	})

	t.Run("empty state file", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, "terraform.tfstate")
		err := os.WriteFile(statePath, []byte{}, 0644)
		require.NoError(t, err)

		m := NewManager(tmpDir)
		assert.False(t, m.HasState())
	})

	t.Run("non-empty state file", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, "terraform.tfstate")
		err := os.WriteFile(statePath, []byte(`{"version": 4}`), 0644)
		require.NoError(t, err)

		m := NewManager(tmpDir)
		assert.True(t, m.HasState())
	})
}

func TestParseOutputJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected map[string]string
	}{
		{
			name: "string outputs",
			json: `{
				"vm_name": {"value": "test-vm", "type": "string"},
				"vm_ip": {"value": "192.168.122.10", "type": "string"}
			}`,
			expected: map[string]string{
				"vm_name": "test-vm",
				"vm_ip":   "192.168.122.10",
			},
		},
		{
			name: "array output",
			json: `{
				"vm_ip": {"value": ["192.168.122.10", "192.168.122.11"], "type": ["list", "string"]}
			}`,
			expected: map[string]string{
				"vm_ip": "192.168.122.10",
			},
		},
		{
			name: "number output",
			json: `{
				"vcpu_count": {"value": 4, "type": "number"}
			}`,
			expected: map[string]string{
				"vcpu_count": "4",
			},
		},
		{
			name: "boolean output",
			json: `{
				"autostart": {"value": true, "type": "bool"}
			}`,
			expected: map[string]string{
				"autostart": "true",
			},
		},
		{
			name:     "empty outputs",
			json:     `{}`,
			expected: map[string]string{},
		},
		{
			name:     "invalid json",
			json:     `not valid json`,
			expected: map[string]string{},
		},
		{
			name: "empty array",
			json: `{
				"ips": {"value": [], "type": ["list", "string"]}
			}`,
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOutputJSON([]byte(tt.json))
			for k, v := range tt.expected {
				assert.Equal(t, v, result[k], "key %s should match", k)
			}
		})
	}
}

func TestVMStatus_String(t *testing.T) {
	// Test that status constants have expected values
	assert.Equal(t, VMStatus("running"), StatusRunning)
	assert.Equal(t, VMStatus("stopped"), StatusStopped)
	assert.Equal(t, VMStatus("paused"), StatusPaused)
	assert.Equal(t, VMStatus("shutoff"), StatusShutoff)
	assert.Equal(t, VMStatus("crashed"), StatusCrashed)
	assert.Equal(t, VMStatus("unknown"), StatusUnknown)
	assert.Equal(t, VMStatus("not-found"), StatusNotFound)
}

func TestVMInfo_Fields(t *testing.T) {
	vm := VMInfo{
		Name:      "test-vm",
		Status:    StatusRunning,
		IP:        "192.168.122.10",
		CPUs:      4,
		MemoryMB:  8192,
		DiskGB:    40,
		Autostart: true,
	}

	assert.Equal(t, "test-vm", vm.Name)
	assert.Equal(t, StatusRunning, vm.Status)
	assert.Equal(t, "192.168.122.10", vm.IP)
	assert.Equal(t, 4, vm.CPUs)
	assert.Equal(t, 8192, vm.MemoryMB)
	assert.Equal(t, 40, vm.DiskGB)
	assert.True(t, vm.Autostart)
}

func TestParseOutputJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected map[string]string
	}{
		{
			name: "null value",
			json: `{
				"optional_field": {"value": null, "type": "string"}
			}`,
			expected: map[string]string{
				"optional_field": "<nil>",
			},
		},
		{
			name: "nested object",
			json: `{
				"metadata": {"value": {"key": "value", "nested": {"deep": "data"}}, "type": "object"}
			}`,
			expected: map[string]string{
				"metadata": "map[key:value nested:map[deep:data]]",
			},
		},
		{
			name: "float number",
			json: `{
				"disk_size": {"value": 20.5, "type": "number"}
			}`,
			expected: map[string]string{
				"disk_size": "20.5",
			},
		},
		{
			name: "integer as float",
			json: `{
				"memory": {"value": 4096.0, "type": "number"}
			}`,
			expected: map[string]string{
				"memory": "4096",
			},
		},
		{
			name: "false boolean",
			json: `{
				"autostart": {"value": false, "type": "bool"}
			}`,
			expected: map[string]string{
				"autostart": "false",
			},
		},
		{
			name: "array with non-string first element",
			json: `{
				"ports": {"value": [8080, 443], "type": ["list", "number"]}
			}`,
			expected: map[string]string{},
		},
		{
			name: "mixed outputs",
			json: `{
				"vm_name": {"value": "prod-server", "type": "string"},
				"vcpu": {"value": 8, "type": "number"},
				"autostart": {"value": true, "type": "bool"},
				"ips": {"value": ["10.0.0.1", "10.0.0.2"], "type": ["list", "string"]}
			}`,
			expected: map[string]string{
				"vm_name":   "prod-server",
				"vcpu":      "8",
				"autostart": "true",
				"ips":       "10.0.0.1",
			},
		},
		{
			name:     "empty byte array",
			json:     "",
			expected: map[string]string{},
		},
		{
			name:     "whitespace only",
			json:     "   \n\t  ",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOutputJSON([]byte(tt.json))
			for k, v := range tt.expected {
				assert.Equal(t, v, result[k], "key %s should match", k)
			}
		})
	}
}

func TestManager_SSHCommand_EdgeCases(t *testing.T) {
	m := NewManager("/test")

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "IPv6 address",
			ip:       "fe80::1",
			expected: "ssh ubuntu@fe80::1",
		},
		{
			name:     "hostname instead of IP",
			ip:       "my-server.local",
			expected: "ssh ubuntu@my-server.local",
		},
		{
			name:     "IP with whitespace",
			ip:       "  192.168.1.1  ",
			expected: "ssh ubuntu@  192.168.1.1  ",
		},
		{
			name:     "PENDING uppercase",
			ip:       "PENDING",
			expected: "ssh ubuntu@PENDING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.SSHCommand(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManager_ConsoleCommand_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		vmName   string
		expected string
	}{
		{
			name:     "VM name with special characters",
			uri:      "qemu:///system",
			vmName:   "test-vm-123",
			expected: "virsh console test-vm-123",
		},
		{
			name:     "remote URI with port",
			uri:      "qemu+ssh://user@192.168.1.100:22/system",
			vmName:   "remote-vm",
			expected: "virsh -c qemu+ssh://user@192.168.1.100:22/system console remote-vm",
		},
		{
			name:     "local session URI",
			uri:      "qemu:///session",
			vmName:   "session-vm",
			expected: "virsh -c qemu:///session console session-vm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager("/test")
			m.SetLibvirtURI(tt.uri)

			result := m.ConsoleCommand(tt.vmName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManager_HasState_EdgeCases(t *testing.T) {
	t.Run("state file is a directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, "terraform.tfstate")
		err := os.MkdirAll(statePath, 0755)
		require.NoError(t, err)

		m := NewManager(tmpDir)
		// Directories have size > 0, so this returns true
		// This is an edge case that could be considered a bug
		assert.True(t, m.HasState())
	})

	t.Run("state file with only whitespace", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, "terraform.tfstate")
		err := os.WriteFile(statePath, []byte("   \n"), 0644)
		require.NoError(t, err)

		m := NewManager(tmpDir)
		assert.True(t, m.HasState()) // Has content (whitespace counts)
	})
}

func TestManager_IsInitialized_EdgeCases(t *testing.T) {
	t.Run(".terraform is a file not directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		tfPath := filepath.Join(tmpDir, ".terraform")
		err := os.WriteFile(tfPath, []byte("not a directory"), 0644)
		require.NoError(t, err)

		m := NewManager(tmpDir)
		// os.Stat succeeds for files too
		assert.True(t, m.IsInitialized())
	})
}

func TestVMInfo_ZeroValues(t *testing.T) {
	vm := VMInfo{}

	assert.Equal(t, "", vm.Name)
	assert.Equal(t, VMStatus(""), vm.Status)
	assert.Equal(t, "", vm.IP)
	assert.Equal(t, 0, vm.CPUs)
	assert.Equal(t, 0, vm.MemoryMB)
	assert.Equal(t, 0, vm.DiskGB)
	assert.False(t, vm.Autostart)
	assert.True(t, vm.CreatedAt.IsZero())
}

func TestVMStatus_Comparison(t *testing.T) {
	// Test that status can be compared
	status := StatusRunning
	assert.True(t, status == StatusRunning)
	assert.False(t, status == StatusStopped)

	// Test that status can be used as map key
	statusMap := map[VMStatus]string{
		StatusRunning: "green",
		StatusStopped: "yellow",
		StatusCrashed: "red",
	}
	assert.Equal(t, "green", statusMap[StatusRunning])
}

func TestNewManager_EmptyPath(t *testing.T) {
	m := NewManager("")
	assert.Equal(t, "", m.workDir)
	assert.Equal(t, "qemu:///system", m.libvirtURI)
}
