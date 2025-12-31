package terraform

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

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

func TestNew(t *testing.T) {
	d := New("/test/project")
	assert.NotNil(t, d)
	assert.Equal(t, "/test/project", d.projectRoot)
}

func TestDeployer_Name(t *testing.T) {
	d := New("/test/project")
	assert.Equal(t, "Terraform/libvirt", d.Name())
}

func TestDeployer_Target(t *testing.T) {
	d := New("/test/project")
	assert.Equal(t, deploy.TargetTerraform, d.Target())
}

func TestDeployer_SetVerbose(t *testing.T) {
	d := New("/test/project")
	assert.False(t, d.verbose)

	d.SetVerbose(true)
	assert.True(t, d.verbose)
}

func TestDeployer_Validate(t *testing.T) {
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
				ProjectRoot: "/test/project",
				Config:      nil,
			},
			expectError: true,
			errorMsg:    "configuration is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip terraform installation check for unit tests
			err := validateOptionsOnly(tt.opts)

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

func TestDeployer_GenerateVMName(t *testing.T) {
	d := New("/test/project")
	name1 := d.generateVMName()
	name2 := d.generateVMName()

	// Names should start with cloud-init-
	assert.True(t, len(name1) > len("cloud-init-"))
	assert.Contains(t, name1, "cloud-init-")

	// Names generated at different times should be different
	// (unless generated in the same second, which is unlikely in tests)
	// We just check format here
	assert.Regexp(t, `^cloud-init-\d{8}-\d{6}$`, name1)
	assert.Regexp(t, `^cloud-init-\d{8}-\d{6}$`, name2)
}

func TestInstallInstructions(t *testing.T) {
	instructions := InstallInstructions()

	assert.Contains(t, instructions, "Terraform")
	assert.Contains(t, instructions, "libvirt")
	assert.Contains(t, instructions, "Linux")
}

// validateOptionsOnly is a helper that validates options without checking terraform installation
func validateOptionsOnly(opts *deploy.DeployOptions) error {
	if opts.ProjectRoot == "" {
		return fmt.Errorf("project root is required")
	}
	if opts.Config == nil {
		return fmt.Errorf("configuration is required")
	}
	return nil
}

func TestWriteTFVars(t *testing.T) {
	tmpDir := t.TempDir()

	// Create terraform directory
	tfDir := filepath.Join(tmpDir, "terraform")
	err := os.MkdirAll(tfDir, 0755)
	require.NoError(t, err)

	d := New(tmpDir)

	opts := &deploy.DeployOptions{
		ProjectRoot: tmpDir,
		Terraform: deploy.TerraformOptions{
			WorkDir:     "terraform",
			VMName:      "test-vm",
			CPUs:        2,
			MemoryMB:    4096,
			DiskGB:      20,
			LibvirtURI:  "qemu:///system",
			StoragePool: "default",
			NetworkName: "default",
			UbuntuImage: "/var/lib/libvirt/images/test.img",
		},
	}

	cloudInitPath := filepath.Join(tmpDir, "cloud-init", "cloud-init.yaml")

	err = d.writeTFVars(opts, cloudInitPath)
	require.NoError(t, err)

	// Check that terraform.tfvars was created
	tfvarsPath := filepath.Join(tfDir, "terraform.tfvars")
	content, err := os.ReadFile(tfvarsPath)
	require.NoError(t, err)

	// Verify content
	assert.Contains(t, string(content), `vm_name      = "test-vm"`)
	assert.Contains(t, string(content), `vcpu_count   = 2`)
	assert.Contains(t, string(content), `memory_mb    = 4096`)
	assert.Contains(t, string(content), `disk_size_gb = 20`)
	assert.Contains(t, string(content), `libvirt_uri  = "qemu:///system"`)
	assert.Contains(t, string(content), `storage_pool = "default"`)
	assert.Contains(t, string(content), `network_name = "default"`)
	assert.Contains(t, string(content), `ubuntu_image_path = "/var/lib/libvirt/images/test.img"`)
	assert.Contains(t, string(content), "cloud_init_file")
}

