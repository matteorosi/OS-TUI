package client

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/remoteconsoles"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

// ComputeClient defines the methods for interacting with OpenStack Compute (Nova) service.
type ComputeClient interface {
	ListInstances() ([]servers.Server, error)
	GetInstance(id string) (servers.Server, error)
	StartInstance(id string) error
	StopInstance(id string) error
	DeleteInstance(id string) error
	ListFlavors() ([]flavors.Flavor, error)
	ListKeypairs() ([]keypairs.KeyPair, error)
	GetConsoleLog(id string, lines int) (string, error)
	GetConsoleURL(ctx context.Context, id, consoleType string) (string, error)
	ListHypervisors(ctx context.Context) ([]hypervisors.Hypervisor, error)
	GetHypervisor(ctx context.Context, id string) (*hypervisors.Hypervisor, error)
	ListAvailabilityZones(ctx context.Context) ([]availabilityzones.AvailabilityZone, error)
	GetFlavor(ctx context.Context, flavorID string) (flavors.Flavor, error)
	GetKeypair(ctx context.Context, name string) (keypairs.KeyPair, error)
	ListServerInterfaces(ctx context.Context, serverID string) ([]ServerInterface, error)
	ListServerVolumes(ctx context.Context, serverID string) ([]ServerVolume, error)
}

type ServerInterface struct {
	PortID     string
	NetworkID  string
	FixedIPs   []string
	MACAddress string
}

type ServerVolume struct {
	ID       string
	VolumeID string
	Device   string
}

// computeClient is a concrete implementation of ComputeClient using gophercloud.
type computeClient struct {
	client *gophercloud.ServiceClient
}

// NewComputeClient creates a new ComputeClient given authentication options.
// It authenticates with OpenStack and returns a client ready to call Compute APIs.
func NewComputeClient(authOpts gophercloud.AuthOptions) (ComputeClient, error) {
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with OpenStack: %w", err)
	}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client: %w", err)
	}
	return &computeClient{client: client}, nil
}

// ListInstances returns all compute instances (servers) visible to the authenticated user.
func (c *computeClient) ListInstances() ([]servers.Server, error) {
	allPages, err := servers.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return servers.ExtractServers(allPages)
}

// GetInstance retrieves a single server by its ID.
func (c *computeClient) GetInstance(id string) (servers.Server, error) {
	result := servers.Get(c.client, id)
	srv, err := result.Extract()
	if err != nil {
		return servers.Server{}, err
	}
	return *srv, nil
}

// StartInstance powers on the specified server.
func (c *computeClient) StartInstance(id string) error {
	return startstop.Start(c.client, id).ExtractErr()
}

// StopInstance powers off the specified server.
func (c *computeClient) StopInstance(id string) error {
	return startstop.Stop(c.client, id).ExtractErr()
}

// DeleteInstance removes the specified server.
func (c *computeClient) DeleteInstance(id string) error {
	return servers.Delete(c.client, id).ExtractErr()
}

