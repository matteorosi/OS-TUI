# Predictive Prefetching Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wrap OpenStack clients with a transparent caching layer and add cross-context
predictive prefetching so that navigating between related views hits the cache instead of
the API.

**Architecture:** Three new `Cached*Client` structs in `internal/client/` implement the
same interfaces as the raw clients, checking `*cache.Cache` on list operations before
calling the API. A shared `*cache.Cache` lives in `AppModel`; when the user opens any
section, background goroutines pre-warm related sections.

**Tech Stack:** Go, `internal/cache` (existing), `golang.org/x/sync/errgroup` (already
in go.mod), bubbletea AppModel.

---

## Task 1: CachedComputeClient

**Files:**
- Create: `internal/client/cached_compute.go`
- Modify: `internal/client/client_test.go`

### Step 1: Write the failing test

Add to `internal/client/client_test.go`:

```go
func TestCachedComputeClient_ListInstances_CacheHit(t *testing.T) {
    c := cache.NewCache(5 * time.Minute)
    // Seed the cache with a known value.
    c.Set("compute", "list_instances", []servers.Server{{ID: "srv-1", Name: "cached"}})
    // Inner client always errors – if cache works, it should never be called.
    svc, ts := newTestClient(t)
    defer ts.Close()
    inner := &computeClient{client: svc}
    cc := client.NewCachedComputeClient(inner, c)
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
    cc := client.NewCachedComputeClient(inner, c)
    // Inner always errors (500), so cache miss → error propagated.
    if _, err := cc.ListInstances(); err == nil {
        t.Fatalf("expected error on cache miss with broken inner client")
    }
}
```

**Note:** you'll need to add imports:
```go
import (
    "ostui/internal/cache"
    "github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
    "time"
)
```

### Step 2: Run tests to verify they fail

```bash
cd ~/Work/ostui && go test ./internal/client/... -run TestCachedCompute -v
```
Expected: compile error — `NewCachedComputeClient` not defined yet.

### Step 3: Implement `internal/client/cached_compute.go`

```go
package client

import (
    "ostui/internal/cache"
)

const cacheKeyComputeInstances = "list_instances"
const cacheResourceCompute = "compute"

// CachedComputeClient wraps a ComputeClient and caches list operations.
type CachedComputeClient struct {
    inner ComputeClient
    cache *cache.Cache
}

// NewCachedComputeClient returns a ComputeClient backed by cache.
func NewCachedComputeClient(inner ComputeClient, c *cache.Cache) ComputeClient {
    return &CachedComputeClient{inner: inner, cache: c}
}

func (c *CachedComputeClient) ListInstances() ([]servers.Server, error) {
    if v, ok := c.cache.Get(cacheResourceCompute, cacheKeyComputeInstances); ok {
        return v.([]servers.Server), nil
    }
    list, err := c.inner.ListInstances()
    if err != nil {
        return nil, err
    }
    c.cache.Set(cacheResourceCompute, cacheKeyComputeInstances, list)
    return list, nil
}

// All other methods delegate directly – no caching needed for single-item fetches
// or mutating operations.
func (c *CachedComputeClient) GetInstance(id string) (servers.Server, error) {
    return c.inner.GetInstance(id)
}
func (c *CachedComputeClient) StartInstance(id string) error {
    return c.inner.StartInstance(id)
}
func (c *CachedComputeClient) StopInstance(id string) error {
    return c.inner.StopInstance(id)
}
func (c *CachedComputeClient) DeleteInstance(id string) error {
    return c.inner.DeleteInstance(id)
}
func (c *CachedComputeClient) RebootInstance(id string) error {
    return c.inner.RebootInstance(id)
}
func (c *CachedComputeClient) ListFlavors() ([]flavors.Flavor, error) {
    return c.inner.ListFlavors()
}
func (c *CachedComputeClient) ListKeypairs() ([]keypairs.KeyPair, error) {
    return c.inner.ListKeypairs()
}
func (c *CachedComputeClient) GetConsoleLog(id string, lines int) (string, error) {
    return c.inner.GetConsoleLog(id, lines)
}
func (c *CachedComputeClient) GetConsoleURL(ctx context.Context, id, consoleType string) (string, error) {
    return c.inner.GetConsoleURL(ctx, id, consoleType)
}
func (c *CachedComputeClient) ListHypervisors(ctx context.Context) ([]hypervisors.Hypervisor, error) {
    return c.inner.ListHypervisors(ctx)
}
func (c *CachedComputeClient) GetHypervisor(ctx context.Context, id string) (*hypervisors.Hypervisor, error) {
    return c.inner.GetHypervisor(ctx, id)
}
func (c *CachedComputeClient) ListAvailabilityZones(ctx context.Context) ([]availabilityzones.AvailabilityZone, error) {
    return c.inner.ListAvailabilityZones(ctx)
}
func (c *CachedComputeClient) GetFlavor(ctx context.Context, flavorID string) (flavors.Flavor, error) {
    return c.inner.GetFlavor(ctx, flavorID)
}
func (c *CachedComputeClient) GetKeypair(ctx context.Context, name string) (keypairs.KeyPair, error) {
    return c.inner.GetKeypair(ctx, name)
}
func (c *CachedComputeClient) ListServerInterfaces(ctx context.Context, serverID string) ([]ServerInterface, error) {
    return c.inner.ListServerInterfaces(ctx, serverID)
}
func (c *CachedComputeClient) ListServerVolumes(ctx context.Context, serverID string) ([]ServerVolume, error) {
    return c.inner.ListServerVolumes(ctx, serverID)
}
```

