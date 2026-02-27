package network

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
)

type SubnetsModel struct {
	table   table.Model
	loading bool
	err     error
	spinner spinner.Model
	client  client.NetworkClient
	filter  textinput.Model
}

type subnetsDataLoadedMsg struct {
	tbl table.Model
	err error
}

// NewSubnetsModel creates a new SubnetsModel.
func NewSubnetsModel(nc client.NetworkClient) SubnetsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return SubnetsModel{client: nc, loading: true, spinner: s, filter: ti}
}

// Init starts async loading of subnets.
func (m SubnetsModel) Init() tea.Cmd {
	return func() tea.Msg {
		subList, err := m.client.ListSubnets()
		if err != nil {
			return subnetsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "CIDR", Width: 20}, {Title: "IPVer", Width: 6}}
		rows := []table.Row{}
		for _, s := range subList {
			rows = append(rows, table.Row{s.ID, s.Name, s.CIDR, fmt.Sprintf("%d", s.IPVersion)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(10),
		)
		t.SetStyles(table.DefaultStyles())
		return subnetsDataLoadedMsg{tbl: t}
	}
}

// Update handles messages.
func (m SubnetsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case subnetsDataLoadedMsg:
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

// View renders the subnets view.
func (m SubnetsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to list subnets: " + m.err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	return m.table.View()
}

// Ensure SubnetsModel implements tea.Model.
// Table returns the underlying table model.
func (m SubnetsModel) Table() table.Model { return m.table }

var _ tea.Model = (*SubnetsModel)(nil)
