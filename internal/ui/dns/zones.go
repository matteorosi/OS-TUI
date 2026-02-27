package dns

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"strings"
)

// ZonesModel implements a subview for listing DNS zones.
type ZonesModel struct {
	table       table.Model
	loading     bool
	err         error
	spinner     spinner.Model
	client      client.DNSClient
	allRows     []table.Row
	filterMode  bool
	filter      textinput.Model
	mode        string // "list" or "detail"
	zoneID      string
	zoneName    string
	detailModel tea.Model
}

// NewZonesModel creates a new ZonesModel with the given DNS client.
func NewZonesModel(dc client.DNSClient) ZonesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return ZonesModel{client: dc, loading: true, spinner: s, filter: ti, mode: "list"}
}

type zonesDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts async loading of DNS zones.
func (m ZonesModel) Init() tea.Cmd {
	return func() tea.Msg {
		zones, err := m.client.ListZones(context.Background())
		if err != nil {
			return zonesDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 40}, {Title: "Status", Width: 12}, {Title: "TTL", Width: 8}}
		rows := []table.Row{}
		for _, z := range zones {
			rows = append(rows, table.Row{z.ID, z.Name, z.Status, fmt.Sprintf("%d", z.TTL)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(10),
		)
		t.SetStyles(table.DefaultStyles())
		return zonesDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update processes messages and user input.
func (m ZonesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case zonesDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.allRows = msg.rows
		return m, nil
	case tea.WindowSizeMsg:
		// No special handling needed.
		return m, nil
	case tea.KeyMsg:
		// If we are in detail mode, forward keys to the detail model.
		if m.mode == "detail" {
			// Handle escape to return to list view.
			if msg.String() == "esc" {
				m.mode = "list"
				m.detailModel = nil
				m.zoneID = ""
				m.zoneName = ""
				return m, nil
			}
			// Forward other keys to the detail model.
			var cmd tea.Cmd
			m.detailModel, cmd = m.detailModel.Update(msg)
			return m, cmd
		}
		// Global loading/error guard.
		if m.loading || m.err != nil {
			return m, nil
		}
		// Filter mode handling.
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
		// Normal navigation.
		if msg.String() == "enter" {
			row := m.table.SelectedRow()
			if len(row) > 0 {
				m.zoneID = row[0]
				m.zoneName = row[1]
				// Switch to detail mode and create the detail model.
				m.mode = "detail"
				m.detailModel = NewRecordSetsModel(m.client, m.zoneID, m.zoneName)
				// Initialise the detail model.
				return m, m.detailModel.Init()
			}
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

// View renders the UI based on the current mode.
func (m ZonesModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	if m.mode == "detail" && m.detailModel != nil {
		// Delegate view to the detail model.
		return m.detailModel.View()
	}
	if m.filterMode {
		filterLine := fmt.Sprintf("Filter: %s", m.filter.View())
		footer := "esc: clear"
		return fmt.Sprintf("%s\n%s\n%s", filterLine, m.table.View(), footer)
	}
	return m.table.View()
}

// Table returns the primary table model (list view).
func (m ZonesModel) Table() table.Model { return m.table }

var _ tea.Model = (*ZonesModel)(nil)
