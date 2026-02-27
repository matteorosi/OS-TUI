package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"ostui/internal/client"
	"ostui/internal/ui/compute"
)

type ResourceType string

const (
	ResourceServer       ResourceType = "server"
	ResourceNetwork      ResourceType = "network"
	ResourceVolume       ResourceType = "volume"
	ResourceFloatingIP   ResourceType = "floatingip"
	ResourceRouter       ResourceType = "router"
	ResourceSubnet       ResourceType = "subnet"
	ResourcePort         ResourceType = "port"
	ResourceLoadBalancer ResourceType = "loadbalancer"
)

type GraphModel struct {
	resourceType ResourceType
	resourceID   string
	resourceName string
	compute      client.ComputeClient
	network      client.NetworkClient
	storage      client.StorageClient
	lb           client.LoadBalancerClient
	loading      bool
	err          error
	content      string
	spinner      spinner.Model
	viewport     viewport.Model
}

type graphDataMsg struct {
	content string
	err     error
}

func NewGraphModel(rt ResourceType, id, name string,
	cc client.ComputeClient, nc client.NetworkClient,
	sc client.StorageClient, lbc client.LoadBalancerClient) GraphModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return GraphModel{
		resourceType: rt, resourceID: id, resourceName: name,
		compute: cc, network: nc, storage: sc, lb: lbc,
		loading: true, spinner: s, viewport: viewport.New(80, 24),
	}
}

func (m GraphModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		content, err := m.buildGraph()
		return graphDataMsg{content: content, err: err}
	})
}

func (m GraphModel) buildGraph() (string, error) {
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	centerStyle := boxStyle.BorderForeground(lipgloss.Color("#5CB85C"))
	portStyle := boxStyle.BorderForeground(lipgloss.Color("#F0AD4E"))
	netStyle := boxStyle.BorderForeground(lipgloss.Color("#5BC0DE"))
	volStyle := boxStyle.BorderForeground(lipgloss.Color("#9B59B6"))
	fipStyle := boxStyle.BorderForeground(lipgloss.Color("#E74C3C"))
	lbStyle := boxStyle.BorderForeground(lipgloss.Color("#1ABC9C"))

	switch m.resourceType {
	case ResourceServer:
		centerBox := centerStyle.Render(fmt.Sprintf("Server\n%s", m.resourceName))
		var row []string
		ifaces, err := m.compute.ListServerInterfaces(context.Background(), m.resourceID)
		if err == nil && len(ifaces) > 0 {
			var portBoxes []string
			var netBoxes []string
			var fipBoxes []string
			fips, _ := m.network.ListFloatingIPs()
			for _, iface := range ifaces {
				portBoxes = append(portBoxes, portStyle.Render(fmt.Sprintf("Port\n%s", strings.Join(iface.FixedIPs, ","))))
				net, _ := m.network.GetNetwork(context.Background(), iface.NetworkID)
				if net != nil {
					netBoxes = append(netBoxes, netStyle.Render(fmt.Sprintf("Net\n%s", net.Name)))
				}
				for _, fip := range fips {
					if fip.PortID == iface.PortID {
						fipBoxes = append(fipBoxes, fipStyle.Render(fmt.Sprintf("FIP\n%s", fip.FloatingIP)))
					}
				}
			}
			row = append(row, centerBox, " ── ", lipgloss.JoinVertical(lipgloss.Left, portBoxes...))
			if len(netBoxes) > 0 {
				row = append(row, " ── ", lipgloss.JoinVertical(lipgloss.Left, netBoxes...))
			}
			if len(fipBoxes) > 0 {
				row = append(row, " ── ", lipgloss.JoinVertical(lipgloss.Left, fipBoxes...))
			}
		} else {
			row = []string{centerBox}
		}
		var sb strings.Builder
		vols, _ := m.compute.ListServerVolumes(context.Background(), m.resourceID)
		if len(vols) > 0 {
			var volBoxes []string
			for _, v := range vols {
				volBoxes = append(volBoxes, volStyle.Render(fmt.Sprintf("Vol\n%s", v.Device)))
			}
			sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, volBoxes...) + "\n  │\n")
		}
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, row...))
		return sb.String(), nil
	case ResourceNetwork:
		centerBox := centerStyle.Render(fmt.Sprintf("Network\n%s", m.resourceName))
		var row []string
		row = append(row, centerBox)
		ports, err := m.network.ListPortsByNetwork(context.Background(), m.resourceID)
		if err == nil && len(ports) > 0 {
			var portBoxes []string
			for _, p := range ports[:min(5, len(ports))] {
				portBoxes = append(portBoxes, portStyle.Render(fmt.Sprintf("Port\n%s", p.MACAddress)))
			}
			row = append(row, " ── ", lipgloss.JoinVertical(lipgloss.Left, portBoxes...))
		}
		return lipgloss.JoinHorizontal(lipgloss.Center, row...), nil
	case ResourceVolume:
		centerBox := centerStyle.Render(fmt.Sprintf("Volume\n%s", m.resourceName))
		var row []string
		row = append(row, centerBox)
		vol, err := m.storage.GetVolume(m.resourceID)
		if err == nil {
			for _, att := range vol.Attachments {
				srv, err := m.compute.GetInstance(att.ServerID)
				if err == nil {
					row = append(row, " ── ", centerStyle.Render(fmt.Sprintf("Server\n%s", srv.Name)))
				}
			}
		}
		return lipgloss.JoinHorizontal(lipgloss.Center, row...), nil
	case ResourceFloatingIP:
		centerBox := fipStyle.Render(fmt.Sprintf("FloatingIP\n%s", m.resourceName))
		return centerBox, nil
	case ResourceLoadBalancer:
		centerBox := lbStyle.Render(fmt.Sprintf("LoadBalancer\n%s", m.resourceName))
		var sb strings.Builder
		sb.WriteString(centerBox)
		if m.lb != nil {
			listeners, err := m.lb.ListListeners(context.Background(), m.resourceID)
			if err == nil && len(listeners) > 0 {
				var lBoxes []string
				for _, l := range listeners {
					lBoxes = append(lBoxes, portStyle.Render(fmt.Sprintf("Listener\n%s:%d", l.Protocol, l.ProtocolPort)))
				}
				sb.WriteString("\n  │\n")
				sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, lBoxes...))
			}
			pools, err := m.lb.ListPools(context.Background(), m.resourceID)
			if err == nil && len(pools) > 0 {
				var pBoxes []string
				for _, p := range pools {
					pBoxes = append(pBoxes, netStyle.Render(fmt.Sprintf("Pool\n%s", p.Name)))
				}
				sb.WriteString("\n  │\n")
				sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, pBoxes...))
			}
		}
		return sb.String(), nil
	default:
		return fmt.Sprintf("Graph not available for %s", m.resourceType), nil
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m GraphModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m, func() tea.Msg { return compute.GoBackMsg{} }
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

func (m GraphModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return m.viewport.View()
}

var _ tea.Model = (*GraphModel)(nil)
