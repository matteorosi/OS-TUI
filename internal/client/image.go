package client

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
)

// ImageClient defines methods for interacting with OpenStack Image (Glance) service via Compute API.
type ImageClient interface {
	ListImages(ctx context.Context) ([]images.Image, error)
	GetImage(ctx context.Context, id string) (*images.Image, error)
	DeleteImage(ctx context.Context, id string) error
}

type imageClient struct {
	client *gophercloud.ServiceClient
}

// NewImageClient creates a new ImageClient given authentication options.
func NewImageClient(authOpts gophercloud.AuthOptions) (ImageClient, error) {
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with OpenStack: %w", err)
	}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client for images: %w", err)
	}
	return &imageClient{client: client}, nil
}

// ListImages returns all images visible to the authenticated project.
func (c *imageClient) ListImages(ctx context.Context) ([]images.Image, error) {
	// Context is currently unused; the underlying gophercloud API does not accept a context.
	_ = ctx
	allPages, err := images.ListDetail(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return images.ExtractImages(allPages)
}

// GetImage retrieves a single image by its ID.
func (c *imageClient) GetImage(ctx context.Context, id string) (*images.Image, error) {
	_ = ctx
	result := images.Get(c.client, id)
	img, err := result.Extract()
	if err != nil {
		return nil, err
	}
	return img, nil
}

// DeleteImage removes the specified image.
func (c *imageClient) DeleteImage(ctx context.Context, id string) error {
	_ = ctx
	return images.Delete(c.client, id).ExtractErr()
}

// Ensure imageClient implements ImageClient.
var _ ImageClient = (*imageClient)(nil)
