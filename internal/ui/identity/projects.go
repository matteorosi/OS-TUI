package identity

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/uiconst"
	"strings"
)

type ProjectsModel struct {
	table      table.Model
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.IdentityClient
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model
	width      int
	height     int
}

type projectsDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// NewProjectsModel creates a new ProjectsModel.
func NewProjectsModel(ic client.IdentityClient) ProjectsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return ProjectsModel{client: ic, loading: true, spinner: s, filter: ti, width: 120, height: 30}
}

// Init starts async loading.
func (m ProjectsModel) Init() tea.Cmd {
	return func() tea.Msg {
		projList, err := m.client.ListProjects()
		if err != nil {
			return projectsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "Domain ID", Width: uiconst.ColWidthName}}
		rows := []table.Row{}
		for _, p := range projList {
			rows = append(rows, table.Row{p.ID, p.Name, p.DomainID})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return projectsDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages.
func (m ProjectsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case projectsDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.updateTableColumns()
		m.table.SetHeight(m.height - 6)
		m.allRows = msg.rows
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

// View renders.
func (m ProjectsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: uiconst.ColWidthError}}
		rows := []table.Row{{"Failed to list projects: " + m.err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	if m.filterMode {
		filterLine := fmt.Sprintf("Filter: %s", m.filter.View())
		footer := "esc: clear"
		return fmt.Sprintf("%s\n%s\n%s", filterLine, m.table.View(), footer)
	}
	return m.table.View()
}

// Ensure ProjectsModel implements tea.Model.
func (m ProjectsModel) Table() table.Model { return m.table }

func (m *ProjectsModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	domainW := uiconst.ColWidthName
	nameW := m.width - idW - domainW - uiconst.TableHeightOffset
	if nameW < 10 {
		nameW = 10
	}
	m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Domain ID", Width: domainW}})
}

var _ tea.Model = (*ProjectsModel)(nil)