Add necessary imports at the top:
```go
import (
    "context"
    "github.com/charmbracelet/bubbles/spinner" // remove if not needed
    "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/attachinterfaces"
    "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
    "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
    "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
    "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
    "github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
    "ostui/internal/cache"
)
```

### Step 4: Run tests

```bash
cd ~/Work/ostui && go test ./internal/client/... -run TestCachedCompute -v
```
Expected: PASS both tests.

### Step 5: Compile check

```bash
go build ./...
```
Expected: no errors.

### Step 6: Commit

```bash
git add internal/client/cached_compute.go internal/client/client_test.go
git commit -m "feat: add CachedComputeClient wrapping compute list operations"
```

---

## Task 2: CachedNetworkClient

**Files:**
- Create: `internal/client/cached_network.go`
- Modify: `internal/client/client_test.go`

### Step 1: Write the failing test

Add to `internal/client/client_test.go`:

```go
func TestCachedNetworkClient_ListNetworks_CacheHit(t *testing.T) {
    c := cache.NewCache(5 * time.Minute)
    c.Set("network", "list_networks", []networks.Network{{ID: "net-1", Name: "cached"}})
    svc, ts := newTestClient(t)
    defer ts.Close()
    inner := &networkClient{client: svc}
    nc := client.NewCachedNetworkClient(inner, c)
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
    c.Set("network", "list_floating_ips", []floatingips.FloatingIP{{ID: "fip-1", FloatingIP: "1.2.3.4"}})
    svc, ts := newTestClient(t)
    defer ts.Close()
    inner := &networkClient{client: svc}
    nc := client.NewCachedNetworkClient(inner, c)
    list, err := nc.ListFloatingIPs()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(list) != 1 || list[0].ID != "fip-1" {
        t.Fatalf("expected cached FIP, got %+v", list)
    }
}
```

Add imports: `networks`, `floatingips` gophercloud packages.

### Step 2: Run tests to verify they fail

```bash
go test ./internal/client/... -run TestCachedNetwork -v
```
Expected: compile error.

### Step 3: Implement `internal/client/cached_network.go`

