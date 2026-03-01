package storage

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
)

type SnapshotsModel struct {
	table   table.Model
	loading bool
	err     error
	spinner spinner.Model
	client  client.StorageClient
	width   int
	height  int
}

type snapshotsDataLoadedMsg struct {
	tbl table.Model
	err error
}

// NewSnapshotsModel creates a new SnapshotsModel.
func NewSnapshotsModel(sc client.StorageClient) SnapshotsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return SnapshotsModel{client: sc, loading: true, spinner: s, width: 120, height: 30}
}

// Init starts async loading of snapshots.
func (m SnapshotsModel) Init() tea.Cmd {
	return func() tea.Msg {
		snapList, err := m.client.ListSnapshots()
		if err != nil {
			return snapshotsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "VolumeID", Width: 36}, {Title: "Size", Width: 6}, {Title: "Status", Width: 12}, {Title: "Created", Width: 20}}
		rows := []table.Row{}
		for _, s := range snapList {
			rows = append(rows, table.Row{s.ID, s.Name, s.VolumeID, fmt.Sprintf("%d", s.Size), s.Status, s.CreatedAt.Format("2006-01-02 15:04:05")})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-6),
		)
		t.SetStyles(table.DefaultStyles())
		return snapshotsDataLoadedMsg{tbl: t}
	}
}

// Update handles messages.
func (m SnapshotsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case snapshotsDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.updateTableColumns()
		m.table.SetHeight(m.height - 6)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.table.Columns() != nil {
			m.table.SetHeight(m.height - 6)
			m.updateTableColumns()
		}
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

// View renders the snapshots view.
func (m SnapshotsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to list snapshots: " + m.err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	return m.table.View()
}

// Ensure SnapshotsModel implements tea.Model.
func (m SnapshotsModel) Table() table.Model { return m.table }

func (m *SnapshotsModel) updateTableColumns() {
	idW := 36
	volIDW := 36
	sizeW := 6
	statusW := 12
	remaining := m.width - idW - volIDW - sizeW - statusW - 6
	if remaining < 20 {
		remaining = 20
	}
	nameW := remaining / 2
	createdW := remaining - nameW
	if nameW < 10 {
		nameW = 10
	}
	if createdW < 10 {
		createdW = 10
	}
	m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "VolumeID", Width: volIDW}, {Title: "Size", Width: sizeW}, {Title: "Status", Width: statusW}, {Title: "Created", Width: createdW}})
}

var _ tea.Model = (*SnapshotsModel)(nil)
