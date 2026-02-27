package compute

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
)

// LogsModel implements a streaming log viewer for a compute server.
// It periodically fetches console logs via the ComputeClient and displays them
// in a viewport. Users can toggle streaming, scroll, adjust the refresh interval,
// and return to the previous view.
type LogsModel struct {
	viewport  viewport.Model
	content   string
	serverID  string
	client    client.ComputeClient
	streaming bool
	interval  time.Duration
	err       error
}

// NewLogsModel creates a new LogsModel for the given server ID.
// The default refresh interval is 1 second and streaming is enabled.
func NewLogsModel(cc client.ComputeClient, serverID string) LogsModel {
	return LogsModel{
		client:    cc,
		serverID:  serverID,
		streaming: true,
		interval:  time.Second,
		viewport:  viewport.New(0, 0),
	}
}

// fetchLogsCmd returns a command that fetches the console log for the server.
func (m LogsModel) fetchLogsCmd() tea.Cmd {
	return func() tea.Msg {
		// Use 0 to fetch the full log (OpenStack API semantics).
		content, err := m.client.GetConsoleLog(m.serverID, 0)
		return logChunkMsg{content: content, err: err}
	}
}

// Init fetches the initial logs and starts the periodic ticker.
func (m LogsModel) Init() tea.Cmd {
	// Fetch logs now and schedule the first tick.
	return tea.Batch(
		m.fetchLogsCmd(),
		tea.Tick(m.interval, func(t time.Time) tea.Msg { return logTickMsg{} }),
	)
}

// nextInterval returns the next interval step based on the current value.
// If increase is true, it moves to a larger interval; otherwise to a smaller one.
func nextInterval(current time.Duration, increase bool) time.Duration {
	steps := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second}
	// Find current index.
	idx := 0
	for i, d := range steps {
		if d == current {
			idx = i
			break
		}
	}
	if increase && idx < len(steps)-1 {
		idx++
	} else if !increase && idx > 0 {
		idx--
	}
	return steps[idx]
}

// Update handles incoming messages and user input.
func (m LogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case logChunkMsg:
		m.content = msg.content
		m.err = msg.err
		// Update viewport content.
		if m.viewport.Width == 0 {
			m.viewport.Width = 80
			m.viewport.Height = 24
		}
		m.viewport.SetContent(m.content)
		if m.streaming {
			m.viewport.GotoBottom()
		}
		return m, nil
	case logTickMsg:
		// On each tick, fetch logs and schedule the next tick.
		return m, tea.Batch(
			m.fetchLogsCmd(),
			tea.Tick(m.interval, func(t time.Time) tea.Msg { return logTickMsg{} }),
		)
	case tea.WindowSizeMsg:
		// Adjust viewport size, leaving space for the header (2 lines).
		m.viewport.Width = msg.Width
		// Ensure we have at least 1 line for viewport.
		if msg.Height > 2 {
			m.viewport.Height = msg.Height - 2
		} else {
			m.viewport.Height = 1
		}
		// Ensure content is set.
		if m.viewport.Width == 0 {
			m.viewport.Width = 80
			m.viewport.Height = 24
		}
		m.viewport.SetContent(m.content)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "p":
			m.streaming = !m.streaming
			return m, nil
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		case "+":
			m.interval = nextInterval(m.interval, true)
			return m, nil
		case "-":
			m.interval = nextInterval(m.interval, false)
			return m, nil
		case "esc":
			// Signal to go back to the previous view.
			return m, func() tea.Msg { return GoBackMsg{} }
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// View renders the header and the viewport.
func (m LogsModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	header := fmt.Sprintf("Server: %s | Streaming: %t | Interval: %s", m.serverID, m.streaming, m.interval)
	footer := fmt.Sprintf(" %3.f%% | [j/k] scroll [g/G] top/bottom [p] pause [esc] back", m.viewport.ScrollPercent()*100)
	return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer)
}

// Ensure LogsModel implements tea.Model.
var _ tea.Model = (*LogsModel)(nil)
