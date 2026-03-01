package network

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
)

type securityGroupJSON struct {
	Group struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Stateful    bool   `json:"stateful"`
	} `json:"group"`
	Rules []client.SecurityGroupRule `json:"rules"`
}

type SecurityGroupDetailModel struct {
	table      table.Model // group details
	rulesTable table.Model // security group rules
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.NetworkClient
	sgID       string
	// JSON view fields
	jsonView     string
	jsonViewport viewport.Model
	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored security group JSON data
	sgJSON securityGroupJSON
	width  int
	height int
}

type securityGroupDetailDataLoadedMsg struct {
	groupTbl table.Model
	rulesTbl table.Model
	err      error
	sgJSON   securityGroupJSON
}

// NewSecurityGroupDetailModel creates a new SecurityGroupDetailModel for the given security group ID.
func NewSecurityGroupDetailModel(nc client.NetworkClient, sgID string) SecurityGroupDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return SecurityGroupDetailModel{client: nc, loading: true, spinner: s, sgID: sgID, width: 120, height: 30}
}

// Init starts async loading of security group details.
func (m SecurityGroupDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		// Load security group details.
		sgList, err := m.client.ListSecurityGroups()
		if err != nil {
			return securityGroupDetailDataLoadedMsg{err: err}
		}
		var sg *struct {
			ID          string
			Name        string
			Description string
			Stateful    bool
		}
		for _, g := range sgList {
			if g.ID == m.sgID {
				sg = &struct {
					ID          string
					Name        string
					Description string
					Stateful    bool
				}{ID: g.ID, Name: g.Name, Description: g.Description, Stateful: g.Stateful}
				break
			}
		}
		if sg == nil {
			return securityGroupDetailDataLoadedMsg{err: fmt.Errorf("security group %s not found", m.sgID)}
		}
		// Build group details table.
		groupCols := []table.Column{{Title: "Field", Width: 20}, {Title: "Value", Width: 60}}
		groupRows := []table.Row{{"ID", sg.ID}, {"Name", sg.Name}, {"Description", sg.Description}, {"Stateful", fmt.Sprintf("%v", sg.Stateful)}}
		groupTbl := table.New(
			table.WithColumns(groupCols),
			table.WithRows(groupRows),
			table.WithFocused(true),
		)
		groupTbl.SetStyles(table.DefaultStyles())
		// Load security group rules.
		rulesList, rErr := m.client.ListSecurityGroupRules(context.Background(), m.sgID)
		var rulesTbl table.Model
		if rErr != nil {
			// If rule loading fails, create an empty table with error row.
			cols := []table.Column{{Title: "Error", Width: 80}}
			rows := []table.Row{{"Failed to load rules: " + rErr.Error()}}
			rulesTbl = table.New(table.WithColumns(cols), table.WithRows(rows))
		} else {
			ruleCols := []table.Column{{Title: "ID", Width: 36}, {Title: "Direction", Width: 8}, {Title: "EtherType", Width: 8}, {Title: "Protocol", Width: 6}, {Title: "PortRange", Width: 12}, {Title: "RemoteIP", Width: 15}, {Title: "RemoteGroup", Width: 36}}
			ruleRows := []table.Row{}
			for _, r := range rulesList {
				portRange := ""
				if r.PortRangeMin != 0 || r.PortRangeMax != 0 {
					portRange = fmt.Sprintf("%d-%d", r.PortRangeMin, r.PortRangeMax)
				}
				ruleRows = append(ruleRows, table.Row{r.ID, r.Direction, r.EtherType, r.Protocol, portRange, r.RemoteIPPrefix, r.RemoteGroupID})
			}
			rulesTbl = table.New(
				table.WithColumns(ruleCols),
				table.WithRows(ruleRows),
				table.WithFocused(true),
				table.WithHeight(m.height-6),
			)
			rulesTbl.SetStyles(table.DefaultStyles())
		}
		sgJSON := securityGroupJSON{Group: struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Stateful    bool   `json:"stateful"`
		}{ID: sg.ID, Name: sg.Name, Description: sg.Description, Stateful: sg.Stateful}, Rules: rulesList}
		return securityGroupDetailDataLoadedMsg{groupTbl: groupTbl, rulesTbl: rulesTbl, err: nil, sgJSON: sgJSON}
	}
}

