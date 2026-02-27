package common

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FormModel struct {
	inputs     []textinput.Model
	focusIndex int
	submitted  bool
}

// NewForm creates a form with the given field placeholders.
func NewForm(fields []string) FormModel {
	inputs := make([]textinput.Model, len(fields))
	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f
		ti.Prompt = f + ": "
		ti.CharLimit = 256
		ti.Width = 30
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
	}
	return FormModel{inputs: inputs}
}

// Init implements tea.Model.
func (m FormModel) Init() tea.Cmd { return textinput.Blink }

// Update handles key events and input updates.
func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.focusIndex < len(m.inputs)-1 {
				m.inputs[m.focusIndex].Blur()
				m.focusIndex++
				m.inputs[m.focusIndex].Focus()
				return m, nil
			}
			// Last field – mark as submitted.
			m.submitted = true
			return m, nil
		case "tab", "shift+tab":
			m.inputs[m.focusIndex].Blur()
			if msg.String() == "tab" {
				m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
			} else {
				m.focusIndex = (m.focusIndex - 1 + len(m.inputs)) % len(m.inputs)
			}
			m.inputs[m.focusIndex].Focus()
			return m, nil
		}
	}

	// Update the currently focused input.
	var cmd tea.Cmd
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	return m, cmd
}

// View renders the form fields.
func (m FormModel) View() string {
	var b strings.Builder
	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteRune('\n')
	}
	if m.submitted {
		b.WriteString("\n[Submitted]")
	}
	return lipgloss.NewStyle().Render(b.String())
}

// Ensure FormModel implements tea.Model.
var _ tea.Model = (*FormModel)(nil)
