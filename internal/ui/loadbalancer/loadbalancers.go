package loadbalancer

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
	"strings"
)

// LoadBalancersModel implements a subview for listing load balancers.
type LoadBalancersModel struct {
	table       table.Model
	loading     bool
	err         error
	spinner     spinner.Model
	client      client.LoadBalancerClient
	width       int
	height      int
	allRows     []table.Row
	filterMode  bool
	filter      textinput.Model
	mode        string // "list" or "detail"
	lbID        string
	lbName      string
	detailModel tea.Model
}

// NewLoadBalancersModel creates a new LoadBalancersModel with the given client.
func NewLoadBalancersModel(lc client.LoadBalancerClient) LoadBalancersModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return LoadBalancersModel{client: lc, loading: true, spinner: s, filter: ti, mode: "list", width: 120, height: 30}
}

type loadBalancersDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts async loading of load balancers.
func (m LoadBalancersModel) Init() tea.Cmd {
	return func() tea.Msg {
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
				m.lbID = row[0]
				m.lbName = row[1]
				m.mode = "detail"
				m.detailModel = NewLoadBalancerDetailModel(m.client, m.lbID, m.lbName)
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
	if m.filterMode {
		filterLine := fmt.Sprintf("Filter: %s", m.filter.View())
		footer := "esc: clear"
		return fmt.Sprintf("%s\n%s\n%s", filterLine, m.table.View(), footer)
	}
	return m.table.View()
}

// Table returns the primary table model (list view).
func (m LoadBalancersModel) Table() table.Model { return m.table }

func (m *LoadBalancersModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	vipW := uiconst.ColWidthVIPAddress
	provW := uiconst.ColWidthProvisioning
	operW := uiconst.ColWidthOperating
	nameW := m.width - idW - vipW - provW - operW - 6
	if nameW < 10 {
		nameW = 10
	}
	m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "VIP Address", Width: vipW}, {Title: "Provisioning", Width: provW}, {Title: "Operating", Width: operW}})
}

var _ tea.Model = (*LoadBalancersModel)(nil)
