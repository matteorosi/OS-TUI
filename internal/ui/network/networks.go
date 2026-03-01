package network

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
	"strings"
)

// NetworksModel implements a subview for listing networks.
type NetworksModel struct {
	table      table.Model
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.NetworkClient
	width      int
	height     int
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model
}

// NewNetworksModel creates a new NetworksModel with the given network client.
func NewNetworksModel(nc client.NetworkClient) NetworksModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return NetworksModel{client: nc, loading: true, spinner: s, filter: ti, width: 120, height: 30}
}

// dataLoadedMsg is sent when network data has been fetched.
type dataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts the async data loading.
func (m NetworksModel) Init() tea.Cmd {
	return func() tea.Msg {
		netList, err := m.client.ListNetworks()
		if err != nil {
			return dataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "Status", Width: uiconst.ColWidthStatus}}
		rows := []table.Row{}
		for _, n := range netList {
			rows = append(rows, table.Row{n.ID, n.Name, n.Status})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return dataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages for the model.
func (m NetworksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.updateTableColumns()
		m.table.SetHeight(m.height - uiconst.TableHeightOffset)
		m.allRows = msg.rows
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
		if m.loading || m.err != nil {
			// ignore key input while loading or on error
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
		// Normal table navigation
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

// View renders the appropriate UI based on state.
func (m NetworksModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	if m.filterMode {
		filterLine := fmt.Sprintf("Filter: %s", m.filter.View())
		footer := "esc: clear"
		return fmt.Sprintf("%s\n%s\n%s", filterLine, m.table.View(), footer)
	}
	return m.table.View()
}

// Ensure NetworksModel implements tea.Model.
// Table returns the underlying table model.
func (m NetworksModel) Table() table.Model { return m.table }

// updateTableColumns adjusts column widths based on the current width.
func (m *NetworksModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	statusW := uiconst.ColWidthStatus
	nameW := m.width - idW - statusW - 6
	if nameW < 10 {
		nameW = 10
	}
	m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Status", Width: statusW}})
}

var _ tea.Model = (*NetworksModel)(nil)
