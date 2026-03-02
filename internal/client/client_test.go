package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"ostui/internal/cache"
)

// newTestClient returns a ServiceClient pointing to a test server that always returns 500.
func newTestClient(t *testing.T) (*gophercloud.ServiceClient, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	provider := &gophercloud.ProviderClient{
		IdentityEndpoint: ts.URL,
		HTTPClient:       *ts.Client(),
	}
	client := &gophercloud.ServiceClient{
		ProviderClient: provider,
		Endpoint:       ts.URL,
	}
	return client, ts
}

// TestComputeClient_ListInstances_Error ensures errors from the underlying service are propagated.
func TestComputeClient_ListInstances_Error(t *testing.T) {
	svc, ts := newTestClient(t)
	defer ts.Close()
	cc := &computeClient{client: svc}
	if _, err := cc.ListInstances(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// TestNetworkClient_ListNetworks_Error ensures errors are propagated.
func TestNetworkClient_ListNetworks_Error(t *testing.T) {
	svc, ts := newTestClient(t)
	defer ts.Close()
	nc := &networkClient{client: svc}
	if _, err := nc.ListNetworks(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// TestStorageClient_ListVolumes_Error ensures errors are propagated.
func TestStorageClient_ListVolumes_Error(t *testing.T) {
	svc, ts := newTestClient(t)
	defer ts.Close()
	sc := &storageClient{client: svc}
	if _, err := sc.ListVolumes(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// TestIdentityClient_ListProjects_Error ensures errors are propagated.
func TestIdentityClient_ListProjects_Error(t *testing.T) {
	svc, ts := newTestClient(t)
	defer ts.Close()
	ic := &identityClient{client: svc}
	if _, err := ic.ListProjects(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCachedComputeClient_ListInstances_CacheHit(t *testing.T) {
	c := cache.NewCache(5 * time.Minute)
	c.Set(cacheResourceCompute, cacheKeyComputeInstances, []servers.Server{{ID: "srv-1", Name: "cached"}})
	svc, ts := newTestClient(t)
	defer ts.Close()
	inner := &computeClient{client: svc}
	cc := NewCachedComputeClient(inner, c)
	list, err := cc.ListInstances()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "srv-1" {
		t.Fatalf("expected cached server, got %+v", list)
	}
}

func TestCachedComputeClient_ListInstances_CacheMiss(t *testing.T) {
	c := cache.NewCache(5 * time.Minute)
	svc, ts := newTestClient(t)
	defer ts.Close()
	inner := &computeClient{client: svc}
	cc := NewCachedComputeClient(inner, c)
	if _, err := cc.ListInstances(); err == nil {
		t.Fatalf("expected error on cache miss with broken inner client")
	}
}

func TestCachedNetworkClient_ListNetworks_CacheHit(t *testing.T) {
	c := cache.NewCache(5 * time.Minute)
	c.Set(cacheResourceNetwork, cacheKeyNetworkNetworks, []networks.Network{{ID: "net-1", Name: "cached"}})
	svc, ts := newTestClient(t)
	defer ts.Close()
	inner := &networkClient{client: svc}
	nc := NewCachedNetworkClient(inner, c)
	list, err := nc.ListNetworks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "net-1" {
		t.Fatalf("expected cached network, got %+v", list)
	}
}

func TestCachedNetworkClient_ListFloatingIPs_CacheHit(t *testing.T) {
	c := cache.NewCache(5 * time.Minute)
	c.Set(cacheResourceNetwork, cacheKeyNetworkFloatingIPs, []floatingips.FloatingIP{{ID: "fip-1", FloatingIP: "1.2.3.4"}})
	svc, ts := newTestClient(t)
	defer ts.Close()
	inner := &networkClient{client: svc}
	nc := NewCachedNetworkClient(inner, c)
	list, err := nc.ListFloatingIPs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "fip-1" {
		t.Fatalf("expected cached FIP, got %+v", list)
	}
}

func TestCachedStorageClient_ListVolumes_CacheHit(t *testing.T) {
	c := cache.NewCache(5 * time.Minute)
	c.Set(cacheResourceStorage, cacheKeyStorageVolumes, []volumes.Volume{{ID: "vol-1", Name: "cached"}})
	svc, ts := newTestClient(t)
	defer ts.Close()
	inner := &storageClient{client: svc}
	sc := NewCachedStorageClient(inner, c)
	list, err := sc.ListVolumes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "vol-1" {
		t.Fatalf("expected cached volume, got %+v", list)
	}
}
