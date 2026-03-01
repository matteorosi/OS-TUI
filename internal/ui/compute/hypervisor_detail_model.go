package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
	"time"
)

// HypervisorDetailModel displays detailed information for a single hypervisor.
type HypervisorDetailModel struct {
	table   table.Model
	loading bool
	err     error
	spinner spinner.Model
	client  client.ComputeClient
	hvID    string
	// JSON view fields
	jsonView     string
	jsonViewport viewport.Model
	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored hypervisor for JSON marshaling
	hypervisor hypervisors.Hypervisor
}

type hypervisorDetailDataLoadedMsg struct {
	tbl table.Model
	err error
	hv  hypervisors.Hypervisor
}

// NewHypervisorDetailModel creates a new HypervisorDetailModel for the given hypervisor ID.
func NewHypervisorDetailModel(cc client.ComputeClient, hvID string) HypervisorDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return HypervisorDetailModel{client: cc, loading: true, spinner: s, hvID: hvID}
}

// Init starts async loading of hypervisor details.
func (m HypervisorDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		hv, err := m.client.GetHypervisor(context.Background(), m.hvID)
		if err != nil {
			return hypervisorDetailDataLoadedMsg{err: err}
		}
		// Build a two‑column table: split fields into two columns, with resource bars for VCPUs and Memory.
		cols := []table.Column{{Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValueShort}, {Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValueShort}}
		rows := []table.Row{{"ID", hv.ID}, {"Hostname", hv.HypervisorHostname}, {"State", hv.State}, {"Status", hv.Status}, {"VCPUs", func() string {
			if hv.VCPUs == 0 {
				return "N/A"
			}
			pct := float64(hv.VCPUsUsed) / float64(hv.VCPUs) * 100
			bar := renderBar(pct)
			return fmt.Sprintf("%s %d/%d", bar, hv.VCPUsUsed, hv.VCPUs)
		}()}, {"Memory", func() string {
			if hv.MemoryMB == 0 {
				return "N/A"
			}
			pct := float64(hv.MemoryMBUsed) / float64(hv.MemoryMB) * 100
			bar := renderBar(pct)
			usedGB := hv.MemoryMBUsed / 1024
			totalGB := hv.MemoryMB / 1024
			return fmt.Sprintf("%s %d/%d GB", bar, usedGB, totalGB)
		}()}, {"Disk GB", fmt.Sprintf("%d", hv.LocalGB)}, {"Disk Used", fmt.Sprintf("%d", hv.LocalGBUsed)}, {"Free RAM MB", fmt.Sprintf("%d", hv.FreeRamMB)}, {"Free Disk GB", fmt.Sprintf("%d", hv.FreeDiskGB)}, {"Host IP", hv.HostIP}, {"Current Workload", fmt.Sprintf("%d", hv.CurrentWorkload)}, {"Running VMs", fmt.Sprintf("%d", hv.RunningVMs)}}
		// Add timestamp for when data was fetched.
		rows = append(rows, table.Row{"Fetched", time.Now().Format(time.RFC3339)})
		// Split rows into two columns.
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
		return hypervisorDetailDataLoadedMsg{tbl: t, hv: *hv}
	}
}

// Update handles messages for the model.
func (m HypervisorDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case hypervisorDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.hypervisor = msg.hv
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
			// Build inspect view for hypervisor.
			content := fmt.Sprintf("=== Hypervisor: %s ===\nID: %s\nHostname: %s\nState: %s\nStatus: %s\nVCPUs: %d\nVCPUs Used: %d\nRAM MB: %d\nRAM Used: %d\nDisk GB: %d\nDisk Used: %d\nFree RAM MB: %d\nFree Disk GB: %d\nHost IP: %s\nCurrent Workload: %d\nRunning VMs: %d\nFetched: %s", m.hypervisor.ID, m.hypervisor.ID, m.hypervisor.HypervisorHostname, m.hypervisor.State, m.hypervisor.Status, m.hypervisor.VCPUs, m.hypervisor.VCPUsUsed, m.hypervisor.MemoryMB, m.hypervisor.MemoryMBUsed, m.hypervisor.LocalGB, m.hypervisor.LocalGBUsed, m.hypervisor.FreeRamMB, m.hypervisor.FreeDiskGB, m.hypervisor.HostIP, m.hypervisor.CurrentWorkload, m.hypervisor.RunningVMs, time.Now().Format(time.RFC3339))
			m.inspectView = content
			m.inspectViewport = viewport.New(80, 24)
			m.inspectViewport.SetContent(m.inspectView)
			return m, nil
		}
		if msg.String() == "y" {
			b, err := json.MarshalIndent(m.hypervisor, "", "  ")
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

// View renders the hypervisor detail view.
func (m HypervisorDetailModel) View() string {
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
		return fmt.Sprintf("Error: %s", m.err)
	}
	return fmt.Sprintf("%s\n[y] json  [i] inspect  [esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m HypervisorDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*HypervisorDetailModel)(nil)
