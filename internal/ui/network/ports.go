package network

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
	"strings"
)

// PortsModel implements a view that lists ports and shows a read‑only detail view for a selected port.
type PortsModel struct {
	// UI components
	table       table.Model // list view
	detailTable table.Model // detail view
	loading     bool
	err         error
	spinner     spinner.Model
	client      client.NetworkClient

	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored port for inspect view
	port client.Port

	// State management
	mode       string // "list" or "detail"
	portID     string // selected port ID for detail view
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model

	// Dynamic sizing
	width  int
	height int
}

// NewPortsModel creates a PortsModel ready to load port data.
func NewPortsModel(nc client.NetworkClient) PortsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return PortsModel{client: nc, loading: true, spinner: s, filter: ti, mode: "list", width: 120, height: 30}
}

// portsListMsg is emitted when the list of ports has been fetched.
type portsListMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// portDetailMsg is emitted when a port's details have been fetched.
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
		m.table = msg.tbl
		m.allRows = msg.rows
		m.updateTableColumns()
		m.table.SetHeight(m.height - uiconst.TableHeightOffset)
		return m, nil
	case portDetailMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.detailTable = msg.tbl
		m.port = msg.port // store the full port for inspect view
		m.mode = "detail"
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.table.Columns() != nil {
			m.table.SetHeight(m.height - uiconst.TableHeightOffset)
			m.updateTableColumns()
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
		// Global escape handling: return to list view from detail.
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
			if !m.filterMode && msg.String() == "/" {
				m.filterMode = true
				m.filter.Focus()
				return m, textinput.Blink
			}
			if m.filterMode && msg.String() == "esc" {
				m.filterMode = false
				m.filter.Blur()
				m.filter.SetValue("")
				m.table.SetRows(m.allRows)
				return m, nil
			}
			if m.filterMode {
				var cmd tea.Cmd
				m.filter, cmd = m.filter.Update(msg)
				filterVal := m.filter.Value()
				if filterVal == "" {
					m.table.SetRows(m.allRows)
				} else {
					lower := strings.ToLower(filterVal)
					filtered := []table.Row{}
					for _, r := range m.allRows {
						for _, c := range r {
							if strings.Contains(strings.ToLower(c), lower) {
								filtered = append(filtered, r)
								break
							}
						}
					}
					m.table.SetRows(filtered)
				}
				return m, cmd
			}
			if msg.String() == "enter" {
				row := m.table.SelectedRow()
				if len(row) > 0 {
					m.portID = row[0]
					m.loading = true
					return m, m.loadPortDetailCmd(m.portID)
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}
		if m.mode == "detail" {
			if msg.String() == "i" {
				// Build inspect view for the selected port.
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
		if m.filterMode {
			filterLine := fmt.Sprintf("Filter: %s", m.filter.View())
			footer := "esc: clear"
			return fmt.Sprintf("%s\n%s\n%s", filterLine, m.table.View(), footer)
		}
		return m.table.View()
	}
	// Detail view
	header := fmt.Sprintf("Port %s details (press esc to go back)", m.portID)
	return fmt.Sprintf("%s\n%s", header, m.detailTable.View())
}

// updateTableColumns adjusts column widths based on the current width.
func (m *PortsModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	netIDW := uiconst.ColWidthUUID
	statusW := uiconst.ColWidthStatus
	nameW := m.width - idW - netIDW - statusW - uiconst.TableHeightOffset
	if nameW < 10 {
		nameW = 10
	}
	m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Network ID", Width: netIDW}, {Title: "Status", Width: statusW}})
}

// Table returns the primary table (list view) – useful for navigation.
func (m PortsModel) Table() table.Model { return m.table }

var _ tea.Model = (*PortsModel)(nil)
