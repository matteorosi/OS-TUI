package compute

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
	"strings"
)

// HypervisorsModel implements a subview for listing OpenStack hypervisors.
type HypervisorsModel struct {
	table      table.Model
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.ComputeClient
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model
	// Dynamic sizing
	width  int
	height int
}

// NewHypervisorsModel creates a new HypervisorsModel.
func NewHypervisorsModel(cc client.ComputeClient) HypervisorsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	// Initialize with reasonable defaults.
	return HypervisorsModel{client: cc, loading: true, spinner: s, filter: ti, width: 120, height: 30}
}

type hypervisorsDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts async loading of hypervisors.
func (m HypervisorsModel) Init() tea.Cmd {
	return func() tea.Msg {
		hvList, err := m.client.ListHypervisors(context.Background())
		if err != nil {
			return hypervisorsDataLoadedMsg{err: err}
		}
		// Define a concise set of columns.
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Hostname", Width: uiconst.ColWidthName}, {Title: "State", Width: uiconst.ColWidthProtocol}, {Title: "Status", Width: uiconst.ColWidthEnabled}, {Title: "VCPUs", Width: uiconst.ColWidthProtocol}, {Title: "VCPUs Used", Width: uiconst.ColWidthType}, {Title: "RAM MB", Width: uiconst.ColWidthEnabled}, {Title: "RAM Used", Width: uiconst.ColWidthRAMUsed}, {Title: "Disk GB", Width: uiconst.ColWidthEnabled}, {Title: "Disk Used", Width: uiconst.ColWidthRAMUsed}}
		rows := []table.Row{}
		for _, hv := range hvList {
			rows = append(rows, table.Row{hv.ID, hv.HypervisorHostname, hv.State, hv.Status, fmt.Sprintf("%d", hv.VCPUs), fmt.Sprintf("%d", hv.VCPUsUsed), fmt.Sprintf("%d", hv.MemoryMB), fmt.Sprintf("%d", hv.MemoryMBUsed), fmt.Sprintf("%d", hv.LocalGB), fmt.Sprintf("%d", hv.LocalGBUsed)})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return hypervisorsDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages for the model.
func (m HypervisorsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case hypervisorsDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.allRows = msg.rows
		// Adjust columns and height based on current dimensions.
		m.updateTableColumns()
		m.table.SetHeight(m.height - uiconst.TableHeightOffset)
		return m, nil
	case tea.WindowSizeMsg:
		// Update stored dimensions and adjust table.
		m.width = msg.Width
		m.height = msg.Height
		if m.table.Columns() != nil {
			m.table.SetHeight(m.height - uiconst.TableHeightOffset)
			m.updateTableColumns()
		}
		return m, nil
	case tea.KeyMsg:
		// Filter mode handling – same pattern as InstancesModel.
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
		// Normal navigation.
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

// View renders the hypervisors view.
func (m HypervisorsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return m.table.View()
}

// updateTableColumns adjusts column widths based on the current width.
func (m *HypervisorsModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	// Fixed column widths.
	stateW := uiconst.ColWidthProtocol
	statusW := uiconst.ColWidthEnabled
	vcpusW := uiconst.ColWidthProtocol
	vcpusUsedW := uiconst.ColWidthType
	ramW := uiconst.ColWidthEnabled
	ramUsedW := uiconst.ColWidthRAMUsed
	diskW := uiconst.ColWidthEnabled
	diskUsedW := uiconst.ColWidthDiskUsed
	// Compute flexible hostname width.
	fixedTotal := idW + stateW + statusW + vcpusW + vcpusUsedW + ramW + ramUsedW + diskW + diskUsedW + uiconst.TableHeightOffset // margin
	hostnameW := m.width - fixedTotal
	if hostnameW < 10 {
		hostnameW = 10
	}
	m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Hostname", Width: hostnameW}, {Title: "State", Width: stateW}, {Title: "Status", Width: statusW}, {Title: "VCPUs", Width: vcpusW}, {Title: "VCPUs Used", Width: vcpusUsedW}, {Title: "RAM MB", Width: ramW}, {Title: "RAM Used", Width: ramUsedW}, {Title: "Disk GB", Width: diskW}, {Title: "Disk Used", Width: diskUsedW}})
}

// Table returns the underlying table model.
func (m HypervisorsModel) Table() table.Model { return m.table }

var _ tea.Model = (*HypervisorsModel)(nil)
