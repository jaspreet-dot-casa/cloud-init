// Package terraform provides Terraform deployment functionality.
package terraform

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// confirmModel is a simple yes/no confirmation dialog using bubbles.
type confirmModel struct {
	question  string
	confirmed bool
	done      bool
}

// Init implements tea.Model.
func (m confirmModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "y", "Y":
			m.confirmed = true
			m.done = true
			return m, tea.Quit
		case "n", "N", "esc", "q", "ctrl+c":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m confirmModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	return fmt.Sprintf("\n%s\n%s ",
		titleStyle.Render(m.question),
		hintStyle.Render("[y/N]"),
	)
}

// runConfirm displays a confirmation prompt and returns the user's choice.
func runConfirm(question string) (bool, error) {
	m := confirmModel{question: question}
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return false, fmt.Errorf("confirm dialog failed: %w", err)
	}
	return result.(confirmModel).confirmed, nil
}