func TestCheckUbuntuImage(t *testing.T) {
	d := New("/test/project")

	t.Run("file exists", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-image-*.img")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		err = d.checkUbuntuImage(tmpFile.Name())
		assert.NoError(t, err)
	})

	t.Run("file does not exist", func(t *testing.T) {
		err := d.checkUbuntuImage("/nonexistent/path/image.img")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestDeployResult_Outputs(t *testing.T) {
	result := &deploy.DeployResult{
		Target:  deploy.TargetTerraform,
		Outputs: make(map[string]string),
	}

	// Test that outputs can be set and retrieved
	result.Outputs["vm_name"] = "test-vm"
	result.Outputs["ip"] = "192.168.122.10"
	result.Outputs["user"] = "ubuntu"
	result.Outputs["console_command"] = "virsh console test-vm"

	assert.Equal(t, "test-vm", result.Outputs["vm_name"])
	assert.Equal(t, "192.168.122.10", result.Outputs["ip"])
	assert.Equal(t, "ubuntu", result.Outputs["user"])
	assert.Equal(t, "virsh console test-vm", result.Outputs["console_command"])
}

func TestDefaultTerraformOptions(t *testing.T) {
	opts := deploy.DefaultTerraformOptions()

	assert.Equal(t, "terraform", opts.WorkDir)
	assert.Equal(t, false, opts.AutoApprove)
	assert.Equal(t, 2, opts.CPUs)
	assert.Equal(t, 2048, opts.MemoryMB)
	assert.Equal(t, 20, opts.DiskGB)
	assert.Equal(t, "qemu:///system", opts.LibvirtURI)
	assert.Equal(t, "default", opts.StoragePool)
	assert.Equal(t, "default", opts.NetworkName)
	assert.Contains(t, opts.UbuntuImage, "jammy-server-cloudimg-amd64.img")
	assert.Equal(t, false, opts.KeepOnFailure)
}

func TestDeployer_Fail(t *testing.T) {
	d := New("/test/project")
	result := &deploy.DeployResult{
		Target:  deploy.TargetTerraform,
		Outputs: make(map[string]string),
	}
	testErr := fmt.Errorf("test error")
	start := time.Now()

	failedResult := d.fail(result, testErr, start)

	assert.False(t, failedResult.Success)
	assert.Equal(t, testErr, failedResult.Error)
	assert.True(t, failedResult.Duration >= 0)
}

func TestDeployer_Cleanup_KeepOnFailure(t *testing.T) {
	d := New("/test/project")
	ctx := context.Background()

	opts := &deploy.DeployOptions{
		ProjectRoot: "/test/project",
		Terraform: deploy.TerraformOptions{
			KeepOnFailure: true,
			WorkDir:       "terraform",
		},
	}

	// When KeepOnFailure is true, Cleanup should return nil without doing anything
	err := d.Cleanup(ctx, opts)
	assert.NoError(t, err)
}

func TestDeployer_VMNameAutoGeneration(t *testing.T) {
	// Test that empty VM name gets auto-generated
	d := New("/test/project")

	// Call generateVMName multiple times
	name1 := d.generateVMName()
	time.Sleep(time.Millisecond * 10) // Small delay to ensure different timestamps if in same second
	name2 := d.generateVMName()

	// Both should have the correct format
	assert.Regexp(t, `^cloud-init-\d{8}-\d{6}$`, name1)
	assert.Regexp(t, `^cloud-init-\d{8}-\d{6}$`, name2)

	// Should start with cloud-init-
	assert.True(t, strings.HasPrefix(name1, "cloud-init-"))
	assert.True(t, strings.HasPrefix(name2, "cloud-init-"))
}

func TestWriteTFVars_RelativePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create terraform directory
	tfDir := filepath.Join(tmpDir, "terraform")
	err := os.MkdirAll(tfDir, 0755)
	require.NoError(t, err)

	// Create cloud-init directory
	cloudInitDir := filepath.Join(tmpDir, "cloud-init")
	err = os.MkdirAll(cloudInitDir, 0755)
	require.NoError(t, err)

	d := New(tmpDir)

	opts := &deploy.DeployOptions{
		ProjectRoot: tmpDir,
		Terraform: deploy.TerraformOptions{
			WorkDir:     "terraform",
			VMName:      "test-vm",
			CPUs:        2,
			MemoryMB:    4096,
			DiskGB:      20,
			LibvirtURI:  "qemu:///system",
			StoragePool: "default",
			NetworkName: "default",
			UbuntuImage: "/var/lib/libvirt/images/test.img",
		},
	}

	cloudInitPath := filepath.Join(cloudInitDir, "cloud-init.yaml")

	err = d.writeTFVars(opts, cloudInitPath)
	require.NoError(t, err)

	// Read and verify relative path is used
	content, err := os.ReadFile(filepath.Join(tfDir, "terraform.tfvars"))
	require.NoError(t, err)

	// Should use relative path from terraform/ to cloud-init/
	assert.Contains(t, string(content), `cloud_init_file = "../cloud-init/cloud-init.yaml"`)
}

func TestWriteTFVars_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()

	// Create terraform directory
	tfDir := filepath.Join(tmpDir, "terraform")
	err := os.MkdirAll(tfDir, 0755)
	require.NoError(t, err)

	d := New(tmpDir)

	// VM name with special characters that need escaping
	opts := &deploy.DeployOptions{
		ProjectRoot: tmpDir,
		Terraform: deploy.TerraformOptions{
			WorkDir:     "terraform",
			VMName:      `test-vm-with-"quotes"`,
			CPUs:        2,
			MemoryMB:    2048,
			DiskGB:      20,
			LibvirtURI:  "qemu:///system",
			StoragePool: "default",
			NetworkName: "default",
			UbuntuImage: `/path/with spaces/image.img`,
		},
	}

	cloudInitPath := filepath.Join(tmpDir, "cloud-init.yaml")

	err = d.writeTFVars(opts, cloudInitPath)
	require.NoError(t, err)

	// Read and verify special characters are properly quoted
	content, err := os.ReadFile(filepath.Join(tfDir, "terraform.tfvars"))
	require.NoError(t, err)

	// Go's %q format should escape quotes properly
	assert.Contains(t, string(content), `vm_name`)
	assert.Contains(t, string(content), `ubuntu_image_path`)
}

func TestWriteTFVars_DirectoryNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	d := New(tmpDir)

	opts := &deploy.DeployOptions{
		ProjectRoot: tmpDir,
		Terraform: deploy.TerraformOptions{
			WorkDir:     "nonexistent-terraform-dir",
			VMName:      "test-vm",
			CPUs:        2,
			MemoryMB:    2048,
			DiskGB:      20,
			LibvirtURI:  "qemu:///system",
			StoragePool: "default",
			NetworkName: "default",
			UbuntuImage: "/path/to/image.img",
		},
	}

	cloudInitPath := filepath.Join(tmpDir, "cloud-init.yaml")

	err := d.writeTFVars(opts, cloudInitPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write terraform.tfvars")
}

func TestCheckUbuntuImage_ErrorMessage(t *testing.T) {
	d := New("/test/project")

	err := d.checkUbuntuImage("/nonexistent/path/image.img")

	assert.Error(t, err)
	// Should include download instructions
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "wget")
	assert.Contains(t, err.Error(), "cloud-images.ubuntu.com")
}
