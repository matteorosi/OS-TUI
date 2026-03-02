package common

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

// ActionItem represents a single action.
type ActionItem struct {
	Key         string // shortcut key
	Label       string // displayed label
	Destructive bool   // if true, requires confirmation
}

// ActionMenuModel shows an overlay with the available actions.
type ActionMenuModel struct {
	actions  []ActionItem
	cursor   int
	title    string
	done     bool
	selected *ActionItem
}

// NewActionMenu creates a new ActionMenuModel with a title and actions.
func NewActionMenu(title string, actions []ActionItem) ActionMenuModel {
	return ActionMenuModel{actions: actions, title: title}
}

// Init implements tea.Model.
func (m ActionMenuModel) Init() tea.Cmd { return nil }

// Update handles key navigation and selection.
func (m ActionMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.actions)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if len(m.actions) > 0 {
				m.selected = &m.actions[m.cursor]
				m.done = true
			}
		case "esc":
			// Cancel without selection.
			m.done = true
		}
	}
	return m, nil
}

// View renders the action menu as a centered overlay with a border.
func (m ActionMenuModel) View() string {
	if m.done {
		// When done, render nothing (the caller will overlay base view).
		return ""
	}
	var b strings.Builder
	// Title (bold)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n")
	// List actions
	for i, a := range m.actions {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		// Style label, red for destructive.
		label := a.Label
		if a.Destructive {
			label = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(label)
		}
		line := fmt.Sprintf("%s%s) %s", cursor, a.Key, label)
		b.WriteString(line)
		b.WriteString("\n")
	}
	// Wrap with border.
	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2)
	return border.Render(b.String())
}

// Selected returns the chosen action after the menu is done.
func (m ActionMenuModel) Selected() *ActionItem { return m.selected }

// IsDone indicates whether the menu has been closed (selection or cancel).
func (m ActionMenuModel) IsDone() bool { return m.done }

// Ensure ActionMenuModel implements tea.Model.
var _ tea.Model = (*ActionMenuModel)(nil)
