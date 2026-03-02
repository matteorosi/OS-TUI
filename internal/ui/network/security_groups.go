package network

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/uiconst"
)

type SecurityGroupsModel struct {
	ft      common.FilterableTable
	loading bool
	err     error
	spinner spinner.Model
	client  client.NetworkClient
	width   int
	height  int
}

type securityGroupsDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// NewSecurityGroupsModel creates a new SecurityGroupsModel.
func NewSecurityGroupsModel(nc client.NetworkClient) SecurityGroupsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return SecurityGroupsModel{client: nc, loading: true, spinner: s, ft: common.NewFilterableTable(), width: 120, height: 30}
}

// Init starts async loading of security groups.
func (m SecurityGroupsModel) Init() tea.Cmd {
	return func() tea.Msg {
		sgList, err := m.client.ListSecurityGroups()
		if err != nil {
			return securityGroupsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "Description", Width: uiconst.ColWidthDescription}, {Title: "Stateful", Width: uiconst.ColWidthStateful}}
		rows := []table.Row{}
		for _, sg := range sgList {
			rows = append(rows, table.Row{sg.ID, sg.Name, sg.Description, fmt.Sprintf("%v", sg.Stateful)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return securityGroupsDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages.
func (m SecurityGroupsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case securityGroupsDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.ft.SetTable(msg.tbl, msg.rows)
		m.updateTableColumns()
		m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.ft.Table.Columns() != nil {
			m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
			m.updateTableColumns()
		}
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}
		if handled, cmd := m.ft.Update(msg); handled {
			return m, cmd
		}
		cmd := m.ft.UpdateTable(msg)
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

// View renders the security groups view.
func (m SecurityGroupsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: uiconst.ColWidthError}}
		rows := []table.Row{{"Failed to list security groups: " + m.err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	return m.ft.View()
}

// updateTableColumns adjusts column widths based on the current width.
func (m *SecurityGroupsModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	statefulW := uiconst.ColWidthStateful
	remaining := m.width - idW - statefulW - 6
	if remaining < 20 {
		remaining = 20
	}
	nameW := remaining / 2
	descW := remaining - nameW
	if nameW < 10 {
		nameW = 10
	}
	if descW < 10 {
		descW = 10
	}
	m.ft.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Description", Width: descW}, {Title: "Stateful", Width: statefulW}})
}

// Table returns the underlying table model.
func (m SecurityGroupsModel) Table() table.Model { return m.ft.Table }

var _ tea.Model = (*SecurityGroupsModel)(nil)
