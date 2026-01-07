package create

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/app"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/deploy"
)

// Ensure app.Tab is used
var _ app.Tab = (*Model)(nil)

// initTargetOptionsPhase initializes the target options phase based on selected target
func (m *Model) initTargetOptionsPhase() {
	switch m.wizard.Data.Target {
	case deploy.TargetMultipass:
		m.initMultipassPhase()
	case deploy.TargetTerragrunt:
		m.initTerragruntPhase()
	case deploy.TargetUSB:
		m.initUSBPhase()
	case deploy.TargetConfigOnly:
		m.initGeneratePhase()
	default:
		// Unknown target - skip initialization
	}
}

// handleTargetOptionsPhase handles input for the target options phase
func (m *Model) handleTargetOptionsPhase(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	switch m.wizard.Data.Target {
	case deploy.TargetMultipass:
		return m.handleMultipassPhase(msg)
	case deploy.TargetTerragrunt:
		return m.handleTerragruntPhase(msg)
	case deploy.TargetUSB:
		return m.handleUSBPhase(msg)
	case deploy.TargetConfigOnly:
		return m.handleGeneratePhase(msg)
	default:
		return m, nil
	}
}

// viewTargetOptionsPhase renders the target options phase
func (m *Model) viewTargetOptionsPhase() string {
	switch m.wizard.Data.Target {
	case deploy.TargetMultipass:
		return m.viewMultipassPhase()
	case deploy.TargetTerragrunt:
		return m.viewTerragruntPhase()
	case deploy.TargetUSB:
		return m.viewUSBPhase()
	case deploy.TargetConfigOnly:
		return m.viewGeneratePhase()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Target Options"))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("No options for this target."))
	return b.String()
}
