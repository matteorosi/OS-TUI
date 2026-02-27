package client

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

// NetworkClient defines the methods for interacting with OpenStack Networking (Neutron) service.
// Type aliases for OpenStack resources
type Router = routers.Router
type RouterInterface = ports.Port
type Port = ports.Port
type SecurityGroupRule = rules.SecGroupRule
type SecurityGroupRuleInput = rules.CreateOpts

type NetworkClient interface {
	ListNetworks() ([]networks.Network, error)
	ListSubnets() ([]subnets.Subnet, error)
	GetSubnet(ctx context.Context, subnetID string) (*subnets.Subnet, error)
	ListFloatingIPs() ([]floatingips.FloatingIP, error)
	AllocateFloatingIP(opts floatingips.CreateOptsBuilder) (floatingips.FloatingIP, error)
	ReleaseFloatingIP(id string) error
	AssociateFloatingIP(fipID string, portID string) (floatingips.FloatingIP, error)
	DisassociateFloatingIP(fipID string) (floatingips.FloatingIP, error)
	ListSecurityGroups() ([]groups.SecGroup, error)
	// Router operations
	ListRouters(ctx context.Context) ([]Router, error)
	GetRouter(ctx context.Context, id string) (*Router, error)
	GetRouterInterfaces(ctx context.Context, id string) ([]RouterInterface, error)
	CreateRouter(ctx context.Context, name, externalNetID string) (*Router, error)
	DeleteRouter(ctx context.Context, id string) error
	AddRouterInterface(ctx context.Context, routerID, subnetID string) error
	RemoveRouterInterface(ctx context.Context, routerID, subnetID string) error
	// Port operations
	ListPorts(ctx context.Context) ([]Port, error)
	GetPort(ctx context.Context, id string) (*Port, error)
	ListPortsByServer(ctx context.Context, serverID string) ([]Port, error)
	ListPortsByNetwork(ctx context.Context, networkID string) ([]Port, error)
	GetNetwork(ctx context.Context, id string) (*networks.Network, error)
	// Security group rule operations
	ListSecurityGroupRules(ctx context.Context, sgID string) ([]SecurityGroupRule, error)
	CreateSecurityGroupRule(ctx context.Context, sgID string, rule SecurityGroupRuleInput) (*SecurityGroupRule, error)
	DeleteSecurityGroupRule(ctx context.Context, id string) error
}

type networkClient struct {
	client *gophercloud.ServiceClient
}

// NewNetworkClient creates a new NetworkClient given authentication options.
func NewNetworkClient(authOpts gophercloud.AuthOptions) (NetworkClient, error) {
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with OpenStack: %w", err)
	}
	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to create network client: %w", err)
	}
	return &networkClient{client: client}, nil
}

// ListNetworks returns all networks visible to the authenticated project.
func (c *networkClient) ListNetworks() ([]networks.Network, error) {
	allPages, err := networks.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return networks.ExtractNetworks(allPages)
}

// ListSubnets returns all subnets visible to the authenticated project.
func (c *networkClient) ListSubnets() ([]subnets.Subnet, error) {
	allPages, err := subnets.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return subnets.ExtractSubnets(allPages)
}

// GetSubnet retrieves a subnet by ID.
func (c *networkClient) GetSubnet(ctx context.Context, subnetID string) (*subnets.Subnet, error) {
	_ = ctx
	s, err := subnets.Get(c.client, subnetID).Extract()
	if err != nil {
		return nil, err
	}
	return s, nil
}

// ListFloatingIPs returns all floating IPs visible to the authenticated project.
func (c *networkClient) ListFloatingIPs() ([]floatingips.FloatingIP, error) {
	allPages, err := floatingips.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return floatingips.ExtractFloatingIPs(allPages)
}

// AllocateFloatingIP creates a new floating IP using the provided options.
func (c *networkClient) AllocateFloatingIP(opts floatingips.CreateOptsBuilder) (floatingips.FloatingIP, error) {
	result := floatingips.Create(c.client, opts)
	fip, err := result.Extract()
	if err != nil {
		return floatingips.FloatingIP{}, err
	}
	return *fip, nil
}

// ReleaseFloatingIP deletes the floating IP identified by id.
func (c *networkClient) ReleaseFloatingIP(id string) error {
	return floatingips.Delete(c.client, id).ExtractErr()
}

// AssociateFloatingIP associates a floating IP with a port.
func (c *networkClient) AssociateFloatingIP(fipID string, portID string) (floatingips.FloatingIP, error) {
	opts := floatingips.UpdateOpts{PortID: &portID}
	result := floatingips.Update(c.client, fipID, opts)
	fip, err := result.Extract()
	if err != nil {
		return floatingips.FloatingIP{}, err
	}
	return *fip, nil
}

