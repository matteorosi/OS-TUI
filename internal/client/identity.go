package client

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
)

// IdentityClient defines methods for interacting with OpenStack Identity (Keystone) service.
type IdentityClient interface {
	ListProjects() ([]projects.Project, error)
	GetCurrentProject() (projects.Project, error)
	ListUsers() ([]users.User, error)
	GetTokenInfo() (*tokens.Token, error)
}

type identityClient struct {
	client *gophercloud.ServiceClient
}

// NewIdentityClient creates a new IdentityClient given authentication options.
func NewIdentityClient(authOpts gophercloud.AuthOptions) (IdentityClient, error) {
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with OpenStack: %w", err)
	}
	client, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to create identity client: %w", err)
	}
	return &identityClient{client: client}, nil
}

// ListProjects returns all projects visible to the authenticated user.
func (c *identityClient) ListProjects() ([]projects.Project, error) {
	allPages, err := projects.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return projects.ExtractProjects(allPages)
}

// GetCurrentProject returns the project associated with the current token.
func (c *identityClient) GetCurrentProject() (projects.Project, error) {
	tokenID := c.client.ProviderClient.TokenID
	if tokenID == "" {
		return projects.Project{}, fmt.Errorf("no token ID available")
	}
	result := tokens.Get(c.client, tokenID)
	proj, err := result.ExtractProject()
	if err != nil {
		return projects.Project{}, err
	}
	if proj == nil {
		return projects.Project{}, fmt.Errorf("project not found in token")
	}
	// Map token.Project to projects.Project (populate ID, Name, DomainID)
	p := projects.Project{
		ID:       proj.ID,
		Name:     proj.Name,
		DomainID: proj.Domain.ID,
	}
	return p, nil
}

// ListUsers returns all users visible to the authenticated user.
func (c *identityClient) ListUsers() ([]users.User, error) {
	allPages, err := users.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return users.ExtractUsers(allPages)
}

// GetTokenInfo retrieves information about the current token.
func (c *identityClient) GetTokenInfo() (*tokens.Token, error) {
	tokenID := c.client.ProviderClient.TokenID
	if tokenID == "" {
		return nil, fmt.Errorf("no token ID available")
	}
	result := tokens.Get(c.client, tokenID)
	return result.ExtractToken()
}

// Ensure identityClient implements IdentityClient.
var _ IdentityClient = (*identityClient)(nil)
