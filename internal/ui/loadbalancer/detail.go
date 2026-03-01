package loadbalancer

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
)

// LoadBalancerDetailModel shows listeners and pools for a load balancer.
type LoadBalancerDetailModel struct {
	// UI components for each view.
	listenersTable table.Model
	poolsTable     table.Model
	loading        bool
	err            error
	spinner        spinner.Model
	client         client.LoadBalancerClient
	lbID           string
	lbName         string
	// mode indicates which table is currently visible: "listeners" or "pools".
	mode string
	// stored data for inspect view.
	listeners []client.Listener
	pools     []client.Pool
	// Inspect view fields.
	inspectView     string
	inspectViewport viewport.Model
}

// ResourceID returns the load balancer ID.
func (m LoadBalancerDetailModel) ResourceID() string { return m.lbID }

// ResourceName returns the load balancer name.
func (m LoadBalancerDetailModel) ResourceName() string { return m.lbName }

type loadBalancerDetailDataLoadedMsg struct {
	listeners []client.Listener
	pools     []client.Pool
	err       error
}

// NewLoadBalancerDetailModel creates a new detail model for the given load balancer.
func NewLoadBalancerDetailModel(lc client.LoadBalancerClient, lbID string, lbName string) LoadBalancerDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return LoadBalancerDetailModel{client: lc, loading: true, spinner: s, lbID: lbID, lbName: lbName, mode: "listeners"}
}

// Init starts async loading of listeners and pools.
func (m LoadBalancerDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		// Load listeners.
		lst, err := m.client.ListListeners(context.Background(), m.lbID)
		if err != nil {
			return loadBalancerDetailDataLoadedMsg{err: err}
		}
		// Load pools.
		p, err := m.client.ListPools(context.Background(), m.lbID)
		if err != nil {
			return loadBalancerDetailDataLoadedMsg{err: err}
		}
		return loadBalancerDetailDataLoadedMsg{listeners: lst, pools: p}
	}
}

// Update processes messages and user input.
func (m LoadBalancerDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadBalancerDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.listeners = msg.listeners
		m.pools = msg.pools
		// Build listeners table.
		lcols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthNameLong}, {Title: "Protocol", Width: uiconst.ColWidthProtocol}, {Title: "Port", Width: uiconst.ColWidthPort}, {Title: "Status", Width: uiconst.ColWidthStatusLong}}
		lrows := []table.Row{}
		for _, l := range m.listeners {
			lrows = append(lrows, table.Row{l.ID, l.Name, l.Protocol, fmt.Sprintf("%d", l.ProtocolPort), l.ProvisioningStatus})
		}
		lt := table.New(
			table.WithColumns(lcols),
			table.WithRows(lrows),
			table.WithFocused(true),
		)
		lt.SetStyles(table.DefaultStyles())
		m.listenersTable = lt
		// Build pools table.
		pcols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthNameLong}, {Title: "Protocol", Width: uiconst.ColWidthProtocol}, {Title: "Algorithm", Width: uiconst.ColWidthAlgorithm}, {Title: "Status", Width: uiconst.ColWidthStatusLong}}
		prows := []table.Row{}
		for _, p := range m.pools {
			prows = append(prows, table.Row{p.ID, p.Name, p.Protocol, p.LBAlgorithm, p.ProvisioningStatus})
		}
		pt := table.New(
			table.WithColumns(pcols),
			table.WithRows(prows),
			table.WithFocused(true),
		)
		pt.SetStyles(table.DefaultStyles())
		m.poolsTable = pt
		return m, nil
	case tea.WindowSizeMsg:
		// Adjust table widths for both tables.
		if !m.loading {
			// Listeners table.
			if len(m.listenersTable.Columns()) > 0 {
				cols := m.listenersTable.Columns()
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
				m.listenersTable.SetColumns(cols)
				m.listenersTable.SetWidth(msg.Width)
			}
			// Pools table.
			if len(m.poolsTable.Columns()) > 0 {
				cols := m.poolsTable.Columns()
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
				m.poolsTable.SetColumns(cols)
				m.poolsTable.SetWidth(msg.Width)
			}
		}
		return m, nil
	case tea.KeyMsg:
		// If inspect view active, handle its keys.
		if m.inspectView != "" {
			if msg.String() == "i" || msg.String() == "esc" {
				m.inspectView = ""
				m.inspectViewport = viewport.Model{}
				return m, nil
			}
			var cmd tea.Cmd
			m.inspectViewport, cmd = m.inspectViewport.Update(msg)
			return m, cmd
		}
		if m.loading || m.err != nil {
			return m, nil
		}
		// Tab switches between listeners and pools.
		if msg.String() == "tab" {
			if m.mode == "listeners" {
				m.mode = "pools"
			} else {
				m.mode = "listeners"
			}
			return m, nil
		}
		// Inspect selected row.
		if msg.String() == "i" {
			if m.mode == "listeners" {
				row := m.listenersTable.SelectedRow()
				if len(row) == 0 {
					return m, nil
				}
				// Find listener by ID (first column).
				id := row[0]
				var l *client.Listener
				for _, li := range m.listeners {
					if li.ID == id {
						l = &li
						break
					}
				}
				if l == nil {
					return m, nil
				}
				content := fmt.Sprintf("=== Listener: %s ===\nID: %s\nName: %s\nProtocol: %s\nPort: %d\nStatus: %s", l.Name, l.ID, l.Name, l.Protocol, l.ProtocolPort, l.ProvisioningStatus)
				m.inspectView = content
				m.inspectViewport = viewport.New(80, 24)
				m.inspectViewport.SetContent(m.inspectView)
				return m, nil
			}
			// Pools mode.
			row := m.poolsTable.SelectedRow()
			if len(row) == 0 {
				return m, nil
			}
			id := row[0]
			var p *client.Pool
			for _, po := range m.pools {
				if po.ID == id {
					p = &po
					break
				}
			}
			if p == nil {
				return m, nil
			}
			content := fmt.Sprintf("=== Pool: %s ===\nID: %s\nName: %s\nProtocol: %s\nAlgorithm: %s\nStatus: %s", p.Name, p.ID, p.Name, p.Protocol, p.LBAlgorithm, p.ProvisioningStatus)
			m.inspectView = content
			m.inspectViewport = viewport.New(80, 24)
			m.inspectViewport.SetContent(m.inspectView)
			return m, nil
		}
		// Forward other keys to the active table.
		var cmd tea.Cmd
		if m.mode == "listeners" {
			m.listenersTable, cmd = m.listenersTable.Update(msg)
		} else {
			m.poolsTable, cmd = m.poolsTable.Update(msg)
		}
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
func (m LoadBalancerDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	if m.inspectView != "" {
		return fmt.Sprintf("%s\n %3.f%% | [j/k] scroll  [esc] close", m.inspectViewport.View(), m.inspectViewport.ScrollPercent()*100)
	}
	// Show the active table with a hint.
	var tableView string
	if m.mode == "listeners" {
		tableView = m.listenersTable.View()
	} else {
		tableView = m.poolsTable.View()
	}
	// Hint line.
	hint := "[tab] switch  [i] inspect  [g] graph  [esc] back"
	return fmt.Sprintf("%s\n%s", tableView, hint)
}

var _ tea.Model = (*LoadBalancerDetailModel)(nil)