// ListFlavors returns the list of available flavors (instance types).
func (c *computeClient) ListFlavors() ([]flavors.Flavor, error) {
	allPages, err := flavors.ListDetail(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return flavors.ExtractFlavors(allPages)
}

// ListKeypairs returns all SSH keypairs defined for the project.
func (c *computeClient) ListKeypairs() ([]keypairs.KeyPair, error) {
	allPages, err := keypairs.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return keypairs.ExtractKeyPairs(allPages)
}

// GetFlavor retrieves a flavor by ID.
func (c *computeClient) GetFlavor(ctx context.Context, flavorID string) (flavors.Flavor, error) {
	_ = ctx // ctx currently unused
	f, err := flavors.Get(c.client, flavorID).Extract()
	if err != nil {
		return flavors.Flavor{}, err
	}
	return *f, nil
}

// GetKeypair retrieves a keypair by name.
func (c *computeClient) GetKeypair(ctx context.Context, name string) (keypairs.KeyPair, error) {
	_ = ctx // ctx currently unused
	// Fallback to listing all keypairs and finding the matching one.
	kpList, err := c.ListKeypairs()
	if err != nil {
		return keypairs.KeyPair{}, err
	}
	for _, kp := range kpList {
		if kp.Name == name {
			return kp, nil
		}
	}
	return keypairs.KeyPair{}, fmt.Errorf("keypair %s not found", name)
}

func (c *computeClient) ListServerInterfaces(ctx context.Context, serverID string) ([]ServerInterface, error) {
	allPages, err := attachinterfaces.List(c.client, serverID).AllPages()
	if err != nil {
		return nil, err
	}
	ifaces, err := attachinterfaces.ExtractInterfaces(allPages)
	if err != nil {
		return nil, err
	}
	var result []ServerInterface
	for _, i := range ifaces {
		var ips []string
		for _, ip := range i.FixedIPs {
			ips = append(ips, ip.IPAddress)
		}
		result = append(result, ServerInterface{
			PortID:     i.PortID,
			NetworkID:  i.NetID,
			FixedIPs:   ips,
			MACAddress: i.MACAddr,
		})
	}
	return result, nil
}

func (c *computeClient) ListServerVolumes(ctx context.Context, serverID string) ([]ServerVolume, error) {
	allPages, err := volumeattach.List(c.client, serverID).AllPages()
	if err != nil {
		return nil, err
	}
	vols, err := volumeattach.ExtractVolumeAttachments(allPages)
	if err != nil {
		return nil, err
	}
	var result []ServerVolume
	for _, v := range vols {
		result = append(result, ServerVolume{
			ID:       v.ID,
			VolumeID: v.VolumeID,
			Device:   v.Device,
		})
	}
	return result, nil
}

// GetConsoleLog fetches the console output for the given server ID.
// It uses the OpenStack Nova API via gophercloud's ShowConsoleOutput call.
// The `lines` argument maps to the `Length` field of the request options –
// a value of 0 returns the entire log.
func (c *computeClient) GetConsoleLog(id string, lines int) (string, error) {
	opts := servers.ShowConsoleOutputOpts{Length: lines}
	result := servers.ShowConsoleOutput(c.client, id, opts)
	return result.Extract()
}

// GetConsoleURL creates a remote console for the given server and returns its URL.
// Currently it uses a default VNC protocol and NoVNC type, ignoring consoleType.
// This can be extended to map consoleType to appropriate protocol/type.
func (c *computeClient) GetConsoleURL(ctx context.Context, id, consoleType string) (string, error) {
	// Use default protocol and type for now.
	opts := remoteconsoles.CreateOpts{
		Protocol: remoteconsoles.ConsoleProtocolVNC,
		Type:     remoteconsoles.ConsoleTypeNoVNC,
	}
	result := remoteconsoles.Create(c.client, id, opts)
	rc, err := result.Extract()
	if err != nil {
		return "", err
	}
	if rc == nil {
		return "", fmt.Errorf("no remote console returned")
	}
	return rc.URL, nil
}

// ListHypervisors returns a list of hypervisors.
func (c *computeClient) ListHypervisors(ctx context.Context) ([]hypervisors.Hypervisor, error) {
	_ = ctx // ctx currently unused
	allPages, err := hypervisors.List(c.client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return hypervisors.ExtractHypervisors(allPages)
}

// GetHypervisor retrieves details of a specific hypervisor by ID.
func (c *computeClient) GetHypervisor(ctx context.Context, id string) (*hypervisors.Hypervisor, error) {
	_ = ctx // ctx currently unused; gophercloud does not accept context for this call.
	h, err := hypervisors.Get(c.client, id).Extract()
	if err != nil {
		return nil, err
	}
	return h, nil
}

// ListAvailabilityZones returns a list of availability zones.
func (c *computeClient) ListAvailabilityZones(ctx context.Context) ([]availabilityzones.AvailabilityZone, error) {
	_ = ctx // ctx currently unused
	allPages, err := availabilityzones.List(c.client).AllPages()
	if err != nil {
		return nil, err
	}
	return availabilityzones.ExtractAvailabilityZones(allPages)
}
