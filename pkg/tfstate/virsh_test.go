package tfstate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVirshClient(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "empty URI uses default",
			uri:      "",
			expected: "qemu:///system",
		},
		{
			name:     "custom URI",
			uri:      "qemu+ssh://user@host/system",
			expected: "qemu+ssh://user@host/system",
		},
		{
			name:     "local system URI",
			uri:      "qemu:///system",
			expected: "qemu:///system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewVirshClient(tt.uri)
			assert.Equal(t, tt.expected, c.uri)
		})
	}
}

func TestVirshClient_Console(t *testing.T) {
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
			name:     "remote URI",
			uri:      "qemu+ssh://root@192.168.1.100/system",
			vmName:   "remote-vm",
			expected: "virsh -c qemu+ssh://root@192.168.1.100/system console remote-vm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewVirshClient(tt.uri)
			result := c.Console(tt.vmName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVirshClient_baseArgs(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		args     []string
		expected []string
	}{
		{
			name:     "with URI",
			uri:      "qemu:///system",
			args:     []string{"list", "--all"},
			expected: []string{"-c", "qemu:///system", "list", "--all"},
		},
		{
			name:     "empty URI",
			uri:      "",
			args:     []string{"start", "my-vm"},
			expected: []string{"start", "my-vm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &VirshClient{uri: tt.uri}
			result := c.baseArgs(tt.args...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVirshList(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []VirshVM
	}{
		{
			name: "running and stopped VMs",
			output: ` Id   Name             State
------------------------------------
 1    running-vm       running
 -    stopped-vm       shut off
 2    another-vm       running
`,
			expected: []VirshVM{
				{ID: 1, Name: "running-vm", Status: StatusRunning},
				{ID: -1, Name: "stopped-vm", Status: StatusShutoff},
				{ID: 2, Name: "another-vm", Status: StatusRunning},
			},
		},
		{
			name: "paused VM",
			output: ` Id   Name             State
------------------------------------
 1    paused-vm        paused
`,
			expected: []VirshVM{
				{ID: 1, Name: "paused-vm", Status: StatusPaused},
			},
		},
		{
			name:     "empty list",
			output:   ` Id   Name             State\n------------------------------------\n`,
			expected: nil,
		},
		{
			name:     "no VMs",
			output:   "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVirshList(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVMStatus(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected VMStatus
	}{
		{"running", "running", StatusRunning},
		{"running uppercase", "Running", StatusRunning},
		{"shut off", "shut off", StatusShutoff},
		{"shutoff", "shutoff", StatusShutoff},
		{"paused", "paused", StatusPaused},
		{"crashed", "crashed", StatusCrashed},
		{"idle", "idle", StatusRunning},
		{"unknown state", "something-else", StatusUnknown},
		{"empty", "", StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVMStatus(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseKeyValue(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected map[string]string
	}{
		{
			name: "dominfo output",
			output: `Id:             1
Name:           my-vm
UUID:           abc123
OS Type:        hvm
State:          running
CPU(s):         2
Max memory:     4194304 KiB
Used memory:    4194304 KiB
Persistent:     yes
Autostart:      enable
`,
			expected: map[string]string{
				"Id":          "1",
				"Name":        "my-vm",
				"UUID":        "abc123",
				"OS Type":     "hvm",
				"State":       "running",
				"CPU(s)":      "2",
				"Max memory":  "4194304 KiB",
				"Used memory": "4194304 KiB",
				"Persistent":  "yes",
				"Autostart":   "enable",
			},
		},
		{
			name:     "empty output",
			output:   "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKeyValue(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseIPFromDomifaddr(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name: "valid output",
			output: ` Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------
 vnet0      52:54:00:12:34:56    ipv4         192.168.122.50/24
`,
			expected: "192.168.122.50",
		},
		{
			name: "multiple interfaces",
			output: ` Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------
 vnet0      52:54:00:12:34:56    ipv4         192.168.122.50/24
 vnet1      52:54:00:12:34:57    ipv4         10.0.0.5/8
`,
			expected: "192.168.122.50",
		},
		{
			name:     "no IPv4",
			output:   ` Name       MAC address          Protocol     Address\n`,
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIPFromDomifaddr(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseMACFromDomiflist(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name: "valid output",
			output: ` Interface  Type       Source     Model       MAC
-------------------------------------------------------
 vnet0      network    default    virtio      52:54:00:AB:CD:EF
`,
			expected: "52:54:00:ab:cd:ef",
		},
		{
			name:     "no MAC address",
			output:   "some output without MAC",
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMACFromDomiflist(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseIPFromLeases(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		mac      string
		expected string
	}{
		{
			name: "matching MAC",
			output: ` Expiry Time           MAC address        Protocol  IP address                Hostname        Client ID or DUID
-------------------------------------------------------------------------------------------------------------------
 2024-01-01 12:00:00   52:54:00:ab:cd:ef  ipv4      192.168.122.100/24        my-vm           -
 2024-01-01 12:00:00   52:54:00:11:22:33  ipv4      192.168.122.101/24        other-vm        -
`,
			mac:      "52:54:00:ab:cd:ef",
			expected: "192.168.122.100",
		},
		{
			name: "MAC not found",
			output: ` Expiry Time           MAC address        Protocol  IP address                Hostname        Client ID or DUID
-------------------------------------------------------------------------------------------------------------------
 2024-01-01 12:00:00   52:54:00:11:22:33  ipv4      192.168.122.101/24        other-vm        -
`,
			mac:      "52:54:00:ab:cd:ef",
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			mac:      "52:54:00:ab:cd:ef",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIPFromLeases(tt.output, tt.mac)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVirshVM_Fields(t *testing.T) {
	vm := VirshVM{
		ID:     1,
		Name:   "test-vm",
		Status: StatusRunning,
	}

	assert.Equal(t, 1, vm.ID)
	assert.Equal(t, "test-vm", vm.Name)
	assert.Equal(t, StatusRunning, vm.Status)
}

func TestVirshVM_ShutoffID(t *testing.T) {
	vm := VirshVM{
		ID:     -1,
		Name:   "stopped-vm",
		Status: StatusShutoff,
	}

	assert.Equal(t, -1, vm.ID)
	assert.Equal(t, StatusShutoff, vm.Status)
}

func TestParseVirshList_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []VirshVM
	}{
		{
			name: "extra whitespace",
			output: `  Id   Name             State
------------------------------------
  1    my-vm            running
`,
			expected: []VirshVM{
				{ID: 1, Name: "my-vm", Status: StatusRunning},
			},
		},
		{
			name: "VM name with numbers",
			output: ` Id   Name             State
------------------------------------
 1    vm-123-test      running
`,
			expected: []VirshVM{
				{ID: 1, Name: "vm-123-test", Status: StatusRunning},
			},
		},
		{
			name: "crashed VM",
			output: ` Id   Name             State
------------------------------------
 -    crashed-vm       crashed
`,
			expected: []VirshVM{
				{ID: -1, Name: "crashed-vm", Status: StatusCrashed},
			},
		},
		{
			name: "only header",
			output: ` Id   Name             State
------------------------------------
`,
			expected: nil,
		},
		{
			name: "malformed line (too few fields)",
			output: ` Id   Name             State
------------------------------------
 1
`,
			expected: nil,
		},
		{
			name: "high ID number",
			output: ` Id   Name             State
------------------------------------
 999  large-id-vm      running
`,
			expected: []VirshVM{
				{ID: 999, Name: "large-id-vm", Status: StatusRunning},
			},
		},
		{
			name: "mixed valid and truly malformed lines",
			output: ` Id   Name             State
------------------------------------
 1    valid-vm         running
 only-two-fields
 2    another-vm       shut off
`,
			expected: []VirshVM{
				{ID: 1, Name: "valid-vm", Status: StatusRunning},
				{ID: 2, Name: "another-vm", Status: StatusShutoff},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVirshList(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVMStatus_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected VMStatus
	}{
		{"RUNNING uppercase", "RUNNING", StatusRunning},
		{"SHUT OFF uppercase", "SHUT OFF", StatusShutoff},
		{"PAUSED uppercase", "PAUSED", StatusPaused},
		{"mixed case", "RuNnInG", StatusRunning},
		{"with extra spaces", "  running  ", StatusUnknown}, // TrimSpace not applied here
		{"pmsuspended", "pmsuspended", StatusUnknown},
		{"in shutdown", "in shutdown", StatusUnknown},
		{"blocked", "blocked", StatusUnknown},
		{"dying", "dying", StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVMStatus(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseKeyValue_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected map[string]string
	}{
		{
			name:   "value with colon",
			output: "URL:           http://example.com:8080/path",
			expected: map[string]string{
				"URL": "http://example.com:8080/path",
			},
		},
		{
			name:   "empty value",
			output: "Empty:         ",
			expected: map[string]string{
				"Empty": "",
			},
		},
		{
			name:   "no colon",
			output: "This is just a line without colon",
			expected: map[string]string{},
		},
		{
			name: "multiple colons in value",
			output: `Time: 12:30:45`,
			expected: map[string]string{
				"Time": "12:30:45",
			},
		},
		{
			name:   "whitespace only",
			output: "   \n   \n   ",
			expected: map[string]string{},
		},
		{
			name: "key with special characters",
			output: `CPU(s):         4
Memory (MB):    2048`,
			expected: map[string]string{
				"CPU(s)":      "4",
				"Memory (MB)": "2048",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKeyValue(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseIPFromDomifaddr_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name: "IPv6 address (should be ignored)",
			output: ` Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------
 vnet0      52:54:00:12:34:56    ipv6         fe80::1/64
`,
			expected: "",
		},
		{
			name: "both IPv4 and IPv6",
			output: ` Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------
 vnet0      52:54:00:12:34:56    ipv6         fe80::1/64
 vnet0      52:54:00:12:34:56    ipv4         192.168.122.50/24
`,
			expected: "192.168.122.50",
		},
		{
			name: "IP with /32 subnet",
			output: ` Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------
 vnet0      52:54:00:12:34:56    ipv4         10.0.0.1/32
`,
			expected: "10.0.0.1",
		},
		{
			name: "IP without subnet (no slash)",
			output: ` Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------
 vnet0      52:54:00:12:34:56    ipv4         192.168.1.1
`,
			expected: "",
		},
		{
			name:     "header only",
			output:   " Name       MAC address          Protocol     Address\n-------------------------------------------------------------------------------\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIPFromDomifaddr(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseMACFromDomiflist_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name: "lowercase MAC",
			output: ` Interface  Type       Source     Model       MAC
-------------------------------------------------------
 vnet0      network    default    virtio      52:54:00:ab:cd:ef
`,
			expected: "52:54:00:ab:cd:ef",
		},
		{
			name: "multiple interfaces",
			output: ` Interface  Type       Source     Model       MAC
-------------------------------------------------------
 vnet0      network    default    virtio      52:54:00:11:22:33
 vnet1      network    internal   virtio      52:54:00:44:55:66
`,
			expected: "52:54:00:11:22:33", // Returns first match
		},
		{
			name:     "partial MAC (invalid)",
			output:   "52:54:00:12:34",
			expected: "",
		},
		{
			name:     "MAC-like but wrong format",
			output:   "52-54-00-AB-CD-EF",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMACFromDomiflist(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseIPFromLeases_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		mac      string
		expected string
	}{
		{
			name: "uppercase MAC in output",
			output: ` Expiry Time           MAC address        Protocol  IP address                Hostname
--------------------------------------------------------------------------------------------
 2024-01-01 12:00:00   52:54:00:AB:CD:EF  ipv4      192.168.122.100/24        my-vm
`,
			mac:      "52:54:00:ab:cd:ef",
			expected: "192.168.122.100",
		},
		{
			name: "uppercase MAC in query",
			output: ` Expiry Time           MAC address        Protocol  IP address                Hostname
--------------------------------------------------------------------------------------------
 2024-01-01 12:00:00   52:54:00:ab:cd:ef  ipv4      192.168.122.100/24        my-vm
`,
			mac:      "52:54:00:AB:CD:EF",
			expected: "192.168.122.100",
		},
		{
			name: "multiple leases same MAC",
			output: ` Expiry Time           MAC address        Protocol  IP address                Hostname
--------------------------------------------------------------------------------------------
 2024-01-01 12:00:00   52:54:00:ab:cd:ef  ipv4      192.168.122.100/24        my-vm
 2024-01-02 12:00:00   52:54:00:ab:cd:ef  ipv4      192.168.122.101/24        my-vm
`,
			mac:      "52:54:00:ab:cd:ef",
			expected: "192.168.122.100", // Returns first match
		},
		{
			name:     "only header",
			output:   " Expiry Time           MAC address        Protocol  IP address                Hostname\n--------------------------------------------------------------------------------------------\n",
			mac:      "52:54:00:ab:cd:ef",
			expected: "",
		},
		{
			name: "IPv6 in leases",
			output: ` Expiry Time           MAC address        Protocol  IP address                Hostname
--------------------------------------------------------------------------------------------
 2024-01-01 12:00:00   52:54:00:ab:cd:ef  ipv6      fe80::1/64                my-vm
`,
			mac:      "52:54:00:ab:cd:ef",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIPFromLeases(tt.output, tt.mac)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVirshClient_Console_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		vmName   string
		expected string
	}{
		{
			name:     "empty VM name",
			uri:      "qemu:///system",
			vmName:   "",
			expected: "virsh console ",
		},
		{
			name:     "VM name with spaces",
			uri:      "qemu:///system",
			vmName:   "my vm",
			expected: "virsh console my vm",
		},
		{
			name:     "session URI",
			uri:      "qemu:///session",
			vmName:   "user-vm",
			expected: "virsh -c qemu:///session console user-vm",
		},
		{
			name:     "TLS URI",
			uri:      "qemu+tls://hypervisor.example.com/system",
			vmName:   "secure-vm",
			expected: "virsh -c qemu+tls://hypervisor.example.com/system console secure-vm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewVirshClient(tt.uri)
			result := c.Console(tt.vmName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVirshClient_baseArgs_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		args     []string
		expected []string
	}{
		{
			name:     "no args",
			uri:      "qemu:///system",
			args:     []string{},
			expected: []string{"-c", "qemu:///system"},
		},
		{
			name:     "single arg",
			uri:      "",
			args:     []string{"version"},
			expected: []string{"version"},
		},
		{
			name:     "many args",
			uri:      "qemu:///system",
			args:     []string{"dominfo", "--domain", "my-vm"},
			expected: []string{"-c", "qemu:///system", "dominfo", "--domain", "my-vm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &VirshClient{uri: tt.uri}
			result := c.baseArgs(tt.args...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVirshVM_ZeroValue(t *testing.T) {
	vm := VirshVM{}

	assert.Equal(t, 0, vm.ID)
	assert.Equal(t, "", vm.Name)
	assert.Equal(t, VMStatus(""), vm.Status)
}
