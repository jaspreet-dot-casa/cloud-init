package create

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy/multipass"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy/terraform"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy/usb"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/generator"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// deployState holds the state for deployment progress
type deployState struct {
	deployer     deploy.Deployer
	spinner      spinner.Model
	progressBar  progress.Model
	events       []deploy.ProgressEvent
	progressChan chan deploy.ProgressEvent
	result       *deploy.DeployResult
	done         bool
}

// getDeployState returns the deploy state with proper type assertion.
// Returns nil if DeployState is nil or holds a different type.
func (m *Model) getDeployState() *deployState {
	if m.wizard.DeployState == nil {
		return nil
	}
	state, ok := m.wizard.DeployState.(*deployState)
	if !ok {
		return nil
	}
	return state
}

// initDeployPhase initializes the Deploy phase
func (m *Model) initDeployPhase() {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
		progress.WithoutPercentage(),
	)

	state := &deployState{
		spinner:      s,
		progressBar:  p,
		events:       make([]deploy.ProgressEvent, 0),
		progressChan: make(chan deploy.ProgressEvent, 100),
	}
	m.wizard.DeployState = state

	// Create the appropriate deployer based on target
	state.deployer = m.createDeployer()
}

// createDeployer creates the appropriate deployer for the selected target
func (m *Model) createDeployer() deploy.Deployer {
	switch m.wizard.Data.Target {
	case deploy.TargetMultipass:
		return multipass.New()
	case deploy.TargetTerraform:
		return terraform.New(m.projectDir)
	case deploy.TargetUSB:
		return usb.New(m.projectDir)
	case deploy.TargetConfigOnly:
		// For config-only, we'll use a simple generator
		return &configOnlyDeployer{
			projectDir:   m.projectDir,
			outputDir:    m.wizard.Data.GenerateOpts.OutputDir,
			generateYAML: m.wizard.Data.GenerateOpts.GenerateCloudInit,
			registry:     m.wizard.Registry,
		}
	default:
		// Fallback to config-only deployer for unknown targets
		return &configOnlyDeployer{
			projectDir: m.projectDir,
			outputDir:  ".",
		}
	}
}

// startDeploy starts the deployment process
func (m *Model) startDeploy() tea.Cmd {
	state := m.getDeployState()
	if state == nil {
		return nil
	}
	return tea.Batch(
		state.spinner.Tick,
		m.runDeployment(),
		m.waitForDeployProgress(),
	)
}

// runDeployment runs the deployment in the background
func (m *Model) runDeployment() tea.Cmd {
	return func() tea.Msg {
		state := m.getDeployState()
		if state == nil || state.deployer == nil {
			return deployCompleteMsg{result: &deploy.DeployResult{
				Success: false,
				Error:   fmt.Errorf("no deployer configured"),
			}}
		}

		// Build deploy options from wizard data
		opts := m.buildDeployOptions()

		// Progress callback that sends to channel
		progressCallback := func(e deploy.ProgressEvent) {
			state.progressChan <- e
		}

		// Run deployment
		ctx := context.Background()
		result, err := state.deployer.Deploy(ctx, opts, progressCallback)

		// Handle error if result is nil
		if err != nil && result == nil {
			result = &deploy.DeployResult{
				Success: false,
				Error:   err,
			}
		}

		// Signal completion
		close(state.progressChan)

		return deployCompleteMsg{result: result}
	}
}

// waitForDeployProgress waits for progress events
func (m *Model) waitForDeployProgress() tea.Cmd {
	return func() tea.Msg {
		state := m.getDeployState()
		if state == nil {
			return nil
		}

		event, ok := <-state.progressChan
		if !ok {
			return nil // Channel closed
		}
		return deployProgressMsg(event)
	}
}

