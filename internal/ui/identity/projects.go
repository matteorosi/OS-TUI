package identity

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/uiconst"
)

type ProjectsModel struct {
	ft      common.FilterableTable
	loading bool
	err     error
	spinner spinner.Model
	client  client.IdentityClient
	// Dynamic sizing
	width  int
	height int
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
	return ProjectsModel{client: ic, loading: true, spinner: s, ft: common.NewFilterableTable(), width: 120, height: 30}
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
		m.ft.SetTable(msg.tbl, msg.rows)
		m.updateTableColumns()
		m.ft.SetHeight(m.height - 6)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.ft.Table.Columns() != nil {
			m.ft.SetHeight(m.height - 6)
			m.updateTableColumns()
		}
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}
		if handled, cmd := m.ft.Update(msg); handled {
			return m, cmd
		}
		// Normal table navigation
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
	return m.ft.View()
}

// Ensure ProjectsModel implements tea.Model.
func (m ProjectsModel) Table() table.Model { return m.ft.Table }

func (m *ProjectsModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	domainW := uiconst.ColWidthName
	nameW := m.width - idW - domainW - uiconst.TableHeightOffset
	if nameW < 10 {
		nameW = 10
	}
	m.ft.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Domain ID", Width: domainW}})
}

var _ tea.Model = (*ProjectsModel)(nil)
