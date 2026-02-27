package compute

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
)

// KeypairDetailModel displays detailed information for a single compute keypair.
// It follows the same pattern as other detail models (e.g., ImageDetailModel).
type KeypairDetailModel struct {
	table       table.Model
	loading     bool
	err         error
	spinner     spinner.Model
	client      client.ComputeClient
	keypairName string
}

type keypairDetailDataLoadedMsg struct {
	tbl table.Model
	err error
}

// NewKeypairDetailModel creates a new KeypairDetailModel for the given keypair name.
func NewKeypairDetailModel(cc client.ComputeClient, keypairName string) KeypairDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return KeypairDetailModel{client: cc, loading: true, spinner: s, keypairName: keypairName}
}

// Init starts the async loading of the keypair details.
func (m KeypairDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		kp, err := m.client.GetKeypair(context.Background(), m.keypairName)
		if err != nil {
			return keypairDetailDataLoadedMsg{err: err}
		}
		// Truncate the public key to 60 characters for display.
		pub := kp.PublicKey
		if len(pub) > 60 {
			pub = pub[:60] + "..."
		}
		cols := []table.Column{{Title: "Field", Width: 20}, {Title: "Value", Width: 60}}
		rows := []table.Row{{"Name", kp.Name}, {"Fingerprint", kp.Fingerprint}, {"Type", kp.Type}, {"PublicKey", pub}}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		return keypairDetailDataLoadedMsg{tbl: t}
	}
}

// Update handles messages for the model.
func (m KeypairDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case keypairDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		return m, nil
	case tea.WindowSizeMsg:
		if !m.loading && len(m.table.Columns()) > 0 {
			cols := m.table.Columns()
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
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
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

// View renders the keypair detail view.
func (m KeypairDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return fmt.Sprintf("%s\n[esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m KeypairDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*KeypairDetailModel)(nil)