// buildDeployOptions builds the deploy options from wizard data
func (m *Model) buildDeployOptions() *deploy.DeployOptions {
	data := &m.wizard.Data

	// Collect all SSH keys (both from GitHub and locally selected)
	var sshKeys []string
	sshKeys = append(sshKeys, data.GitHubSSHKeys...)
	sshKeys = append(sshKeys, data.SSHKeys...)

	// Build the config
	cfg := &config.FullConfig{
		Username:         data.Username,
		Hostname:         data.Hostname,
		FullName:         data.GitName,
		Email:            data.GitEmail,
		MachineName:      data.DisplayName,
		SSHPublicKeys:    sshKeys,
		EnabledPackages:  data.Packages,
		TailscaleAuthKey: data.TailscaleKey,
		GithubUser:       data.GitHubUser,
		GithubPAT:        data.GitHubPAT,
	}

	// Calculate disabled packages
	if m.wizard.Registry != nil {
		allPackages := m.wizard.Registry.Names()
		enabledSet := make(map[string]bool)
		for _, pkg := range data.Packages {
			enabledSet[pkg] = true
		}
		for _, pkg := range allPackages {
			if !enabledSet[pkg] {
				cfg.DisabledPackages = append(cfg.DisabledPackages, pkg)
			}
		}
	}

	opts := &deploy.DeployOptions{
		ProjectRoot: m.projectDir,
		Config:      cfg,
	}

	// Add target-specific options
	switch data.Target {
	case deploy.TargetMultipass:
		opts.Multipass = data.MultipassOpts

	case deploy.TargetTerraform:
		opts.Terraform = data.TerraformOpts

	case deploy.TargetUSB:
		opts.USB = deploy.USBOptions{
			SourceISO:     data.USBOpts.SourceISO,
			OutputISO:     data.USBOpts.OutputPath,
			StorageLayout: data.USBOpts.StorageLayout,
		}
	}

	return opts
}

// handleDeployPhase handles input for the Deploy phase
func (m *Model) handleDeployPhase(msg tea.Msg) (app.Tab, tea.Cmd) {
	state := m.getDeployState()
	if state == nil {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if state.done {
				// Move to complete phase
				m.wizard.Advance()
				m.initPhase(m.wizard.Phase)
				return m, nil
			}
		}

	case spinner.TickMsg:
		if !state.done {
			var cmd tea.Cmd
			state.spinner, cmd = state.spinner.Update(msg)
			return m, cmd
		}

	case progress.FrameMsg:
		progressModel, cmd := state.progressBar.Update(msg)
		state.progressBar = progressModel.(progress.Model)
		return m, cmd

	case deployProgressMsg:
		state.events = append(state.events, deploy.ProgressEvent(msg))
		// Continue listening for more progress events
		return m, tea.Batch(
			m.waitForDeployProgress(),
			state.progressBar.SetPercent(float64(msg.Percent)/100.0),
		)

	case deployCompleteMsg:
		state.done = true
		state.result = msg.result
		return m, nil
	}

	return m, nil
}

// viewDeployPhase renders the Deploy phase
func (m *Model) viewDeployPhase() string {
	state := m.getDeployState()
	if state == nil {
		return "Initializing deployment...\n"
	}

	var b strings.Builder

	// Header
	deployerName := "Unknown"
	if state.deployer != nil {
		deployerName = state.deployer.Name()
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf("Deploying to %s", deployerName)))
	b.WriteString("\n\n")

	// Progress bar
	if len(state.events) > 0 {
		lastEvent := state.events[len(state.events)-1]
		percent := lastEvent.Percent
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}

		barView := state.progressBar.ViewAs(float64(percent) / 100.0)
		b.WriteString(progressBarStyle.Render(barView))
		b.WriteString(fmt.Sprintf(" %d%%", percent))
		b.WriteString("\n\n")
	}

	// Event log
	for i, event := range state.events {
		isLast := i == len(state.events)-1 && !state.done

		icon := "  "
		msgStyle := dimStyle

		if event.IsError {
			icon = errorStyle.Render("  ")
			msgStyle = errorStyle
		} else if event.Stage == deploy.StageComplete {
			icon = successStyle.Render("  ")
			msgStyle = successStyle
		} else if isLast {
			icon = activeStyle.Render("  ")
			msgStyle = lipgloss.NewStyle()
		} else {
			icon = successStyle.Render("  ")
		}

		b.WriteString(icon)
		b.WriteString(msgStyle.Render(event.Message))
		b.WriteString("\n")

		// Show command if present (for the active step or errors)
		if event.Command != "" && (isLast || event.IsError) {
			b.WriteString("     ")
			b.WriteString(commandStyle.Render(" " + event.Command))
			b.WriteString("\n")
		}

		// Show detail if present
		if event.Detail != "" && (isLast || event.IsError) {
			b.WriteString("     ")
			b.WriteString(dimStyle.Render(event.Detail))
			b.WriteString("\n")
		}
	}

	// Spinner if still deploying
	if !state.done && len(state.events) > 0 {
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(state.spinner.View())
		b.WriteString(" Working...")
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	if state.done {
		if state.result != nil && state.result.Success {
			b.WriteString(dimStyle.Render("Press Enter to view results"))
		} else {
			b.WriteString(dimStyle.Render("Press Enter to continue"))
		}
	} else {
		b.WriteString(dimStyle.Render("Deployment in progress..."))
	}
	b.WriteString("\n")

	return b.String()
}

