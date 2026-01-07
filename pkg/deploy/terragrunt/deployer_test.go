package terragrunt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

func TestNew(t *testing.T) {
	g := New("/test/project")
	assert.NotNil(t, g)
	assert.Equal(t, "/test/project", g.projectRoot)
}

func TestGenerator_Name(t *testing.T) {
	g := New("/test/project")
	assert.Equal(t, "Terragrunt/libvirt", g.Name())
}

func TestGenerator_Target(t *testing.T) {
	g := New("/test/project")
	assert.Equal(t, deploy.TargetTerragrunt, g.Target())
}

func TestGenerator_Validate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create terragrunt modules directory structure
	modulesDir := filepath.Join(tmpDir, "terragrunt", "modules", "libvirt-vm")
	err := os.MkdirAll(modulesDir, 0755)
	require.NoError(t, err)

	tests := []struct {
		name        string
		opts        *deploy.DeployOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing project root",
			opts: &deploy.DeployOptions{
				ProjectRoot: "",
				Config:      nil,
			},
			expectError: true,
			errorMsg:    "project root is required",
		},
		{
			name: "missing config",
			opts: &deploy.DeployOptions{
				ProjectRoot: tmpDir,
				Config:      nil,
			},
			expectError: true,
			errorMsg:    "configuration is required",
		},
		{
			name: "terragrunt module not found",
			opts: &deploy.DeployOptions{
				ProjectRoot: "/nonexistent/path",
				Config:      &config.FullConfig{},
			},
			expectError: true,
			errorMsg:    "terragrunt module not found",
		},
		{
			name: "valid options",
			opts: &deploy.DeployOptions{
				ProjectRoot: tmpDir,
				Config:      &config.FullConfig{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New(tt.opts.ProjectRoot)
			err := g.Validate(tt.opts)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerator_GenerateVMName(t *testing.T) {
	g := New("/test/project")

	// Generate multiple names
	names := make(map[string]bool)
	for i := 0; i < 10; i++ {
		name := g.generateVMName()

		// Names should start with vm-
		assert.True(t, strings.HasPrefix(name, "vm-"), "Name should start with vm-: %s", name)

		// Names should have random suffix (format: vm-MMDD-HHMM-XXXX)
		parts := strings.Split(name, "-")
		assert.GreaterOrEqual(t, len(parts), 4, "Name should have at least 4 parts: %s", name)

		// Track for uniqueness (with random suffix, collisions should be rare)
		names[name] = true
	}

	// With random suffix, we should have unique names
	assert.GreaterOrEqual(t, len(names), 5, "Should have mostly unique names with random suffix")
}

func TestInstallInstructions(t *testing.T) {
	instructions := InstallInstructions()

	assert.Contains(t, instructions, "Terragrunt")
	assert.Contains(t, instructions, "OpenTofu")
	assert.Contains(t, instructions, "libvirt")
	assert.Contains(t, instructions, "Linux")
}

func TestGenerator_Fail(t *testing.T) {
	g := New("/test/project")
	result := &deploy.DeployResult{
		Target:  deploy.TargetTerragrunt,
		Outputs: make(map[string]string),
	}
	testErr := fmt.Errorf("test error")
	start := time.Now()

	failedResult := g.fail(result, testErr, start)

	assert.False(t, failedResult.Success)
	assert.Equal(t, testErr, failedResult.Error)
	assert.True(t, failedResult.Duration >= 0)
}

func TestGenerator_Cleanup(t *testing.T) {
	g := New("/test/project")

	// Cleanup is a no-op for the generator
	err := g.Cleanup(nil, nil)
	assert.NoError(t, err)
}

func TestGenerator_ConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	g := New(tmpDir)

	t.Run("non-existent directory", func(t *testing.T) {
		exists, err := g.configExists(filepath.Join(tmpDir, "nonexistent"))
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		err := os.MkdirAll(emptyDir, 0755)
		require.NoError(t, err)

		exists, err := g.configExists(emptyDir)
		assert.NoError(t, err)
		assert.False(t, exists) // Empty dir is considered available
	})

	t.Run("directory with terragrunt.hcl", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, "existing-config")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Create terragrunt.hcl
		err = os.WriteFile(filepath.Join(configDir, "terragrunt.hcl"), []byte("# config"), 0644)
		require.NoError(t, err)

		exists, err := g.configExists(configDir)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("directory with cloud-init.yaml only", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, "partial-config")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Create only cloud-init.yaml (partial failure scenario)
		err = os.WriteFile(filepath.Join(configDir, "cloud-init.yaml"), []byte("# config"), 0644)
		require.NoError(t, err)

		exists, err := g.configExists(configDir)
		assert.NoError(t, err)
		assert.True(t, exists) // Should detect partial config
	})

	t.Run("file instead of directory", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "is-a-file")
		err := os.WriteFile(filePath, []byte("content"), 0644)
		require.NoError(t, err)

		_, err = g.configExists(filePath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})
}

