package network

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

type mockNetworkClient struct {
	networks []networks.Network
	netErr   error

	subnets []subnets.Subnet
	subErr  error

	floatingIPs []floatingips.FloatingIP
	fipErr      error

	allocated floatingips.FloatingIP
	allocErr  error

	releaseErr error

	associate floatingips.FloatingIP
	assocErr  error

	disassociate floatingips.FloatingIP
	disassocErr  error

	secGroups []groups.SecGroup
	secErr    error
}

func (m *mockNetworkClient) ListNetworks() ([]networks.Network, error) {
	return m.networks, m.netErr
}
func (m *mockNetworkClient) ListSubnets() ([]subnets.Subnet, error) {
	return m.subnets, m.subErr
}

// GetSubnet returns a subnet by ID from the mock data.
func (m *mockNetworkClient) GetSubnet(ctx context.Context, subnetID string) (*subnets.Subnet, error) {
	for _, s := range m.subnets {
		if s.ID == subnetID {
			subCopy := s
			return &subCopy, nil
		}
	}
	return nil, fmt.Errorf("subnet not found")
}
func (m *mockNetworkClient) ListFloatingIPs() ([]floatingips.FloatingIP, error) {
	return m.floatingIPs, m.fipErr
}
func (m *mockNetworkClient) AllocateFloatingIP(opts floatingips.CreateOptsBuilder) (floatingips.FloatingIP, error) {
	return m.allocated, m.allocErr
}
func (m *mockNetworkClient) ReleaseFloatingIP(id string) error {
	return m.releaseErr
}
func (m *mockNetworkClient) AssociateFloatingIP(fipID string, portID string) (floatingips.FloatingIP, error) {
	return m.associate, m.assocErr
}
func (m *mockNetworkClient) DisassociateFloatingIP(fipID string) (floatingips.FloatingIP, error) {
	return m.disassociate, m.disassocErr
}
func (m *mockNetworkClient) ListSecurityGroups() ([]groups.SecGroup, error) {
	return m.secGroups, m.secErr
}

// Stub implementations for new NetworkClient methods.
func (m *mockNetworkClient) ListRouters(ctx context.Context) ([]routers.Router, error) {
	return []routers.Router{}, nil
}
func (m *mockNetworkClient) GetRouter(ctx context.Context, id string) (*routers.Router, error) {
	return nil, nil
}
func (m *mockNetworkClient) GetRouterInterfaces(ctx context.Context, id string) ([]ports.Port, error) {
	return []ports.Port{}, nil
}
func (m *mockNetworkClient) CreateRouter(ctx context.Context, name, externalNetID string) (*routers.Router, error) {
	return nil, nil
}
func (m *mockNetworkClient) DeleteRouter(ctx context.Context, id string) error {
	return nil
}
func (m *mockNetworkClient) AddRouterInterface(ctx context.Context, routerID, subnetID string) error {
	return nil
}
func (m *mockNetworkClient) RemoveRouterInterface(ctx context.Context, routerID, subnetID string) error {
	return nil
}
func (m *mockNetworkClient) ListPorts(ctx context.Context) ([]ports.Port, error) {
	return []ports.Port{}, nil
}

// ListPortsByServer returns ports for a given server ID (mock implementation).
func (m *mockNetworkClient) ListPortsByServer(ctx context.Context, serverID string) ([]ports.Port, error) {
	return []ports.Port{}, nil
}

// ListPortsByNetwork returns ports for a given network ID (mock implementation).
func (m *mockNetworkClient) ListPortsByNetwork(ctx context.Context, networkID string) ([]ports.Port, error) {
	return []ports.Port{}, nil
}

// GetNetwork returns a network by ID from the mock data.
func (m *mockNetworkClient) GetNetwork(ctx context.Context, id string) (*networks.Network, error) {
	for _, n := range m.networks {
		if n.ID == id {
			netCopy := n
			return &netCopy, nil
		}
	}
	return nil, fmt.Errorf("network not found")
}
func (m *mockNetworkClient) GetPort(ctx context.Context, id string) (*ports.Port, error) {
	return nil, nil
}
func (m *mockNetworkClient) ListSecurityGroupRules(ctx context.Context, sgID string) ([]rules.SecGroupRule, error) {
	return []rules.SecGroupRule{}, nil
}
func (m *mockNetworkClient) CreateSecurityGroupRule(ctx context.Context, sgID string, rule rules.CreateOpts) (*rules.SecGroupRule, error) {
	return nil, nil
}
func (m *mockNetworkClient) DeleteSecurityGroupRule(ctx context.Context, id string) error {
	return nil
}

