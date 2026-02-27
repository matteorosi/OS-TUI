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

type SecurityGroupsModel struct {
	table      table.Model
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.NetworkClient
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model
}

type securityGroupsDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// NewSecurityGroupsModel creates a new SecurityGroupsModel.
func NewSecurityGroupsModel(nc client.NetworkClient) SecurityGroupsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return SecurityGroupsModel{client: nc, loading: true, spinner: s, filter: ti}
}

// Init starts async loading of security groups.
func (m SecurityGroupsModel) Init() tea.Cmd {
	return func() tea.Msg {
		sgList, err := m.client.ListSecurityGroups()
		if err != nil {
			return securityGroupsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "Description", Width: 30}, {Title: "Stateful", Width: 8}}
		rows := []table.Row{}
		for _, sg := range sgList {
			rows = append(rows, table.Row{sg.ID, sg.Name, sg.Description, fmt.Sprintf("%v", sg.Stateful)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(10),
		)
		t.SetStyles(table.DefaultStyles())
		return securityGroupsDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages.
func (m SecurityGroupsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case securityGroupsDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.allRows = msg.rows
		return m, nil
	case tea.WindowSizeMsg:
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

// View renders the security groups view.
func (m SecurityGroupsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to list security groups: " + m.err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	if m.filterMode {
		filterLine := fmt.Sprintf("Filter: %s", m.filter.View())
		footer := "esc: clear"
		return fmt.Sprintf("%s\n%s\n%s", filterLine, m.table.View(), footer)
	}
	return m.table.View()
}

// Ensure SecurityGroupsModel implements tea.Model.
// Table returns the underlying table model.
func (m SecurityGroupsModel) Table() table.Model { return m.table }

var _ tea.Model = (*SecurityGroupsModel)(nil)
