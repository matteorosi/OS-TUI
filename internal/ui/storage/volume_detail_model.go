package storage

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"ostui/internal/client"
)

type VolumeDetailModel struct {
	table    table.Model
	loading  bool
	err      error
	spinner  spinner.Model
	client   client.StorageClient
	volumeID string
	// JSON view fields
	jsonView     string
	jsonViewport viewport.Model
	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored volume for JSON marshaling
	volume volumes.Volume
}

// ResourceID returns the volume ID.
func (m VolumeDetailModel) ResourceID() string { return m.volumeID }

// ResourceName returns the volume name.
func (m VolumeDetailModel) ResourceName() string { return m.volume.Name }

type volumeDetailDataLoadedMsg struct {
	tbl    table.Model
	err    error
	volume volumes.Volume
}

// NewVolumeDetailModel creates a new VolumeDetailModel for the given volume ID.
func NewVolumeDetailModel(sc client.StorageClient, volumeID string) VolumeDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return VolumeDetailModel{client: sc, loading: true, spinner: s, volumeID: volumeID}
}

// Init starts async loading of volume details.
func (m VolumeDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		vol, err := m.client.GetVolume(m.volumeID)
		if err != nil {
			return volumeDetailDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "Field", Width: 20}, {Title: "Value", Width: 30}, {Title: "Field", Width: 20}, {Title: "Value", Width: 30}}
		rows := []table.Row{{"ID", vol.ID}, {"Name", vol.Name}, {"Size", fmt.Sprintf("%d", vol.Size)}, {"Status", vol.Status}, {"Description", vol.Description}}
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
		return volumeDetailDataLoadedMsg{tbl: t, volume: vol}
	}
}

// Update handles messages.
func (m VolumeDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case volumeDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.volume = msg.volume
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
			// Build inspect view for volume.
			content := fmt.Sprintf("=== Volume: %s ===\nID: %s\nName: %s\nSize: %d\nStatus: %s\nDescription: %s", m.volume.Name, m.volume.ID, m.volume.Name, m.volume.Size, m.volume.Status, m.volume.Description)
			m.inspectView = content
			m.inspectViewport = viewport.New(80, 24)
			m.inspectViewport.SetContent(m.inspectView)
			return m, nil
		}
		if msg.String() == "y" {
			b, err := json.MarshalIndent(m.volume, "", "  ")
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

// View renders the volume detail view.
func (m VolumeDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.jsonView != "" {
		return fmt.Sprintf("%s\nPress 'y' or 'esc' to close", m.jsonViewport.View())
	}
	if m.inspectView != "" {
		return fmt.Sprintf("%s\n %3.f%% | [j/k] scroll  [esc] close", m.inspectViewport.View(), m.inspectViewport.ScrollPercent()*100)
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to load volume: " + m.err.Error()}}
		return table.New(table.WithColumns(cols), table.WithRows(rows)).View()
	}
	return fmt.Sprintf("%s\n[y] json  [i] inspect  [g] graph  [esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m VolumeDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*VolumeDetailModel)(nil)
