package client

import (
	"context"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"ostui/internal/cache"
)

const cacheResourceNetwork = "network"

const (
	cacheKeyNetworkNetworks       = "list_networks"
	cacheKeyNetworkSubnets        = "list_subnets"
	cacheKeyNetworkFloatingIPs    = "list_floating_ips"
	cacheKeyNetworkRouters        = "list_routers"
	cacheKeyNetworkSecurityGroups = "list_security_groups"
)

// CachedNetworkClient wraps a NetworkClient and caches list operations.
type CachedNetworkClient struct {
	inner NetworkClient
	cache *cache.Cache
}

// NewCachedNetworkClient returns a NetworkClient backed by cache.
func NewCachedNetworkClient(inner NetworkClient, c *cache.Cache) NetworkClient {
	return &CachedNetworkClient{inner: inner, cache: c}
}

func (c *CachedNetworkClient) ListNetworks() ([]networks.Network, error) {
	if v, ok := c.cache.Get(cacheResourceNetwork, cacheKeyNetworkNetworks); ok {
		if list, ok := v.([]networks.Network); ok {
			return list, nil
		}
		c.cache.Delete(cacheResourceNetwork, cacheKeyNetworkNetworks)
	}
	list, err := c.inner.ListNetworks()
	if err != nil {
		return nil, err
	}
	c.cache.Set(cacheResourceNetwork, cacheKeyNetworkNetworks, list)
	return list, nil
}

func (c *CachedNetworkClient) ListSubnets() ([]subnets.Subnet, error) {
	if v, ok := c.cache.Get(cacheResourceNetwork, cacheKeyNetworkSubnets); ok {
		if list, ok := v.([]subnets.Subnet); ok {
			return list, nil
		}
		c.cache.Delete(cacheResourceNetwork, cacheKeyNetworkSubnets)
	}
	list, err := c.inner.ListSubnets()
	if err != nil {
		return nil, err
	}
	c.cache.Set(cacheResourceNetwork, cacheKeyNetworkSubnets, list)
	return list, nil
}

func (c *CachedNetworkClient) ListFloatingIPs() ([]floatingips.FloatingIP, error) {
	if v, ok := c.cache.Get(cacheResourceNetwork, cacheKeyNetworkFloatingIPs); ok {
		if list, ok := v.([]floatingips.FloatingIP); ok {
			return list, nil
		}
		c.cache.Delete(cacheResourceNetwork, cacheKeyNetworkFloatingIPs)
	}
	list, err := c.inner.ListFloatingIPs()
	if err != nil {
		return nil, err
	}
	c.cache.Set(cacheResourceNetwork, cacheKeyNetworkFloatingIPs, list)
	return list, nil
}

func (c *CachedNetworkClient) ListRouters(ctx context.Context) ([]Router, error) {
	if v, ok := c.cache.Get(cacheResourceNetwork, cacheKeyNetworkRouters); ok {
		if list, ok := v.([]Router); ok {
			return list, nil
		}
		c.cache.Delete(cacheResourceNetwork, cacheKeyNetworkRouters)
	}
	list, err := c.inner.ListRouters(ctx)
	if err != nil {
		return nil, err
	}
	c.cache.Set(cacheResourceNetwork, cacheKeyNetworkRouters, list)
	return list, nil
}

func (c *CachedNetworkClient) ListSecurityGroups() ([]groups.SecGroup, error) {
	if v, ok := c.cache.Get(cacheResourceNetwork, cacheKeyNetworkSecurityGroups); ok {
		if list, ok := v.([]groups.SecGroup); ok {
			return list, nil
		}
		c.cache.Delete(cacheResourceNetwork, cacheKeyNetworkSecurityGroups)
	}
	list, err := c.inner.ListSecurityGroups()
	if err != nil {
		return nil, err
	}
	c.cache.Set(cacheResourceNetwork, cacheKeyNetworkSecurityGroups, list)
	return list, nil
}

// Remaining methods — delegate directly, no caching.
func (c *CachedNetworkClient) GetSubnet(ctx context.Context, id string) (*subnets.Subnet, error) {
	return c.inner.GetSubnet(ctx, id)
}
func (c *CachedNetworkClient) AllocateFloatingIP(opts floatingips.CreateOptsBuilder) (floatingips.FloatingIP, error) {
	return c.inner.AllocateFloatingIP(opts)
}
func (c *CachedNetworkClient) ReleaseFloatingIP(id string) error {
	return c.inner.ReleaseFloatingIP(id)
}
func (c *CachedNetworkClient) AssociateFloatingIP(fipID, portID string) (floatingips.FloatingIP, error) {
	return c.inner.AssociateFloatingIP(fipID, portID)
}
func (c *CachedNetworkClient) DisassociateFloatingIP(fipID string) (floatingips.FloatingIP, error) {
	return c.inner.DisassociateFloatingIP(fipID)
}
func (c *CachedNetworkClient) GetRouter(ctx context.Context, id string) (*Router, error) {
	return c.inner.GetRouter(ctx, id)
}
func (c *CachedNetworkClient) GetRouterInterfaces(ctx context.Context, id string) ([]RouterInterface, error) {
	return c.inner.GetRouterInterfaces(ctx, id)
}
func (c *CachedNetworkClient) CreateRouter(ctx context.Context, name, externalNetID string) (*Router, error) {
	return c.inner.CreateRouter(ctx, name, externalNetID)
}
func (c *CachedNetworkClient) DeleteRouter(ctx context.Context, id string) error {
	return c.inner.DeleteRouter(ctx, id)
}
func (c *CachedNetworkClient) DeleteNetwork(id string) error {
	return c.inner.DeleteNetwork(id)
}
func (c *CachedNetworkClient) AddRouterInterface(ctx context.Context, routerID, subnetID string) error {
	return c.inner.AddRouterInterface(ctx, routerID, subnetID)
}
func (c *CachedNetworkClient) RemoveRouterInterface(ctx context.Context, routerID, subnetID string) error {
	return c.inner.RemoveRouterInterface(ctx, routerID, subnetID)
}
func (c *CachedNetworkClient) ListPorts(ctx context.Context) ([]Port, error) {
	return c.inner.ListPorts(ctx)
}
func (c *CachedNetworkClient) GetPort(ctx context.Context, id string) (*Port, error) {
	return c.inner.GetPort(ctx, id)
}
func (c *CachedNetworkClient) ListPortsByServer(ctx context.Context, serverID string) ([]Port, error) {
	return c.inner.ListPortsByServer(ctx, serverID)
}
func (c *CachedNetworkClient) ListPortsByNetwork(ctx context.Context, networkID string) ([]Port, error) {
	return c.inner.ListPortsByNetwork(ctx, networkID)
}
func (c *CachedNetworkClient) GetNetwork(ctx context.Context, id string) (*networks.Network, error) {
	return c.inner.GetNetwork(ctx, id)
}
func (c *CachedNetworkClient) ListSecurityGroupRules(ctx context.Context, sgID string) ([]SecurityGroupRule, error) {
	return c.inner.ListSecurityGroupRules(ctx, sgID)
}
func (c *CachedNetworkClient) CreateSecurityGroupRule(ctx context.Context, sgID string, rule SecurityGroupRuleInput) (*SecurityGroupRule, error) {
	return c.inner.CreateSecurityGroupRule(ctx, sgID, rule)
}
func (c *CachedNetworkClient) DeleteSecurityGroupRule(ctx context.Context, id string) error {
	return c.inner.DeleteSecurityGroupRule(ctx, id)
}

var _ NetworkClient = (*CachedNetworkClient)(nil)
