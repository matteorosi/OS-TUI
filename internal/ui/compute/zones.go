package compute

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
)

// ZonesModel implements a subview for listing OpenStack availability zones.
type ZonesModel struct {
	table   table.Model
	loading bool
	err     error
	spinner spinner.Model
	client  client.ComputeClient
}

// NewZonesModel creates a new ZonesModel.
func NewZonesModel(cc client.ComputeClient) ZonesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return ZonesModel{client: cc, loading: true, spinner: s}
}

type zonesDataLoadedMsg struct {
	tbl table.Model
	err error
}

// Init starts async loading of availability zones.
func (m ZonesModel) Init() tea.Cmd {
	return func() tea.Msg {
		zones, err := m.client.ListAvailabilityZones(context.Background())
		if err != nil {
			return zonesDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "Name", Width: 20}, {Title: "Available", Width: 10}}
		rows := []table.Row{}
		for _, z := range zones {
			rows = append(rows, table.Row{z.ZoneName, fmt.Sprintf("%t", z.ZoneState.Available)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(10),
		)
		t.SetStyles(table.DefaultStyles())
		return zonesDataLoadedMsg{tbl: t}
	}
}

// Update handles messages for the model.
func (m ZonesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case zonesDataLoadedMsg:
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

// View renders the zones view.
func (m ZonesModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return m.table.View()
}

// Table returns the underlying table model.
func (m ZonesModel) Table() table.Model { return m.table }

var _ tea.Model = (*ZonesModel)(nil)
