package identity

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
)

type UsersModel struct {
	table   table.Model
	loading bool
	err     error
	spinner spinner.Model
	client  client.IdentityClient
	filter  textinput.Model
}

type usersDataLoadedMsg struct {
	tbl table.Model
	err error
}

// NewUsersModel creates a new UsersModel.
func NewUsersModel(ic client.IdentityClient) UsersModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return UsersModel{client: ic, loading: true, spinner: s, filter: ti}
}

// Init starts async loading.
func (m UsersModel) Init() tea.Cmd {
	return func() tea.Msg {
		userList, err := m.client.ListUsers()
		if err != nil {
			return usersDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "Domain ID", Width: 20}, {Title: "Enabled", Width: 8}}
		rows := []table.Row{}
		for _, u := range userList {
			rows = append(rows, table.Row{u.ID, u.Name, u.DomainID, fmt.Sprintf("%t", u.Enabled)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(10),
		)
		t.SetStyles(table.DefaultStyles())
		return usersDataLoadedMsg{tbl: t}
	}
}

// Update handles messages.
func (m UsersModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case usersDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		return m, nil
	case tea.WindowSizeMsg:
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	default:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// View renders.
func (m UsersModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to list users: " + m.err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	return m.table.View()
}

// Ensure UsersModel implements tea.Model.
func (m UsersModel) Table() table.Model { return m.table }

var _ tea.Model = (*UsersModel)(nil)