// configOnlyDeployer is a simple deployer for config-only generation
type configOnlyDeployer struct {
	projectDir   string
	outputDir    string
	generateYAML bool
	registry     *packages.Registry
}

func (d *configOnlyDeployer) Name() string {
	return "Config Generator"
}

func (d *configOnlyDeployer) Target() deploy.DeploymentTarget {
	return deploy.TargetConfigOnly
}

func (d *configOnlyDeployer) Validate(opts *deploy.DeployOptions) error {
	if d.registry == nil {
		return fmt.Errorf("package registry not available - cannot generate summary")
	}
	return nil
}

func (d *configOnlyDeployer) Deploy(ctx context.Context, opts *deploy.DeployOptions, progress deploy.ProgressCallback) (*deploy.DeployResult, error) {
	cfg := opts.Config
	outputDir := d.outputDir
	if outputDir == "" {
		outputDir = "."
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Helper to safely call progress callback
	reportProgress := func(event deploy.ProgressEvent) {
		if progress != nil {
			progress(event)
		}
	}

	reportProgress(deploy.ProgressEvent{
		Stage:   deploy.StageConfig,
		Message: "Generating configuration files...",
		Percent: 10,
	})

	// Generate config.env
	reportProgress(deploy.ProgressEvent{
		Stage:   deploy.StageConfig,
		Message: "Writing config.env...",
		Percent: 25,
	})

	configEnvPath := filepath.Join(outputDir, "config.env")
	if err := d.writeConfigEnv(configEnvPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to write config.env: %w", err)
	}

	// Generate summary.md
	reportProgress(deploy.ProgressEvent{
		Stage:   deploy.StageConfig,
		Message: "Writing summary.md...",
		Percent: 40,
	})

	summaryPath := filepath.Join(outputDir, "summary.md")
	if err := generator.GenerateSummary(cfg, d.registry, summaryPath); err != nil {
		return nil, fmt.Errorf("failed to write summary.md: %w", err)
	}

	// Generate secrets.env
	reportProgress(deploy.ProgressEvent{
		Stage:   deploy.StageConfig,
		Message: "Writing cloud-init/secrets.env...",
		Percent: 55,
	})

	secretsDir := filepath.Join(outputDir, "cloud-init")
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cloud-init directory: %w", err)
	}

	secretsEnvPath := filepath.Join(secretsDir, "secrets.env")
	if err := d.writeSecretsEnv(secretsEnvPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to write secrets.env: %w", err)
	}

	// Generate cloud-init.yaml if requested
	if d.generateYAML {
		reportProgress(deploy.ProgressEvent{
			Stage:   deploy.StageConfig,
			Message: "Writing cloud-init/cloud-init.yaml...",
			Percent: 75,
		})

		outputPath := filepath.Join(secretsDir, "cloud-init.yaml")

		if err := generator.Generate(cfg, outputPath); err != nil {
			return nil, fmt.Errorf("failed to generate cloud-init.yaml: %w", err)
		}
	}

	reportProgress(deploy.ProgressEvent{
		Stage:   deploy.StageComplete,
		Message: "Configuration files generated successfully",
		Percent: 100,
	})

	outputs := map[string]string{
		"config.env":  configEnvPath,
		"secrets.env": secretsEnvPath,
		"summary.md":  summaryPath,
	}
	if d.generateYAML {
		outputs["cloud-init.yaml"] = filepath.Join(secretsDir, "cloud-init.yaml")
	}

	return &deploy.DeployResult{
		Success: true,
		Outputs: outputs,
	}, nil
}

