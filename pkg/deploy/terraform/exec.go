package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// checkTerraformInstalled verifies terraform is available.
func (d *Deployer) checkTerraformInstalled() error {
	path, err := exec.LookPath("terraform")
	if err != nil {
		return fmt.Errorf("terraform is not installed; %s", InstallInstructions())
	}

	// Verify it works
	cmd := exec.Command(path, "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform is installed but not working: %w", err)
	}

	return nil
}

// checkUbuntuImage verifies the Ubuntu cloud image exists.
func (d *Deployer) checkUbuntuImage(imagePath string) error {
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return fmt.Errorf("Ubuntu cloud image not found at %s\n\nDownload with:\n  wget https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img -O %s",
			imagePath, imagePath)
	}
	return nil
}

// terraformInit runs terraform init if needed.
func (d *Deployer) terraformInit(ctx context.Context, workDir string) error {
	// Check if .terraform directory exists
	terraformDir := filepath.Join(workDir, ".terraform")
	if _, err := os.Stat(terraformDir); err == nil {
		// Already initialized
		return nil
	}

	cmd := exec.CommandContext(ctx, "terraform", "init")
	cmd.Dir = workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return fmt.Errorf("terraform init failed: %s", strings.TrimSpace(errMsg))
		}
		return fmt.Errorf("terraform init failed: %w", err)
	}

	return nil
}

// terraformPlan runs terraform plan and returns the output.
func (d *Deployer) terraformPlan(ctx context.Context, workDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "terraform", "plan", "-no-color")
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return "", fmt.Errorf("terraform plan failed: %s", strings.TrimSpace(errMsg))
		}
		return "", fmt.Errorf("terraform plan failed: %w", err)
	}

	return stdout.String(), nil
}

// confirmApply prompts the user to confirm the terraform apply.
func (d *Deployer) confirmApply(planOutput string) (bool, error) {
	// Display the plan output
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("Terraform Plan Output:")
	fmt.Println(strings.Repeat("─", 60))

	// Show a summary of changes (first 50 lines or so)
	lines := strings.Split(planOutput, "\n")
	maxLines := 50
	if len(lines) > maxLines {
		for _, line := range lines[:maxLines] {
			fmt.Println(line)
		}
		fmt.Printf("\n... (%d more lines)\n", len(lines)-maxLines)
	} else {
		fmt.Println(planOutput)
	}

	fmt.Println(strings.Repeat("─", 60))

	// Prompt for confirmation
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Apply this plan?").
				Description("This will create the resources shown above").
				Affirmative("Yes, apply").
				Negative("No, cancel").
				Value(&confirmed),
		),
	).WithTheme(tui.Theme())

	if err := form.Run(); err != nil {
		return false, fmt.Errorf("confirmation cancelled: %w", err)
	}

	return confirmed, nil
}

// terraformApply runs terraform apply.
func (d *Deployer) terraformApply(ctx context.Context, workDir string) error {
	cmd := exec.CommandContext(ctx, "terraform", "apply", "-auto-approve", "-no-color")
	cmd.Dir = workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Stream stdout to show progress
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return fmt.Errorf("terraform apply failed: %s", strings.TrimSpace(errMsg))
		}
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	return nil
}

// terraformOutput gets the terraform outputs as a map.
func (d *Deployer) terraformOutput(ctx context.Context, workDir string) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return nil, fmt.Errorf("terraform output failed: %s", strings.TrimSpace(errMsg))
		}
		return nil, fmt.Errorf("terraform output failed: %w", err)
	}

	// Parse JSON output
	var rawOutputs map[string]struct {
		Value interface{} `json:"value"`
		Type  interface{} `json:"type"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &rawOutputs); err != nil {
		return nil, fmt.Errorf("failed to parse terraform output: %w", err)
	}

	// Convert to string map
	outputs := make(map[string]string)
	for k, v := range rawOutputs {
		switch val := v.Value.(type) {
		case string:
			outputs[k] = val
		case []interface{}:
			// Handle arrays (like IP addresses)
			if len(val) > 0 {
				if str, ok := val[0].(string); ok {
					outputs[k] = str
				}
			}
		default:
			outputs[k] = fmt.Sprintf("%v", val)
		}
	}

	// Map terraform output names to our expected names
	if vmIP, ok := outputs["vm_ip"]; ok {
		outputs["ip"] = vmIP
	}

	return outputs, nil
}

// terraformDestroy destroys terraform resources.
func (d *Deployer) terraformDestroy(ctx context.Context, workDir string) error {
	cmd := exec.CommandContext(ctx, "terraform", "destroy", "-auto-approve", "-no-color")
	cmd.Dir = workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return fmt.Errorf("terraform destroy failed: %s", strings.TrimSpace(errMsg))
		}
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	return nil
}
