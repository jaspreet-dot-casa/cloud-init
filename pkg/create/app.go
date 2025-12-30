package create

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy/multipass"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy/usb"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/tui"
)

// Styles for the create TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("40")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 2)

	progressBarStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				PaddingRight(2)

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
)

// Run executes the create command workflow.
func Run(projectRoot string) error {
	// Step 1: Welcome and discover packages
	fmt.Println(titleStyle.Render("Ubuntu Cloud-Init Setup"))
	fmt.Println()

	registry, err := packages.DiscoverFromProjectRoot(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}

	// Step 2: TARGET SELECTION FIRST
	target, err := runTargetSelection()
	if err != nil {
		return err
	}

	// Step 3: Target-specific options (before config wizard)
	var targetOpts interface{}
	switch target {
	case deploy.TargetMultipass:
		targetOpts, err = runMultipassOptions()
		if err != nil {
			return err
		}
	case deploy.TargetUSB:
		targetOpts, err = runUSBOptions()
		if err != nil {
			return err
		}
	case deploy.TargetSSH:
		return fmt.Errorf("SSH target not yet implemented")
	case deploy.TargetTerraform:
		return fmt.Errorf("Terraform target not yet implemented")
	}

	// Step 4: Run the configuration wizard (SSH, Git, Host, Packages, Optional)
	// Skip output mode question since target was already selected
	formResult, err := tui.RunForm(registry, &tui.FormOptions{SkipOutputMode: true})
	if err != nil {
		return err
	}

	// Step 5: Show review and confirm
	if !confirmDeployment(formResult, target, targetOpts) {
		fmt.Println("\n" + dimStyle.Render("Deployment cancelled."))
		return nil
	}

	// Step 6: Generate config and deploy
	cfg := config.NewFullConfigFromFormResult(formResult)

	opts := &deploy.DeployOptions{
		ProjectRoot: projectRoot,
		Config:      cfg,
	}

	// Merge target-specific options
	switch target {
	case deploy.TargetMultipass:
		opts.Multipass = targetOpts.(deploy.MultipassOptions)
	case deploy.TargetUSB:
		opts.USB = targetOpts.(deploy.USBOptions)
	}

	// Step 7: Run deployment with progress UI
	return runDeployment(target, opts)
}

// runTargetSelection prompts the user to select a deployment target.
func runTargetSelection() (deploy.DeploymentTarget, error) {
	var target deploy.DeploymentTarget

	targetForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[deploy.DeploymentTarget]().
				Title("What would you like to create?").
				Description("Select your deployment target").
				Options(
					huh.NewOption("Multipass VM (local testing)", deploy.TargetMultipass),
					huh.NewOption("Bootable ISO (bare metal install)", deploy.TargetUSB),
					huh.NewOption("Remote SSH (existing server)", deploy.TargetSSH),
					huh.NewOption("Terraform/libvirt (local KVM)", deploy.TargetTerraform),
				).
				Value(&target),
		).Title("Deployment Target"),
	).WithTheme(tui.Theme())

	if err := targetForm.Run(); err != nil {
		return "", fmt.Errorf("target selection cancelled: %w", err)
	}

	return target, nil
}

// OSImage represents an available OS image for Multipass.
type OSImage struct {
	Name  string // Display name
	Image string // Multipass image identifier
}

// Available OS images for Multipass (from `multipass find`)
var osImages = []OSImage{
	{"Ubuntu 24.04 LTS (Noble Numbat)", "24.04"},
	{"Ubuntu 25.04 (Plucky Puffin)", "25.04"},
	{"Ubuntu 25.10 (Questing Quail)", "25.10"},
	{"Ubuntu 22.04 LTS (Jammy Jellyfish)", "22.04"},
	{"Ubuntu 26.04 LTS Daily (Resolute)", "daily:26.04"},
}