```go
package client

import (
    "context"

    "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
    "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
    "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
    "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
    "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
    "github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
    "github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
    "ostui/internal/cache"
)

const cacheResourceNetwork = "network"

// CachedNetworkClient wraps a NetworkClient and caches list operations.
type CachedNetworkClient struct {
    inner NetworkClient
    cache *cache.Cache
}

// NewCachedNetworkClient returns a NetworkClient backed by cache.
func NewCachedNetworkClient(inner NetworkClient, c *cache.Cache) NetworkClient {
    return &CachedNetworkClient{inner: inner, cache: c}
}

func (c *CachedNetworkClient) ListNetworks() ([]networks.Network, error) {
    if v, ok := c.cache.Get(cacheResourceNetwork, "list_networks"); ok {
        return v.([]networks.Network), nil
    }
    list, err := c.inner.ListNetworks()
    if err != nil {
        return nil, err
    }
    c.cache.Set(cacheResourceNetwork, "list_networks", list)
    return list, nil
}

func (c *CachedNetworkClient) ListSubnets() ([]subnets.Subnet, error) {
    if v, ok := c.cache.Get(cacheResourceNetwork, "list_subnets"); ok {
        return v.([]subnets.Subnet), nil
    }
    list, err := c.inner.ListSubnets()
    if err != nil {
        return nil, err
    }
    c.cache.Set(cacheResourceNetwork, "list_subnets", list)
    return list, nil
}

func (c *CachedNetworkClient) ListFloatingIPs() ([]floatingips.FloatingIP, error) {
    if v, ok := c.cache.Get(cacheResourceNetwork, "list_floating_ips"); ok {
        return v.([]floatingips.FloatingIP), nil
    }
    list, err := c.inner.ListFloatingIPs()
    if err != nil {
        return nil, err
    }
    c.cache.Set(cacheResourceNetwork, "list_floating_ips", list)
    return list, nil
}

func (c *CachedNetworkClient) ListRouters(ctx context.Context) ([]Router, error) {
    if v, ok := c.cache.Get(cacheResourceNetwork, "list_routers"); ok {
        return v.([]Router), nil
    }
    list, err := c.inner.ListRouters(ctx)
    if err != nil {
        return nil, err
    }
    c.cache.Set(cacheResourceNetwork, "list_routers", list)
    return list, nil
}

func (c *CachedNetworkClient) ListSecurityGroups() ([]groups.SecGroup, error) {
    if v, ok := c.cache.Get(cacheResourceNetwork, "list_security_groups"); ok {
        return v.([]groups.SecGroup), nil
    }
    list, err := c.inner.ListSecurityGroups()
    if err != nil {
        return nil, err
    }
    c.cache.Set(cacheResourceNetwork, "list_security_groups", list)
    return list, nil
}

// Remaining methods — delegate directly, no caching.
func (c *CachedNetworkClient) GetSubnet(ctx context.Context, id string) (*subnets.Subnet, error) {
    return c.inner.GetSubnet(ctx, id)
}
func (c *CachedNetworkClient) AllocateFloatingIP(opts floatingips.CreateOptsBuilder) (floatingips.FloatingIP, error) {
    return c.inner.AllocateFloatingIP(opts)
}
func (c *CachedNetworkClient) ReleaseFloatingIP(id string) error {
    return c.inner.ReleaseFloatingIP(id)
}
func (c *CachedNetworkClient) AssociateFloatingIP(fipID, portID string) (floatingips.FloatingIP, error) {
    return c.inner.AssociateFloatingIP(fipID, portID)
}
func (c *CachedNetworkClient) DisassociateFloatingIP(fipID string) (floatingips.FloatingIP, error) {
    return c.inner.DisassociateFloatingIP(fipID)
}
func (c *CachedNetworkClient) GetRouter(ctx context.Context, id string) (*Router, error) {
    return c.inner.GetRouter(ctx, id)
}
func (c *CachedNetworkClient) GetRouterInterfaces(ctx context.Context, id string) ([]RouterInterface, error) {
    return c.inner.GetRouterInterfaces(ctx, id)
}
func (c *CachedNetworkClient) CreateRouter(ctx context.Context, name, externalNetID string) (*Router, error) {
    return c.inner.CreateRouter(ctx, name, externalNetID)
}
func (c *CachedNetworkClient) DeleteRouter(ctx context.Context, id string) error {
    return c.inner.DeleteRouter(ctx, id)
}
func (c *CachedNetworkClient) DeleteNetwork(id string) error {
    return c.inner.DeleteNetwork(id)
}
func (c *CachedNetworkClient) AddRouterInterface(ctx context.Context, routerID, subnetID string) error {
    return c.inner.AddRouterInterface(ctx, routerID, subnetID)
}
func (c *CachedNetworkClient) RemoveRouterInterface(ctx context.Context, routerID, subnetID string) error {
    return c.inner.RemoveRouterInterface(ctx, routerID, subnetID)
}
func (c *CachedNetworkClient) ListPorts(ctx context.Context) ([]Port, error) {
    return c.inner.ListPorts(ctx)
}
func (c *CachedNetworkClient) GetPort(ctx context.Context, id string) (*Port, error) {
    return c.inner.GetPort(ctx, id)
}
func (c *CachedNetworkClient) ListPortsByServer(ctx context.Context, serverID string) ([]Port, error) {
    return c.inner.ListPortsByServer(ctx, serverID)
}
func (c *CachedNetworkClient) ListPortsByNetwork(ctx context.Context, networkID string) ([]Port, error) {
    return c.inner.ListPortsByNetwork(ctx, networkID)
}
func (c *CachedNetworkClient) GetNetwork(ctx context.Context, id string) (*networks.Network, error) {
    return c.inner.GetNetwork(ctx, id)
}
func (c *CachedNetworkClient) ListSecurityGroupRules(ctx context.Context, sgID string) ([]SecurityGroupRule, error) {
    return c.inner.ListSecurityGroupRules(ctx, sgID)
}
func (c *CachedNetworkClient) CreateSecurityGroupRule(ctx context.Context, sgID string, rule SecurityGroupRuleInput) (*SecurityGroupRule, error) {
    return c.inner.CreateSecurityGroupRule(ctx, sgID, rule)
}
func (c *CachedNetworkClient) DeleteSecurityGroupRule(ctx context.Context, id string) error {
    return c.inner.DeleteSecurityGroupRule(ctx, id)
}

// compile-time check
var _ NetworkClient = (*CachedNetworkClient)(nil)
```