func TestGenerator_CheckUbuntuImage(t *testing.T) {
	g := New("/test/project")

	t.Run("empty path", func(t *testing.T) {
		warning := g.checkUbuntuImage("")
		assert.Contains(t, warning, "Warning")
		assert.Contains(t, warning, "No Ubuntu image path specified")
	})

	t.Run("non-existent path", func(t *testing.T) {
		warning := g.checkUbuntuImage("/nonexistent/image.img")
		assert.Contains(t, warning, "Warning")
		assert.Contains(t, warning, "not found")
		assert.Contains(t, warning, "wget") // Should include download instructions
	})

	t.Run("existing file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-image-*.img")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		warning := g.checkUbuntuImage(tmpFile.Name())
		assert.Empty(t, warning) // No warning for existing file
	})
}

func TestGenerator_EnsureRootConfig(t *testing.T) {
	tmpDir := t.TempDir()
	g := New(tmpDir)

	tfDir := filepath.Join(tmpDir, "tf")

	t.Run("creates directory and root config", func(t *testing.T) {
		err := g.ensureRootConfig(tmpDir, tfDir)
		assert.NoError(t, err)

		// Check directory was created
		info, err := os.Stat(tfDir)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())

		// Check root terragrunt.hcl was created
		rootHCL := filepath.Join(tfDir, "terragrunt.hcl")
		content, err := os.ReadFile(rootHCL)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "Root Terragrunt Configuration")
		assert.Contains(t, string(content), "../terragrunt/modules/libvirt-vm")
	})

	t.Run("does not overwrite existing root config", func(t *testing.T) {
		rootHCL := filepath.Join(tfDir, "terragrunt.hcl")
		customContent := "# Custom config - should not be overwritten"
		err := os.WriteFile(rootHCL, []byte(customContent), 0644)
		require.NoError(t, err)

		err = g.ensureRootConfig(tmpDir, tfDir)
		assert.NoError(t, err)

		// Check content was not overwritten
		content, err := os.ReadFile(rootHCL)
		assert.NoError(t, err)
		assert.Equal(t, customContent, string(content))
	})
}

func TestGenerator_WriteTerragruntHCL(t *testing.T) {
	tmpDir := t.TempDir()
	g := New(tmpDir)

	machineDir := filepath.Join(tmpDir, "test-vm")
	err := os.MkdirAll(machineDir, 0755)
	require.NoError(t, err)

	opts := &deploy.DeployOptions{
		Terragrunt: deploy.TerragruntOptions{
			VMName:      "test-vm",
			CPUs:        4,
			MemoryMB:    8192,
			DiskGB:      50,
			Autostart:   true,
			UbuntuImage: "/var/lib/libvirt/images/test.img",
			LibvirtURI:  "qemu+ssh://user@host/system",
			StoragePool: "mypool",
			NetworkName: "mynet",
		},
	}

	err = g.writeTerragruntHCL(opts, machineDir)
	assert.NoError(t, err)

	// Read and verify content
	content, err := os.ReadFile(filepath.Join(machineDir, "terragrunt.hcl"))
	assert.NoError(t, err)
	contentStr := string(content)

	// Check structure
	assert.Contains(t, contentStr, "include \"root\"")
	assert.Contains(t, contentStr, "find_in_parent_folders()")
	assert.Contains(t, contentStr, "../../terragrunt/modules/libvirt-vm")

	// Check provider with custom libvirt URI
	assert.Contains(t, contentStr, "generate \"provider\"")
	assert.Contains(t, contentStr, "qemu+ssh://user@host/system")

	// Check inputs
	assert.Contains(t, contentStr, `vm_name           = "test-vm"`)
	assert.Contains(t, contentStr, "vcpu_count        = 4")
	assert.Contains(t, contentStr, "memory_mb         = 8192")
	assert.Contains(t, contentStr, "disk_size_gb      = 50")
	assert.Contains(t, contentStr, "autostart         = true")
	assert.Contains(t, contentStr, `ubuntu_image_path = "/var/lib/libvirt/images/test.img"`)
	assert.Contains(t, contentStr, `storage_pool      = "mypool"`)
	assert.Contains(t, contentStr, `network_name      = "mynet"`)
	assert.Contains(t, contentStr, "${get_terragrunt_dir()}/cloud-init.yaml")
}