// runMultipassOptions prompts for Multipass-specific options.
func runMultipassOptions() (deploy.MultipassOptions, error) {
	opts := deploy.DefaultMultipassOptions()

	// Generate default VM name
	opts.VMName = fmt.Sprintf("cloud-init-%s", time.Now().Format("20060102-150405"))

	// Build OS image options
	imageOptions := make([]huh.Option[string], len(osImages))
	for i, img := range osImages {
		imageOptions[i] = huh.NewOption(img.Name, img.Image)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("VM Name").
				Description("Name for the Multipass VM").
				Value(&opts.VMName),

			huh.NewSelect[string]().
				Title("OS Image").
				Description("Ubuntu version to install (type to filter)").
				Options(imageOptions...).
				Filtering(true).
				Value(&opts.UbuntuVersion),

			huh.NewSelect[int]().
				Title("CPUs").
				Options(
					huh.NewOption("1 CPU", 1),
					huh.NewOption("2 CPUs (recommended)", 2),
					huh.NewOption("4 CPUs", 4),
				).
				Value(&opts.CPUs),

			huh.NewSelect[int]().
				Title("Memory").
				Options(
					huh.NewOption("2 GB", 2048),
					huh.NewOption("4 GB (recommended)", 4096),
					huh.NewOption("8 GB", 8192),
				).
				Value(&opts.MemoryMB),

			huh.NewSelect[int]().
				Title("Disk Size").
				Options(
					huh.NewOption("10 GB", 10),
					huh.NewOption("20 GB (recommended)", 20),
					huh.NewOption("40 GB", 40),
				).
				Value(&opts.DiskGB),

			huh.NewConfirm().
				Title("Keep VM on failure?").
				Description("Keep the VM for debugging if deployment fails").
				Value(&opts.KeepOnFailure),
		).Title("Multipass VM Options"),
	).WithTheme(tui.Theme())

	if err := form.Run(); err != nil {
		return opts, fmt.Errorf("multipass options cancelled: %w", err)
	}

	return opts, nil
}

// runUSBOptions prompts for USB/ISO-specific options.
func runUSBOptions() (deploy.USBOptions, error) {
	opts := deploy.DefaultUSBOptions()

	// Check for required tools before showing the form
	if _, err := exec.LookPath("xorriso"); err != nil {
		if err := offerToInstallXorriso(); err != nil {
			return opts, err
		}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Source Ubuntu ISO").
				Description("Path to Ubuntu Server ISO file").
				Placeholder("/path/to/ubuntu-24.04-live-server-amd64.iso").
				Value(&opts.SourceISO).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("source ISO path is required")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Storage Layout").
				Description("Disk partitioning scheme for installation").
				Options(
					huh.NewOption("LVM - Flexible partitions, snapshots, easy resizing", "lvm"),
					huh.NewOption("Direct - Simple partitions, no overhead, full disk access", "direct"),
					huh.NewOption("ZFS - Advanced filesystem, built-in snapshots, compression", "zfs"),
				).
				Value(&opts.StorageLayout),
		).Title("Bootable ISO Options"),
	).WithTheme(tui.Theme())

	if err := form.Run(); err != nil {
		return opts, fmt.Errorf("USB options cancelled: %w", err)
	}

	return opts, nil
}