### Step 4: Run tests

```bash
go test ./internal/client/... -run TestCachedNetwork -v
```
Expected: PASS.

### Step 5: Build check

```bash
go build ./...
```

### Step 6: Commit

```bash
git add internal/client/cached_network.go internal/client/client_test.go
git commit -m "feat: add CachedNetworkClient wrapping network list operations"
```

---

## Task 3: CachedStorageClient

**Files:**
- Create: `internal/client/cached_storage.go`
- Modify: `internal/client/client_test.go`

### Step 1: Write the failing test

Add to `internal/client/client_test.go`:

```go
func TestCachedStorageClient_ListVolumes_CacheHit(t *testing.T) {
    c := cache.NewCache(5 * time.Minute)
    c.Set("storage", "list_volumes", []volumes.Volume{{ID: "vol-1", Name: "cached"}})
    svc, ts := newTestClient(t)
    defer ts.Close()
    inner := &storageClient{client: svc}
    sc := client.NewCachedStorageClient(inner, c)
    list, err := sc.ListVolumes()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(list) != 1 || list[0].ID != "vol-1" {
        t.Fatalf("expected cached volume, got %+v", list)
    }
}
```

Add import: `"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"`

### Step 2: Run to verify failure

```bash
go test ./internal/client/... -run TestCachedStorage -v
```
Expected: compile error.

### Step 3: Implement `internal/client/cached_storage.go`

