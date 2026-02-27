package client

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	vLimits "github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/limits"
	cLimits "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/limits"
)

// Limits aggregates compute and volume limits.
type Limits struct {
	Compute *cLimits.Limits
	Volume  *vLimits.Limits
}

// LimitsClient defines a method to retrieve limits for both compute and volume services.
type LimitsClient interface {
	GetLimits(ctx context.Context) (*Limits, error)
}

type limitsClient struct {
	compute *gophercloud.ServiceClient
	volume  *gophercloud.ServiceClient
}

// NewLimitsClient creates a new LimitsClient given authentication options.
func NewLimitsClient(authOpts gophercloud.AuthOptions) (LimitsClient, error) {
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with OpenStack: %w", err)
	}
	computeClient, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client for limits: %w", err)
	}
	volumeClient, err := openstack.NewBlockStorageV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to create block storage client for limits: %w", err)
	}
	return &limitsClient{compute: computeClient, volume: volumeClient}, nil
}

// GetLimits retrieves compute and volume limits.
func (c *limitsClient) GetLimits(ctx context.Context) (*Limits, error) {
	// ctx currently unused; gophercloud APIs do not accept context.
	_ = ctx
	// Compute limits
	compRes := cLimits.Get(c.compute, nil)
	compLimits, err := compRes.Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get compute limits: %w", err)
	}
	// Volume limits
	volRes := vLimits.Get(c.volume)
	volLimits, err := volRes.Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get volume limits: %w", err)
	}
	return &Limits{Compute: compLimits, Volume: volLimits}, nil
}

// Ensure limitsClient implements LimitsClient.
var _ LimitsClient = (*limitsClient)(nil)
