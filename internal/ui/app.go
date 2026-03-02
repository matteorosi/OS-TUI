package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	appcache "ostui/internal/cache"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/compute"
	"ostui/internal/ui/dns"
	"ostui/internal/ui/graph"
	"ostui/internal/ui/identity"
	"ostui/internal/ui/image"
	"ostui/internal/ui/loadbalancer"
	"ostui/internal/ui/network"
	"ostui/internal/ui/search"
	"ostui/internal/ui/shell"
	"ostui/internal/ui/storage"
	"ostui/internal/ui/topology"
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
	stateTopology    = "topology"
	stateSearch      = "search"
	stateAction      = "action"
	stateConfirm     = "confirm"
	stateInput       = "input"
)

type actionResultMsg struct {
	success bool
	message string
}

// AppModel is the root model of the TUI, managing a simple state machine.
type AppModel struct {
	provider       *gophercloud.ProviderClient
	cloudName      string
	computeClient  client.ComputeClient
	networkClient  client.NetworkClient
	storageClient  client.StorageClient
	cache          *appcache.Cache
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
	// topologyModel holds the topology view model.
	topologyModel *topology.TopologyModel
	searchModel   *search.SearchModel
	// commandBar is the text input for command mode.
	commandBar textinput.Model
	// commandMap maps command strings to section titles.
	commandMap map[string]string
	// tabMatches holds autocomplete suggestions for the current prefix.
	tabMatches []string
	tabIndex   int
	// Action handling fields
	actionMenu          *common.ActionMenuModel
	actionTarget        string
	actionResource      string
	confirmModal        *common.ConfirmModel
	pendingAction       *common.ActionItem
	pendingInput        string
	actionResult        string
	actionResultSuccess bool
	inputModel          *common.FormModel
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
		// Topology section
		item{title: "=== TOPOLOGY ===", description: ""},
		item{title: "Topology", description: "View topology of resources"},
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
		"lb": "Load Balancers", "loadbalancers": "Load Balancers", "topology": "Topology", "topo": "Topology",
		"search": "__search__",
	}
	c := appcache.NewCache(5 * time.Minute)
	return AppModel{
		provider:       provider,
		cloudName:      cloudName,
		computeClient:  client.NewCachedComputeClient(compute, c),
		networkClient:  client.NewCachedNetworkClient(network, c),
		storageClient:  client.NewCachedStorageClient(storage, c),
		identityClient: identity,
		imageClient:    image,
		limitsClient:   limits,
		dnsClient:      dns,
		lbClient:       lb,
		cache:          c,
		sidebar:        l,
		state:          stateSidebar,
		prevState:      "",
		commandBar:     cmdBar,
		commandMap:     cmdMap,
	}
}

