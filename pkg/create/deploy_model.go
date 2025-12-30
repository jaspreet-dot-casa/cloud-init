package create

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// Note: deployProgressMsg and deployCompleteMsg are defined in messages.go

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
