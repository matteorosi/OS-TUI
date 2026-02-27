package client

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
)

// ObjectStorageClient defines methods for interacting with OpenStack Object Storage (Swift) service.
type ObjectStorageClient interface {
	// ListBuckets returns a slice of containers (buckets) with their metadata.
	ListBuckets() ([]containers.Container, error)
}

type objectStorageClient struct {
	client *gophercloud.ServiceClient
}

// NewObjectStorageClient creates a new ObjectStorageClient given authentication options.
func NewObjectStorageClient(authOpts gophercloud.AuthOptions) (ObjectStorageClient, error) {
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with OpenStack: %w", err)
	}
	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to create object storage client: %w", err)
	}
	return &objectStorageClient{client: client}, nil
}

// ListBuckets retrieves all containers (buckets) visible to the authenticated project.
func (c *objectStorageClient) ListBuckets() ([]containers.Container, error) {
	allPages, err := containers.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return containers.ExtractInfo(allPages)
}

// Ensure objectStorageClient implements ObjectStorageClient.
var _ ObjectStorageClient = (*objectStorageClient)(nil)
