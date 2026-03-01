package dns

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
	"strings"
)

// RecordSetsModel displays DNS record sets for a specific zone.
type RecordSetsModel struct {
	table    table.Model
	loading  bool
	err      error
	spinner  spinner.Model
	client   client.DNSClient
	zoneID   string
	zoneName string
	// stored recordsets for inspect view
	recordsets []client.RecordSet
	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
}

// NewRecordSetsModel creates a new RecordSetsModel for the given zone.
func NewRecordSetsModel(dc client.DNSClient, zoneID string, zoneName string) RecordSetsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return RecordSetsModel{client: dc, loading: true, spinner: s, zoneID: zoneID, zoneName: zoneName}
}

type recordSetsDataLoadedMsg struct {
	tbl        table.Model
	err        error
	recordsets []client.RecordSet
}

// Init starts async loading of record sets for the zone.
func (m RecordSetsModel) Init() tea.Cmd {
	return func() tea.Msg {
		rs, err := m.client.ListRecordSets(context.Background(), m.zoneID)
		if err != nil {
			return recordSetsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "Name", Width: uiconst.ColWidthNameDNS}, {Title: "Type", Width: uiconst.ColWidthType}, {Title: "TTL", Width: uiconst.ColWidthTTL}, {Title: "Status", Width: uiconst.ColWidthStatus}, {Title: "Records", Width: uiconst.ColWidthRecords}}
		rows := []table.Row{}
		for _, r := range rs {
			records := strings.Join(r.Records, ",")
			rows = append(rows, table.Row{r.Name, r.Type, fmt.Sprintf("%d", r.TTL), r.Status, records})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		return recordSetsDataLoadedMsg{tbl: t, recordsets: rs}
	}
}

// Update handles messages and user input.
func (m RecordSetsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case recordSetsDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.recordsets = msg.recordsets
		return m, nil
	case tea.WindowSizeMsg:
		// Adjust table width to fill terminal.
		if !m.loading && len(m.table.Columns()) > 0 {
			cols := m.table.Columns()
			if len(cols) > 0 {
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
			// Forward other keys to viewport for scrolling.
			var cmd tea.Cmd
			m.inspectViewport, cmd = m.inspectViewport.Update(msg)
			return m, cmd
		}
		if m.loading || m.err != nil {
			return m, nil
		}
		if msg.String() == "i" {
			// Inspect the selected record set.
			row := m.table.SelectedRow()
			if len(row) == 0 {
				return m, nil
			}
			// Find the record set by name (first column).
			name := row[0]
			var rs *client.RecordSet
			for _, r := range m.recordsets {
				if r.Name == name {
					rs = &r
					break
				}
			}
			if rs == nil {
				return m, nil
			}
			content := fmt.Sprintf("=== RecordSet: %s ===\nID: %s\nName: %s\nType: %s\nTTL: %d\nStatus: %s\nRecords: %s", rs.Name, rs.ID, rs.Name, rs.Type, rs.TTL, rs.Status, strings.Join(rs.Records, ", "))
			m.inspectView = content
			m.inspectViewport = viewport.New(80, 24)
			m.inspectViewport.SetContent(m.inspectView)
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

// View renders the record sets view.
func (m RecordSetsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.inspectView != "" {
		return fmt.Sprintf("%s\n %3.f%% | [j/k] scroll  [esc] close", m.inspectViewport.View(), m.inspectViewport.ScrollPercent()*100)
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	// Show table with a hint for inspect and back.
	return fmt.Sprintf("%s\n[i] inspect  [esc] back", m.table.View())
}

var _ tea.Model = (*RecordSetsModel)(nil)