func TestGenerator_Deploy_CleanupOnFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create terragrunt modules directory
	modulesDir := filepath.Join(tmpDir, "terragrunt", "modules", "libvirt-vm")
	err := os.MkdirAll(modulesDir, 0755)
	require.NoError(t, err)

	g := New(tmpDir)

	// Create options with nil config to cause failure during cloud-init generation
	opts := &deploy.DeployOptions{
		ProjectRoot: tmpDir,
		Config:      nil, // This will cause Validate to fail
		Terragrunt: deploy.TerragruntOptions{
			VMName: "test-cleanup-vm",
		},
	}

	progressCalled := false
	progress := func(event deploy.ProgressEvent) {
		progressCalled = true
	}

	_, err = g.Deploy(context.Background(), opts, progress)
	assert.Error(t, err)
	assert.True(t, progressCalled)

	// Directory should not exist because validation failed before creation
	machineDir := filepath.Join(tmpDir, "tf", "test-cleanup-vm")
	_, statErr := os.Stat(machineDir)
	assert.True(t, os.IsNotExist(statErr), "Directory should not exist after validation failure")
}

func TestDeployResult_Outputs(t *testing.T) {
	result := &deploy.DeployResult{
		Target:  deploy.TargetTerragrunt,
		Outputs: make(map[string]string),
	}

	// Test that outputs can be set and retrieved
	result.Outputs["vm_name"] = "test-vm"
	result.Outputs["config_dir"] = "/path/to/config"
	result.Outputs["next_steps"] = "cd tf/test-vm && terragrunt init && terragrunt apply"

	assert.Equal(t, "test-vm", result.Outputs["vm_name"])
	assert.Equal(t, "/path/to/config", result.Outputs["config_dir"])
	assert.Contains(t, result.Outputs["next_steps"], "terragrunt")
}

func TestDefaultTerragruntOptions(t *testing.T) {
	opts := deploy.DefaultTerragruntOptions()

	assert.Equal(t, 2, opts.CPUs)
	assert.Equal(t, 2048, opts.MemoryMB)
	assert.Equal(t, 20, opts.DiskGB)
	assert.Equal(t, "qemu:///system", opts.LibvirtURI)
	assert.Equal(t, "default", opts.StoragePool)
	assert.Equal(t, "default", opts.NetworkName)
	assert.Contains(t, opts.UbuntuImage, "noble-server-cloudimg-amd64.img")
}

