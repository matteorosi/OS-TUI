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

type RouterModel struct {
	// UI components
	ft         common.FilterableTable // list view table
	ifaceTable table.Model            // detail view table (router interfaces)
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
	mode     string // "list" or "detail"
	routerID string // selected router ID when in detail mode
}

// NewRoutersModel creates a RouterModel ready to load router data.
func NewRoutersModel(nc client.NetworkClient) RouterModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return RouterModel{client: nc, loading: true, spinner: s, ft: common.NewFilterableTable(), mode: "list", width: 120, height: 30}
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
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "Status", Width: uiconst.ColWidthStatus}}
		rows := []table.Row{}
		for _, r := range routers {
			rows = append(rows, table.Row{r.ID, r.Name, fmt.Sprintf("%v", r.Status)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
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
		cols := []table.Column{{Title: "Interface ID", Width: uiconst.ColWidthUUID}, {Title: "Subnet ID", Width: uiconst.ColWidthUUID}}
		rows := []table.Row{}
		for _, i := range ifaces {
			rows = append(rows, table.Row{i.ID, i.NetworkID})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
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
		m.ft.SetTable(msg.tbl, msg.rows)
		m.updateTableColumns()
		m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
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
		if m.ft.Table.Columns() != nil {
			m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
			m.updateTableColumns()
		}
		return m, nil
	case tea.KeyMsg:
		// Global escape handling: return to list view.
		if msg.String() == "esc" && m.mode == "detail" {
			m.mode = "list"
			m.routerID = ""
			m.ifaceTable = table.Model{}
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
					m.routerID = row[0]
					m.loading = true
					return m, m.loadInterfacesCmd(m.routerID)
				}
				return m, nil
			}
			cmd := m.ft.UpdateTable(msg)
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
		return m.ft.View()
	}
	// Detail view – show router interfaces.
	header := fmt.Sprintf("Router %s interfaces (press esc to go back)", m.routerID)
	return fmt.Sprintf("%s\n%s", header, m.ifaceTable.View())
}

// Table returns the primary table (list view) – useful for navigation.
func (m RouterModel) Table() table.Model { return m.ft.Table }

// updateTableColumns adjusts column widths based on the current width.
func (m *RouterModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	statusW := uiconst.ColWidthStatus
	nameW := m.width - idW - statusW - uiconst.TableHeightOffset
	if nameW < 10 {
		nameW = 10
	}
	m.ft.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Status", Width: statusW}})
}

var _ tea.Model = (*RouterModel)(nil)
