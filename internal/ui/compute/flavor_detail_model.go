package compute

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
)

// FlavorDetailModel displays detailed information for a single compute flavor.
// It follows the same pattern as ImageDetailModel.
type FlavorDetailModel struct {
	table    table.Model
	loading  bool
	err      error
	spinner  spinner.Model
	client   client.ComputeClient
	flavorID string
}

type flavorDetailDataLoadedMsg struct {
	tbl table.Model
	err error
}

// NewFlavorDetailModel creates a new FlavorDetailModel for the given flavor ID.
func NewFlavorDetailModel(cc client.ComputeClient, flavorID string) FlavorDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return FlavorDetailModel{client: cc, loading: true, spinner: s, flavorID: flavorID}
}

// Init starts the async loading of the flavor details.
func (m FlavorDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		f, err := m.client.GetFlavor(context.Background(), m.flavorID)
		if err != nil {
			return flavorDetailDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValue}}
		rows := []table.Row{{"ID", f.ID}, {"Name", f.Name}, {"VCPUs", fmt.Sprintf("%d", f.VCPUs)}, {"RAM (MB)", fmt.Sprintf("%d", f.RAM)}, {"Disk (GB)", fmt.Sprintf("%d", f.Disk)}, {"Swap", fmt.Sprintf("%d", f.Swap)}, {"IsPublic", fmt.Sprintf("%v", f.IsPublic)}}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		return flavorDetailDataLoadedMsg{tbl: t}
	}
}

// Update handles messages.
func (m FlavorDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case flavorDetailDataLoadedMsg:
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

// View renders the flavor detail view.
func (m FlavorDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return fmt.Sprintf("%s\n[esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m FlavorDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*FlavorDetailModel)(nil)
