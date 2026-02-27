package identity

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"time"
)

type TokenModel struct {
	token   *tokens.Token
	loading bool
	err     error
	spinner spinner.Model
	client  client.IdentityClient
}

type tokenDataLoadedMsg struct {
	token *tokens.Token
	err   error
}

// NewTokenModel creates a new TokenModel.
func NewTokenModel(ic client.IdentityClient) TokenModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return TokenModel{client: ic, loading: true, spinner: s}
}

// Init starts async loading of token info.
func (m TokenModel) Init() tea.Cmd {
	return func() tea.Msg {
		token, err := m.client.GetTokenInfo()
		return tokenDataLoadedMsg{token: token, err: err}
	}
}

// Update handles messages.
func (m TokenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tokenDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.token = msg.token
		return m, nil
	case tea.WindowSizeMsg:
		return m, nil
	case tea.KeyMsg:
		// No key handling needed for token view.
		return m, nil
	default:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// View renders the token information.
func (m TokenModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to get token info: " + m.err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	// Compute remaining time.
	remaining := time.Until(m.token.ExpiresAt)
	var remainingStr string
	if remaining > 0 {
		hours := int(remaining.Hours())
		minutes := int(remaining.Minutes()) % 60
		seconds := int(remaining.Seconds()) % 60
		remainingStr = fmt.Sprintf("%02dh %02dm %02ds", hours, minutes, seconds)
	} else {
		remainingStr = "Expired"
	}
	fields := map[string]string{
		"Token ID":   m.token.ID,
		"Expires At": m.token.ExpiresAt.Format(time.RFC3339),
		"Remaining":  remainingStr,
	}
	return common.NewDetail("Token Info", fields).View()
}

// Ensure TokenModel implements tea.Model.
var _ tea.Model = (*TokenModel)(nil)
