package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"ostui/internal/client"
	"ostui/internal/ui/compute"
	"ostui/internal/ui/dns"
	"ostui/internal/ui/graph"
	"ostui/internal/ui/identity"
	"ostui/internal/ui/image"
	"ostui/internal/ui/loadbalancer"
	"ostui/internal/ui/network"
	"ostui/internal/ui/shell"
	"ostui/internal/ui/storage"
)

// item represents a selectable entry in the sidebar.
type item struct {
	title       string
	description string
}

// item implements list.Item
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

type cloudItem struct {
	name string
}

// cloudItem implements list.Item
func (c cloudItem) Title() string       { return c.name }
func (c cloudItem) Description() string { return "" }
func (c cloudItem) FilterValue() string { return c.name }

// UI states for the root model.
const (
	stateSidebar     = "sidebar"
	stateMain        = "main"
	stateModal       = "modal"
	stateHelp        = "help"
	stateCloudSelect = "cloudSelect"
	stateDetail      = "detail"
	stateLogs        = "logs"
	stateCommand     = "command"
	stateShell       = "shell"
	stateGraph       = "graph"
)

// AppModel is the root model of the TUI, managing a simple state machine.
type AppModel struct {
	provider       *gophercloud.ProviderClient
	cloudName      string
	computeClient  client.ComputeClient
	networkClient  client.NetworkClient
	storageClient  client.StorageClient
	identityClient client.IdentityClient
	imageClient    client.ImageClient
	limitsClient   client.LimitsClient
	dnsClient      client.DNSClient
	lbClient       client.LoadBalancerClient
	sidebar        list.Model
	width          int
	height         int
	state          string
	prevState      string
	// selectedItem holds the item chosen from the sidebar when entering the main view.
	selectedItem item
	// modalActive indicates whether a modal overlay is shown.
	modalActive bool
	// cloudList holds the list of clouds for selection.
	cloudList list.Model
	// mainModel holds the currently active subview model (e.g., InstancesModel, NetworksModel).
	// It implements tea.Model and is updated/rendered when the user navigates into a
	// sidebar entry. When no subview is active (e.g., in the sidebar state) this field
	// is nil.
	mainModel tea.Model
	// detailModel holds the active drill-down view.
	detailModel tea.Model
	graphModel  tea.Model
	// logsModel holds the logs view for a server.
	logsModel tea.Model
	// shellModel holds the shell passthrough model.
	shellModel *shell.ShellModel
	// commandBar is the text input for command mode.
	commandBar textinput.Model
	// commandMap maps command strings to section titles.
	commandMap map[string]string
	// tabMatches holds autocomplete suggestions for the current prefix.
	tabMatches []string
	tabIndex   int
}