// offerToInstallXorriso checks for xorriso and offers to install it.
func offerToInstallXorriso() error {
	fmt.Println()
	fmt.Println(errorStyle.Render("  Missing Required Tool"))
	fmt.Println()
	fmt.Println("  xorriso is required to create bootable ISOs but was not found.")
	fmt.Println()

	// Determine install command based on platform
	var installCmd string
	var installArgs []string
	var manualInstructions string

	switch runtime.GOOS {
	case "darwin":
		// Check if brew is available
		if _, err := exec.LookPath("brew"); err == nil {
			installCmd = "brew"
			installArgs = []string{"install", "xorriso"}
			manualInstructions = "brew install xorriso"
		} else {
			fmt.Println("  Homebrew is not installed. Please install it first:")
			fmt.Println("    /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
			fmt.Println()
			fmt.Println("  Then run:")
			fmt.Println("    brew install xorriso")
			fmt.Println()
			return fmt.Errorf("xorriso not found - please install Homebrew and xorriso")
		}
	case "linux":
		// Try to detect package manager
		if _, err := exec.LookPath("apt"); err == nil {
			installCmd = "sudo"
			installArgs = []string{"apt", "install", "-y", "xorriso"}
			manualInstructions = "sudo apt install xorriso"
		} else if _, err := exec.LookPath("dnf"); err == nil {
			installCmd = "sudo"
			installArgs = []string{"dnf", "install", "-y", "xorriso"}
			manualInstructions = "sudo dnf install xorriso"
		} else if _, err := exec.LookPath("pacman"); err == nil {
			installCmd = "sudo"
			installArgs = []string{"pacman", "-S", "--noconfirm", "libisoburn"}
			manualInstructions = "sudo pacman -S libisoburn"
		} else {
			fmt.Println("  Could not detect package manager. Please install xorriso manually.")
			fmt.Println()
			return fmt.Errorf("xorriso not found - please install it manually")
		}
	default:
		fmt.Println("  Unsupported platform. Please install xorriso manually.")
		fmt.Println()
		return fmt.Errorf("xorriso not found - unsupported platform")
	}

	// Ask user if they want to install
	var install bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Install xorriso?").
				Description(fmt.Sprintf("Run: %s", manualInstructions)).
				Value(&install),
		),
	).WithTheme(tui.Theme())

	if err := form.Run(); err != nil {
		return fmt.Errorf("cancelled")
	}

	if !install {
		fmt.Println()
		fmt.Println("  To install manually, run:")
		fmt.Println("    " + manualInstructions)
		fmt.Println()
		return fmt.Errorf("xorriso not found - please install it and try again")
	}

	// Run the install command
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Installing xorriso..."))
	fmt.Println()

	cmd := exec.Command(installCmd, installArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("  Installation failed"))
		fmt.Println()
		fmt.Println("  Please install manually:")
		fmt.Println("    " + manualInstructions)
		fmt.Println()
		return fmt.Errorf("failed to install xorriso: %w", err)
	}

	// Verify installation
	if _, err := exec.LookPath("xorriso"); err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("  Installation completed but xorriso not found in PATH"))
		fmt.Println()
		return fmt.Errorf("xorriso installed but not found in PATH")
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  ✓ xorriso installed successfully"))
	fmt.Println()

	return nil
}

// confirmDeployment shows a review and asks for confirmation.
func confirmDeployment(result *tui.FormResult, target deploy.DeploymentTarget, targetOpts interface{}) bool {
	fmt.Println()
	fmt.Println(titleStyle.Render("Review Configuration"))
	fmt.Println()

	// Build target-specific details
	var targetDetails string
	switch target {
	case deploy.TargetMultipass:
		if opts, ok := targetOpts.(deploy.MultipassOptions); ok {
			targetDetails = fmt.Sprintf(`
%s
  Target:    %s
  VM Name:   %s
  OS Image:  %s
  Resources: %d CPU, %d MB RAM, %d GB disk`,
				successStyle.Render("Deployment"),
				target.DisplayName(),
				opts.VMName,
				opts.UbuntuVersion,
				opts.CPUs,
				opts.MemoryMB,
				opts.DiskGB,
			)
		}
	case deploy.TargetUSB:
		if opts, ok := targetOpts.(deploy.USBOptions); ok {
			targetDetails = fmt.Sprintf(`
%s
  Target:    %s
  Source:    %s
  Storage:   %s`,
				successStyle.Render("Deployment"),
				target.DisplayName(),
				opts.SourceISO,
				opts.StorageLayout,
			)
		}
	default:
		targetDetails = fmt.Sprintf(`
%s
  Target:    %s`,
			successStyle.Render("Deployment"),
			target.DisplayName(),
		)
	}

	review := fmt.Sprintf(`%s
  Username:  %s
  Hostname:  %s
  Name:      %s
  Email:     %s
  SSH Keys:  %d configured

%s
  Selected:  %d packages%s`,
		successStyle.Render("User Configuration"),
		result.User.Username,
		result.User.Hostname,
		result.User.FullName,
		result.User.Email,
		len(result.User.SSHPublicKeys),
		successStyle.Render("Packages"),
		len(result.SelectedPackages),
		targetDetails,
	)

	fmt.Println(boxStyle.Render(review))
	fmt.Println()

	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Start deployment?").
				Affirmative("Yes, deploy!").
				Negative("Cancel").
				Value(&confirm),
		),
	).WithTheme(tui.Theme())

	if err := confirmForm.Run(); err != nil {
		return false
	}

	return confirm
}