// writeConfigEnv writes the config.env file
func (d *configOnlyDeployer) writeConfigEnv(path string, cfg *config.FullConfig) error {
	var b strings.Builder

	b.WriteString("# Generated by ucli - Configuration File\n")
	b.WriteString("# This file contains non-sensitive configuration\n\n")

	// User configuration
	b.WriteString("# User Configuration\n")
	b.WriteString(fmt.Sprintf("USERNAME=%q\n", cfg.Username))
	b.WriteString(fmt.Sprintf("HOSTNAME=%q\n", cfg.Hostname))
	b.WriteString(fmt.Sprintf("USER_NAME=%q\n", cfg.FullName))
	b.WriteString(fmt.Sprintf("USER_EMAIL=%q\n", cfg.Email))
	b.WriteString(fmt.Sprintf("MACHINE_USER_NAME=%q\n", cfg.MachineName))
	b.WriteString("\n")

	// Git configuration
	b.WriteString("# Git Configuration\n")
	b.WriteString(fmt.Sprintf("GIT_DEFAULT_BRANCH=%q\n", "main"))
	b.WriteString(fmt.Sprintf("GIT_PUSH_AUTO_SETUP_REMOTE=%t\n", true))
	b.WriteString(fmt.Sprintf("GIT_PULL_REBASE=%t\n", true))
	b.WriteString("\n")

	// Package configuration
	b.WriteString("# Package Configuration\n")
	for _, pkg := range cfg.EnabledPackages {
		envVar := "PACKAGE_" + strings.ToUpper(strings.ReplaceAll(pkg, "-", "_")) + "_ENABLED"
		b.WriteString(fmt.Sprintf("export %s=true\n", envVar))
	}
	for _, pkg := range cfg.DisabledPackages {
		envVar := "PACKAGE_" + strings.ToUpper(strings.ReplaceAll(pkg, "-", "_")) + "_ENABLED"
		b.WriteString(fmt.Sprintf("export %s=false\n", envVar))
	}

	return os.WriteFile(path, []byte(b.String()), 0644)
}

// writeSecretsEnv writes the secrets.env file
func (d *configOnlyDeployer) writeSecretsEnv(path string, cfg *config.FullConfig) error {
	var b strings.Builder

	b.WriteString("# Generated by ucli - Secrets File\n")
	b.WriteString("# This file contains sensitive configuration - DO NOT COMMIT\n\n")

	// SSH keys
	b.WriteString("# SSH Keys\n")
	for i, key := range cfg.SSHPublicKeys {
		b.WriteString(fmt.Sprintf("SSH_PUBLIC_KEY_%d=%q\n", i+1, key))
	}
	if len(cfg.SSHPublicKeys) > 0 {
		b.WriteString(fmt.Sprintf("SSH_PUBLIC_KEY=%q\n", cfg.SSHPublicKeys[0]))
	}
	b.WriteString("\n")

	// Tailscale
	if cfg.TailscaleAuthKey != "" {
		b.WriteString("# Tailscale\n")
		b.WriteString(fmt.Sprintf("TAILSCALE_AUTH_KEY=%q\n", cfg.TailscaleAuthKey))
		b.WriteString("\n")
	}

	// GitHub
	if cfg.GithubUser != "" || cfg.GithubPAT != "" {
		b.WriteString("# GitHub\n")
		if cfg.GithubUser != "" {
			b.WriteString(fmt.Sprintf("GITHUB_USER=%q\n", cfg.GithubUser))
		}
		if cfg.GithubPAT != "" {
			b.WriteString(fmt.Sprintf("GITHUB_PAT=%q\n", cfg.GithubPAT))
		}
		b.WriteString("\n")
	}

	return os.WriteFile(path, []byte(b.String()), 0600) // More restrictive permissions for secrets
}

func (d *configOnlyDeployer) Cleanup(ctx context.Context, opts *deploy.DeployOptions) error {
	// Nothing to cleanup for config-only generation
	return nil
}
