package network

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
)

type floatingIPInfo struct {
	ID                string `json:"id"`
	FloatingNetworkID string `json:"floating_network_id"`
	FixedIP           string `json:"fixed_ip"`
	PortID            string `json:"port_id"`
	Status            string `json:"status"`
}

type FloatingIPDetailModel struct {
	table   table.Model
	loading bool
	err     error
	spinner spinner.Model
	client  client.NetworkClient
	fipID   string
	// JSON view fields
	jsonView     string
	jsonViewport viewport.Model
	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored floating IP for JSON marshaling
	fipInfo floatingIPInfo
}

// ResourceID returns the floating IP ID.
func (m FloatingIPDetailModel) ResourceID() string { return m.fipID }

// ResourceName returns a display name for the floating IP (using ID).
func (m FloatingIPDetailModel) ResourceName() string { return m.fipID }

type floatingIPDetailDataLoadedMsg struct {
	tbl     table.Model
	err     error
	fipInfo floatingIPInfo
}

// NewFloatingIPDetailModel creates a new FloatingIPDetailModel for the given floating IP ID.
func NewFloatingIPDetailModel(nc client.NetworkClient, fipID string) FloatingIPDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return FloatingIPDetailModel{client: nc, loading: true, spinner: s, fipID: fipID}
}

// Init starts async loading of floating IP details.
func (m FloatingIPDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		fipList, err := m.client.ListFloatingIPs()
		if err != nil {
			return floatingIPDetailDataLoadedMsg{err: err}
		}
		var fip *struct {
			ID                string
			FloatingNetworkID string
			FixedIP           string
			PortID            string
			Status            string
		}
		// Find the floating IP with matching ID.
		for _, f := range fipList {
			if f.ID == m.fipID {
				// Use a temporary struct to hold needed fields.
				fip = &struct {
					ID                string
					FloatingNetworkID string
					FixedIP           string
					PortID            string
					Status            string
				}{ID: f.ID, FloatingNetworkID: f.FloatingNetworkID, FixedIP: f.FixedIP, PortID: f.PortID, Status: f.Status}
				break
			}
		}
		if fip == nil {
			return floatingIPDetailDataLoadedMsg{err: fmt.Errorf("floating IP %s not found", m.fipID)}
		}
		cols := []table.Column{{Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValueShort}, {Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValueShort}}
		rows := []table.Row{{"ID", fip.ID}, {"FloatingNetworkID", fip.FloatingNetworkID}, {"FixedIP", fip.FixedIP}, {"PortID", fip.PortID}, {"Status", fip.Status}}
		half := (len(rows) + 1) / 2
		newRows := []table.Row{}
		for i := 0; i < half; i++ {
			left := rows[i]
			var right table.Row
			if i+half < len(rows) {
				right = rows[i+half]
			} else {
				right = table.Row{"", ""}
			}
			newRows = append(newRows, table.Row{left[0], left[1], right[0], right[1]})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(newRows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		fipInfo := floatingIPInfo{ID: fip.ID, FloatingNetworkID: fip.FloatingNetworkID, FixedIP: fip.FixedIP, PortID: fip.PortID, Status: fip.Status}
		return floatingIPDetailDataLoadedMsg{tbl: t, fipInfo: fipInfo}
	}
}

// Update handles messages.
func (m FloatingIPDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case floatingIPDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.fipInfo = msg.fipInfo
		return m, nil
	case tea.WindowSizeMsg:
		if m.jsonView != "" {
			m.jsonViewport.Width = msg.Width
			m.jsonViewport.Height = msg.Height
			m.jsonViewport.SetContent(m.jsonView)
			return m, nil
		}
		// Adjust table width to fill terminal
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
			// Forward other keys to viewport for scrolling
			var cmd tea.Cmd
			m.inspectViewport, cmd = m.inspectViewport.Update(msg)
			return m, cmd
		}
		// If JSON view is active, handle its keys.
		if m.jsonView != "" {
			if msg.String() == "y" || msg.String() == "esc" {
				m.jsonView = ""
				m.jsonViewport = viewport.Model{}
				return m, nil
			}
			// ignore other keys while JSON view is active
			return m, nil
		}
		if m.loading || m.err != nil {
			return m, nil
		}
		if msg.String() == "i" {
			// Build inspect view for floating IP.
			content := fmt.Sprintf("=== Floating IP: %s ===\nID: %s\nFloatingNetworkID: %s\nFixedIP: %s\nPortID: %s\nStatus: %s", m.fipInfo.ID, m.fipInfo.ID, m.fipInfo.FloatingNetworkID, m.fipInfo.FixedIP, m.fipInfo.PortID, m.fipInfo.Status)
			m.inspectView = content
			m.inspectViewport = viewport.New(80, 24)
			m.inspectViewport.SetContent(m.inspectView)
			return m, nil
		}
		if msg.String() == "y" {
			b, err := json.MarshalIndent(m.fipInfo, "", "  ")
			if err != nil {
				m.err = err
				return m, nil
			}
			m.jsonView = string(b)
			m.jsonViewport = viewport.New(80, 24)
			m.jsonViewport.SetContent(m.jsonView)
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

// View renders the floating IP detail view.
func (m FloatingIPDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.inspectView != "" {
		return fmt.Sprintf("%s\n %3.f%% | [j/k] scroll  [esc] close", m.inspectViewport.View(), m.inspectViewport.ScrollPercent()*100)
	}
	if m.jsonView != "" {
		return fmt.Sprintf("%s\nPress 'y' or 'esc' to close", m.jsonViewport.View())
	}

	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to load floating IP: " + m.err.Error()}}
		return table.New(table.WithColumns(cols), table.WithRows(rows)).View()
	}
	return fmt.Sprintf("%s\n[y] json  [i] inspect  [g] graph  [esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m FloatingIPDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*FloatingIPDetailModel)(nil)
