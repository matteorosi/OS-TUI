package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"ostui/internal/client"
)

// InstanceDetailModel displays detailed information for a single compute instance.
// It follows the same pattern as other subview models: async loading, spinner while loading,
// and a table view once data is available.
type InstanceDetailModel struct {
	table   table.Model
	loading bool
	err     error
	spinner spinner.Model
	// client holds the compute client used for instance operations.
	client client.ComputeClient
	// network and storage clients are required for the server graph view.
	network client.NetworkClient
	storage client.StorageClient
	// instanceID identifies the instance to fetch.
	instanceID string
	// console handling fields
	consoleURL     string
	showConsole    bool
	consoleLoading bool
	consoleErr     error
	// JSON view fields
	jsonView     string
	jsonViewport viewport.Model
	// Inspect view fields
	inspectView     string
	inspectViewport viewport.Model
	// stored instance for JSON marshaling and for graph view.
	instance servers.Server
	// graphModel renders the server relationship graph.
	graphModel *ServerGraphModel
	// showGraph toggles the graph view.
	showGraph bool
}

// IsShowingGraph returns true if the graph view is currently displayed.
func (m InstanceDetailModel) IsShowingGraph() bool { return m.showGraph }

type instanceDetailDataLoadedMsg struct {
	tbl      table.Model
	err      error
	instance servers.Server
}

type consoleURLLoadedMsg struct {
	url string
	err error
}

// NewInstanceDetailModel creates a new InstanceDetailModel for the given instance ID.
func NewInstanceDetailModel(cc client.ComputeClient, nc client.NetworkClient, sc client.StorageClient, instanceID string) InstanceDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	// Initialise with loading true; the table will be set after data is loaded.
	return InstanceDetailModel{client: cc, network: nc, storage: sc, loading: true, spinner: s, instanceID: instanceID}
}

// Init starts the async loading of the instance details.
func (m InstanceDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		srv, err := m.client.GetInstance(m.instanceID)
		if err != nil {
			return instanceDetailDataLoadedMsg{err: err}
		}
		// Build a two‑column table: split fields into two columns.
		cols := []table.Column{{Title: "Field", Width: 20}, {Title: "Value", Width: 30}, {Title: "Field", Width: 20}, {Title: "Value", Width: 30}}
		rows := []table.Row{{"ID", srv.ID}, {"Name", srv.Name}, {"Status", srv.Status}, {"Flavor", fmt.Sprintf("%v", srv.Flavor["id"])}, {"Image", fmt.Sprintf("%v", srv.Image["id"])}, {"Created", srv.Created.Format(time.RFC3339)}, {"Updated", srv.Updated.Format(time.RFC3339)}, {"HostID", srv.HostID}, {"KeyName", srv.KeyName}, {"UserID", srv.UserID}, {"TenantID", srv.TenantID}}
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
		return instanceDetailDataLoadedMsg{tbl: t, instance: srv}
	}
}

// Update handles messages for the model.
func (m InstanceDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If graph view is active, forward messages to the graph model.
	if m.showGraph && m.graphModel != nil {
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd = m.graphModel.Update(msg)
		if gm, ok := newModel.(ServerGraphModel); ok {
			*m.graphModel = gm
		}
		return m, cmd
	}
	switch msg := msg.(type) {
	case instanceDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.instance = msg.instance
		return m, nil
	case consoleURLLoadedMsg:
		m.consoleLoading = false
		if msg.err != nil {
			m.consoleErr = msg.err
		} else {
			m.consoleURL = msg.url
		}
		m.showConsole = true
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
		// If console view is active, handle its keys.
		if m.showConsole {
			if msg.String() == "o" && m.consoleURL != "" {
				// Open URL in default browser.
				var cmd *exec.Cmd
				if runtime.GOOS == "darwin" {
					cmd = exec.Command("open", m.consoleURL)
				} else {
					cmd = exec.Command("xdg-open", m.consoleURL)
				}
				// Run command asynchronously; ignore errors.
				_ = cmd.Start()
				return m, nil
			}
			// Any other key closes the console view.
			m.showConsole = false
			return m, nil
		}
		if m.loading || m.err != nil {
			// Ignore key input while loading or on error.
			return m, nil
		}
		// Custom key handling for opening logs, inspect, and console.
		if msg.String() == "l" {
			// Emit openLogsMsg with the instance ID.
			return m, func() tea.Msg { return OpenLogsMsg{ServerID: m.instanceID} }
		}
		if msg.String() == "i" {
			// Build inspect view for instance.
			content := fmt.Sprintf("=== Instance: %s ===\nID: %s\nName: %s\nStatus: %s\nFlavor: %s\nImage: %s\nCreated: %s\nUpdated: %s\nHostID: %s\nKeyName: %s\nUserID: %s\nTenantID: %s", m.instance.Name, m.instance.ID, m.instance.Name, m.instance.Status, fmt.Sprintf("%v", m.instance.Flavor["id"]), fmt.Sprintf("%v", m.instance.Image["id"]), m.instance.Created.Format(time.RFC3339), m.instance.Updated.Format(time.RFC3339), m.instance.HostID, m.instance.KeyName, m.instance.UserID, m.instance.TenantID)
			m.inspectView = content
			m.inspectViewport = viewport.New(80, 24)
			m.inspectViewport.SetContent(m.inspectView)
			return m, nil
		}
		if msg.String() == "v" {
			// Fetch console URL.
			m.consoleLoading = true
			return m, func() tea.Msg {
				url, err := m.client.GetConsoleURL(context.Background(), m.instanceID, "vnc")
				return consoleURLLoadedMsg{url: url, err: err}
			}
		}
		if msg.String() == "g" {
			// Initialize graph model if not already
			if m.graphModel == nil {
				gm := NewServerGraphModel(m.client, m.network, m.storage, m.instanceID, m.instance.Name)
				m.graphModel = &gm
			}
			m.showGraph = true
			return m, m.graphModel.Init()
		}
		if msg.String() == "y" {
			// Marshal instance to JSON.
			b, err := json.MarshalIndent(m.instance, "", "  ")
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
	case GoBackMsg:
		// Hide graph view
		m.showGraph = false
		m.graphModel = nil
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

// View renders the model: spinner while loading, error message on failure, or the table.
func (m InstanceDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.showGraph && m.graphModel != nil {
		return m.graphModel.View()
	}
	if m.jsonView != "" {
		return fmt.Sprintf("%s\nPress 'y' or 'esc' to close", m.jsonViewport.View())
	}
	if m.inspectView != "" {
		return fmt.Sprintf("%s\n %3.f%% | [j/k] scroll  [esc] close", m.inspectViewport.View(), m.inspectViewport.ScrollPercent()*100)
	}
	if m.consoleLoading {
		return "Fetching console URL..."
	}
	if m.showConsole {
		if m.consoleErr != nil {
			return fmt.Sprintf("Error fetching console URL: %s\nPress any key to return", m.consoleErr)
		}
		return fmt.Sprintf("Console URL: %s\nPress 'o' to open in browser, any other key to return", m.consoleURL)
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return fmt.Sprintf("%s\n[l] logs  [y] json  [i] inspect  [v] console  [g] graph  [esc] back", m.table.View())
}

// Ensure InstanceDetailModel implements tea.Model.
var _ tea.Model = (*InstanceDetailModel)(nil)
