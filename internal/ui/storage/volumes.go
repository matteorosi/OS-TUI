package storage

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/uiconst"
)

type VolumesModel struct {
	ft      common.FilterableTable
	loading bool
	err     error
	spinner spinner.Model
	client  client.StorageClient
	width   int
	height  int
}

type dataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// NewVolumesModel creates a new VolumesModel with the given storage client.
func NewVolumesModel(sc client.StorageClient) VolumesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return VolumesModel{client: sc, loading: true, spinner: s, ft: common.NewFilterableTable(), width: 120, height: 30}
}

// Init starts the async data loading.
func (m VolumesModel) Init() tea.Cmd {
	return func() tea.Msg {
		volList, err := m.client.ListVolumes()
		if err != nil {
			return dataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "Size", Width: uiconst.ColWidthSize}, {Title: "Status", Width: uiconst.ColWidthStatus}}
		rows := []table.Row{}
		for _, v := range volList {
			rows = append(rows, table.Row{v.ID, v.Name, fmt.Sprintf("%d", v.Size), v.Status})
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
func (m VolumesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// View renders the appropriate UI based on state.
func (m VolumesModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return m.ft.View()
}

// updateTableColumns adjusts column widths based on the current width.
func (m *VolumesModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	sizeW := uiconst.ColWidthSize
	statusW := uiconst.ColWidthStatus
	nameW := m.width - idW - sizeW - statusW - uiconst.TableHeightOffset
	if nameW < 10 {
		nameW = 10
	}
	m.ft.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Size", Width: sizeW}, {Title: "Status", Width: statusW}})
}

// Table returns the underlying table model.
func (m VolumesModel) Table() table.Model { return m.ft.Table }

var _ tea.Model = (*VolumesModel)(nil)
