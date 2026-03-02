package client

import (
	"context"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"ostui/internal/cache"
)

const cacheKeyComputeInstances = "list_instances"
const cacheResourceCompute = "compute"

// CachedComputeClient wraps a ComputeClient and caches list operations.
type CachedComputeClient struct {
	inner ComputeClient
	cache *cache.Cache
}

// NewCachedComputeClient returns a ComputeClient backed by cache.
func NewCachedComputeClient(inner ComputeClient, c *cache.Cache) ComputeClient {
	return &CachedComputeClient{inner: inner, cache: c}
}

func (c *CachedComputeClient) ListInstances() ([]servers.Server, error) {
	if v, ok := c.cache.Get(cacheResourceCompute, cacheKeyComputeInstances); ok {
		if list, ok := v.([]servers.Server); ok {
			return list, nil
		}
		// Type mismatch: evict and fall through to live fetch.
		c.cache.Delete(cacheResourceCompute, cacheKeyComputeInstances)
	}
	list, err := c.inner.ListInstances()
	if err != nil {
		return nil, err
	}
	c.cache.Set(cacheResourceCompute, cacheKeyComputeInstances, list)
	return list, nil
}

func (c *CachedComputeClient) GetInstance(id string) (servers.Server, error) {
	return c.inner.GetInstance(id)
}
func (c *CachedComputeClient) StartInstance(id string) error {
	return c.inner.StartInstance(id)
}
func (c *CachedComputeClient) StopInstance(id string) error {
	return c.inner.StopInstance(id)
}
func (c *CachedComputeClient) DeleteInstance(id string) error {
	return c.inner.DeleteInstance(id)
}
func (c *CachedComputeClient) RebootInstance(id string) error {
	return c.inner.RebootInstance(id)
}
func (c *CachedComputeClient) ListFlavors() ([]flavors.Flavor, error) {
	return c.inner.ListFlavors()
}
func (c *CachedComputeClient) ListKeypairs() ([]keypairs.KeyPair, error) {
	return c.inner.ListKeypairs()
}
func (c *CachedComputeClient) GetConsoleLog(id string, lines int) (string, error) {
	return c.inner.GetConsoleLog(id, lines)
}
func (c *CachedComputeClient) GetConsoleURL(ctx context.Context, id, consoleType string) (string, error) {
	return c.inner.GetConsoleURL(ctx, id, consoleType)
}
func (c *CachedComputeClient) ListHypervisors(ctx context.Context) ([]hypervisors.Hypervisor, error) {
	return c.inner.ListHypervisors(ctx)
}
func (c *CachedComputeClient) GetHypervisor(ctx context.Context, id string) (*hypervisors.Hypervisor, error) {
	return c.inner.GetHypervisor(ctx, id)
}
func (c *CachedComputeClient) ListAvailabilityZones(ctx context.Context) ([]availabilityzones.AvailabilityZone, error) {
	return c.inner.ListAvailabilityZones(ctx)
}
func (c *CachedComputeClient) GetFlavor(ctx context.Context, flavorID string) (flavors.Flavor, error) {
	return c.inner.GetFlavor(ctx, flavorID)
}
func (c *CachedComputeClient) GetKeypair(ctx context.Context, name string) (keypairs.KeyPair, error) {
	return c.inner.GetKeypair(ctx, name)
}
func (c *CachedComputeClient) ListServerInterfaces(ctx context.Context, serverID string) ([]ServerInterface, error) {
	return c.inner.ListServerInterfaces(ctx, serverID)
}
func (c *CachedComputeClient) ListServerVolumes(ctx context.Context, serverID string) ([]ServerVolume, error) {
	return c.inner.ListServerVolumes(ctx, serverID)
}

var _ ComputeClient = (*CachedComputeClient)(nil)