func TestRenderNetworksSuccess(t *testing.T) {
	mock := &mockNetworkClient{networks: []networks.Network{{ID: "net-1", Name: "net1", Status: "ACTIVE"}}}
	out := RenderNetworks(mock)
	if !strings.Contains(out, "net1") {
		t.Fatalf("expected network name in output, got %s", out)
	}
}

func TestRenderNetworksError(t *testing.T) {
	mock := &mockNetworkClient{netErr: errors.New("list error")}
	out := RenderNetworks(mock)
	if !strings.Contains(out, "Failed to list networks") {
		t.Fatalf("expected error message, got %s", out)
	}
}

func TestRenderSubnetsSuccess(t *testing.T) {
	mock := &mockNetworkClient{subnets: []subnets.Subnet{{ID: "sub-1", Name: "sub1", CIDR: "10.0.0.0/24", IPVersion: 4}}}
	out := RenderSubnets(mock)
	if !strings.Contains(out, "sub1") {
		t.Fatalf("expected subnet name in output, got %s", out)
	}
}

func TestRenderSubnetsError(t *testing.T) {
	mock := &mockNetworkClient{subErr: errors.New("list error")}
	out := RenderSubnets(mock)
	if !strings.Contains(out, "Failed to list subnets") {
		t.Fatalf("expected error message, got %s", out)
	}
}

func TestRenderFloatingIPsSuccess(t *testing.T) {
	mock := &mockNetworkClient{floatingIPs: []floatingips.FloatingIP{{ID: "fip-1", FloatingNetworkID: "net-1", FixedIP: "10.0.0.5", PortID: "port-1", Status: "ACTIVE"}}}
	out := RenderFloatingIPs(mock)
	if !strings.Contains(out, "fip-1") {
		t.Fatalf("expected floating IP ID in output, got %s", out)
	}
}

func TestRenderFloatingIPsError(t *testing.T) {
	mock := &mockNetworkClient{fipErr: errors.New("list error")}
	out := RenderFloatingIPs(mock)
	if !strings.Contains(out, "Failed to list floating IPs") {
		t.Fatalf("expected error message, got %s", out)
	}
}

func TestRenderSecurityGroupsSuccess(t *testing.T) {
	mock := &mockNetworkClient{secGroups: []groups.SecGroup{{ID: "sg-1", Name: "sg1", Description: "desc", Stateful: true}}}
	out := RenderSecurityGroups(mock)
	if !strings.Contains(out, "sg1") {
		t.Fatalf("expected security group name in output, got %s", out)
	}
}

func TestRenderSecurityGroupsError(t *testing.T) {
	mock := &mockNetworkClient{secErr: errors.New("list error")}
	out := RenderSecurityGroups(mock)
	if !strings.Contains(out, "Failed to list security groups") {
		t.Fatalf("expected error message, got %s", out)
	}
}

func TestRenderSecurityGroupDetailSuccess(t *testing.T) {
	mock := &mockNetworkClient{secGroups: []groups.SecGroup{{ID: "sg-1", Name: "sg1", Description: "desc", Stateful: true, TenantID: "tenant", ProjectID: "proj", CreatedAt: time.Now(), UpdatedAt: time.Now(), Tags: []string{"tag"}}}}
	out := RenderSecurityGroupDetail(mock, "sg-1")
	if !strings.Contains(out, "Security Group Details") {
		t.Fatalf("expected detail title, got %s", out)
	}
	if !strings.Contains(out, "sg1") {
		t.Fatalf("expected security group name, got %s", out)
	}
}

func TestRenderSecurityGroupDetailNotFound(t *testing.T) {
	mock := &mockNetworkClient{secGroups: []groups.SecGroup{{ID: "sg-2", Name: "sg2"}}}
	out := RenderSecurityGroupDetail(mock, "sg-1")
	if !strings.Contains(out, "Security group not found") {
		t.Fatalf("expected not found message, got %s", out)
	}
}

func TestRenderAllocateFloatingIPForm(t *testing.T) {
	out := RenderAllocateFloatingIPForm()
	if !strings.Contains(out, "FloatingNetworkID:") {
		t.Fatalf("expected form field, got %s", out)
	}
}
