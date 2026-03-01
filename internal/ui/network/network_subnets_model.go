package network

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"strings"
)

type NetworkSubnetsModel struct {
	table      table.Model
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.NetworkClient
	networkID  string
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model
	width      int
	height     int
}

// ResourceID returns the network ID.
func (m NetworkSubnetsModel) ResourceID() string { return m.networkID }

// ResourceName returns the network ID (used as name).
func (m NetworkSubnetsModel) ResourceName() string { return m.networkID }

type networkSubnetsDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// NewNetworkSubnetsModel creates a new NetworkSubnetsModel for the given network ID.
func NewNetworkSubnetsModel(nc client.NetworkClient, networkID string) NetworkSubnetsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return NetworkSubnetsModel{client: nc, loading: true, spinner: s, networkID: networkID, filter: ti, width: 120, height: 30}
}

// Init starts async loading of subnets for the specified network.
func (m NetworkSubnetsModel) Init() tea.Cmd {
	return func() tea.Msg {
		subList, err := m.client.ListSubnets()
		if err != nil {
			return networkSubnetsDataLoadedMsg{err: err}
		}
		// Filter subnets belonging to the network.
		cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "CIDR", Width: 20}, {Title: "IPVer", Width: 6}}
		rows := []table.Row{}
		for _, s := range subList {
			if s.NetworkID == m.networkID {
				rows = append(rows, table.Row{s.ID, s.Name, s.CIDR, fmt.Sprintf("%d", s.IPVersion)})
			}
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-6),
		)
		t.SetStyles(table.DefaultStyles())
		return networkSubnetsDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages.
func (m NetworkSubnetsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case networkSubnetsDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.allRows = msg.rows
		m.updateTableColumns()
		m.table.SetHeight(m.height - 6)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.loading {
			m.updateTableColumns()
			m.table.SetHeight(m.height - 6)
		}
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}
		// Filter mode handling
		if !m.filterMode && msg.String() == "/" {
			m.filterMode = true
			m.filter.Focus()
			return m, textinput.Blink
		}
		if m.filterMode && msg.String() == "esc" {
			// clear filter
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

// View renders the subnets view.
func (m NetworkSubnetsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to list subnets: " + m.err.Error()}}
		return table.New(table.WithColumns(cols), table.WithRows(rows)).View()
	}
	return fmt.Sprintf("%s\n[g] graph  [esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m NetworkSubnetsModel) Table() table.Model { return m.table }

func (m *NetworkSubnetsModel) updateTableColumns() {
	if len(m.table.Columns()) > 0 {
		// Fixed widths
		idW := 36
		cidrW := 20
		ipverW := 6
		// Remaining width for Name column
		nameW := m.width - idW - cidrW - ipverW - 6
		if nameW < 10 {
			nameW = 10
		}
		m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "CIDR", Width: cidrW}, {Title: "IPVer", Width: ipverW}})
	}
}

var _ tea.Model = (*NetworkSubnetsModel)(nil)