func TestValidateVMName(t *testing.T) {
	tests := []struct {
		name        string
		vmName      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid simple name",
			vmName:      "my-vm",
			expectError: false,
		},
		{
			name:        "valid single char",
			vmName:      "a",
			expectError: false,
		},
		{
			name:        "valid with numbers",
			vmName:      "vm-123-test",
			expectError: false,
		},
		{
			name:        "valid generated name format",
			vmName:      "vm-0107-1234-ab12",
			expectError: false,
		},
		{
			name:        "empty name",
			vmName:      "",
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name:        "too long name",
			vmName:      strings.Repeat("a", 65),
			expectError: true,
			errorMsg:    "too long",
		},
		{
			name:        "path traversal attempt",
			vmName:      "../etc/passwd",
			expectError: true,
			errorMsg:    "slashes",
		},
		{
			name:        "contains slash",
			vmName:      "my/vm",
			expectError: true,
			errorMsg:    "slashes",
		},
		{
			name:        "contains backslash",
			vmName:      "my\\vm",
			expectError: true,
			errorMsg:    "slashes",
		},
		{
			name:        "starts with hyphen",
			vmName:      "-myvm",
			expectError: true,
			errorMsg:    "cannot start or end with a hyphen",
		},
		{
			name:        "ends with hyphen",
			vmName:      "myvm-",
			expectError: true,
			errorMsg:    "cannot start or end with a hyphen",
		},
		{
			name:        "contains uppercase",
			vmName:      "MyVM",
			expectError: true,
			errorMsg:    "lowercase",
		},
		{
			name:        "contains spaces",
			vmName:      "my vm",
			expectError: true,
			errorMsg:    "lowercase",
		},
		{
			name:        "contains underscore",
			vmName:      "my_vm",
			expectError: true,
			errorMsg:    "lowercase",
		},
		{
			name:        "dot directory",
			vmName:      ".",
			expectError: true,
			errorMsg:    "'.'",
		},
		{
			name:        "double dot directory",
			vmName:      "..",
			expectError: true,
			errorMsg:    "'..'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVMName(tt.vmName)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerator_Deploy_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create terragrunt modules directory
	modulesDir := filepath.Join(tmpDir, "terragrunt", "modules", "libvirt-vm")
	err := os.MkdirAll(modulesDir, 0755)
	require.NoError(t, err)

	// Create a minimal main.tf so the module looks valid
	err = os.WriteFile(filepath.Join(modulesDir, "main.tf"), []byte("# placeholder"), 0644)
	require.NoError(t, err)

	g := New(tmpDir)

	opts := &deploy.DeployOptions{
		ProjectRoot: tmpDir,
		Config: &config.FullConfig{
			Username: "testuser",
			Hostname: "testhost",
		},
		Terragrunt: deploy.TerragruntOptions{
			VMName:      "test-vm",
			CPUs:        2,
			MemoryMB:    2048,
			DiskGB:      20,
			UbuntuImage: "/var/lib/libvirt/images/test.img",
			LibvirtURI:  "qemu:///system",
			StoragePool: "default",
			NetworkName: "default",
		},
	}

	events := make([]deploy.ProgressEvent, 0)
	progress := func(event deploy.ProgressEvent) {
		events = append(events, event)
	}

	result, err := g.Deploy(context.Background(), opts, progress)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Check outputs
	assert.Equal(t, "test-vm", result.Outputs["vm_name"])
	assert.Contains(t, result.Outputs["config_dir"], "tf/test-vm")
	assert.Contains(t, result.Outputs["next_steps"], "terragrunt")

	// Check files were created
	machineDir := filepath.Join(tmpDir, "tf", "test-vm")
	assert.DirExists(t, machineDir)
	assert.FileExists(t, filepath.Join(machineDir, "terragrunt.hcl"))
	assert.FileExists(t, filepath.Join(machineDir, "cloud-init.yaml"))

	// Check root terragrunt.hcl was created
	assert.FileExists(t, filepath.Join(tmpDir, "tf", "terragrunt.hcl"))

	// Check progress events
	assert.GreaterOrEqual(t, len(events), 4)
	assert.Equal(t, deploy.StageComplete, events[len(events)-1].Stage)
}

func TestGenerator_Validate_InvalidVMName(t *testing.T) {
	tmpDir := t.TempDir()

	// Create terragrunt modules directory structure
	modulesDir := filepath.Join(tmpDir, "terragrunt", "modules", "libvirt-vm")
	err := os.MkdirAll(modulesDir, 0755)
	require.NoError(t, err)

	g := New(tmpDir)

	opts := &deploy.DeployOptions{
		ProjectRoot: tmpDir,
		Config:      &config.FullConfig{},
		Terragrunt: deploy.TerragruntOptions{
			VMName: "../malicious-vm",
		},
	}

	err = g.Validate(opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slashes")
}