// navigationMap returns a map of sidebar titles to model constructors.
func (m AppModel) navigationMap() map[string]func() tea.Model {
	return map[string]func() tea.Model{
		"Servers":            func() tea.Model { return compute.NewInstancesModel(m.computeClient) },
		"Networks":           func() tea.Model { return network.NewNetworksModel(m.networkClient) },
		"Floating IPs":       func() tea.Model { return network.NewFloatingIPsModel(m.networkClient) },
		"Security Groups":    func() tea.Model { return network.NewSecurityGroupsModel(m.networkClient) },
		"Routers":            func() tea.Model { return network.NewRoutersModel(m.networkClient) },
		"Ports":              func() tea.Model { return network.NewPortsModel(m.networkClient) },
		"Volumes":            func() tea.Model { return storage.NewVolumesModel(m.storageClient) },
		"Projects":           func() tea.Model { return identity.NewProjectsModel(m.identityClient) },
		"Users":              func() tea.Model { return identity.NewUsersModel(m.identityClient) },
		"Token":              func() tea.Model { return identity.NewTokenModel(m.identityClient) },
		"Images":             func() tea.Model { return image.NewImagesModel(m.imageClient) },
		"Limits":             func() tea.Model { return compute.NewLimitsModel(m.limitsClient) },
		"Hypervisors":        func() tea.Model { return compute.NewHypervisorsModel(m.computeClient) },
		"Availability Zones": func() tea.Model { return compute.NewZonesModel(m.computeClient) },
		"Subnets":            func() tea.Model { return network.NewSubnetsModel(m.networkClient) },
		"Flavors":            func() tea.Model { return compute.NewFlavorsModel(m.computeClient) },
		"Keypairs":           func() tea.Model { return compute.NewKeypairsModel(m.computeClient) },
		"Zones":              func() tea.Model { return dns.NewZonesModel(m.dnsClient) },
		"Load Balancers":     func() tea.Model { return loadbalancer.NewLoadBalancersModel(m.lbClient) },
		"Topology":           func() tea.Model { return topology.NewTopologyModel(m.computeClient, m.networkClient, m.storageClient) },
	}
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

// navigateTo instantiates the appropriate submodel based on the given section title.
func (m *AppModel) navigateTo(section string) {
	// Use navigationMap for most sections.
	navMap := m.navigationMap()
	if constructor, ok := navMap[section]; ok {
		// Special handling for Topology which uses a dedicated model and state.
		if section == "Topology" {
			tm := topology.NewTopologyModel(m.computeClient, m.networkClient, m.storageClient)
			m.topologyModel = &tm
			m.state = stateTopology
			return
		}
		m.mainModel = constructor()
		return
	}
	// No submodel for unknown sections.
}

// prefetchMap defines which resources to warm when a section is opened.
var prefetchMap = map[string][]string{
	"Servers":      {"Networks", "Volumes", "Floating IPs"},
	"Networks":     {"Subnets", "Routers", "Floating IPs"},
	"Floating IPs": {"Networks"},
	"Routers":      {"Networks", "Subnets"},
	"Subnets":      {"Networks"},
}

// triggerPrefetch fires background goroutines to warm the cache for sections
// related to the one just opened. Safe to call from Update(); goroutines are
// fire-and-forget — they write into the thread-safe cache.
func (m AppModel) triggerPrefetch(section string) {
	targets, ok := prefetchMap[section]
	if !ok {
		return
	}
	for _, t := range targets {
		target := t
		go func() {
			switch target {
			case "Networks":
				_, _ = m.networkClient.ListNetworks()
			case "Subnets":
				_, _ = m.networkClient.ListSubnets()
			case "Floating IPs":
				_, _ = m.networkClient.ListFloatingIPs()
			case "Routers":
				_, _ = m.networkClient.ListRouters(context.Background())
			case "Volumes":
				_, _ = m.storageClient.ListVolumes()
			}
		}()
	}
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case search.SearchDoneMsg:
		m.state = stateSidebar
		m.searchModel = nil
		return m, nil
	case search.SearchSelectedMsg:
		navMap := m.navigationMap()
		if constructor, ok := navMap[msg.Result.Category]; ok {
			m.mainModel = constructor()
			m.state = stateMain
			m.searchModel = nil
			return m, m.mainModel.Init()
		}
		m.state = stateSidebar
		m.searchModel = nil
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sidebar.SetSize(34, msg.Height-4)
		// Forward the window size message to the active submodel (if any).
		var cmds []tea.Cmd
		if m.mainModel != nil {
			var cmd tea.Cmd
			m.mainModel, cmd = m.mainModel.Update(msg)
			cmds = append(cmds, cmd)
		}
		if m.state == stateSearch && m.searchModel != nil {
			var cmd tea.Cmd
			var newModel tea.Model
			newModel, cmd = m.searchModel.Update(msg)
			if sm, ok := newModel.(search.SearchModel); ok {
				m.searchModel = &sm
			} else {
				m.searchModel = nil
			}
			return m, cmd
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
		// Clear any previous action result on any key press
		m.actionResult = ""
		// Forward ALL keys to search model when in search state.
		if m.state == stateSearch && m.searchModel != nil {
			var cmd tea.Cmd
			var newModel tea.Model
			newModel, cmd = m.searchModel.Update(msg)
			if sm, ok := newModel.(search.SearchModel); ok {
				m.searchModel = &sm
			}
			return m, cmd
		}
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
		case "/":
			if m.state == stateSidebar {
				sm := search.NewSearchModel(m.computeClient, m.networkClient, m.storageClient, m.imageClient, m.width, m.height)
				m.searchModel = &sm
				m.state = stateSearch
				return m, sm.Init()
			}
		case "c":
			// Load cloud names and show selection list (original)
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
		case "T":
			// Open topology view
			tm := topology.NewTopologyModel(m.computeClient, m.networkClient, m.storageClient)
			m.topologyModel = &tm
			m.state = stateTopology
			return m, m.topologyModel.Init()
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

		case "a":
			if m.state == stateMain && m.mainModel != nil {
				var actions []common.ActionItem
				var id string
				switch model := m.mainModel.(type) {
				case compute.InstancesModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id = row[0]
						actions = []common.ActionItem{{Key: "start", Label: "Start", Destructive: false}, {Key: "stop", Label: "Stop", Destructive: true}, {Key: "reboot", Label: "Reboot", Destructive: false}, {Key: "delete", Label: "Delete", Destructive: true}}
						m.actionResource = "instance"
					}

				case storage.VolumesModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id = row[0]
						actions = []common.ActionItem{{Key: "delete", Label: "Delete", Destructive: true}, {Key: "extend", Label: "Extend Size", Destructive: false}}
						m.actionResource = "volume"
					}

				case network.NetworksModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id = row[0]
						actions = []common.ActionItem{{Key: "delete", Label: "Delete", Destructive: true}}
					}
				case network.FloatingIPsModel:
					row := model.Table().SelectedRow()
					if len(row) > 0 {
						id = row[0]
						actions = []common.ActionItem{{Key: "associate", Label: "Associate", Destructive: false}, {Key: "disassociate", Label: "Disassociate", Destructive: false}, {Key: "delete", Label: "Delete/Release", Destructive: true}}
					}
				default:
					// No actions for other models
				}
				if id != "" && len(actions) > 0 {
					m.actionTarget = id
					am := common.NewActionMenu("Actions", actions)
					m.actionMenu = &am
					m.state = stateAction
				}
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
					navMap := m.navigationMap()
					if constructor, ok := navMap[i.title]; ok {
						if i.title == "Topology" {
							// Use navigateTo for Topology to handle state and model.
							m.navigateTo(i.title)
						} else {
							m.mainModel = constructor()
							m.state = stateMain
						}
					} else {
						// Fallback: no submodel – keep nil.
					}

					// If a submodel was created, invoke its Init to start async loading.
					if m.mainModel != nil {
						m.triggerPrefetch(i.title)
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
	case topology.CloseMsg:
		m.state = stateSidebar
		m.topologyModel = nil
		return m, nil
	case shell.CloseMsg:
		m.state = stateSidebar
		m.shellModel = nil
		return m, nil
	case actionResultMsg:
		m.actionResult = msg.message
		m.actionResultSuccess = msg.success
		m.pendingAction = nil
		m.actionTarget = ""
		m.actionResource = ""
		m.pendingInput = ""
		m.state = stateMain
		if m.mainModel != nil {
			return m, m.mainModel.Init()
		}
		return m, nil
	}
	// Handle action menu, confirm modal, input form, and action result messages
	if m.state == stateAction && m.actionMenu != nil {
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd = (*m.actionMenu).Update(msg)
		if am, ok := newModel.(common.ActionMenuModel); ok {
			*m.actionMenu = am
		}
		if m.actionMenu.IsDone() {
			selected := m.actionMenu.Selected()
			if selected != nil {
				m.pendingAction = selected
				if selected.Destructive {
					confirmMsg := fmt.Sprintf("Confirm %s on %s? (y/n)", selected.Label, m.actionTarget)
					cm := common.NewConfirm(confirmMsg)
					m.confirmModal = &cm
					m.state = stateConfirm
				} else {
					if selected.Key == "extend" {
						fm := common.NewForm([]string{"New size (GB)"})
						m.inputModel = &fm
						m.state = stateInput
					} else if selected.Key == "associate" {
						fm := common.NewForm([]string{"Port ID"})
						m.inputModel = &fm
						m.state = stateInput
					} else {
						return m, executeAction(m)
					}
				}
			} else {
				m.state = stateMain
			}
			m.actionMenu = nil
		}
		return m, cmd
	}
	if m.state == stateConfirm && m.confirmModal != nil {
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd = (*m.confirmModal).Update(msg)
		if cm, ok := newModel.(common.ConfirmModel); ok {
			*m.confirmModal = cm
		}
		if m.confirmModal.Done() {
			if m.confirmModal.Result() == "Yes" {
				return m, executeAction(m)
			} else {
				m.state = stateMain
				m.confirmModal = nil
				m.pendingAction = nil
				m.actionTarget = ""
				m.actionResource = ""
			}
		}
		return m, cmd
	}
	if m.state == stateInput && m.inputModel != nil {
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd = (*m.inputModel).Update(msg)
		if fm, ok := newModel.(common.FormModel); ok {
			*m.inputModel = fm
		}
		if m.inputModel.Submitted() {
			vals := m.inputModel.Values()
			if len(vals) > 0 {
				m.pendingInput = vals[0]
			}
			m.inputModel = nil
			return m, executeAction(m)
		}
		return m, cmd
	}

	// Handle custom messages
	/*
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
	           case topology.CloseMsg:
	               m.state = stateSidebar
	               m.topologyModel = nil
	               return m, nil
	           case shell.CloseMsg:
	               m.state = stateSidebar
	               m.shellModel = nil
	               return m, nil
	           case actionResultMsg:
	               // Set result message and success flag
	               m.actionResult = msg.message
	               m.actionResultSuccess = msg.success
	               // Reset pending state
	               m.pendingAction = nil
	               m.actionTarget = ""
	               m.actionResource = ""
	               m.pendingInput = ""
	               // Return to main view and reload list
	               m.state = stateMain
	               if m.mainModel != nil {
	                   return m, m.mainModel.Init()
	               }
	               return m, nil
	           }
	           // Command mode handling
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
	   					if cmd == "topology" || cmd == "topo" {
	   						// Open topology view using navigateTo
	   						m.navigateTo("Topology")
	   						m.commandBar.SetValue("")
	   						m.commandBar.Blur()
	   						// reset tab autocomplete state
	   						m.tabMatches = nil
	   						m.tabIndex = 0
	   						if m.topologyModel != nil {
	   							return m, m.topologyModel.Init()
	   						}
	   						return m, nil
	   					}
	   					if cmd == "__search__" {
	   						sm := search.NewSearchModel(m.computeClient, m.networkClient, m.storageClient, m.imageClient, m.width, m.height)
	   						m.searchModel = &sm
	   						m.state = stateSearch
	   						m.commandBar.SetValue("")
	   						m.commandBar.Blur()
	   						// reset tab autocomplete state
	   						m.tabMatches = nil
	   						m.tabIndex = 0
	   						return m, sm.Init()
	   					}
	   					if section, ok := m.commandMap[cmd]; ok {
	   						if section == "__quit__" {
	   							return m, tea.Quit
	   						}
	   						m.navigateTo(section)
	   						if section == "Topology" {
	   							m.commandBar.SetValue("")
	   							m.commandBar.Blur()
	   							// reset tab autocomplete state
	   							m.tabMatches = nil
	   							m.tabIndex = 0
	   							if m.topologyModel != nil {
	   								return m, m.topologyModel.Init()
	   							}
	   							return m, nil
	   						}
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
	   			// ignore other messages

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
	   		var cmd tea.Cmd
	   		m.detailModel, cmd = m.detailModel.Update(msg)
	   		return m, cmd
	   	}
	   	if m.state == stateGraph && m.graphModel != nil {
	   		var cmd tea.Cmd
	   		m.graphModel, cmd = m.graphModel.Update(msg)
	   		return m, cmd
	   	}
	   	if m.state == stateTopology && m.topologyModel != nil {
	   		var cmd tea.Cmd
	   		var newModel tea.Model
	   		newModel, cmd = m.topologyModel.Update(msg)
	   		if tm, ok := newModel.(topology.TopologyModel); ok {
	   			*m.topologyModel = tm
	   		}
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
	   	// Catch-all: forward any unhandled message to search model.
	   	if m.state == stateSearch && m.searchModel != nil {
	   		var cmd tea.Cmd
	   		var newModel tea.Model
	   		newModel, cmd = m.searchModel.Update(msg)
	   		if sm, ok := newModel.(search.SearchModel); ok {
	   			m.searchModel = &sm
	   		}
	   		return m, cmd
	   	}
	   	// Catch-all: forward any unhandled message to search model.
	   	if m.state == stateSearch && m.searchModel != nil {
	   		var cmd tea.Cmd
	   		var newModel tea.Model
	   		newModel, cmd = m.searchModel.Update(msg)
	   		if sm, ok := newModel.(search.SearchModel); ok {
	   			m.searchModel = &sm
	   		}
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
	   	//				}
	   	//				// Return to sidebar.
	   	//				m.state = stateSidebar
	   	//				m.modalActive = false
	   	//				return m, nil
	   	//			}
	   	//		}
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
	   	case topology.CloseMsg:
	   		m.state = stateSidebar
	   		m.topologyModel = nil
	   		return m, nil
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
	   			// Clear any previous action result on any key press
	   			m.actionResult = ""
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
	   					if cmd == "topology" || cmd == "topo" {
	   						// Open topology view using navigateTo
	   						m.navigateTo("Topology")
	   						m.commandBar.SetValue("")
	   						m.commandBar.Blur()
	   						// reset tab autocomplete state
	   						m.tabMatches = nil
	   						m.tabIndex = 0
	   						if m.topologyModel != nil {
	   							return m, m.topologyModel.Init()
	   						}
	   						return m, nil
	   					}
	   					if cmd == "__search__" {
	   						sm := search.NewSearchModel(m.computeClient, m.networkClient, m.storageClient, m.imageClient, m.width, m.height)
	   						m.searchModel = &sm
	   						m.state = stateSearch
	   						m.commandBar.SetValue("")
	   						m.commandBar.Blur()
	   						// reset tab autocomplete state
	   						m.tabMatches = nil
	   						m.tabIndex = 0
	   						return m, sm.Init()
	   					}
	   					if section, ok := m.commandMap[cmd]; ok {
	   						if section == "__quit__" {
	   							return m, tea.Quit
	   						}
	   						m.navigateTo(section)
	   						if section == "Topology" {
	   							m.commandBar.SetValue("")
	   							m.commandBar.Blur()
	   							// reset tab autocomplete state
	   							m.tabMatches = nil
	   							m.tabIndex = 0
	   							if m.topologyModel != nil {
	   								return m, m.topologyModel.Init()
	   							}
	   							return m, nil
	   						}
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
	*/
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
		var cmd tea.Cmd
		m.detailModel, cmd = m.detailModel.Update(msg)
		return m, cmd
	}
	if m.state == stateGraph && m.graphModel != nil {
		var cmd tea.Cmd
		m.graphModel, cmd = m.graphModel.Update(msg)
		return m, cmd
	}
	if m.state == stateTopology && m.topologyModel != nil {
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd = m.topologyModel.Update(msg)
		if tm, ok := newModel.(topology.TopologyModel); ok {
			*m.topologyModel = tm
		}
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
	// Catch-all: forward any unhandled message to search model.
	if m.state == stateSearch && m.searchModel != nil {
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd = m.searchModel.Update(msg)
		if sm, ok := newModel.(search.SearchModel); ok {
			m.searchModel = &sm
		}
		return m, cmd
	}
	// Catch-all: forward any unhandled message to search model.
	if m.state == stateSearch && m.searchModel != nil {
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd = m.searchModel.Update(msg)
		if sm, ok := newModel.(search.SearchModel); ok {
			m.searchModel = &sm
		}
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

// executeAction performs the pending action asynchronously and returns a command that yields an actionResultMsg.
func executeAction(m AppModel) tea.Cmd {
	pending := m.pendingAction
	target := m.actionTarget
	resource := m.actionResource
	input := m.pendingInput

	return func() tea.Msg {
		var err error
		switch resource {
		case "instance":
			switch pending.Key {
			case "start":
				err = m.computeClient.StartInstance(target)
			case "stop":
				err = m.computeClient.StopInstance(target)
			case "reboot":
				err = m.computeClient.RebootInstance(target)
			case "delete":
				err = m.computeClient.DeleteInstance(target)
			}
		case "volume":
			switch pending.Key {
			case "delete":
				err = m.storageClient.DeleteVolume(target)
			case "extend":
				size, convErr := strconv.Atoi(input)
				if convErr != nil {
					err = fmt.Errorf("invalid size: %v", convErr)
				} else {
					err = m.storageClient.ExtendVolume(target, size)
				}
			}
		case "network":
			if pending.Key == "delete" {
				err = m.networkClient.DeleteNetwork(target)
			}
		case "floatingip":
			switch pending.Key {
			case "associate":
				_, err = m.networkClient.AssociateFloatingIP(target, input)
			case "disassociate":
				_, err = m.networkClient.DisassociateFloatingIP(target)
			case "delete":
				err = m.networkClient.ReleaseFloatingIP(target)
			}
		default:
			err = fmt.Errorf("unsupported resource %s for action %s", resource, pending.Key)
		}
		if err != nil {
			return actionResultMsg{success: false, message: err.Error()}
		}
		msg := fmt.Sprintf("%s %s succeeded", pending.Label, target)
		return actionResultMsg{success: true, message: msg}
	}
}

// View implements tea.Model.
func (m AppModel) View() string {
	footer := fmt.Sprintf("\n[%s] Press : for command mode  [T] topology  [/]", m.state) + " search"
	switch m.state {
	case stateSidebar:
		sidebarWidth := 36
		rightWidth := m.width - sidebarWidth - 4
		if rightWidth < 20 {
			// Terminal too narrow: fallback to single column
			return "\n" + m.sidebar.View() + "\n" + footer
		}
		sideStyle := lipgloss.NewStyle().
			Width(sidebarWidth).
			Height(m.height - 4).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
		rightStyle := lipgloss.NewStyle().
			Width(rightWidth).
			Height(m.height - 4).
			PaddingLeft(2).
			PaddingTop(1)
		help := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render
		accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render
		rightContent := accent("Cloud: ") + m.cloudName + "\n\n" +
			accent("Navigation") + "\n" +
			help("  ↑/k  up          ↓/j  down") + "\n" +
			help("  enter  open      esc  back") + "\n\n" +
			accent("Global keys") + "\n" +
			help("  ?   help         c   switch cloud") + "\n" +
			help("  T   topology     :   command mode") + "\n" +
			help("  g   graph        y   JSON view") + "\n" +
			help("  i   inspect      l   logs (servers)") + "\n\n" +
			accent("Commands") + "\n" +
			help("  :servers  :networks  :volumes") + "\n" +
			help("  :images   :limits    :dns") + "\n" +
			help("  :routers  :ports     :fip") + "\n" +
			help("  :topology / :topo") + "\n" +
			help("  :!<cmd>  → openstack CLI") + "\n\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("ostui v0.1.0")
		layout := lipgloss.JoinHorizontal(lipgloss.Top,
			sideStyle.Render(m.sidebar.View()),
			rightStyle.Render(rightContent),
		)
		return layout + "\n" + footer
	case stateMain:
		if m.mainModel != nil {
			return m.mainModel.View() + footer
		}
		return fmt.Sprintf("\n%s view – press esc to return\n", m.selectedItem.title) + footer
	case stateModal:
		return "\n[Modal] Press esc to close\n" + footer
	case stateAction:
		if m.actionMenu != nil {
			return m.actionMenu.View() + footer
		}
		return "" + footer
	case stateConfirm:
		if m.confirmModal != nil {
			return m.confirmModal.View() + footer
		}
		return "" + footer
	case stateInput:
		if m.inputModel != nil {
			return m.inputModel.View() + footer
		}
		return "" + footer
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
	case stateTopology:
		if m.topologyModel != nil {
			return m.topologyModel.View() + footer
		}
		return "" + footer
	case stateShell:
		if m.shellModel != nil {
			return m.shellModel.View() + footer
		}
		return "" + footer
	case stateSearch:
		if m.searchModel != nil {
			return m.searchModel.View() + footer
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
	b.WriteString(key("/", "Global search (from sidebar)"))

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
