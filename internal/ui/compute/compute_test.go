package compute

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"ostui/internal/client"
)

type mockComputeClient struct {
	listInstances []servers.Server
	listErr       error
	getInstance   servers.Server
	getErr        error
}

func (m *mockComputeClient) ListInstances() ([]servers.Server, error) {
	return m.listInstances, m.listErr
}

func (m *mockComputeClient) GetInstance(id string) (servers.Server, error) {
	return m.getInstance, m.getErr
}

// Unused methods for the ComputeClient interface.
func (m *mockComputeClient) GetConsoleLog(id string, lines int) (string, error) { return "", nil }

// Stub implementations for the remaining ComputeClient methods.
func (m *mockComputeClient) StartInstance(id string) error             { return nil }
func (m *mockComputeClient) StopInstance(id string) error              { return nil }
func (m *mockComputeClient) DeleteInstance(id string) error            { return nil }
func (m *mockComputeClient) ListFlavors() ([]flavors.Flavor, error)    { return nil, nil }
func (m *mockComputeClient) ListKeypairs() ([]keypairs.KeyPair, error) { return nil, nil }

// Additional stub methods for new ComputeClient interface methods.
func (m *mockComputeClient) GetConsoleURL(ctx context.Context, id, consoleType string) (string, error) {
	return "", nil
}
func (m *mockComputeClient) ListHypervisors(ctx context.Context) ([]hypervisors.Hypervisor, error) {
	return nil, nil
}
func (m *mockComputeClient) GetHypervisor(ctx context.Context, id string) (*hypervisors.Hypervisor, error) {
	return nil, nil
}
func (m *mockComputeClient) ListAvailabilityZones(ctx context.Context) ([]availabilityzones.AvailabilityZone, error) {
	return nil, nil
}

// GetFlavor returns a stub flavor.
func (m *mockComputeClient) GetFlavor(ctx context.Context, flavorID string) (flavors.Flavor, error) {
	return flavors.Flavor{}, nil
}

// GetKeypair returns a stub keypair.
func (m *mockComputeClient) GetKeypair(ctx context.Context, name string) (keypairs.KeyPair, error) {
	return keypairs.KeyPair{}, nil
}

// ListServerInterfaces returns an empty slice (mock).
func (m *mockComputeClient) ListServerInterfaces(ctx context.Context, serverID string) ([]client.ServerInterface, error) {
	return []client.ServerInterface{}, nil
}

// ListServerVolumes returns an empty slice (mock).
func (m *mockComputeClient) ListServerVolumes(ctx context.Context, serverID string) ([]client.ServerVolume, error) {
	return []client.ServerVolume{}, nil
}

func TestRenderInstancesSuccess(t *testing.T) {
	mock := &mockComputeClient{
		listInstances: []servers.Server{{ID: "123", Name: "test-instance", Status: "ACTIVE"}},
	}
	out := RenderInstances(mock)
	if !strings.Contains(out, "test-instance") {
		t.Fatalf("expected instance name in output, got %s", out)
	}
}

func TestRenderInstancesError(t *testing.T) {
	mock := &mockComputeClient{listErr: errors.New("list error")}
	out := RenderInstances(mock)
	if !strings.Contains(out, "Failed to list instances") {
		t.Fatalf("expected error message in output, got %s", out)
	}
}

func TestRenderInstanceDetailSuccess(t *testing.T) {
	mock := &mockComputeClient{getInstance: servers.Server{
		ID:       "123",
		Name:     "test-instance",
		Status:   "ACTIVE",
		Flavor:   map[string]interface{}{"id": "flavor-1"},
		Image:    map[string]interface{}{"id": "image-1"},
		Created:  time.Now(),
		Updated:  time.Now(),
		HostID:   "host-1",
		KeyName:  "keypair-1",
		UserID:   "user-1",
		TenantID: "tenant-1",
	}}
	out := RenderInstanceDetail(mock, "123")
	if !strings.Contains(out, "Instance Details") {
		t.Fatalf("expected detail title, got %s", out)
	}
	if !strings.Contains(out, "test-instance") {
		t.Fatalf("expected instance name, got %s", out)
	}
}

func TestRenderInstanceDetailError(t *testing.T) {
	mock := &mockComputeClient{getErr: errors.New("get error")}
	out := RenderInstanceDetail(mock, "123")
	if !strings.Contains(out, "Failed to get instance") {
		t.Fatalf("expected error message, got %s", out)
	}
}

func TestRenderInstanceForm(t *testing.T) {
	out := RenderInstanceForm()
	// The form view should contain the field prompts.
	if !strings.Contains(out, "Name:") || !strings.Contains(out, "Image:") {
		t.Fatalf("expected form fields in output, got %s", out)
	}
}
