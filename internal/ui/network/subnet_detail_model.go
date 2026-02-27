package network

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
)

// SubnetDetailModel displays detailed information for a single network subnet.
// It follows the same pattern as other detail models.
type SubnetDetailModel struct {
	table    table.Model
	loading  bool
	err      error
	spinner  spinner.Model
	client   client.NetworkClient
	subnetID string
}

type subnetDetailDataLoadedMsg struct {
	tbl table.Model
	err error
}

// NewSubnetDetailModel creates a new SubnetDetailModel for the given subnet ID.
func NewSubnetDetailModel(nc client.NetworkClient, subnetID string) SubnetDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return SubnetDetailModel{client: nc, loading: true, spinner: s, subnetID: subnetID}
}

// Init starts async loading of subnet details.
func (m SubnetDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		s, err := m.client.GetSubnet(context.Background(), m.subnetID)
		if err != nil {
			return subnetDetailDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "Field", Width: 20}, {Title: "Value", Width: 60}}
		rows := []table.Row{{"ID", s.ID}, {"Name", s.Name}, {"NetworkID", s.NetworkID}, {"CIDR", s.CIDR}, {"IPVersion", fmt.Sprintf("%d", s.IPVersion)}, {"GatewayIP", s.GatewayIP}, {"EnableDHCP", fmt.Sprintf("%v", s.EnableDHCP)}}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		return subnetDetailDataLoadedMsg{tbl: t}
	}
}

// Update handles messages.
func (m SubnetDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case subnetDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		return m, nil
	case tea.WindowSizeMsg:
		if !m.loading && len(m.table.Columns()) > 0 {
			cols := m.table.Columns()
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

// View renders the subnet detail view.
func (m SubnetDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return fmt.Sprintf("%s\n[esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m SubnetDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*SubnetDetailModel)(nil)
