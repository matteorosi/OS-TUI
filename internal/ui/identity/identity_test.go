package identity

import (
	"errors"
	"ostui/internal/ui/uiconst"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
)

type mockIdentityClient struct {
	projList []projects.Project
	projErr  error

	userList []users.User
	userErr  error

	token    *tokens.Token
	tokenErr error
}

func (m *mockIdentityClient) ListProjects() ([]projects.Project, error) {
	return m.projList, m.projErr
}

func (m *mockIdentityClient) GetCurrentProject() (projects.Project, error) {
	// Not used in UI tests
	return projects.Project{}, nil
}

func (m *mockIdentityClient) ListUsers() ([]users.User, error) {
	return m.userList, m.userErr
}

func (m *mockIdentityClient) GetTokenInfo() (*tokens.Token, error) {
	return m.token, m.tokenErr
}

// Helper to create a table model for projects.
func newProjectsTable(rows []table.Row) table.Model {
	cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "Domain ID", Width: uiconst.ColWidthName}}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(uiconst.TableHeightDefault),
	)
	t.SetStyles(table.DefaultStyles())
	return t
}

// Helper to create a table model for users.
func newUsersTable(rows []table.Row) table.Model {
	cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "Domain ID", Width: uiconst.ColWidthName}, {Title: "Enabled", Width: uiconst.ColWidthEnabled}}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(uiconst.TableHeightDefault),
	)
	t.SetStyles(table.DefaultStyles())
	return t
}

func TestProjectsModelSuccess(t *testing.T) {
	mock := &mockIdentityClient{projList: []projects.Project{{ID: "proj-1", Name: "proj1", DomainID: "domain-1"}}}
	m := NewProjectsModel(mock)
	// Simulate loaded state.
	m.loading = false
	rows := []table.Row{{"proj-1", "proj1", "domain-1"}}
	m.table = newProjectsTable(rows)
	m.allRows = rows
	view := m.View()
	if !strings.Contains(view, "proj1") {
		t.Fatalf("expected project name in view, got %s", view)
	}
}

func TestProjectsModelError(t *testing.T) {
	mock := &mockIdentityClient{projErr: errors.New("list error")}
	m := NewProjectsModel(mock)
	m.loading = false
	m.err = errors.New("list error")
	view := m.View()
	if !strings.Contains(view, "Failed to list projects") {
		t.Fatalf("expected error message in view, got %s", view)
	}
}

func TestProjectsModelFilterMode(t *testing.T) {
	mock := &mockIdentityClient{projList: []projects.Project{{ID: "proj-1", Name: "proj1", DomainID: "domain-1"}}}
	m := NewProjectsModel(mock)
	m.loading = false
	rows := []table.Row{{"proj-1", "proj1", "domain-1"}}
	m.table = newProjectsTable(rows)
	m.allRows = rows
	// Enable filter mode.
	m.filterMode = true
	m.filter = textinput.New()
	view := m.View()
	if !strings.Contains(view, "Filter:") {
		t.Fatalf("expected filter line in view, got %s", view)
	}
}

func TestUsersModelSuccess(t *testing.T) {
	mock := &mockIdentityClient{userList: []users.User{{ID: "user-1", Name: "user1", DomainID: "domain-1", Enabled: true}}}
	m := NewUsersModel(mock)
	m.loading = false
	rows := []table.Row{{"user-1", "user1", "domain-1", "true"}}
	m.table = newUsersTable(rows)
	view := m.View()
	if !strings.Contains(view, "user1") {
		t.Fatalf("expected user name in view, got %s", view)
	}
}

func TestUsersModelError(t *testing.T) {
	mock := &mockIdentityClient{userErr: errors.New("list error")}
	m := NewUsersModel(mock)
	m.loading = false
	m.err = errors.New("list error")
	view := m.View()
	if !strings.Contains(view, "Failed to list users") {
		t.Fatalf("expected error message in view, got %s", view)
	}
}

func TestTokenModelSuccess(t *testing.T) {
	mock := &mockIdentityClient{token: &tokens.Token{ID: "token-1", ExpiresAt: time.Now().Add(1 * time.Hour)}}
	m := NewTokenModel(mock)
	m.loading = false
	m.token = mock.token
	view := m.View()
	if !strings.Contains(view, "token-1") {
		t.Fatalf("expected token ID in view, got %s", view)
	}
}

func TestTokenModelError(t *testing.T) {
	mock := &mockIdentityClient{tokenErr: errors.New("token error")}
	m := NewTokenModel(mock)
	m.loading = false
	m.err = errors.New("token error")
	view := m.View()
	if !strings.Contains(view, "Failed to get token info") {
		t.Fatalf("expected error message in view, got %s", view)
	}
}