```go
package client

import (
    "github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumeactions"
    "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
    "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
    "ostui/internal/cache"
)

const cacheResourceStorage = "storage"

// CachedStorageClient wraps a StorageClient and caches list operations.
type CachedStorageClient struct {
    inner StorageClient
    cache *cache.Cache
}

// NewCachedStorageClient returns a StorageClient backed by cache.
func NewCachedStorageClient(inner StorageClient, c *cache.Cache) StorageClient {
    return &CachedStorageClient{inner: inner, cache: c}
}

func (c *CachedStorageClient) ListVolumes() ([]volumes.Volume, error) {
    if v, ok := c.cache.Get(cacheResourceStorage, "list_volumes"); ok {
        return v.([]volumes.Volume), nil
    }
    list, err := c.inner.ListVolumes()
    if err != nil {
        return nil, err
    }
    c.cache.Set(cacheResourceStorage, "list_volumes", list)
    return list, nil
}

func (c *CachedStorageClient) GetVolume(id string) (volumes.Volume, error) {
    return c.inner.GetVolume(id)
}
func (c *CachedStorageClient) DeleteVolume(id string) error {
    return c.inner.DeleteVolume(id)
}
func (c *CachedStorageClient) ListSnapshots() ([]snapshots.Snapshot, error) {
    return c.inner.ListSnapshots()
}
func (c *CachedStorageClient) CreateSnapshot(opts snapshots.CreateOptsBuilder) (snapshots.Snapshot, error) {
    return c.inner.CreateSnapshot(opts)
}
func (c *CachedStorageClient) ExtendVolume(id string, newSizeGB int) error {
    return c.inner.ExtendVolume(id, newSizeGB)
}

var _ StorageClient = (*CachedStorageClient)(nil)
```

### Step 4: Run tests

```bash
go test ./internal/client/... -run TestCachedStorage -v
```
Expected: PASS.

### Step 5: Build check

```bash
go build ./...
```

### Step 6: Commit

```bash
git add internal/client/cached_storage.go internal/client/client_test.go
git commit -m "feat: add CachedStorageClient wrapping storage list operations"
```

---

## Task 4: Wire cache into AppModel + add prefetch goroutines

**Files:**
- Modify: `internal/ui/app.go`

### Step 1: Add `cache` field to `AppModel`

In `AppModel` struct (around line 75), add:
```go
cache *appcache.Cache
```

Add import at the top of `app.go`:
```go
"ostui/internal/cache" // import as appcache to avoid collision with local vars
```

Alias in import block:
```go
appcache "ostui/internal/cache"
```

### Step 2: Wrap raw clients in `NewModel`

`NewModel` currently ends with (line ~202):
```go
return AppModel{provider: provider, cloudName: cloudName, computeClient: compute, ...}
```

Replace with:
```go
c := appcache.NewCache(5 * time.Minute)
return AppModel{
    provider:       provider,
    cloudName:      cloudName,
    computeClient:  client.NewCachedComputeClient(compute, c),
    networkClient:  client.NewCachedNetworkClient(network, c),
    storageClient:  client.NewCachedStorageClient(storage, c),
    identityClient: identity,
    imageClient:    image,
    limitsClient:   limits,
    dnsClient:      dns,
    lbClient:       lb,
    cache:          c,
    sidebar:        l,
    state:          stateSidebar,
    prevState:      "",
    commandBar:     cmdBar,
    commandMap:     cmdMap,
}
```

Add `"time"` to imports if not already present.

### Step 3: Add `prefetchMap` and `triggerPrefetch` helper

Add after the `navigateTo` function (around line 250):

