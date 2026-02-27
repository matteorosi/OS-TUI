package client

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	dnsRecordsets "github.com/gophercloud/gophercloud/v2/openstack/dns/v2/recordsets"
	dnsZones "github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
)

// Zone represents a DNS zone in a simplified form used by the application.
type Zone struct {
	ID          string
	Name        string
	Email       string
	Status      string
	TTL         int
	Description string
}

// RecordSet represents a DNS record set (RRset) within a zone.
type RecordSet struct {
	ID      string
	Name    string
	Type    string
	TTL     int
	Status  string
	Records []string
}

// DNSClient defines the methods for interacting with the OpenStack Designate (DNS) service.
type DNSClient interface {
	// ListZones returns all DNS zones visible to the authenticated project.
	ListZones(ctx context.Context) ([]Zone, error)
	// ListRecordSets returns all record sets for a given zone ID.
	ListRecordSets(ctx context.Context, zoneID string) ([]RecordSet, error)
}

// DNSClientImpl is the concrete implementation of DNSClient using gophercloud.
type DNSClientImpl struct {
	client *gophercloud.ServiceClient
}

// NewDNSClient creates a new DNS client given an authenticated provider and endpoint options.
// It mirrors the pattern used in other client implementations but receives a ProviderClient
// directly (instead of AuthOptions) as required by the Designate service.
func NewDNSClient(provider *gophercloud.ProviderClient, opts gophercloud.EndpointOpts) (*DNSClientImpl, error) {
	client, err := openstack.NewDNSV2(provider, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS client: %w", err)
	}
	return &DNSClientImpl{client: client}, nil
}

// ListZones returns all DNS zones visible to the authenticated project.
func (c *DNSClientImpl) ListZones(ctx context.Context) ([]Zone, error) {
	allPages, err := dnsZones.List(c.client, nil).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	gopherZones, err := dnsZones.ExtractZones(allPages)
	if err != nil {
		return nil, err
	}
	zones := make([]Zone, len(gopherZones))
	for i, gz := range gopherZones {
		zones[i] = Zone{
			ID:          gz.ID,
			Name:        gz.Name,
			Email:       gz.Email,
			Status:      gz.Status,
			TTL:         gz.TTL,
			Description: gz.Description,
		}
	}
	return zones, nil
}

// ListRecordSets returns all record sets for the specified zone.
func (c *DNSClientImpl) ListRecordSets(ctx context.Context, zoneID string) ([]RecordSet, error) {
	allPages, err := dnsRecordsets.ListByZone(c.client, zoneID, nil).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	gopherRS, err := dnsRecordsets.ExtractRecordSets(allPages)
	if err != nil {
		return nil, err
	}
	recsets := make([]RecordSet, len(gopherRS))
	for i, rs := range gopherRS {
		recsets[i] = RecordSet{
			ID:      rs.ID,
			Name:    rs.Name,
			Type:    rs.Type,
			TTL:     rs.TTL,
			Status:  rs.Status,
			Records: rs.Records,
		}
	}
	return recsets, nil
}

// Ensure DNSClientImpl implements DNSClient.
var _ DNSClient = (*DNSClientImpl)(nil)
