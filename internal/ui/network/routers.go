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
	"strings"
)

// RouterModel implements a view that lists routers and, on selection, shows the
// interfaces attached to a router. It mirrors the behaviour of the existing
// NetworksModel (filterable list) but adds a simple detail view.
type RouterModel struct {
	// UI components
	table      table.Model // list view table
	ifaceTable table.Model // detail view table (router interfaces)
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.NetworkClient
	width      int
	height     int

	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored router details for inspect view
	routerName   string
	routerStatus string

	// State management
	mode       string // "list" or "detail"
	routerID   string // selected router ID when in detail mode
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model
}

// NewRoutersModel creates a RouterModel ready to load router data.
func NewRoutersModel(nc client.NetworkClient) RouterModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return RouterModel{client: nc, loading: true, spinner: s, filter: ti, mode: "list", width: 120, height: 30}
}

// routersListMsg is emitted when the list of routers has been fetched.
type routersListMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// routerIfacesMsg is emitted when router interfaces have been fetched.
type routerIfacesMsg struct {
	tbl table.Model
	err error
}

// Init starts the asynchronous loading of routers.
func (m RouterModel) Init() tea.Cmd {
	return func() tea.Msg {
		routers, err := m.client.ListRouters(context.Background())
		if err != nil {
			return routersListMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "Status", Width: 12}}
		rows := []table.Row{}
		for _, r := range routers {
			// The Router type is an alias for gophercloud's routers.Router which has a Status field.
			// Use fmt.Sprintf to safely handle any zero values.
			rows = append(rows, table.Row{r.ID, r.Name, fmt.Sprintf("%v", r.Status)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-6),
		)
		t.SetStyles(table.DefaultStyles())
		return routersListMsg{tbl: t, rows: rows}
	}
}

// loadInterfacesCmd returns a command that fetches interfaces for the given router.
func (m RouterModel) loadInterfacesCmd(routerID string) tea.Cmd {
	return func() tea.Msg {
		ifaces, err := m.client.GetRouterInterfaces(context.Background(), routerID)
		if err != nil {
			return routerIfacesMsg{err: err}
		}
		// Build a simple table: Interface ID and Subnet ID.
		cols := []table.Column{{Title: "Interface ID", Width: 36}, {Title: "Subnet ID", Width: 36}}
		rows := []table.Row{}
		for _, i := range ifaces {
			rows = append(rows, table.Row{i.ID, i.NetworkID})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-6),
		)
		t.SetStyles(table.DefaultStyles())
		return routerIfacesMsg{tbl: t}
	}
}

// Update processes incoming messages and user input.
func (m RouterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case routersListMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.updateTableColumns()
		m.table.SetHeight(m.height - 6)
		m.allRows = msg.rows
		return m, nil
	case routerIfacesMsg:
		// Switch to detail mode after interfaces are loaded.
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.ifaceTable = msg.tbl
		m.mode = "detail"
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
		// Global escape handling: return to list view.
		if msg.String() == "esc" && m.mode == "detail" {
			// Reset to list view.
			m.mode = "list"
			m.routerID = ""
			m.ifaceTable = table.Model{}
			return m, nil
		}
		// If we are still loading or have an error, ignore key input.
		if m.loading || m.err != nil {
			return m, nil
		}
		// Filter handling only in list mode.
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
			// Normal navigation / selection.
			if msg.String() == "enter" {
				// User selected a router – load its interfaces.
				row := m.table.SelectedRow()
				if len(row) > 0 {
					m.routerID = row[0]
					m.loading = true
					return m, m.loadInterfacesCmd(m.routerID)
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}
		// Detail mode – forward key handling to the interface table.
		if m.mode == "detail" {
			var cmd tea.Cmd
			m.ifaceTable, cmd = m.ifaceTable.Update(msg)
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
func (m RouterModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	if m.mode == "list" {
		if m.filterMode {
			filterLine := fmt.Sprintf("Filter: %s", m.filter.View())
			footer := "esc: clear"
			return fmt.Sprintf("%s\n%s\n%s", filterLine, m.table.View(), footer)
		}
		return m.table.View()
	}
	// Detail view – show router interfaces.
	header := fmt.Sprintf("Router %s interfaces (press esc to go back)", m.routerID)
	return fmt.Sprintf("%s\n%s", header, m.ifaceTable.View())
}

// Table returns the primary table (list view) – useful for navigation.
func (m RouterModel) Table() table.Model { return m.table }

// updateTableColumns adjusts column widths based on the current width.
func (m *RouterModel) updateTableColumns() {
	idW := 36
	statusW := 12
	nameW := m.width - idW - statusW - 6
	if nameW < 10 {
		nameW = 10
	}
	m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Status", Width: statusW}})
}

var _ tea.Model = (*RouterModel)(nil)
