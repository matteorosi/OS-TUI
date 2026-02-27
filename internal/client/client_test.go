package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophercloud/gophercloud"
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
