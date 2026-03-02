package network

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/uiconst"
)

type PortsModel struct {
	ft      common.FilterableTable
	loading bool
	err     error
	spinner spinner.Model
	client  client.NetworkClient
	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored port for inspect view
	port client.Port
	// State management
	mode   string // "list" or "detail"
	portID string // selected port ID for detail view
	// Detail view table
	detailTable table.Model
	// Dynamic sizing
	width  int
	height int
}

// NewPortsModel creates a PortsModel ready to load port data.
func NewPortsModel(nc client.NetworkClient) PortsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return PortsModel{client: nc, loading: true, spinner: s, ft: common.NewFilterableTable(), mode: "list", width: 120, height: 30}
}

type portsListMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

type portDetailMsg struct {
	tbl  table.Model
	err  error
	port client.Port
}

// Init starts the asynchronous loading of ports.
func (m PortsModel) Init() tea.Cmd {
	return func() tea.Msg {
		ports, err := m.client.ListPorts(context.Background())
		if err != nil {
			return portsListMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "Network ID", Width: uiconst.ColWidthUUID}, {Title: "Status", Width: uiconst.ColWidthStatus}}
		rows := []table.Row{}
		for _, p := range ports {
			rows = append(rows, table.Row{p.ID, p.Name, p.NetworkID, fmt.Sprintf("%v", p.Status)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return portsListMsg{tbl: t, rows: rows}
	}
}

// loadPortDetailCmd returns a command that fetches details for the given port.
func (m PortsModel) loadPortDetailCmd(portID string) tea.Cmd {
	return func() tea.Msg {
		p, err := m.client.GetPort(context.Background(), portID)
		if err != nil {
			return portDetailMsg{err: err}
		}
		cols := []table.Column{{Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValue}}
		rows := []table.Row{{"ID", p.ID}, {"Name", p.Name}, {"Network ID", p.NetworkID}, {"Status", fmt.Sprintf("%v", p.Status)}, {"MAC Address", p.MACAddress}, {"Device ID", p.DeviceID}}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		return portDetailMsg{tbl: t, port: *p}
	}
}

// Update processes incoming messages and user input.
func (m PortsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case portsListMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.ft.SetTable(msg.tbl, msg.rows)
		m.updateTableColumns()
		m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
		return m, nil
	case portDetailMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.detailTable = msg.tbl
		m.port = msg.port
		m.mode = "detail"
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
		// Inspect view handling
		if m.inspectView != "" {
			if msg.String() == "i" || msg.String() == "esc" {
				m.inspectView = ""
				m.inspectViewport = viewport.Model{}
				return m, nil
			}
			var cmd tea.Cmd
			m.inspectViewport, cmd = m.inspectViewport.Update(msg)
			return m, cmd
		}
		// Global escape handling from detail view
		if msg.String() == "esc" && m.mode == "detail" {
			m.mode = "list"
			m.portID = ""
			m.detailTable = table.Model{}
			m.port = client.Port{}
			return m, nil
		}
		if m.loading || m.err != nil {
			return m, nil
		}
		if m.mode == "list" {
			if handled, cmd := m.ft.Update(msg); handled {
				return m, cmd
			}
			if msg.String() == "enter" {
				row := m.ft.SelectedRow()
				if len(row) > 0 {
					m.portID = row[0]
					m.loading = true
					return m, m.loadPortDetailCmd(m.portID)
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.ft.Table, cmd = m.ft.Table.Update(msg)
			return m, cmd
		}
		if m.mode == "detail" {
			if msg.String() == "i" {
				content := fmt.Sprintf("=== Port: %s ===\nID: %s\nName: %s\nNetworkID: %s\nStatus: %v\nMACAddress: %s\nDeviceID: %s",
					m.port.Name, m.port.ID, m.port.Name, m.port.NetworkID, m.port.Status, m.port.MACAddress, m.port.DeviceID)
				m.inspectView = content
				m.inspectViewport = viewport.New(80, 24)
				m.inspectViewport.SetContent(m.inspectView)
				return m, nil
			}
			var cmd tea.Cmd
			m.detailTable, cmd = m.detailTable.Update(msg)
			return m, cmd
		}
	default:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// View renders the appropriate UI based on the current mode.
func (m PortsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	if m.inspectView != "" {
		return fmt.Sprintf("%s\n %3.f%% | [j/k] scroll  [esc] close", m.inspectViewport.View(), m.inspectViewport.ScrollPercent()*100)
	}
	if m.mode == "list" {
		return m.ft.View()
	}
	// Detail view
	header := fmt.Sprintf("Port %s details (press esc to go back)", m.portID)
	return fmt.Sprintf("%s\n%s", header, m.detailTable.View())
}

func (m *PortsModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	netIDW := uiconst.ColWidthUUID
	statusW := uiconst.ColWidthStatus
	nameW := m.width - idW - netIDW - statusW - uiconst.TableHeightOffset
	if nameW < 10 {
		nameW = 10
	}
	m.ft.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Network ID", Width: netIDW}, {Title: "Status", Width: statusW}})
}

// Table returns the primary table (list view) – useful for navigation.
func (m PortsModel) Table() table.Model { return m.ft.Table }

var _ tea.Model = (*PortsModel)(nil)
