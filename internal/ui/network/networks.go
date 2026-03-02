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

// NetworksModel implements a subview for listing networks.
type NetworksModel struct {
	ft      common.FilterableTable
	loading bool
	err     error
	spinner spinner.Model
	client  client.NetworkClient
	width   int
	height  int
}

// NewNetworksModel creates a new NetworksModel with the given network client.
func NewNetworksModel(nc client.NetworkClient) NetworksModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return NetworksModel{client: nc, loading: true, spinner: s, ft: common.NewFilterableTable(), width: 120, height: 30}
}

type dataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts the async data loading.
func (m NetworksModel) Init() tea.Cmd {
	return func() tea.Msg {
		netList, err := m.client.ListNetworks()
		if err != nil {
			return dataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "Status", Width: uiconst.ColWidthStatus}}
		rows := []table.Row{}
		for _, n := range netList {
			rows = append(rows, table.Row{n.ID, n.Name, n.Status})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return dataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages for the model.
func (m NetworksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dataLoadedMsg:
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
		if !m.loading {
			m.updateTableColumns()
			m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
		}
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}
		if handled, cmd := m.ft.Update(msg); handled {
			return m, cmd
		}
		var cmd tea.Cmd
		m.ft.Table, cmd = m.ft.Table.Update(msg)
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

// View renders the appropriate UI based on state.
func (m NetworksModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return m.ft.View()
}

// Table returns the underlying table model.
func (m NetworksModel) Table() table.Model { return m.ft.Table }

func (m *NetworksModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	statusW := uiconst.ColWidthStatus
	nameW := m.width - idW - statusW - 6
	if nameW < 10 {
		nameW = 10
	}
	m.ft.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Status", Width: statusW}})
}

var _ tea.Model = (*NetworksModel)(nil)
