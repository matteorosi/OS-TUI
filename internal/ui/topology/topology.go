package topology

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"ostui/internal/client"
)

type TopologyModel struct {
	compute  client.ComputeClient
	network  client.NetworkClient
	storage  client.StorageClient
	loading  bool
	err      error
	content  string
	viewport viewport.Model
	spinner  spinner.Model
}

type topologyDataMsg struct {
	content string
	err     error
}

func NewTopologyModel(cc client.ComputeClient, nc client.NetworkClient, sc client.StorageClient) TopologyModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return TopologyModel{compute: cc, network: nc, storage: sc, loading: true, spinner: s, viewport: viewport.New(80, 24)}
}

func (m TopologyModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		content, err := m.buildTopology()
		return topologyDataMsg{content: content, err: err}
	})
}

func (m *TopologyModel) buildTopology() (string, error) {
	ctx := context.Background()
	var (
		srvList    []servers.Server
		netList    []networks.Network
		subList    []subnets.Subnet
		portList   []ports.Port
		fipList    []floatingips.FloatingIP
		volList    []volumes.Volume
		routerList []client.Router
	)
	errChan := make(chan error, 7)
	var wg sync.WaitGroup
	wg.Add(7)
	go func() {
		defer wg.Done()
		var err error
		srvList, err = m.compute.ListInstances()
		if err != nil {
			errChan <- fmt.Errorf("list instances: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		netList, err = m.network.ListNetworks()
		if err != nil {
			errChan <- fmt.Errorf("list networks: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		subList, err = m.network.ListSubnets()
		if err != nil {
			errChan <- fmt.Errorf("list subnets: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		portList, err = m.network.ListPorts(ctx)
		if err != nil {
			errChan <- fmt.Errorf("list ports: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		fipList, err = m.network.ListFloatingIPs()
		if err != nil {
			errChan <- fmt.Errorf("list floating IPs: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		volList, err = m.storage.ListVolumes()
		if err != nil {
			errChan <- fmt.Errorf("list volumes: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		routerList, err = m.network.ListRouters(ctx)
		if err != nil {
			errChan <- fmt.Errorf("list routers: %w", err)
		}
	}()
	wg.Wait()
	close(errChan)
	for e := range errChan {
		if e != nil {
			return "", e
		}
	}

	// Build lookup maps
	netMap := make(map[string]networks.Network)
	for _, n := range netList {
		netMap[n.ID] = n
	}
	subnetMap := make(map[string]subnets.Subnet)
	for _, s := range subList {
		subnetMap[s.ID] = s
	}
	// server map
	serverMap := make(map[string]servers.Server)
	for _, s := range srvList {
		serverMap[s.ID] = s
	}
	// ports per server and per network
	netServers := make(map[string]map[string]bool) // networkID -> set of server IDs
	serverPorts := make(map[string][]ports.Port)
	for _, p := range portList {
		if p.DeviceID != "" {
			serverPorts[p.DeviceID] = append(serverPorts[p.DeviceID], p)
			if _, ok := netServers[p.NetworkID]; !ok {
				netServers[p.NetworkID] = make(map[string]bool)
			}
			netServers[p.NetworkID][p.DeviceID] = true
		}
	}
	// floating IPs per port
	portFIPs := make(map[string][]floatingips.FloatingIP)
	for _, f := range fipList {
		if f.PortID != "" {
			portFIPs[f.PortID] = append(portFIPs[f.PortID], f)
		}
	}
	// volumes per server
	serverVolumes := make(map[string][]volumes.Volume)
	for _, v := range volList {
		for _, att := range v.Attachments {
			if att.ServerID != "" {
				serverVolumes[att.ServerID] = append(serverVolumes[att.ServerID], v)
			}
		}
	}
	// routers per network (using external gateway network ID)
	netRouters := make(map[string][]client.Router)
	for _, r := range routerList {
		if r.GatewayInfo.NetworkID != "" {
			netRouters[r.GatewayInfo.NetworkID] = append(netRouters[r.GatewayInfo.NetworkID], r)
		}
	}

	// Styles
	networkStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5BC0DE"))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5CB85C"))
	shutoffStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F0AD4E"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E74C3C"))
	fipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#C678DD"))
	volStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9B59B6"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	// Helper for server status style
	serverStatusStyle := func(status string) lipgloss.Style {
		switch status {
		case "ACTIVE":
			return activeStyle
		case "SHUTOFF":
			return shutoffStyle
		case "ERROR":
			return errorStyle
		default:
			return dimStyle
		}
	}

	// Tree characters
	branch := dimStyle.Render("├── ")
	lastBranch := dimStyle.Render("└── ")
	indent := dimStyle.Render("│   ")

	var sb strings.Builder
	// Sort networks by name for deterministic output
	netIDs := make([]string, 0, len(netList))
	for _, n := range netList {
		netIDs = append(netIDs, n.ID)
	}
	sort.Slice(netIDs, func(i, j int) bool {
		return netMap[netIDs[i]].Name < netMap[netIDs[j]].Name
	})

	for _, nid := range netIDs {
		n := netMap[nid]
		// Determine CIDR from first subnet if available
		cidr := ""
		if len(n.Subnets) > 0 {
			if s, ok := subnetMap[n.Subnets[0]]; ok {
				cidr = s.CIDR
			}
		}
		header := fmt.Sprintf("Network: %s (%s)", n.Name, cidr)
		sb.WriteString(networkStyle.Render(header))
		sb.WriteString("\n")
		// Servers in this network
		serverSet := netServers[nid]
		// Convert set to slice
		srvIDs := make([]string, 0, len(serverSet))
		for sid := range serverSet {
			srvIDs = append(srvIDs, sid)
		}
		sort.Slice(srvIDs, func(i, j int) bool {
			return serverMap[srvIDs[i]].Name < serverMap[srvIDs[j]].Name
		})
		for si, sid := range srvIDs {
			srv := serverMap[sid]
			// Determine prefix for server line
			isLastServer := si == len(srvIDs)-1 && len(netRouters[nid]) == 0
			prefix := branch
			if isLastServer {
				prefix = lastBranch
			}
			srvLine := fmt.Sprintf("Server: %s [%s]", srv.Name, srv.Status)
			sb.WriteString(prefix + serverStatusStyle(srv.Status).Render(srvLine))
			sb.WriteString("\n")
			// Ports for server
			ports := serverPorts[srv.ID]
			sort.Slice(ports, func(i, j int) bool { return ports[i].ID < ports[j].ID })
			for pi, p := range ports {
				// Determine prefix for port line
				portIsLast := pi == len(ports)-1 && len(serverVolumes[srv.ID]) == 0 && len(portFIPs[p.ID]) == 0
				portPrefix := indent
				if portIsLast {
					portPrefix += lastBranch
				} else {
					portPrefix += branch
				}
				ip := ""
				if len(p.FixedIPs) > 0 {
					ip = p.FixedIPs[0].IPAddress
				}
				sb.WriteString(portPrefix + fmt.Sprintf("Port: %s", ip))
				sb.WriteString("\n")
				// Floating IPs attached to this port
				fips := portFIPs[p.ID]
				for fi, f := range fips {
					fipPrefix := indent + "    "
					if fi == len(fips)-1 {
						fipPrefix += lastBranch
					} else {
						fipPrefix += branch
					}
					sb.WriteString(fipPrefix + fipStyle.Render(fmt.Sprintf("FIP: %s", f.FloatingIP)))
					sb.WriteString("\n")
				}
			}
			// Volumes attached to server
			vols := serverVolumes[srv.ID]
			for vi, v := range vols {
				volIsLast := vi == len(vols)-1
				volPrefix := indent
				if volIsLast {
					volPrefix += lastBranch
				} else {
					volPrefix += branch
				}
				device := ""
				if len(v.Attachments) > 0 {
					device = v.Attachments[0].Device
				}
				sb.WriteString(volPrefix + volStyle.Render(fmt.Sprintf("Vol: %s %dGB", device, v.Size)))
				sb.WriteString("\n")
			}
		}
		// Routers for this network
		routers := netRouters[nid]
		for ri, r := range routers {
			routerIsLast := ri == len(routers)-1
			routerPrefix := branch
			if routerIsLast {
				routerPrefix = lastBranch
			}
			sb.WriteString(routerPrefix + fmt.Sprintf("Router: %s", r.Name))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	// Unattached resources
	var unattachedFIPs []floatingips.FloatingIP
	for _, f := range fipList {
		if f.PortID == "" {
			unattachedFIPs = append(unattachedFIPs, f)
		}
	}
	var unattachedVols []volumes.Volume
	for _, v := range volList {
		if len(v.Attachments) == 0 {
			unattachedVols = append(unattachedVols, v)
		}
	}
	if len(unattachedFIPs) > 0 || len(unattachedVols) > 0 {
		sb.WriteString("Unattached resources:\n")
		for i, f := range unattachedFIPs {
			isLast := i == len(unattachedFIPs)-1 && len(unattachedVols) == 0
			prefix := branch
			if isLast {
				prefix = lastBranch
			}
			sb.WriteString(prefix + fipStyle.Render(fmt.Sprintf("FIP: %s (not associated)", f.FloatingIP)))
			sb.WriteString("\n")
		}
		for i, v := range unattachedVols {
			isLast := i == len(unattachedVols)-1
			prefix := branch
			if isLast {
				prefix = lastBranch
			}
			sb.WriteString(prefix + volStyle.Render(fmt.Sprintf("Vol: %s %dGB (available)", v.Name, v.Size)))
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

func (m TopologyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case topologyDataMsg:
		m.loading = false
		m.content = msg.content
		m.err = msg.err
		m.viewport.SetContent(m.content)
		return m, nil
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 3
		m.viewport.SetContent(m.content)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return CloseMsg{} }
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

func (m TopologyModel) View() string {
	if m.loading {
		return m.spinner.View() + " Loading topology..."
	}
	header := "Topology"
	footer := fmt.Sprintf(" %3.f%% | [j/k] scroll  [esc] close", m.viewport.ScrollPercent()*100)
	return header + "\n" + m.viewport.View() + "\n" + footer
}

type CloseMsg struct{}

var _ tea.Model = (*TopologyModel)(nil)
