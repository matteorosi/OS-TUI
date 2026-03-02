package compute

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/uiconst"
)

// KeypairsModel implements a subview for listing OpenStack compute keypairs.
// It follows the same pattern as InstancesModel: async loading, spinner while
// loading, optional filter mode, and a table view once data is available.
type KeypairsModel struct {
	ft      common.FilterableTable
	loading bool
	err     error
	spinner spinner.Model
	client  client.ComputeClient
	// Dynamic sizing
	width  int
	height int
}

// NewKeypairsModel creates a new KeypairsModel with the given compute client.
func NewKeypairsModel(cc client.ComputeClient) KeypairsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return KeypairsModel{client: cc, loading: true, spinner: s, ft: common.NewFilterableTable(), width: 120, height: 30}
}

type keypairsDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts the async loading of keypair data.
func (m KeypairsModel) Init() tea.Cmd {
	return func() tea.Msg {
		kpList, err := m.client.ListKeypairs()
		if err != nil {
			return keypairsDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "Name", Width: uiconst.ColWidthName}, {Title: "Fingerprint", Width: uiconst.ColWidthFingerprint}, {Title: "Type", Width: uiconst.ColWidthType}, {Title: "UserID", Width: uiconst.ColWidthUUID}}
		rows := []table.Row{}
		for _, kp := range kpList {
			rows = append(rows, table.Row{kp.Name, kp.Fingerprint, kp.Type, kp.UserID})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return keypairsDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages for the model, including data load, window resize,
// and key handling for filtering.
func (m KeypairsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case keypairsDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.ft.SetTable(msg.tbl, msg.rows)
		m.updateTableColumns()
		m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.ft.Table.Columns() != nil {
			m.ft.SetHeight(m.height - uiconst.TableHeightOffset)
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
		// Normal table navigation.
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

// View renders the model: spinner while loading, error if any, filter UI or the table.
func (m KeypairsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return m.ft.View()
}

// updateTableColumns adjusts column widths based on the current width.
func (m *KeypairsModel) updateTableColumns() {
	fingerprintW := uiconst.ColWidthFingerprint
	typeW := 10
	userIDW := 36
	// Name column gets remaining space.
	nameW := m.width - fingerprintW - typeW - userIDW - 6
	if nameW < 10 {
		nameW = 10
	}
	m.ft.SetColumns([]table.Column{{Title: "Name", Width: nameW}, {Title: "Fingerprint", Width: fingerprintW}, {Title: "Type", Width: typeW}, {Title: "UserID", Width: userIDW}})
}

// Table returns the underlying table model for external callers.
func (m KeypairsModel) Table() table.Model { return m.ft.Table }

var _ tea.Model = (*KeypairsModel)(nil)