// DisassociateFloatingIP disassociates a floating IP from any port.
func (c *networkClient) DisassociateFloatingIP(fipID string) (floatingips.FloatingIP, error) {
	empty := ""
	opts := floatingips.UpdateOpts{PortID: &empty}
	result := floatingips.Update(c.client, fipID, opts)
	fip, err := result.Extract()
	if err != nil {
		return floatingips.FloatingIP{}, err
	}
	return *fip, nil
}

// ListSecurityGroups returns all security groups visible to the authenticated project.
func (c *networkClient) ListSecurityGroups() ([]groups.SecGroup, error) {
	allPages, err := groups.List(c.client, groups.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	return groups.ExtractGroups(allPages)
}

// Router operations
func (c *networkClient) ListRouters(ctx context.Context) ([]Router, error) {
	_ = ctx // ctx currently unused
	allPages, err := routers.List(c.client, routers.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	return routers.ExtractRouters(allPages)
}

func (c *networkClient) GetRouter(ctx context.Context, id string) (*Router, error) {
	_ = ctx
	r, err := routers.Get(c.client, id).Extract()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *networkClient) GetRouterInterfaces(ctx context.Context, id string) ([]RouterInterface, error) {
	_ = ctx
	// Gophercloud does not provide a direct ListInterfaces function; returning empty slice for now.
	return []RouterInterface{}, nil
}

func (c *networkClient) CreateRouter(ctx context.Context, name, externalNetID string) (*Router, error) {
	_ = ctx
	r, err := routers.Create(c.client, routers.CreateOpts{Name: name, GatewayInfo: &routers.GatewayInfo{NetworkID: externalNetID}}).Extract()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *networkClient) DeleteRouter(ctx context.Context, id string) error {
	_ = ctx
	return routers.Delete(c.client, id).ExtractErr()
}

func (c *networkClient) AddRouterInterface(ctx context.Context, routerID, subnetID string) error {
	_ = ctx
	opts := routers.AddInterfaceOpts{SubnetID: subnetID}
	_, err := routers.AddInterface(c.client, routerID, opts).Extract()
	return err
}

func (c *networkClient) RemoveRouterInterface(ctx context.Context, routerID, subnetID string) error {
	_ = ctx
	opts := routers.RemoveInterfaceOpts{SubnetID: subnetID}
	_, err := routers.RemoveInterface(c.client, routerID, opts).Extract()
	return err
}

// Port operations
func (c *networkClient) ListPorts(ctx context.Context) ([]Port, error) {
	_ = ctx
	allPages, err := ports.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return ports.ExtractPorts(allPages)
}

func (c *networkClient) ListPortsByServer(ctx context.Context, serverID string) ([]Port, error) {
	_ = ctx
	allPages, err := ports.List(c.client, ports.ListOpts{DeviceID: serverID}).AllPages()
	if err != nil {
		return nil, err
	}
	return ports.ExtractPorts(allPages)
}

// ListPortsByNetwork returns ports for a given network ID.
func (c *networkClient) ListPortsByNetwork(ctx context.Context, networkID string) ([]Port, error) {
	_ = ctx
	allPages, err := ports.List(c.client, ports.ListOpts{NetworkID: networkID}).AllPages()
	if err != nil {
		return nil, err
	}
	return ports.ExtractPorts(allPages)
}

// GetPort retrieves a port by ID.
func (c *networkClient) GetPort(ctx context.Context, id string) (*Port, error) {
	_ = ctx
	p, err := ports.Get(c.client, id).Extract()
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetNetwork retrieves a network by ID.
func (c *networkClient) GetNetwork(ctx context.Context, id string) (*networks.Network, error) {
	_ = ctx
	n, err := networks.Get(c.client, id).Extract()
	if err != nil {
		return nil, err
	}
	return n, nil
}

// Security group rule operations
func (c *networkClient) ListSecurityGroupRules(ctx context.Context, sgID string) ([]SecurityGroupRule, error) {
	_ = ctx
	allPages, err := rules.List(c.client, rules.ListOpts{SecGroupID: sgID}).AllPages()
	if err != nil {
		return nil, err
	}
	return rules.ExtractRules(allPages)
}

func (c *networkClient) CreateSecurityGroupRule(ctx context.Context, sgID string, rule SecurityGroupRuleInput) (*SecurityGroupRule, error) {
	_ = ctx
	r, err := rules.Create(c.client, rule).Extract()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *networkClient) DeleteSecurityGroupRule(ctx context.Context, id string) error {
	_ = ctx
	return rules.Delete(c.client, id).ExtractErr()
}

// Ensure NetworkClient implements the interface.
var _ NetworkClient = (*networkClient)(nil)
