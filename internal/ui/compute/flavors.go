package compute

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/uiconst"
)

// FlavorsModel implements a subview for listing OpenStack compute flavors.
// It follows the same pattern as InstancesModel: async loading, spinner while
// loading, optional filter mode, and a table view once data is available.
type FlavorsModel struct {
	ft      common.FilterableTable
	loading bool
	err     error
	spinner spinner.Model
	client  client.ComputeClient
	// Dynamic sizing
	width  int
	height int
}

// NewFlavorsModel creates a new FlavorsModel with the given compute client.
func NewFlavorsModel(cc client.ComputeClient) FlavorsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return FlavorsModel{client: cc, loading: true, spinner: s, ft: common.NewFilterableTable(), width: 120, height: 30}
}

type flavorsDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts the async loading of flavor data.
func (m FlavorsModel) Init() tea.Cmd {
	return func() tea.Msg {
		flavorList, err := m.client.ListFlavors()
		if err != nil {
			return flavorsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "VCPUs", Width: uiconst.ColWidthProtocol}, {Title: "RAM (MB)", Width: uiconst.ColWidthEnabled}, {Title: "Disk (GB)", Width: uiconst.ColWidthEnabled}}
		rows := []table.Row{}
		for _, f := range flavorList {
			rows = append(rows, table.Row{f.ID, f.Name, fmt.Sprintf("%d", f.VCPUs), fmt.Sprintf("%d", f.RAM), fmt.Sprintf("%d", f.Disk)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return flavorsDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages for the model, including data load, window resize,
// and key handling for filtering.
func (m FlavorsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case flavorsDataLoadedMsg:
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
		// Normal table navigation.
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

// View renders the model: spinner while loading, error if any, filter UI or the table.
func (m FlavorsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return m.ft.View()
}

// updateTableColumns adjusts column widths based on the current width.
func (m *FlavorsModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	vcpusW := uiconst.ColWidthProtocol
	ramW := uiconst.ColWidthEnabled
	diskW := uiconst.ColWidthEnabled
	// Name column gets remaining space.
	nameW := m.width - idW - vcpusW - ramW - diskW - uiconst.TableHeightOffset
	if nameW < 10 {
		nameW = 10
	}
	m.ft.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "VCPUs", Width: vcpusW}, {Title: "RAM (MB)", Width: ramW}, {Title: "Disk (GB)", Width: diskW}})
}

// Table returns the underlying table model for external callers.
func (m FlavorsModel) Table() table.Model { return m.ft.Table }

var _ tea.Model = (*FlavorsModel)(nil)
