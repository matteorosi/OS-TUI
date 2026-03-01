package network

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
)

// RouterDetailModel displays detailed information for a single router.
// It follows the same pattern as ImageDetailModel.
type RouterDetailModel struct {
	table    table.Model
	loading  bool
	err      error
	spinner  spinner.Model
	client   client.NetworkClient
	routerID string
}

type routerDetailDataLoadedMsg struct {
	tbl table.Model
	err error
}

// NewRouterDetailModel creates a new RouterDetailModel for the given router ID.
func NewRouterDetailModel(nc client.NetworkClient, routerID string) RouterDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return RouterDetailModel{client: nc, loading: true, spinner: s, routerID: routerID}
}

// Init starts the async loading of router details.
func (m RouterDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		r, err := m.client.GetRouter(context.Background(), m.routerID)
		if err != nil {
			return routerDetailDataLoadedMsg{err: err}
		}
		// Build rows: ID, Name, Status, AdminStateUp, ExternalGateway (network ID)
		external := ""
		if r != nil {
			external = r.GatewayInfo.NetworkID
		}
		cols := []table.Column{{Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValue}}
		rows := []table.Row{{"ID", r.ID}, {"Name", r.Name}, {"Status", fmt.Sprintf("%v", r.Status)}, {"AdminStateUp", fmt.Sprintf("%v", r.AdminStateUp)}, {"ExternalGateway", external}}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		return routerDetailDataLoadedMsg{tbl: t}
	}
}

// Update handles messages.
func (m RouterDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case routerDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		return m, nil
	case tea.WindowSizeMsg:
		if !m.loading && len(m.table.Columns()) > 0 {
			cols := m.table.Columns()
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
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
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

// View renders the router detail view.
func (m RouterDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return fmt.Sprintf("%s\n[esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m RouterDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*RouterDetailModel)(nil)
