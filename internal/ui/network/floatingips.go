package network

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"strings"
)

type FloatingIPsModel struct {
	table      table.Model
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.NetworkClient
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model

	// Dynamic sizing
	width  int
	height int
}

type floatingIPsDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// NewFloatingIPsModel creates a new FloatingIPsModel.
func NewFloatingIPsModel(nc client.NetworkClient) FloatingIPsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return FloatingIPsModel{client: nc, loading: true, spinner: s, filter: ti, width: 120, height: 30}
}

// Init starts async loading of floating IPs.
func (m FloatingIPsModel) Init() tea.Cmd {
	return func() tea.Msg {
		fipList, err := m.client.ListFloatingIPs()
		if err != nil {
			return floatingIPsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: 36}, {Title: "FloatingNetworkID", Width: 36}, {Title: "FixedIP", Width: 20}, {Title: "PortID", Width: 36}, {Title: "Status", Width: 12}}
		rows := []table.Row{}
		for _, f := range fipList {
			rows = append(rows, table.Row{f.ID, f.FloatingNetworkID, f.FixedIP, f.PortID, f.Status})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-6),
		)
		t.SetStyles(table.DefaultStyles())
		return floatingIPsDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages.
func (m FloatingIPsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case floatingIPsDataLoadedMsg:
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
		if m.table.Columns() != nil {
			m.table.SetHeight(m.height - 6)
			m.updateTableColumns()
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

// View renders the floating IPs view.
func (m FloatingIPsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to list floating IPs: " + m.err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	if m.filterMode {
		filterLine := fmt.Sprintf("Filter: %s", m.filter.View())
		footer := "esc: clear"
		return fmt.Sprintf("%s\n%s\n%s", filterLine, m.table.View(), footer)
	}
	return m.table.View()
}

// updateTableColumns adjusts column widths based on the current width.
func (m *FloatingIPsModel) updateTableColumns() {
	idW := 36
	fnetW := 36
	portIDW := 36
	statusW := 12
	// FixedIP column gets remaining space
	fixedIPW := m.width - idW - fnetW - portIDW - statusW - 6
	if fixedIPW < 10 {
		fixedIPW = 10
	}
	m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "FloatingNetworkID", Width: fnetW}, {Title: "FixedIP", Width: fixedIPW}, {Title: "PortID", Width: portIDW}, {Title: "Status", Width: statusW}})
}

// Ensure FloatingIPsModel implements tea.Model.
// Table returns the underlying table model.
func (m FloatingIPsModel) Table() table.Model { return m.table }

var _ tea.Model = (*FloatingIPsModel)(nil)