// Update handles messages.
func (m SecurityGroupDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case securityGroupDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.groupTbl
		m.rulesTable = msg.rulesTbl
		m.sgJSON = msg.sgJSON
		m.updateTableColumns()
		m.table.SetHeight(m.height - 6)
		m.rulesTable.SetHeight(m.height - 6)
		return m, nil
	case tea.WindowSizeMsg:
		if m.jsonView != "" {
			m.jsonViewport.Width = msg.Width
			m.jsonViewport.Height = msg.Height
			m.jsonViewport.SetContent(m.jsonView)
			return m, nil
		}
		m.width = msg.Width
		m.height = msg.Height
		if !m.loading {
			m.updateTableColumns()
			m.table.SetHeight(m.height - 6)
			m.rulesTable.SetHeight(m.height - 6)
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
		// Handle new/delete actions (currently no-op).
		if msg.String() == "n" || msg.String() == "d" {
			// Placeholder for future implementation.
			return m, nil
		}
		if msg.String() == "i" {
			// Build inspect view for security group.
			content := fmt.Sprintf("=== Security Group: %s ===\nID: %s\nName: %s\nDescription: %s\nStateful: %v\nRules: %d", m.sgJSON.Group.Name, m.sgJSON.Group.ID, m.sgJSON.Group.Name, m.sgJSON.Group.Description, m.sgJSON.Group.Stateful, len(m.sgJSON.Rules))
			m.inspectView = content
			m.inspectViewport = viewport.New(80, 24)
			m.inspectViewport.SetContent(m.inspectView)
			return m, nil
		}
		if msg.String() == "y" {
			b, err := json.MarshalIndent(m.sgJSON, "", "  ")
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
		// Forward navigation to the rules table.
		m.rulesTable, cmd = m.rulesTable.Update(msg)
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

// View renders the security group detail view.
func (m SecurityGroupDetailModel) View() string {
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
		rows := []table.Row{{"Failed to load security group: " + m.err.Error()}}
		return table.New(table.WithColumns(cols), table.WithRows(rows)).View()
	}
	// Render group details and rules.
	groupView := m.table.View()
	rulesView := m.rulesTable.View()
	footer := "[n]ew [d]elete [y] json [i] inspect [esc] back"
	return fmt.Sprintf("%s\n\nRules:\n%s\n%s", groupView, rulesView, footer)
}

// Table returns the underlying table model.
func (m SecurityGroupDetailModel) Table() table.Model { return m.table }

func (m *SecurityGroupDetailModel) updateTableColumns() {
	// Update group details table columns proportionally.
	if len(m.table.Columns()) > 0 {
		cols := m.table.Columns()
		totalWidth := m.width - 4
		if totalWidth < 0 {
			totalWidth = m.width
		}
		colWidth := totalWidth / len(cols)
		if colWidth < 5 {
			colWidth = 5
		}
		for i := range cols {
			cols[i].Width = colWidth
		}
		m.table.SetColumns(cols)
		m.table.SetWidth(m.width)
	}
	// Update rules table columns proportionally.
	if len(m.rulesTable.Columns()) > 0 {
		cols := m.rulesTable.Columns()
		totalWidth := m.width - 4
		if totalWidth < 0 {
			totalWidth = m.width
		}
		colWidth := totalWidth / len(cols)
		if colWidth < 5 {
			colWidth = 5
		}
		for i := range cols {
			cols[i].Width = colWidth
		}
		m.rulesTable.SetColumns(cols)
		m.rulesTable.SetWidth(m.width)
	}
}

var _ tea.Model = (*SecurityGroupDetailModel)(nil)
