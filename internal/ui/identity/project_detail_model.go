package identity

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
)

type projectInfo struct {
	ID       string
	Name     string
	DomainID string
	Enabled  bool
}

type ProjectDetailModel struct {
	table     table.Model
	loading   bool
	err       error
	spinner   spinner.Model
	client    client.IdentityClient
	projectID string
	// JSON view fields
	jsonView     string
	jsonViewport viewport.Model
	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored project for JSON marshaling
	project projectInfo
}

type projectDetailDataLoadedMsg struct {
	tbl  table.Model
	err  error
	proj projectInfo
}

// NewProjectDetailModel creates a new ProjectDetailModel for the given project ID.
func NewProjectDetailModel(ic client.IdentityClient, projectID string) ProjectDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return ProjectDetailModel{client: ic, loading: true, spinner: s, projectID: projectID}
}

// Init starts async loading of project details.
func (m ProjectDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		// Since the client does not provide GetProject, we fetch all projects and find the matching one.
		projList, err := m.client.ListProjects()
		if err != nil {
			return projectDetailDataLoadedMsg{err: err}
		}
		var proj *struct {
			ID       string
			Name     string
			DomainID string
			Enabled  bool
		}
		for _, p := range projList {
			if p.ID == m.projectID {
				proj = &struct {
					ID       string
					Name     string
					DomainID string
					Enabled  bool
				}{ID: p.ID, Name: p.Name, DomainID: p.DomainID, Enabled: p.Enabled}
				break
			}
		}
		if proj == nil {
			return projectDetailDataLoadedMsg{err: fmt.Errorf("project %s not found", m.projectID)}
		}
		cols := []table.Column{{Title: "Field", Width: 20}, {Title: "Value", Width: 30}, {Title: "Field", Width: 20}, {Title: "Value", Width: 30}}
		rows := []table.Row{{"ID", proj.ID}, {"Name", proj.Name}, {"DomainID", proj.DomainID}, {"Enabled", fmt.Sprintf("%v", proj.Enabled)}}
		half := (len(rows) + 1) / 2
		newRows := []table.Row{}
		for i := 0; i < half; i++ {
			left := rows[i]
			var right table.Row
			if i+half < len(rows) {
				right = rows[i+half]
			} else {
				right = table.Row{"", ""}
			}
			newRows = append(newRows, table.Row{left[0], left[1], right[0], right[1]})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(newRows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		pInfo := projectInfo{ID: proj.ID, Name: proj.Name, DomainID: proj.DomainID, Enabled: proj.Enabled}
		return projectDetailDataLoadedMsg{tbl: t, proj: pInfo}
	}
}

// Update handles messages.
func (m ProjectDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case projectDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.project = msg.proj
		return m, nil
	case tea.WindowSizeMsg:
		if m.jsonView != "" {
			m.jsonViewport.Width = msg.Width
			m.jsonViewport.Height = msg.Height
			m.jsonViewport.SetContent(m.jsonView)
			return m, nil
		}
		// Adjust table width to fill terminal
		if !m.loading && len(m.table.Columns()) > 0 {
			cols := m.table.Columns()
			if len(cols) > 0 {
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
		if msg.String() == "i" {
			// Build inspect view for project.
			content := fmt.Sprintf("=== Project: %s ===\nID: %s\nName: %s\nDomainID: %s\nEnabled: %v", m.project.Name, m.project.ID, m.project.Name, m.project.DomainID, m.project.Enabled)
			m.inspectView = content
			m.inspectViewport = viewport.New(80, 24)
			m.inspectViewport.SetContent(m.inspectView)
			return m, nil
		}
		if msg.String() == "y" {
			// Marshal project to JSON.
			b, err := json.MarshalIndent(m.project, "", "  ")
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

// View renders the project detail view.
func (m ProjectDetailModel) View() string {
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
		rows := []table.Row{{"Failed to load project: " + m.err.Error()}}
		return table.New(table.WithColumns(cols), table.WithRows(rows)).View()
	}
	return fmt.Sprintf("%s\n[y] json  [i] inspect  [esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m ProjectDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*ProjectDetailModel)(nil)
