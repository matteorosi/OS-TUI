package compute

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"strings"
)

// InstancesModel implements a subview for listing compute instances.
type InstancesModel struct {
	table      table.Model
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.ComputeClient
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model

	// Dynamic sizing
	width  int
	height int
}

// NewInstancesModel creates a new InstancesModel with the given compute client.
func NewInstancesModel(cc client.ComputeClient) InstancesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	// Use default style (no explicit style set).
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return InstancesModel{client: cc, loading: true, spinner: s, filter: ti, width: 120, height: 30}
}

// dataLoadedMsg is sent when instance data has been fetched.
type dataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts the async data loading.
func (m InstancesModel) Init() tea.Cmd {
	return func() tea.Msg {
		srvList, err := m.client.ListInstances()
		if err != nil {
			return dataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "Status", Width: 12}}
		rows := []table.Row{}
		for _, s := range srvList {
			rows = append(rows, table.Row{s.ID, s.Name, s.Status})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-6),
		)
		t.SetStyles(table.DefaultStyles())
		return dataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages for the model.
func (m InstancesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.allRows = msg.rows
		m.updateTableColumns()
		m.table.SetHeight(m.height - 6)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.table.Columns() != nil {
			m.table.SetHeight(m.height - 6)
			m.updateTableColumns()
		}
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			// ignore key input while loading or on error
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
		// Normal table navigation
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

// View renders the appropriate UI based on state.
func (m InstancesModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	if m.filterMode {
		filterLine := fmt.Sprintf("Filter: %s", m.filter.View())
		footer := "esc: clear"
		return fmt.Sprintf("%s\n%s\n%s", filterLine, m.table.View(), footer)
	}
	return m.table.View()
}

// updateTableColumns adjusts column widths based on the current width.
func (m *InstancesModel) updateTableColumns() {
	idW := 36
	statusW := 12
	nameW := m.width - idW - statusW - 6
	if nameW < 10 {
		nameW = 10
	}
	m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Status", Width: statusW}})
}

// Ensure InstancesModel implements tea.Model.
// Table returns the underlying table model.
func (m InstancesModel) Table() table.Model { return m.table }

var _ tea.Model = (*InstancesModel)(nil)
