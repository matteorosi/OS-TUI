package common

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type TableModel struct {
	table table.Model
}

// NewTable creates a table component with given columns and rows.
func NewTable(columns []table.Column, rows []table.Row) TableModel {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	// Apply a simple header style.
	t.SetStyles(table.DefaultStyles()) // Use default styles
	return TableModel{table: t}
}

// Init implements tea.Model.
func (m TableModel) Init() tea.Cmd { return nil }

// Update forwards messages to the underlying table.
func (m TableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the table.
func (m TableModel) View() string { return m.table.View() }

// Ensure TableModel implements tea.Model.
var _ tea.Model = (*TableModel)(nil)
