package identity

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
)

type userInfo struct {
	ID       string
	Name     string
	Email    string
	DomainID string
	Enabled  bool
}

type UserDetailModel struct {
	table   table.Model
	loading bool
	err     error
	spinner spinner.Model
	client  client.IdentityClient
	userID  string
	// JSON view fields
	jsonView     string
	jsonViewport viewport.Model
	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored user for JSON marshaling
	user userInfo
}

type userDetailDataLoadedMsg struct {
	tbl  table.Model
	err  error
	user userInfo
}

// NewUserDetailModel creates a new UserDetailModel for the given user ID.
func NewUserDetailModel(ic client.IdentityClient, userID string) UserDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return UserDetailModel{client: ic, loading: true, spinner: s, userID: userID}
}

// Init starts async loading of user details.
func (m UserDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		// Since the client does not provide GetUser, we fetch all users and find the matching one.
		userList, err := m.client.ListUsers()
		if err != nil {
			return userDetailDataLoadedMsg{err: err}
		}
		var user *struct {
			ID       string
			Name     string
			Email    string
			DomainID string
			Enabled  bool
		}
		for _, u := range userList {
			if u.ID == m.userID {
				user = &struct {
					ID       string
					Name     string
					Email    string
					DomainID string
					Enabled  bool
				}{ID: u.ID, Name: u.Name, Email: "", DomainID: u.DomainID, Enabled: u.Enabled}

				break
			}
		}
		if user == nil {
			return userDetailDataLoadedMsg{err: fmt.Errorf("user %s not found", m.userID)}
		}
		cols := []table.Column{{Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValueShort}, {Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValueShort}}
		rows := []table.Row{{"ID", user.ID}, {"Name", user.Name}, {"Email", user.Email}, {"DomainID", user.DomainID}, {"Enabled", fmt.Sprintf("%v", user.Enabled)}}
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
		uInfo := userInfo{ID: user.ID, Name: user.Name, Email: user.Email, DomainID: user.DomainID, Enabled: user.Enabled}
		return userDetailDataLoadedMsg{tbl: t, user: uInfo}
	}
}

// Update handles messages.
func (m UserDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case userDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.user = msg.user
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
			// Build inspect view for user.
			content := fmt.Sprintf("=== User: %s ===\nID: %s\nName: %s\nEmail: %s\nDomainID: %s\nEnabled: %v", m.user.Name, m.user.ID, m.user.Name, m.user.Email, m.user.DomainID, m.user.Enabled)
			m.inspectView = content
			m.inspectViewport = viewport.New(80, 24)
			m.inspectViewport.SetContent(m.inspectView)
			return m, nil
		}
		if msg.String() == "y" {
			b, err := json.MarshalIndent(m.user, "", "  ")
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

// View renders the user detail view.
func (m UserDetailModel) View() string {
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
		rows := []table.Row{{"Failed to load user: " + m.err.Error()}}
		return table.New(table.WithColumns(cols), table.WithRows(rows)).View()
	}
	return fmt.Sprintf("%s\n[y] json  [i] inspect  [esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m UserDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*UserDetailModel)(nil)