// runDeployment runs the deployment with a Bubble Tea progress UI.
func runDeployment(target deploy.DeploymentTarget, opts *deploy.DeployOptions) error {
	// Create deployer
	var deployer deploy.Deployer
	switch target {
	case deploy.TargetMultipass:
		deployer = multipass.New()
	case deploy.TargetUSB:
		deployer = usb.New(opts.ProjectRoot)
	default:
		return fmt.Errorf("deployer not implemented for target: %s", target)
	}

	// Run the deployment UI in alt-screen
	m := newDeployModel(deployer, opts)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("deployment UI error: %w", err)
	}

	// Get the result from the model
	model, ok := finalModel.(deployModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	// Print final results outside of alt-screen (so they're scrollable in terminal)
	printDeploymentResults(model.result, deployer.Name())

	// Return error if deployment failed
	if model.result != nil && !model.result.Success {
		return model.result.Error
	}

	return nil
}

// printDeploymentResults prints the deployment results to the terminal (outside alt-screen).
func printDeploymentResults(result *deploy.DeployResult, deployerName string) {
	fmt.Println()

	if result == nil {
		fmt.Println(errorStyle.Render("Deployment did not complete."))
		return
	}

	if result.Success {
		// Success banner
		fmt.Println(successStyle.Render("  Deployment Complete!"))
		fmt.Println()
		fmt.Printf("  Duration: %s\n", result.Duration.Round(time.Second))
		fmt.Println()

		// Show outputs
		if len(result.Outputs) > 0 {
			fmt.Println(subtitleStyle.Render("  Details:"))

			// Order outputs nicely based on target type
			var orderedKeys []string
			if result.Target == deploy.TargetUSB {
				orderedKeys = []string{"iso_path", "iso_size", "storage_layout", "source_iso"}
			} else {
				orderedKeys = []string{"vm_name", "ip", "user", "ssh_command", "multipass_shell", "cloud_init_path"}
			}
			printedKeys := make(map[string]bool)

			for _, key := range orderedKeys {
				if value, ok := result.Outputs[key]; ok {
					label := formatOutputLabel(key)
					fmt.Printf("    %s: %s\n", label, value)
					printedKeys[key] = true
				}
			}

			// Print any remaining outputs
			for key, value := range result.Outputs {
				if !printedKeys[key] {
					label := formatOutputLabel(key)
					fmt.Printf("    %s: %s\n", label, value)
				}
			}
		}

		// Show helpful commands based on target type
		if vmName, ok := result.Outputs["vm_name"]; ok {
			fmt.Println()
			fmt.Println(dimStyle.Render("  To access the VM:"))
			fmt.Printf("    multipass shell %s\n", vmName)
		}

		if isoPath, ok := result.Outputs["iso_path"]; ok {
			fmt.Println()
			fmt.Println(dimStyle.Render("  To write to USB:"))
			fmt.Printf("    sudo dd if=%s of=/dev/sdX bs=4M status=progress\n", isoPath)
			fmt.Println()
			fmt.Println(dimStyle.Render("  Replace /dev/sdX with your USB device (use 'diskutil list' on macOS)"))
		}

		// Show any warnings from logs
		if len(result.Logs) > 0 {
			fmt.Println()
			fmt.Println(dimStyle.Render("  Notes:"))
			for _, log := range result.Logs {
				fmt.Printf("    %s\n", log)
			}
		}
	} else {
		// Failure banner
		fmt.Println(errorStyle.Render("  Deployment Failed"))
		fmt.Println()
		if result.Error != nil {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		fmt.Println()
		fmt.Println(dimStyle.Render("  Run with --verbose for more details"))
	}

	fmt.Println()
}

// formatOutputLabel formats an output key as a human-readable label.
func formatOutputLabel(key string) string {
	switch key {
	case "vm_name":
		return "VM Name"
	case "ip":
		return "IP Address"
	case "user":
		return "Username"
	case "ssh_command":
		return "SSH Command"
	case "multipass_shell":
		return "Shell Command"
	case "cloud_init_path":
		return "Cloud-Init"
	case "iso_path":
		return "ISO Path"
	case "source_iso":
		return "Source ISO"
	case "storage_layout":
		return "Storage"
	case "iso_size":
		return "ISO Size"
	default:
		return strings.Title(strings.ReplaceAll(key, "_", " "))
	}
}

// deployModel is a Bubble Tea model for deployment progress.
type deployModel struct {
	deployer deploy.Deployer
	opts     *deploy.DeployOptions

	spinner      spinner.Model
	progressBar  progress.Model
	events       []deploy.ProgressEvent
	progressChan chan deploy.ProgressEvent
	result       *deploy.DeployResult
	done         bool
	quitting     bool

	width  int
	height int
}

func newDeployModel(deployer deploy.Deployer, opts *deploy.DeployOptions) deployModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
		progress.WithoutPercentage(),
	)

	return deployModel{
		deployer:     deployer,
		opts:         opts,
		spinner:      s,
		progressBar:  p,
		events:       make([]deploy.ProgressEvent, 0),
		progressChan: make(chan deploy.ProgressEvent, 100),
	}
}

