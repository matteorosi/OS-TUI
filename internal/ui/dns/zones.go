package dns

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/uiconst"
)

// ZonesModel implements a subview for listing DNS zones.
type ZonesModel struct {
	ft      common.FilterableTable
	loading bool
	err     error
	spinner spinner.Model
	client  client.DNSClient
	// Dynamic sizing
	width       int
	height      int
	mode        string // "list" or "detail"
	zoneID      string
	zoneName    string
	detailModel tea.Model
}

// NewZonesModel creates a new ZonesModel with the given DNS client.
func NewZonesModel(dc client.DNSClient) ZonesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return ZonesModel{client: dc, loading: true, spinner: s, ft: common.NewFilterableTable(), mode: "list", width: 120, height: 30}
}

type zonesDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts async loading of DNS zones.
func (m ZonesModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return zonesDataLoadedMsg{err: fmt.Errorf("DNS service unavailable (check credentials or service endpoint)")}
		}
		if m.client == nil {
			return zonesDataLoadedMsg{err: fmt.Errorf("DNS service unavailable (check credentials or service endpoint)")}
		}
		zones, err := m.client.ListZones(context.Background())
		if err != nil {
			return zonesDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthNameDNS}, {Title: "Status", Width: uiconst.ColWidthStatus}, {Title: "TTL", Width: uiconst.ColWidthTTL}}
		rows := []table.Row{}
		for _, z := range zones {
			rows = append(rows, table.Row{z.ID, z.Name, z.Status, fmt.Sprintf("%d", z.TTL)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
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
		m.ft.SetTable(msg.tbl, msg.rows)
		m.updateTableColumns()
		m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.loading {
			m.updateTableColumns()
			m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
		}
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
		// Filter handling delegated to ft.
		if handled, cmd := m.ft.Update(msg); handled {
			return m, cmd
		}
		// Normal navigation.
		if msg.String() == "enter" {
			row := m.ft.SelectedRow()
			if len(row) > 0 {
				m.zoneID = row[0]
				m.zoneName = row[1]
				m.mode = "detail"
				m.detailModel = NewRecordSetsModel(m.client, m.zoneID, m.zoneName)
				return m, m.detailModel.Init()
			}
			return m, nil
		}
		cmd := m.ft.UpdateTable(msg)
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
		return m.detailModel.View()
	}
	return m.ft.View()
}

// Table returns the primary table model (list view).
func (m ZonesModel) Table() table.Model { return m.ft.Table }

func (m *ZonesModel) updateTableColumns() {
	if len(m.ft.Table.Columns()) > 0 {
		idW := uiconst.ColWidthUUID
		statusW := uiconst.ColWidthStatus
		ttlW := uiconst.ColWidthTTL
		nameW := m.width - idW - statusW - ttlW - 6
		if nameW < 10 {
			nameW = 10
		}
		m.ft.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Status", Width: statusW}, {Title: "TTL", Width: ttlW}})
	}
}

var _ tea.Model = (*ZonesModel)(nil)
