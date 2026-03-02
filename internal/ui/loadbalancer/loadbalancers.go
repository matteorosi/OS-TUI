package loadbalancer

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

// LoadBalancersModel implements a subview for listing load balancers.
type LoadBalancersModel struct {
	ft          common.FilterableTable
	loading     bool
	err         error
	spinner     spinner.Model
	client      client.LoadBalancerClient
	width       int
	height      int
	mode        string // "list" or "detail"
	lbID        string
	lbName      string
	detailModel tea.Model
}

// NewLoadBalancersModel creates a new LoadBalancersModel with the given client.
func NewLoadBalancersModel(lc client.LoadBalancerClient) LoadBalancersModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return LoadBalancersModel{client: lc, loading: true, spinner: s, ft: common.NewFilterableTable(), mode: "list", width: 120, height: 30}
}

type loadBalancersDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts async loading of load balancers.
func (m LoadBalancersModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return loadBalancersDataLoadedMsg{err: fmt.Errorf("Load Balancer service unavailable (check credentials or service endpoint)")}
		}
		lbs, err := m.client.ListLoadBalancers(context.Background())
		if err != nil {
			return loadBalancersDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthNameLong}, {Title: "VIP Address", Width: uiconst.ColWidthVIPAddress}, {Title: "Provisioning", Width: uiconst.ColWidthProvisioning}, {Title: "Operating", Width: uiconst.ColWidthOperating}}
		rows := []table.Row{}
		for _, lb := range lbs {
			rows = append(rows, table.Row{lb.ID, lb.Name, lb.VipAddress, lb.ProvisioningStatus, lb.OperatingStatus})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return loadBalancersDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update processes messages and user input.
func (m LoadBalancersModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadBalancersDataLoadedMsg:
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
		if m.ft.Table.Columns() != nil {
			m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
			m.updateTableColumns()
		}
		return m, nil
	case tea.KeyMsg:
		// If we are in detail mode, forward keys to the detail model.
		if m.mode == "detail" {
			if msg.String() == "esc" {
				// Return to list view.
				m.mode = "list"
				m.detailModel = nil
				m.lbID = ""
				m.lbName = ""
				return m, nil
			}
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
				m.lbID = row[0]
				m.lbName = row[1]
				m.mode = "detail"
				m.detailModel = NewLoadBalancerDetailModel(m.client, m.lbID, m.lbName)
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
func (m LoadBalancersModel) View() string {
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
func (m LoadBalancersModel) Table() table.Model { return m.ft.Table }

func (m *LoadBalancersModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	vipW := uiconst.ColWidthVIPAddress
	provW := uiconst.ColWidthProvisioning
	operW := uiconst.ColWidthOperating
	nameW := m.width - idW - vipW - provW - operW - 6
	if nameW < 10 {
		nameW = 10
	}
	m.ft.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "VIP Address", Width: vipW}, {Title: "Provisioning", Width: provW}, {Title: "Operating", Width: operW}})
}

var _ tea.Model = (*LoadBalancersModel)(nil)
