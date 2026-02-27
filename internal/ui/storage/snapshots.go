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
}

type snapshotsDataLoadedMsg struct {
	tbl table.Model
	err error
}

// NewSnapshotsModel creates a new SnapshotsModel.
func NewSnapshotsModel(sc client.StorageClient) SnapshotsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return SnapshotsModel{client: sc, loading: true, spinner: s}
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
			table.WithHeight(10),
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

var _ tea.Model = (*SnapshotsModel)(nil)
