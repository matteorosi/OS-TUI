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

// FlavorsModel implements a subview for listing OpenStack compute flavors.
// It follows the same pattern as InstancesModel: async loading, spinner while
// loading, optional filter mode, and a table view once data is available.
type FlavorsModel struct {
	table      table.Model
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.ComputeClient
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model
}

// NewFlavorsModel creates a new FlavorsModel with the given compute client.
func NewFlavorsModel(cc client.ComputeClient) FlavorsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return FlavorsModel{client: cc, loading: true, spinner: s, filter: ti}
}

type flavorsDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts the async loading of flavor data.
func (m FlavorsModel) Init() tea.Cmd {
	return func() tea.Msg {
		flavorList, err := m.client.ListFlavors()
		if err != nil {
			return flavorsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "VCPUs", Width: 6}, {Title: "RAM (MB)", Width: 8}, {Title: "Disk (GB)", Width: 8}}
		rows := []table.Row{}
		for _, f := range flavorList {
			rows = append(rows, table.Row{f.ID, f.Name, fmt.Sprintf("%d", f.VCPUs), fmt.Sprintf("%d", f.RAM), fmt.Sprintf("%d", f.Disk)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(10),
		)
		t.SetStyles(table.DefaultStyles())
		return flavorsDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages for the model, including data load, window resize,
// and key handling for filtering.
func (m FlavorsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case flavorsDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.allRows = msg.rows
		return m, nil
	case tea.WindowSizeMsg:
		// No special resize handling needed.
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}
		// Filter mode handling – same behaviour as InstancesModel.
		if !m.filterMode && msg.String() == "/" {
			m.filterMode = true
			m.filter.Focus()
			return m, textinput.Blink
		}
		if m.filterMode && msg.String() == "esc" {
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
		// Normal table navigation.
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

// View renders the model: spinner while loading, error if any, filter UI or the table.
func (m FlavorsModel) View() string {
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

// Table returns the underlying table model for external callers.
func (m FlavorsModel) Table() table.Model { return m.table }

var _ tea.Model = (*FlavorsModel)(nil)
