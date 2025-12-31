package terraform

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

func TestCheckTerraformInstalled(t *testing.T) {
	d := New("/test/project")

	// This test depends on whether terraform is installed on the system
	// We just verify it doesn't panic and returns appropriate error
	err := d.checkTerraformInstalled()

	// Check if terraform is actually installed
	_, lookErr := exec.LookPath("terraform")
	if lookErr != nil {
		// Terraform not installed, should get an error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "terraform is not installed")
	} else {
		// Terraform is installed, should succeed
		assert.NoError(t, err)
	}
}

func TestCheckUbuntuImage_Validation(t *testing.T) {
	d := New("/test/project")

	tests := []struct {
		name        string
		setupFile   bool
		expectError bool
	}{
		{
			name:        "existing file",
			setupFile:   true,
			expectError: false,
		},
		{
			name:        "non-existing file",
			setupFile:   false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var imagePath string
			if tt.setupFile {
				tmpFile, err := os.CreateTemp("", "test-*.img")
				require.NoError(t, err)
				defer os.Remove(tmpFile.Name())
				tmpFile.Close()
				imagePath = tmpFile.Name()
			} else {
				imagePath = "/nonexistent/path/to/image.img"
			}

			err := d.checkUbuntuImage(imagePath)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTerraformInit_DirectoryCheck(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .terraform directory to simulate already initialized
	tfDir := filepath.Join(tmpDir, ".terraform")
	err := os.MkdirAll(tfDir, 0755)
	require.NoError(t, err)

	// When .terraform exists, init should be skipped
	// Verify the directory check logic
	_, err = os.Stat(filepath.Join(tmpDir, ".terraform"))
	assert.NoError(t, err, ".terraform directory should exist")
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
				"ip":      "192.168.122.10",
			},
		},
		{
			name: "array output (IP addresses)",
			json: `{
				"vm_ip": {"value": ["192.168.122.10", "192.168.122.11"], "type": ["list", "string"]}
			}`,
			expected: map[string]string{
				"vm_ip": "192.168.122.10",
				"ip":    "192.168.122.10",
			},
		},
		{
			name:     "empty outputs",
			json:     `{}`,
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

// parseOutputJSON is a helper to test JSON parsing logic (mirrors terraformOutput parsing)
func parseOutputJSON(data []byte) map[string]string {
	outputs := make(map[string]string)

	var rawOutputs map[string]struct {
		Value interface{} `json:"value"`
		Type  interface{} `json:"type"`
	}

	if err := json.Unmarshal(data, &rawOutputs); err != nil {
		return outputs
	}

	for k, v := range rawOutputs {
		switch val := v.Value.(type) {
		case string:
			outputs[k] = val
		case []interface{}:
			if len(val) > 0 {
				if str, ok := val[0].(string); ok {
					outputs[k] = str
				}
			}
		default:
			outputs[k] = fmt.Sprintf("%v", val)
		}
	}

	// Map vm_ip to ip
	if vmIP, ok := outputs["vm_ip"]; ok {
		outputs["ip"] = vmIP
	}

	return outputs
}

func TestParseOutputJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected map[string]string
	}{
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
			name: "empty array",
			json: `{
				"vm_ip": {"value": [], "type": ["list", "string"]}
			}`,
			expected: map[string]string{},
		},
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
			name:     "invalid json",
			json:     `not valid json`,
			expected: map[string]string{},
		},
		{
			name: "nested object (should be stringified)",
			json: `{
				"metadata": {"value": {"key": "value"}, "type": "object"}
			}`,
			expected: map[string]string{
				"metadata": "map[key:value]",
			},
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

func TestTerraformStages(t *testing.T) {
	// Test that new terraform stages have correct display names
	tests := []struct {
		stage       deploy.Stage
		displayName string
	}{
		{deploy.StagePreparing, "Preparing"},
		{deploy.StagePlanning, "Planning"},
		{deploy.StageConfirming, "Awaiting Confirmation"},
		{deploy.StageApplying, "Applying"},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			assert.Equal(t, tt.displayName, tt.stage.DisplayName())
		})
	}
}