func (m deployModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startDeployment(),
		m.waitForProgress(),
	)
}

func (m deployModel) startDeployment() tea.Cmd {
	return func() tea.Msg {
		// Progress callback that sends to channel
		progressCallback := func(e deploy.ProgressEvent) {
			m.progressChan <- e
		}

		// Run deployment
		ctx := context.Background()
		result, _ := m.deployer.Deploy(ctx, m.opts, progressCallback)

		// Signal completion
		close(m.progressChan)

		return deployCompleteMsg{result: result}
	}
}

func (m deployModel) waitForProgress() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.progressChan
		if !ok {
			return nil // Channel closed
		}
		return deployProgressMsg(event)
	}
}

func (m deployModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progressBar.Width = min(msg.Width-10, 60)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if m.done {
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		if !m.done {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case progress.FrameMsg:
		progressModel, cmd := m.progressBar.Update(msg)
		m.progressBar = progressModel.(progress.Model)
		return m, cmd

	case deployProgressMsg:
		m.events = append(m.events, deploy.ProgressEvent(msg))
		// Continue listening for more progress events
		return m, tea.Batch(
			m.waitForProgress(),
			m.progressBar.SetPercent(float64(msg.Percent)/100.0),
		)

	case deployCompleteMsg:
		m.done = true
		m.result = msg.result
		return m, nil
	}

	return m, nil
}

func (m deployModel) View() string {
	if m.quitting && !m.done {
		return "\n  Cancelling...\n"
	}

	var s strings.Builder

	// Calculate available width
	width := m.width
	if width < 60 {
		width = 80
	}
	if width > 100 {
		width = 100
	}

	// Header
	header := titleStyle.Render(fmt.Sprintf(" Deploying to %s ", m.deployer.Name()))
	s.WriteString("\n")
	s.WriteString(header)
	s.WriteString("\n\n")

	// Progress bar
	if len(m.events) > 0 {
		lastEvent := m.events[len(m.events)-1]
		percent := lastEvent.Percent
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}

		barView := m.progressBar.ViewAs(float64(percent) / 100.0)
		s.WriteString(progressBarStyle.Render(barView))
		s.WriteString(fmt.Sprintf(" %d%%", percent))
		s.WriteString("\n\n")
	}

	// Event log
	for i, event := range m.events {
		isLast := i == len(m.events)-1 && !m.done

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

		s.WriteString(icon)
		s.WriteString(msgStyle.Render(event.Message))
		s.WriteString("\n")

		// Show command if present (for the active step or errors)
		if event.Command != "" && (isLast || event.IsError) {
			s.WriteString("     ")
			s.WriteString(commandStyle.Render(" " + event.Command))
			s.WriteString("\n")
		}

		// Show detail if present
		if event.Detail != "" && (isLast || event.IsError) {
			s.WriteString("     ")
			s.WriteString(dimStyle.Render(event.Detail))
			s.WriteString("\n")
		}
	}

	// Spinner if still deploying
	if !m.done && len(m.events) > 0 {
		s.WriteString("\n")
		s.WriteString("  ")
		s.WriteString(m.spinner.View())
		s.WriteString(" Working...")
		s.WriteString("\n")
	}

	// Footer
	s.WriteString("\n")
	if m.done {
		if m.result != nil && m.result.Success {
			s.WriteString(dimStyle.Render("  Press Enter to view results"))
		} else {
			s.WriteString(dimStyle.Render("  Press Enter to exit"))
		}
	} else {
		s.WriteString(dimStyle.Render("  Press Ctrl+C to cancel"))
	}
	s.WriteString("\n")

	return s.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
