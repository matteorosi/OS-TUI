package network

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/uiconst"
)

type FloatingIPsModel struct {
	ft      common.FilterableTable
	loading bool
	err     error
	spinner spinner.Model
	client  client.NetworkClient
	// Dynamic sizing
	width  int
	height int
}

type floatingIPsDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// NewFloatingIPsModel creates a new FloatingIPsModel.
func NewFloatingIPsModel(nc client.NetworkClient) FloatingIPsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return FloatingIPsModel{client: nc, loading: true, spinner: s, ft: common.NewFilterableTable(), width: 120, height: 30}
}

// Init starts async loading of floating IPs.
func (m FloatingIPsModel) Init() tea.Cmd {
	return func() tea.Msg {
		fipList, err := m.client.ListFloatingIPs()
		if err != nil {
			return floatingIPsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "FloatingNetworkID", Width: uiconst.ColWidthUUID}, {Title: "FixedIP", Width: uiconst.ColWidthFixed}, {Title: "PortID", Width: uiconst.ColWidthUUID}, {Title: "Status", Width: uiconst.ColWidthStatus}}
		rows := []table.Row{}
		for _, f := range fipList {
			rows = append(rows, table.Row{f.ID, f.FloatingNetworkID, f.FixedIP, f.PortID, f.Status})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return floatingIPsDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages.
func (m FloatingIPsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case floatingIPsDataLoadedMsg:
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
		// Normal navigation
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

// View renders the floating IPs view.
func (m FloatingIPsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: uiconst.ColWidthError}}
		rows := []table.Row{{"Failed to list floating IPs: " + m.err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	return m.ft.View()
}

// updateTableColumns adjusts column widths based on the current width.
func (m *FloatingIPsModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	fnetW := uiconst.ColWidthUUID
	portIDW := uiconst.ColWidthUUID
	statusW := uiconst.ColWidthStatus
	// FixedIP column gets remaining space
	fixedIPW := m.width - idW - fnetW - portIDW - statusW - uiconst.TableHeightOffset
	if fixedIPW < 10 {
		fixedIPW = 10
	}
	m.ft.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "FloatingNetworkID", Width: fnetW}, {Title: "FixedIP", Width: fixedIPW}, {Title: "PortID", Width: portIDW}, {Title: "Status", Width: statusW}})
}

// Ensure FloatingIPsModel implements tea.Model.
// Table returns the underlying table model.
func (m FloatingIPsModel) Table() table.Model { return m.ft.Table }

var _ tea.Model = (*FloatingIPsModel)(nil)
