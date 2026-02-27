package client

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
)

// LoadBalancer represents a simplified load balancer.
type LoadBalancer struct {
	ID                 string
	Name               string
	Description        string
	ProvisioningStatus string
	OperatingStatus    string
	VipAddress         string
	VipSubnetID        string
}

// Listener represents a simplified listener.
type Listener struct {
	ID                 string
	Name               string
	Protocol           string
	ProtocolPort       int
	ProvisioningStatus string
}

// Pool represents a simplified pool.
type Pool struct {
	ID                 string
	Name               string
	Protocol           string
	LBAlgorithm        string
	ProvisioningStatus string
}

// LoadBalancerClient defines methods for interacting with Octavia load balancer service.
type LoadBalancerClient interface {
	ListLoadBalancers(ctx context.Context) ([]LoadBalancer, error)
	ListListeners(ctx context.Context, lbID string) ([]Listener, error)
	ListPools(ctx context.Context, lbID string) ([]Pool, error)
}

// LoadBalancerClientImpl is the concrete implementation using gophercloud.
type LoadBalancerClientImpl struct {
	client *gophercloud.ServiceClient
}

// NewLoadBalancerClient creates a new client for the Octavia load balancer service.
func NewLoadBalancerClient(provider *gophercloud.ProviderClient, opts gophercloud.EndpointOpts) (*LoadBalancerClientImpl, error) {
	client, err := openstack.NewLoadBalancerV2(provider, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer client: %w", err)
	}
	return &LoadBalancerClientImpl{client: client}, nil
}

// ListLoadBalancers returns all load balancers visible to the project.
func (c *LoadBalancerClientImpl) ListLoadBalancers(ctx context.Context) ([]LoadBalancer, error) {
	allPages, err := loadbalancers.List(c.client, nil).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	gopherLBs, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		return nil, err
	}
	lbs := make([]LoadBalancer, len(gopherLBs))
	for i, glb := range gopherLBs {
		lbs[i] = LoadBalancer{
			ID:                 glb.ID,
			Name:               glb.Name,
			Description:        glb.Description,
			ProvisioningStatus: glb.ProvisioningStatus,
			OperatingStatus:    glb.OperatingStatus,
			VipAddress:         glb.VipAddress,
			VipSubnetID:        glb.VipSubnetID,
		}
	}
	return lbs, nil
}

// ListListeners returns listeners for a specific load balancer.
func (c *LoadBalancerClientImpl) ListListeners(ctx context.Context, lbID string) ([]Listener, error) {
	opts := listeners.ListOpts{LoadbalancerID: lbID}
	allPages, err := listeners.List(c.client, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	gopherListeners, err := listeners.ExtractListeners(allPages)
	if err != nil {
		return nil, err
	}
	lst := make([]Listener, len(gopherListeners))
	for i, gl := range gopherListeners {
		lst[i] = Listener{
			ID:                 gl.ID,
			Name:               gl.Name,
			Protocol:           gl.Protocol,
			ProtocolPort:       gl.ProtocolPort,
			ProvisioningStatus: gl.ProvisioningStatus,
		}
	}
	return lst, nil
}

// ListPools returns pools for a specific load balancer.
func (c *LoadBalancerClientImpl) ListPools(ctx context.Context, lbID string) ([]Pool, error) {
	opts := pools.ListOpts{LoadbalancerID: lbID}
	allPages, err := pools.List(c.client, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	gopherPools, err := pools.ExtractPools(allPages)
	if err != nil {
		return nil, err
	}
	ps := make([]Pool, len(gopherPools))
	for i, gp := range gopherPools {
		ps[i] = Pool{
			ID:                 gp.ID,
			Name:               gp.Name,
			Protocol:           gp.Protocol,
			LBAlgorithm:        gp.LBMethod,
			ProvisioningStatus: gp.ProvisioningStatus,
		}
	}
	return ps, nil
}

// Ensure LoadBalancerClientImpl implements LoadBalancerClient.
var _ LoadBalancerClient = (*LoadBalancerClientImpl)(nil)
