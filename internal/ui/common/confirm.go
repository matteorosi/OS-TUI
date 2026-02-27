package common

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmItem struct {
	choice string
}

func (c confirmItem) Title() string       { return c.choice }
func (c confirmItem) Description() string { return "" }
func (c confirmItem) FilterValue() string { return c.choice }

type ConfirmModel struct {
	list   list.Model
	result string // "Yes" or "No"
	done   bool
}

// NewConfirm creates a confirm dialog with a message.
func NewConfirm(message string) ConfirmModel {
	items := []list.Item{
		confirmItem{choice: "Yes"},
		confirmItem{choice: "No"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 20, 5)
	l.Title = message
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)
	return ConfirmModel{list: l}
}

// Init implements tea.Model.
func (m ConfirmModel) Init() tea.Cmd { return nil }

// Update forwards messages to the list and captures selection.
func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.list.SelectedItem().(confirmItem); ok {
				m.result = i.choice
				m.done = true
				return m, nil
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the confirm dialog.
func (m ConfirmModel) View() string {
	if m.done {
		return "You selected: " + m.result
	}
	return m.list.View()
}

// Ensure ConfirmModel implements tea.Model.
var _ tea.Model = (*ConfirmModel)(nil)