```go
// prefetchMap defines which resources to warm when a section is opened.
var prefetchMap = map[string][]string{
    "Servers":      {"Networks", "Volumes", "Floating IPs"},
    "Networks":     {"Subnets", "Routers", "Floating IPs"},
    "Floating IPs": {"Networks"},
    "Routers":      {"Networks", "Subnets"},
    "Subnets":      {"Networks"},
}

// triggerPrefetch fires background goroutines to warm the cache for sections
// related to the one just opened. Safe to call from Update(); goroutines are
// fire-and-forget — they write into the thread-safe cache.
func (m AppModel) triggerPrefetch(section string) {
    targets, ok := prefetchMap[section]
    if !ok {
        return
    }
    for _, t := range targets {
        target := t // capture loop var
        go func() {
            switch target {
            case "Networks":
                _, _ = m.networkClient.ListNetworks()
            case "Subnets":
                _, _ = m.networkClient.ListSubnets()
            case "Floating IPs":
                _, _ = m.networkClient.ListFloatingIPs()
            case "Routers":
                _, _ = m.networkClient.ListRouters(context.Background())
            case "Volumes":
                _, _ = m.storageClient.ListVolumes()
            }
        }()
    }
}
```

Add `"context"` to imports if not already present.

### Step 4: Call `triggerPrefetch` when a section is opened

In the `case "enter":` handler (around line 487), after the `stateMain` block where
`m.mainModel = constructor()` is called, add `m.triggerPrefetch(i.title)`:

```go
if m.mainModel != nil {
    m.triggerPrefetch(i.title)   // ← add this line
    return m, m.mainModel.Init()
}
```

### Step 5: Build and run all tests

```bash
cd ~/Work/ostui && go build ./... && go test ./...
```
Expected: all pass.

### Step 6: Commit

```bash
git add internal/ui/app.go
git commit -m "feat: wire cache into AppModel and add cross-context prefetch goroutines"
```

---

## Task 5: Invalidate cache after mutating actions

**Files:**
- Modify: `internal/ui/app.go` — `executeAction` function (line ~1330)

### Step 1: Add invalidation calls

After each successful mutating operation in `executeAction`, invalidate the relevant
cache key so the next list call goes to the API.

Replace the `return func() tea.Msg {` body with:

```go
return func() tea.Msg {
    var err error
    switch resource {
    case "instance":
        switch pending.Key {
        case "start":
            err = m.computeClient.StartInstance(target)
        case "stop":
            err = m.computeClient.StopInstance(target)
        case "reboot":
            err = m.computeClient.RebootInstance(target)
        case "delete":
            err = m.computeClient.DeleteInstance(target)
        }
        if err == nil {
            m.cache.Delete("compute", "list_instances")
        }
    case "volume":
        switch pending.Key {
        case "delete":
            err = m.storageClient.DeleteVolume(target)
        case "extend":
            size, convErr := strconv.Atoi(input)
            if convErr != nil {
                err = fmt.Errorf("invalid size: %v", convErr)
            } else {
                err = m.storageClient.ExtendVolume(target, size)
            }
        }
        if err == nil {
            m.cache.Delete("storage", "list_volumes")
        }
    case "network":
        if pending.Key == "delete" {
            err = m.networkClient.DeleteNetwork(target)
        }
        if err == nil {
            m.cache.Delete("network", "list_networks")
        }
    case "floatingip":
        switch pending.Key {
        case "associate":
            _, err = m.networkClient.AssociateFloatingIP(target, input)
        case "disassociate":
            _, err = m.networkClient.DisassociateFloatingIP(target)
        case "delete":
            err = m.networkClient.ReleaseFloatingIP(target)
        }
        if err == nil {
            m.cache.Delete("network", "list_floating_ips")
        }
    default:
        err = fmt.Errorf("unsupported resource %s for action %s", resource, pending.Key)
    }
    if err != nil {
        return actionResultMsg{success: false, message: err.Error()}
    }
    return actionResultMsg{success: true, message: fmt.Sprintf("%s %s succeeded", pending.Label, target)}
}
```

### Step 2: Build and run all tests

```bash
go build ./... && go test ./...
```
Expected: all pass.

### Step 3: Commit

```bash
git add internal/ui/app.go
git commit -m "feat: invalidate cache after mutating actions (delete/stop/reboot/extend)"
```

---

## Verification

```bash
go build ./... && go test ./...
```

All tests green, binary builds cleanly. The cache is transparent — no model files were
modified.
