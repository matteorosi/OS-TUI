package network

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
)

// PortDetailModel displays detailed information for a single network port.
// It follows the same pattern as other detail models.
type PortDetailModel struct {
	table   table.Model
	loading bool
	err     error
	spinner spinner.Model
	client  client.NetworkClient
	portID  string
}

type portDetailDataLoadedMsg struct {
	tbl table.Model
	err error
}

// NewPortDetailModel creates a new PortDetailModel for the given port ID.
func NewPortDetailModel(nc client.NetworkClient, portID string) PortDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return PortDetailModel{client: nc, loading: true, spinner: s, portID: portID}
}

// Init starts async loading of port details.
func (m PortDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		p, err := m.client.GetPort(context.Background(), m.portID)
		if err != nil {
			return portDetailDataLoadedMsg{err: err}
		}
		// FixedIPs: format as a comma‑separated list of "subnetID:IP"
		fixedIPs := ""
		if len(p.FixedIPs) > 0 {
			parts := []string{}
			for _, ip := range p.FixedIPs {
				parts = append(parts, fmt.Sprintf("%s:%s", ip.SubnetID, ip.IPAddress))
			}
			fixedIPs = fmt.Sprintf("%s", fmt.Sprint(parts))
		}
		cols := []table.Column{{Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValue}}
		rows := []table.Row{{"ID", p.ID}, {"Name", p.Name}, {"Status", fmt.Sprintf("%v", p.Status)}, {"NetworkID", p.NetworkID}, {"MACAddress", p.MACAddress}, {"DeviceOwner", p.DeviceOwner}, {"FixedIPs", fixedIPs}}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		return portDetailDataLoadedMsg{tbl: t}
	}
}

// Update handles messages.
func (m PortDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case portDetailDataLoadedMsg:
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

// View renders the port detail view.
func (m PortDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return fmt.Sprintf("%s\n[esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m PortDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*PortDetailModel)(nil)
