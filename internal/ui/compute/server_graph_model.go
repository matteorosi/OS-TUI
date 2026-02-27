package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"ostui/internal/client"
)

type graphNode struct {
	title string
	lines []string
}

type ServerGraphModel struct {
	serverID   string
	serverName string
	loading    bool
	err        error
	spinner    spinner.Model
	content    string
	viewport   viewport.Model
	compute    client.ComputeClient
	network    client.NetworkClient
	storage    client.StorageClient
}

type graphDataMsg struct {
	content string
	err     error
}

func NewServerGraphModel(cc client.ComputeClient, nc client.NetworkClient, sc client.StorageClient, serverID, serverName string) ServerGraphModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	vp := viewport.New(80, 24)
	return ServerGraphModel{compute: cc, network: nc, storage: sc, serverID: serverID, serverName: serverName, loading: true, spinner: s, viewport: vp}
}

func (m ServerGraphModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		return m.buildGraph()
	})
}

func (m ServerGraphModel) buildGraph() tea.Msg {
	// 1. Get server interfaces (ports)
	ifaces, err := m.compute.ListServerInterfaces(context.Background(), m.serverID)
	if err != nil {
		return graphDataMsg{err: err}
	}

	// 2. Get server volumes
	vols, err := m.compute.ListServerVolumes(context.Background(), m.serverID)
	if err != nil {
		return graphDataMsg{err: err}
	}

	// 3. Get floating IPs (all, filter by port)
	fips, _ := m.network.ListFloatingIPs()

	// 4. Build boxes using lipgloss
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	serverStyle := boxStyle.BorderForeground(lipgloss.Color("#5CB85C"))
	portStyle := boxStyle.BorderForeground(lipgloss.Color("#F0AD4E"))
	netStyle := boxStyle.BorderForeground(lipgloss.Color("#5BC0DE"))
	volStyle := boxStyle.BorderForeground(lipgloss.Color("#9B59B6"))
	fipStyle := boxStyle.BorderForeground(lipgloss.Color("#E74C3C"))

	// Build server box
	serverBox := serverStyle.Render(fmt.Sprintf("Server: %s", m.serverName))

	// Build volume boxes
	var volBoxes []string
	for _, v := range vols {
		// Show first 8 chars of volume ID for brevity
		id := v.VolumeID
		if len(id) > 8 {
			id = id[:8]
		}
		volBoxes = append(volBoxes, volStyle.Render(fmt.Sprintf("Vol: %s %s", v.Device, id)))
	}

	// Build port+network+fip columns
	var portCol, netCol, fipCol []string
	for _, iface := range ifaces {
		portBox := portStyle.Render(fmt.Sprintf("Port\nIP: %s", strings.Join(iface.FixedIPs, ", ")))
		portCol = append(portCol, portBox)

		net, _ := m.network.GetNetwork(context.Background(), iface.NetworkID)
		if net != nil {
			netBox := netStyle.Render(fmt.Sprintf("Net: %s", net.Name))
			netCol = append(netCol, netBox)
		}

		for _, fip := range fips {
			if fip.PortID == iface.PortID {
				fipCol = append(fipCol, fipStyle.Render(fmt.Sprintf("FIP: %s", fip.FloatingIP)))
			}
		}
	}

	// Compose layout: volumes on top (wrapped), then columns with overflow handling
	var sb strings.Builder
	if len(volBoxes) > 0 {
		// Helper to chunk strings into rows of max 4
		chunkStrings := func(s []string, size int) [][]string {
			var chunks [][]string
			for size < len(s) {
				s, chunks = s[size:], append(chunks, s[0:size:size])
			}
			return append(chunks, s)
		}
		var volRows []string
		for _, chunk := range chunkStrings(volBoxes, 4) {
			volRows = append(volRows, lipgloss.JoinHorizontal(lipgloss.Top, chunk...))
		}
		sb.WriteString(lipgloss.JoinVertical(lipgloss.Left, volRows...))
		sb.WriteString("\n")
		sb.WriteString("  │\n")
	}

	// Limit ports display to maxPorts with a "+N more" indicator
	const maxPorts = 8
	if len(portCol) > maxPorts {
		extra := len(portCol) - maxPorts
		portCol = append(portCol[:maxPorts], portStyle.Render(fmt.Sprintf("+%d more", extra)))
	}

	// Determine if we need to stack columns vertically based on viewport width
	shouldStack := false
	if m.viewport.Width > 0 {
		// Compute total width of horizontal layout
		totalWidth := lipgloss.Width(serverBox)
		maxColWidth := func(col []string) int {
			max := 0
			for _, s := range col {
				if w := lipgloss.Width(s); w > max {
					max = w
				}
			}
			return max
		}
		if len(portCol) > 0 {
			w := maxColWidth(portCol)
			totalWidth += 3 + w // separator width approx 3
		}
		if len(netCol) > 0 {
			w := maxColWidth(netCol)
			totalWidth += 3 + w
		}
		if len(fipCol) > 0 {
			w := maxColWidth(fipCol)
			totalWidth += 3 + w
		}
		if totalWidth > m.viewport.Width {
			shouldStack = true
		}
	}

	if shouldStack {
		// Stack columns vertically
		var sections []string
		sections = append(sections, serverBox)
		if len(portCol) > 0 {
			sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, portCol...))
		}
		if len(netCol) > 0 {
			sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, netCol...))
		}
		if len(fipCol) > 0 {
			sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, fipCol...))
		}
		sb.WriteString(lipgloss.JoinVertical(lipgloss.Left, sections...))
	} else {
		// Horizontal layout as before
		row := []string{serverBox}
		if len(portCol) > 0 {
			row = append(row, " ── ")
			row = append(row, lipgloss.JoinVertical(lipgloss.Left, portCol...))
		}
		if len(netCol) > 0 {
			row = append(row, " ── ")
			row = append(row, lipgloss.JoinVertical(lipgloss.Left, netCol...))
		}
		if len(fipCol) > 0 {
			row = append(row, " ── ")
			row = append(row, lipgloss.JoinVertical(lipgloss.Left, fipCol...))
		}
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, row...))
	}
	sb.WriteString("\n\n [g] close  [j/k] scroll")

	return graphDataMsg{content: sb.String()}
}

func (m ServerGraphModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case graphDataMsg:
		m.loading = false
		m.err = msg.err
		m.content = msg.content
		m.viewport.SetContent(m.content)
		return m, nil
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 2
		m.viewport.SetContent(m.content)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "g", "esc":
			return m, func() tea.Msg { return GoBackMsg{} }
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	default:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m ServerGraphModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return m.viewport.View()
}

var _ tea.Model = (*ServerGraphModel)(nil)
