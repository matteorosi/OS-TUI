package storage

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"ostui/internal/client"
)

type SnapshotDetailModel struct {
	table      table.Model
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.StorageClient
	snapshotID string
	// JSON view fields
	jsonView     string
	jsonViewport viewport.Model
	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored snapshot for JSON marshaling
	snapshot snapshots.Snapshot
}

type snapshotDetailDataLoadedMsg struct {
	tbl      table.Model
	err      error
	snapshot snapshots.Snapshot
}

// NewSnapshotDetailModel creates a new SnapshotDetailModel for the given snapshot ID.
func NewSnapshotDetailModel(sc client.StorageClient, snapshotID string) SnapshotDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return SnapshotDetailModel{client: sc, loading: true, spinner: s, snapshotID: snapshotID}
}

// Init starts async loading of snapshot details.
func (m SnapshotDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		snapList, err := m.client.ListSnapshots()
		if err != nil {
			return snapshotDetailDataLoadedMsg{err: err}
		}
		var snap *snapshots.Snapshot
		for _, s := range snapList {
			if s.ID == m.snapshotID {
				snap = &s
				break
			}
		}
		if snap == nil {
			return snapshotDetailDataLoadedMsg{err: fmt.Errorf("snapshot %s not found", m.snapshotID)}
		}
		cols := []table.Column{{Title: "Field", Width: 20}, {Title: "Value", Width: 30}, {Title: "Field", Width: 20}, {Title: "Value", Width: 30}}
		rows := []table.Row{{"ID", snap.ID}, {"Name", snap.Name}, {"VolumeID", snap.VolumeID}, {"Size", fmt.Sprintf("%d", snap.Size)}, {"Status", snap.Status}, {"CreatedAt", snap.CreatedAt.Format("2006-01-02 15:04:05")}}
		half := (len(rows) + 1) / 2
		newRows := []table.Row{}
		for i := 0; i < half; i++ {
			left := rows[i]
			var right table.Row
			if i+half < len(rows) {
				right = rows[i+half]
			} else {
				right = table.Row{"", ""}
			}
			newRows = append(newRows, table.Row{left[0], left[1], right[0], right[1]})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(newRows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		return snapshotDetailDataLoadedMsg{tbl: t, snapshot: *snap}
	}
}

// Update handles messages.
func (m SnapshotDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case snapshotDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.snapshot = msg.snapshot
		return m, nil
	case tea.WindowSizeMsg:
		if m.jsonView != "" {
			m.jsonViewport.Width = msg.Width
			m.jsonViewport.Height = msg.Height
			m.jsonViewport.SetContent(m.jsonView)
			return m, nil
		}
		// Adjust table width to fill terminal
		if !m.loading && len(m.table.Columns()) > 0 {
			cols := m.table.Columns()
			if len(cols) > 0 {
				totalWidth := msg.Width - 4
				if totalWidth < 0 {
					totalWidth = msg.Width
				}
				colWidth := totalWidth / len(cols)
				if colWidth < 5 {
					colWidth = 5
				}
				for i := range cols {
					cols[i].Width = colWidth
				}
				m.table.SetColumns(cols)
				m.table.SetWidth(msg.Width)
			}
		}
		return m, nil
	case tea.KeyMsg:
		// If Inspect view is active, handle its keys.
		if m.inspectView != "" {
			if msg.String() == "i" || msg.String() == "esc" {
				m.inspectView = ""
				m.inspectViewport = viewport.Model{}
				return m, nil
			}
			// Forward other keys to viewport for scrolling
			var cmd tea.Cmd
			m.inspectViewport, cmd = m.inspectViewport.Update(msg)
			return m, cmd
		}
		// If JSON view is active, handle its keys.
		if m.jsonView != "" {
			if msg.String() == "y" || msg.String() == "esc" {
				m.jsonView = ""
				m.jsonViewport = viewport.Model{}
				return m, nil
			}
			// ignore other keys while JSON view is active
			return m, nil
		}
		if m.loading || m.err != nil {
			return m, nil
		}
		if msg.String() == "i" {
			// Build inspect view for snapshot.
			content := fmt.Sprintf("=== Snapshot: %s ===\nID: %s\nName: %s\nVolumeID: %s\nSize: %d\nStatus: %s\nCreatedAt: %s", m.snapshot.Name, m.snapshot.ID, m.snapshot.Name, m.snapshot.VolumeID, m.snapshot.Size, m.snapshot.Status, m.snapshot.CreatedAt.Format("2006-01-02 15:04:05"))
			m.inspectView = content
			m.inspectViewport = viewport.New(80, 24)
			m.inspectViewport.SetContent(m.inspectView)
			return m, nil
		}
		if msg.String() == "y" {
			b, err := json.MarshalIndent(m.snapshot, "", "  ")
			if err != nil {
				m.err = err
				return m, nil
			}
			m.jsonView = string(b)
			m.jsonViewport = viewport.New(80, 24)
			m.jsonViewport.SetContent(m.jsonView)
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

// View renders the snapshot detail view.
func (m SnapshotDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.inspectView != "" {
		return fmt.Sprintf("%s\n %3.f%% | [j/k] scroll  [esc] close", m.inspectViewport.View(), m.inspectViewport.ScrollPercent()*100)
	}
	if m.jsonView != "" {
		return fmt.Sprintf("%s\nPress 'y' or 'esc' to close", m.jsonViewport.View())
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to load snapshot: " + m.err.Error()}}
		return table.New(table.WithColumns(cols), table.WithRows(rows)).View()
	}
	return fmt.Sprintf("%s\n[y] json  [i] inspect  [esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m SnapshotDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*SnapshotDetailModel)(nil)
