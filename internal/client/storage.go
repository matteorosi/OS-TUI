package client

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
)

// StorageClient defines the methods for interacting with OpenStack Block Storage (Cinder) service.
type StorageClient interface {
	ListVolumes() ([]volumes.Volume, error)
	GetVolume(id string) (volumes.Volume, error)
	DeleteVolume(id string) error
	ListSnapshots() ([]snapshots.Snapshot, error)
	CreateSnapshot(opts snapshots.CreateOptsBuilder) (snapshots.Snapshot, error)
}

type storageClient struct {
	client *gophercloud.ServiceClient
}

// NewStorageClient creates a new StorageClient given authentication options.
func NewStorageClient(authOpts gophercloud.AuthOptions) (StorageClient, error) {
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with OpenStack: %w", err)
	}
	client, err := openstack.NewBlockStorageV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to create block storage client: %w", err)
	}
	return &storageClient{client: client}, nil
}

// ListVolumes returns all block storage volumes visible to the authenticated project.
func (c *storageClient) ListVolumes() ([]volumes.Volume, error) {
	allPages, err := volumes.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return volumes.ExtractVolumes(allPages)
}

// GetVolume retrieves a single volume by its ID.
func (c *storageClient) GetVolume(id string) (volumes.Volume, error) {
	result := volumes.Get(c.client, id)
	vol, err := result.Extract()
	if err != nil {
		return volumes.Volume{}, err
	}
	return *vol, nil
}

// DeleteVolume removes the specified volume.
func (c *storageClient) DeleteVolume(id string) error {
	return volumes.Delete(c.client, id, nil).ExtractErr()
}

// ListSnapshots returns all volume snapshots visible to the authenticated project.
func (c *storageClient) ListSnapshots() ([]snapshots.Snapshot, error) {
	allPages, err := snapshots.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return snapshots.ExtractSnapshots(allPages)
}

// CreateSnapshot creates a new snapshot for a volume using the provided options.
func (c *storageClient) CreateSnapshot(opts snapshots.CreateOptsBuilder) (snapshots.Snapshot, error) {
	result := snapshots.Create(c.client, opts)
	snap, err := result.Extract()
	if err != nil {
		return snapshots.Snapshot{}, err
	}
	return *snap, nil
}

// Ensure storageClient implements the StorageClient interface.
var _ StorageClient = (*storageClient)(nil)