// NewModel creates a new AppModel with a sidebar list.
func NewModel(provider *gophercloud.ProviderClient, cloudName string, compute client.ComputeClient, network client.NetworkClient, storage client.StorageClient, identity client.IdentityClient, image client.ImageClient, limits client.LimitsClient, dns client.DNSClient, lb client.LoadBalancerClient) AppModel {
	items := []list.Item{
		// Compute section
		item{title: "=== COMPUTE ===", description: ""},
		item{title: "Servers", description: "List and manage servers"},
		item{title: "Images", description: "List and manage images"},
		item{title: "Flavors", description: "List and manage flavors"},
		item{title: "Keypairs", description: "List and manage keypairs"},
		item{title: "Hypervisors", description: "List hypervisors"},
		item{title: "Availability Zones", description: "Availability zones"},
		item{title: "Limits", description: "Show compute and volume quotas"},
		// Network section
		item{title: "=== NETWORK ===", description: ""},
		item{title: "Networks", description: "List and manage networks"},
		item{title: "Subnets", description: "List and manage subnets"},
		item{title: "Routers", description: "List and manage routers"},
		item{title: "Ports", description: "List and manage ports"},
		item{title: "Floating IPs", description: "List and manage floating IPs"},
		item{title: "Security Groups", description: "List and manage security groups"},
		item{title: "Load Balancers", description: "List load balancers"},
		// Storage section
		item{title: "=== STORAGE ===", description: ""},
		item{title: "Volumes", description: "List and manage volumes"},
		item{title: "Snapshots", description: "List and manage snapshots"},
		// Identity section
		item{title: "=== IDENTITY ===", description: ""},
		item{title: "Projects", description: "List OpenStack projects"},
		item{title: "Users", description: "List OpenStack users"},
		item{title: "Token", description: "Show token info"},
		// Exit
		item{title: "=== DNS ===", description: ""},
		item{title: "Zones", description: "List DNS zones"},
		item{title: "Exit", description: "Quit the application"},
	}
	const defaultWidth = 30
	const defaultHeight = 14
	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	l.Title = "OSTUI – OpenStack TUI"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	// Initialize command mode text input.
	cmdBar := textinput.New()
	cmdBar.Placeholder = "command"
	// Command map: aliases to section titles.
	cmdMap := map[string]string{
		"servers": "Servers", "srv": "Servers",
		"networks": "Networks", "net": "Networks",
		"floatingips": "Floating IPs", "fip": "Floating IPs",
		"secgroups": "Security Groups", "sg": "Security Groups",
		"routers": "Routers", "rt": "Routers",
		"ports": "Ports", "port": "Ports",
		"volumes": "Volumes", "vol": "Volumes",
		"snapshots": "Snapshots",
		"projects":  "Projects",
		"users":     "Users",
		"token":     "Token",
		"images":    "Images", "img": "Images",
		"limits": "Limits", "quota": "Limits",
		"hypervisors": "Hypervisors", "hyp": "Hypervisors", "hv": "Hypervisors",
		"az":      "Availability Zones",
		"flavors": "Flavors", "flavor": "Flavors",
		"keypairs": "Keypairs", "kp": "Keypairs",
		"quit":  "__quit__",
		"zones": "Zones", "dns": "Zones",
		"lb": "Load Balancers", "loadbalancers": "Load Balancers",
	}
	return AppModel{provider: provider, cloudName: cloudName, computeClient: compute, networkClient: network, storageClient: storage, identityClient: identity, imageClient: image, limitsClient: limits, dnsClient: dns, lbClient: lb, sidebar: l, state: stateSidebar, prevState: "", commandBar: cmdBar, commandMap: cmdMap}
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

// navigateTo instantiates the appropriate submodel based on the given section title.
func (m *AppModel) navigateTo(section string) {
	switch section {
	case "Servers":
		m.mainModel = compute.NewInstancesModel(m.computeClient)
	case "Networks":
		m.mainModel = network.NewNetworksModel(m.networkClient)
	case "Floating IPs":
		m.mainModel = network.NewFloatingIPsModel(m.networkClient)
	case "Security Groups":
		m.mainModel = network.NewSecurityGroupsModel(m.networkClient)
	case "Volumes":
		m.mainModel = storage.NewVolumesModel(m.storageClient)
	case "Snapshots":
		m.mainModel = storage.NewSnapshotsModel(m.storageClient)
	case "Projects":
		m.mainModel = identity.NewProjectsModel(m.identityClient)
	case "Users":
		m.mainModel = identity.NewUsersModel(m.identityClient)
	case "Token":
		m.mainModel = identity.NewTokenModel(m.identityClient)
	case "Images":
		m.mainModel = image.NewImagesModel(m.imageClient)
	case "Limits":
		m.mainModel = compute.NewLimitsModel(m.limitsClient)
	case "Hypervisors":
		m.mainModel = compute.NewHypervisorsModel(m.computeClient)
	case "Availability Zones":
		m.mainModel = compute.NewZonesModel(m.computeClient)
	case "Subnets":
		m.mainModel = network.NewSubnetsModel(m.networkClient)
	case "Flavors":
		m.mainModel = compute.NewFlavorsModel(m.computeClient)
	case "Keypairs":
		m.mainModel = compute.NewKeypairsModel(m.computeClient)
	case "Zones":
		m.mainModel = dns.NewZonesModel(m.dnsClient)
	case "Load Balancers":
		m.mainModel = loadbalancer.NewLoadBalancersModel(m.lbClient)
	default:
		// No submodel for unknown sections.
	}
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sidebar.SetSize(msg.Width/5, msg.Height-4)
		// Forward the window size message to the active submodel (if any).
		var cmds []tea.Cmd
		if m.mainModel != nil {
			var cmd tea.Cmd
			m.mainModel, cmd = m.mainModel.Update(msg)
			cmds = append(cmds, cmd)
		}
		if m.state == stateLogs && m.logsModel != nil {
			var cmd tea.Cmd
			m.logsModel, cmd = m.logsModel.Update(msg)
			cmds = append(cmds, cmd)
		}
		if m.state == stateShell && m.shellModel != nil {
			var cmd tea.Cmd
			var newModel tea.Model
			newModel, cmd = m.shellModel.Update(msg)
			if sm, ok := newModel.(shell.ShellModel); ok {
				m.shellModel = &sm
			} else {
				m.shellModel = nil
			}
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?":
			if m.state != stateHelp {
				m.prevState = m.state
				m.state = stateHelp
			}
		case "esc":
			if m.state == stateHelp {
				// Return to previous state.
				m.state = m.prevState
				m.prevState = ""
				return m, nil
			}
			// Return to sidebar from any other state.
			if m.state == stateDetail {
				m.state = stateMain
				m.modalActive = false
				return m, nil
			} else if m.state != stateSidebar {
				m.state = stateSidebar
				m.modalActive = false
				m.mainModel = nil
				return m, nil
			}
		case "c":
			// Load cloud names and show selection list.
			clouds, err := clientconfig.LoadCloudsYAML()
			if err != nil {
				// ignore error, stay in current state
				return m, nil
			}
			var items []list.Item
			for name := range clouds {
				items = append(items, cloudItem{name: name})
			}
			const cloudListWidth = 30
			const cloudListHeight = 10
			l := list.New(items, list.NewDefaultDelegate(), cloudListWidth, cloudListHeight)
			l.Title = "Select Cloud"
			l.SetShowStatusBar(false)
			l.SetFilteringEnabled(false)
			l.Styles.Title = lipgloss.NewStyle().Bold(true)
			m.cloudList = l
			m.state = stateCloudSelect
			return m, nil
		case ":":
			// Enter command mode
			m.prevState = m.state
			m.state = stateCommand
			m.commandBar.Focus()
			m.commandBar.SetValue("")
			return m, nil
		case "g":
			// If InstanceDetailModel is showing graph, forward g to close it
			if m.state == stateDetail && m.detailModel != nil {
				if im, ok := m.detailModel.(compute.InstanceDetailModel); ok && im.IsShowingGraph() {
					var cmd tea.Cmd
					m.detailModel, cmd = m.detailModel.Update(msg)
					return m, cmd
				}
			}
			if m.state == stateDetail && m.detailModel != nil {
				// Determine resource type from detailModel type
				var rt graph.ResourceType
				var resID, resName string
				switch dm := m.detailModel.(type) {
				case network.FloatingIPDetailModel:
					rt = graph.ResourceFloatingIP
					resID = dm.ResourceID()
					resName = dm.ResourceName()
				case storage.VolumeDetailModel:
					rt = graph.ResourceVolume
					resID = dm.ResourceID()
					resName = dm.ResourceName()
				case network.NetworkSubnetsModel:
					rt = graph.ResourceNetwork
					resID = dm.ResourceID()
					resName = dm.ResourceName()
				case loadbalancer.LoadBalancerDetailModel:
					rt = graph.ResourceLoadBalancer
					resID = dm.ResourceID()
					resName = dm.ResourceName()
				default:
					// Forward to detail model (e.g. server graph)
					if m.detailModel != nil {
						var cmd tea.Cmd
						m.detailModel, cmd = m.detailModel.Update(msg)
						return m, cmd
					}
				}
				gm := graph.NewGraphModel(rt, resID, resName, m.computeClient, m.networkClient, m.storageClient, m.lbClient)
				m.graphModel = &gm
				m.state = stateGraph
				return m, m.graphModel.Init()
			}

		case "enter":
			if m.state == stateSidebar {
				if i, ok := m.sidebar.SelectedItem().(item); ok {
					if i.title == "Exit" {
						return m, tea.Quit
					}
					m.selectedItem = i
					// Transition to the main view and initialise the appropriate submodel.
					m.state = stateMain
					switch i.title {
					case "Servers":
						m.mainModel = compute.NewInstancesModel(m.computeClient)
					case "Networks":
						m.mainModel = network.NewNetworksModel(m.networkClient)
					case "Floating IPs":
						m.mainModel = network.NewFloatingIPsModel(m.networkClient)
					case "Security Groups":
						m.mainModel = network.NewSecurityGroupsModel(m.networkClient)
					case "Routers":
						m.mainModel = network.NewRoutersModel(m.networkClient)
					case "Ports":
						m.mainModel = network.NewPortsModel(m.networkClient)
					case "Volumes":
						m.mainModel = storage.NewVolumesModel(m.storageClient)
					case "Projects":
						m.mainModel = identity.NewProjectsModel(m.identityClient)
					case "Token":
						m.mainModel = identity.NewTokenModel(m.identityClient)
					case "Users":
						m.mainModel = identity.NewUsersModel(m.identityClient)
					case "Images":
						m.mainModel = image.NewImagesModel(m.imageClient)
					case "Limits":
						m.mainModel = compute.NewLimitsModel(m.limitsClient)
					case "Hypervisors":
						m.mainModel = compute.NewHypervisorsModel(m.computeClient)
					case "Availability Zones":
						m.mainModel = compute.NewZonesModel(m.computeClient)
					case "Flavors":
						m.mainModel = compute.NewFlavorsModel(m.computeClient)
					case "Keypairs":
						m.mainModel = compute.NewKeypairsModel(m.computeClient)
					case "Zones":
						m.mainModel = dns.NewZonesModel(m.dnsClient)
					case "Load Balancers":
						m.mainModel = loadbalancer.NewLoadBalancersModel(m.lbClient)
					default:
						// Fallback: no submodel – keep nil.
					}
					// If a submodel was created, invoke its Init to start async loading.
					if m.mainModel != nil {
						return m, m.mainModel.Init()
					}
					return m, nil
				}
				return m, nil
			} else if m.state == stateMain && m.mainModel != nil {
				// Handle drill-down Enter on submodel rows.
				switch model := m.mainModel.(type) {
				case compute.InstancesModel:
					// Get selected server ID.
					if model.Table().Rows() == nil {
						return m, nil
					}
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = compute.NewInstanceDetailModel(m.computeClient, m.networkClient, m.storageClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case network.NetworksModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						// Show subnets for this network.
						m.detailModel = network.NewNetworkSubnetsModel(m.networkClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case network.FloatingIPsModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = network.NewFloatingIPDetailModel(m.networkClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case network.SecurityGroupsModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = network.NewSecurityGroupDetailModel(m.networkClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case storage.VolumesModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = storage.NewVolumeDetailModel(m.storageClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case storage.SnapshotsModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = storage.NewSnapshotDetailModel(m.storageClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case identity.ProjectsModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = identity.NewProjectDetailModel(m.identityClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case identity.UsersModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = identity.NewUserDetailModel(m.identityClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
					return m, nil
				case image.ImagesModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = image.NewImageDetailModel(m.imageClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case compute.FlavorsModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = compute.NewFlavorDetailModel(m.computeClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case compute.KeypairsModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						name := row[0]
						m.detailModel = compute.NewKeypairDetailModel(m.computeClient, name)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				// Hypervisors drill-down
				case compute.HypervisorsModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = compute.NewHypervisorDetailModel(m.computeClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				// Load Balancers drill-down
				case loadbalancer.LoadBalancersModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						name := row[1]
						m.detailModel = loadbalancer.NewLoadBalancerDetailModel(m.lbClient, id, name)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				// DNS Zones drill-down
				case dns.ZonesModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						name := row[1]
						m.detailModel = dns.NewRecordSetsModel(m.dnsClient, id, name)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case network.RouterModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = network.NewRouterDetailModel(m.networkClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case network.SubnetsModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = network.NewSubnetDetailModel(m.networkClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				case network.PortsModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id := row[0]
						m.detailModel = network.NewPortDetailModel(m.networkClient, id)
						m.state = stateDetail
						return m, m.detailModel.Init()
					}
				}
			}
		}
	}
	// Handle custom messages
	switch msg := msg.(type) {
	case compute.OpenLogsMsg:
		m.logsModel = compute.NewLogsModel(m.computeClient, msg.ServerID)
		m.state = stateLogs
		return m, m.logsModel.Init()
	case compute.GoBackMsg:
		if m.state == stateLogs {
			m.state = stateDetail
			m.logsModel = nil
			return m, nil
		} else if m.state == stateDetail && m.detailModel != nil {
			var cmd tea.Cmd
			m.detailModel, cmd = m.detailModel.Update(msg)
			return m, cmd
		} else if m.state == stateGraph {
			m.state = stateDetail
			m.graphModel = nil
			return m, nil
		}
	case shell.CloseMsg:
		m.state = stateSidebar
		m.shellModel = nil
		return m, nil
	}
	// Command mode handling
	if m.state == stateCommand {
		// handle command mode key events
		switch msg := msg.(type) {
		case tea.KeyMsg:
			{
				switch msg.String() {
				case "esc":
					// exit command mode
					m.state = m.prevState
					m.prevState = ""
					m.commandBar.Blur()
					m.commandBar.SetValue("")
					// reset tab autocomplete state
					m.tabMatches = nil
					m.tabIndex = 0
					return m, nil
				case "enter":
					cmd := strings.TrimSpace(m.commandBar.Value())
					// Shell passthrough command mode: prefix '!'
					if strings.HasPrefix(cmd, "!") {
						command := strings.TrimPrefix(cmd, "!")
						sm := shell.NewShellModel(m.cloudName, command)
						m.shellModel = &sm
						m.state = stateShell
						m.commandBar.SetValue("")
						m.commandBar.Blur()
						// reset tab autocomplete state
						m.tabMatches = nil
						m.tabIndex = 0
						return m, m.shellModel.Init()
					}
					if section, ok := m.commandMap[cmd]; ok {
						if section == "__quit__" {
							return m, tea.Quit
						}
						m.navigateTo(section)
						m.state = stateMain
						m.commandBar.SetValue("")
						m.commandBar.Blur()
						// reset tab autocomplete state
						m.tabMatches = nil
						m.tabIndex = 0
						return m, m.mainModel.Init()
					}
					// unknown command: clear input
					m.commandBar.SetValue("")
					// reset tab autocomplete state
					m.tabMatches = nil
					m.tabIndex = 0
					return m, nil
				case "tab":
					prefix := strings.TrimSpace(m.commandBar.Value())
					// Collect and sort all matches
					var matches []string
					for k := range m.commandMap {
						if strings.HasPrefix(k, prefix) {
							matches = append(matches, k)
						}
					}
					sort.Strings(matches)
					if len(matches) == 0 {
						return m, nil
					}
					// If prefix changed, reset cycle
					if len(m.tabMatches) == 0 || m.commandBar.Value() != m.tabMatches[m.tabIndex] {
						m.tabMatches = matches
						m.tabIndex = 0
					} else {
						m.tabIndex = (m.tabIndex + 1) % len(m.tabMatches)
					}
					m.commandBar.SetValue(m.tabMatches[m.tabIndex])
					return m, nil
				default:
					var cmd tea.Cmd
					m.commandBar, cmd = m.commandBar.Update(msg)
					return m, cmd
				}
			}
		}
		// ignore other messages
		return m, nil
	}
	// When in sidebar state, forward updates to the list component.
	if m.state == stateSidebar {
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		return m, cmd
	}

	if m.state == stateCloudSelect {
		var cmd tea.Cmd
		m.cloudList, cmd = m.cloudList.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			if _, ok := m.cloudList.SelectedItem().(cloudItem); ok {
				m.state = stateSidebar
			}
		}
		return m, cmd
	}
	if m.state == stateMain && m.mainModel != nil {
		var cmd tea.Cmd
		m.mainModel, cmd = m.mainModel.Update(msg)
		return m, cmd
	}
	if m.state == stateDetail && m.detailModel != nil {
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			var cmd tea.Cmd
			m.detailModel, cmd = m.detailModel.Update(msg)
			return m, cmd
		}
	}
	if m.state == stateGraph && m.graphModel != nil {
		var cmd tea.Cmd
		m.graphModel, cmd = m.graphModel.Update(msg)
		return m, cmd
	}
	if m.state == stateShell && m.shellModel != nil {
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd = m.shellModel.Update(msg)
		if sm, ok := newModel.(shell.ShellModel); ok {
			m.shellModel = &sm
		} else {
			m.shellModel = nil
		}
		return m, cmd
	}
	if m.state == stateLogs && m.logsModel != nil {
		var cmd tea.Cmd
		m.logsModel, cmd = m.logsModel.Update(msg)
		return m, cmd
	}
	// When in cloud select state, forward updates to the cloud list component.
	//if m.state == stateCloudSelect {
	//	var cmd tea.Cmd
	//	m.cloudList, cmd = m.cloudList.Update(msg)
	//	// If Enter pressed, handle selection.
	//	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
	//		if _, ok := m.cloudList.SelectedItem().(cloudItem); ok {
	//			cloudsPath := os.Getenv("OS_CLIENT_CONFIG_FILE")
	//			authOpts, err := config.LoadAuthOptions(i.name, cloudsPath)
	//			if err == nil {
	//				provider, err := openstack.AuthenticatedClient(authOpts)
	//				if err == nil {
	//					m.provider = provider
	//					// Recreate clients.
	//					if computeClient, err := client.NewComputeClient(authOpts); err == nil {
	//						m.computeClient = computeClient
	//					}
	//					if networkClient, err := client.NewNetworkClient(authOpts); err == nil {
	//						m.networkClient = networkClient
	//					}
	//					if storageClient, err := client.NewStorageClient(authOpts); err == nil {
	//						m.storageClient = storageClient
	//					}
	//					if identityClient, err := client.NewIdentityClient(authOpts); err == nil {
	//						m.identityClient = identityClient
	//					}
	//				}
	//			}
	//			// Return to sidebar.
	//			m.state = stateSidebar
	//			m.modalActive = false
	//			return m, nil
	//		}
	//	}
	//	return m, cmd
	//}

	// When in the main view, forward all messages to the active submodel.
	//if m.state == stateMain && m.mainModel != nil {
	//	var cmd tea.Cmd
	//	m.mainModel, cmd = m.mainModel.Update(msg)
	//	return m, cmd
	//}
	return m, nil
}

// View implements tea.Model.
func (m AppModel) View() string {
	footer := fmt.Sprintf("\n[%s] Press : for command mode", m.state)
	switch m.state {
	case stateSidebar:
		return "\n" + m.sidebar.View() + "\n" + footer
	case stateMain:
		if m.mainModel != nil {
			return m.mainModel.View() + footer
		}
		return fmt.Sprintf("\n%s view – press esc to return\n", m.selectedItem.title) + footer
	case stateModal:
		return "\n[Modal] Press esc to close\n" + footer
	case stateDetail:
		if m.detailModel != nil {
			return m.detailModel.View() + footer
		}
		return "" + footer
	case stateLogs:
		if m.logsModel != nil {
			return m.logsModel.View() + footer
		}
		return "" + footer
	case stateHelp:
		return m.helpView() + footer
	case stateGraph:
		if m.graphModel != nil {
			return m.graphModel.View() + footer
		}
		return "" + footer
	case stateShell:
		if m.shellModel != nil {
			return m.shellModel.View() + footer
		}
		return "" + footer
	case stateCommand:
		// Render previous view plus command bar overlay, with autocomplete suggestions.
		var base string
		switch m.prevState {
		case stateSidebar:
			base = "\n" + m.sidebar.View() + "\n"
		case stateMain:
			if m.mainModel != nil {
				base = m.mainModel.View()
			} else {
				base = fmt.Sprintf("\n%s view – press esc to return\n", m.selectedItem.title)
			}
		case stateDetail:
			if m.detailModel != nil {
				base = m.detailModel.View()
			} else {
				base = ""
			}
		case stateLogs:
			if m.logsModel != nil {
				base = m.logsModel.View()
			} else {
				base = ""
			}
		case stateHelp:
			base = m.helpView()
		default:
			base = ""
		}
		// Command bar view
		view := base + "\n" + m.commandBar.View()
		// Show suggestions if multiple matches are available.
		if len(m.tabMatches) > 1 {
			suggestions := strings.Join(m.tabMatches, "  ")
			view += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(suggestions)
		}
		return view + footer
	default:
		return ""
	}
}

// Ensure AppModel implements tea.Model.
func (m AppModel) helpView() string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#AAAAAA"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5CB85C"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))

	key := func(k, desc string) string {
		return keyStyle.Render(fmt.Sprintf("  %-12s", k)) + descStyle.Render(desc) + "\n"
	}

	b.WriteString(titleStyle.Render("\n  Global") + "\n")
	b.WriteString(key("q / ctrl+c", "Quit"))
	b.WriteString(key("?", "Toggle help"))
	b.WriteString(key("c", "Switch cloud"))
	b.WriteString(key(":", "Command mode"))

	switch m.prevState {
	case stateMain:
		b.WriteString(titleStyle.Render("\n  List view") + "\n")
		b.WriteString(key("j / k", "Move down / up"))
		b.WriteString(key("enter", "Open detail"))
		b.WriteString(key("/", "Filter"))
		b.WriteString(key("esc", "Back to sidebar"))
		b.WriteString(key("r", "Refresh"))
		// Extra keys for Servers
		if _, ok := m.mainModel.(compute.InstancesModel); ok {
			b.WriteString(titleStyle.Render("\n  Servers (detail)\n") + "\n")
			b.WriteString(key("l", "View logs"))
			b.WriteString(key("i", "Inspect"))
			b.WriteString(key("y", "JSON view"))
			b.WriteString(key("v", "Console URL"))
		}
	case stateDetail:
		b.WriteString(titleStyle.Render("\n  Detail view") + "\n")
		b.WriteString(key("j / k", "Scroll"))
		b.WriteString(key("i", "Inspect"))
		b.WriteString(key("y", "JSON view"))
		b.WriteString(key("esc", "Back to list"))
	case stateLogs:
		b.WriteString(titleStyle.Render("\n  Log viewer") + "\n")
		b.WriteString(key("j / k", "Scroll"))
		b.WriteString(key("g / G", "Top / bottom"))
		b.WriteString(key("p", "Pause / resume streaming"))
		b.WriteString(key("+  /  -", "Increase / decrease interval"))
		b.WriteString(key("esc", "Back"))
	case stateCommand:
		b.WriteString(titleStyle.Render("\n  Command mode") + "\n")
		b.WriteString(key("tab", "Autocomplete (cycle)"))
		b.WriteString(key("enter", "Execute command"))
		b.WriteString(key("esc", "Cancel"))
		b.WriteString(titleStyle.Render("\n  Commands") + "\n")
		b.WriteString(key("servers / srv", "Servers"))
		b.WriteString(key("networks / net", "Networks"))
		b.WriteString(key("volumes / vol", "Volumes"))
		b.WriteString(key("images / img", "Images"))
		b.WriteString(key("limits / quota", "Limits"))
		b.WriteString(key("dns / zones", "DNS Zones"))
		b.WriteString(key("lb", "Load Balancers"))
		b.WriteString(key("quit", "Exit"))
	default:
		b.WriteString(titleStyle.Render("\n  Sidebar") + "\n")
		b.WriteString(key("j / k", "Move down / up"))
		b.WriteString(key("enter", "Open section"))
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("\n  [?] close help\n"))
	return b.String()
}

// Ensure AppModel implements tea.Model.
var _ tea.Model = (*AppModel)(nil)
