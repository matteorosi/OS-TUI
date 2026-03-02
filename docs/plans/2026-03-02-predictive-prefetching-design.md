# Predictive Prefetching — Design

Date: 2026-03-02

## Problem

All views load data on-demand via `tea.Cmd`. On remote OpenStack environments each API
call costs 200ms–2s. When the user opens a server's graph view, 3–4 sequential fetches
produce 1–4s of spinner time.

An existing `internal/cache/cache.go` TTL cache is defined but never used.

## Approach: Cached Client Wrappers (drop-in)

A single `*cache.Cache` lives in `AppModel`. The raw OpenStack clients are wrapped in
thin cached versions that implement the same interfaces — zero changes to existing model
files.

## Architecture

```
AppModel
├── cache       *cache.Cache          (5 min default TTL, shared)
├── computeClient  CachedComputeClient   (wraps raw computeClient)
├── networkClient  CachedNetworkClient   (wraps raw networkClient)
└── storageClient  CachedStorageClient   (wraps raw storageClient)
```

### New files

| File | Responsibility |
|------|---------------|
| `internal/client/cached_compute.go` | Wraps `ComputeClient`, caches `ListInstances` |
| `internal/client/cached_network.go` | Wraps `NetworkClient`, caches `ListNetworks`, `ListSubnets`, `ListFloatingIPs`, `ListRouters`, `ListSecurityGroups` |
| `internal/client/cached_storage.go` | Wraps `StorageClient`, caches `ListVolumes` |

All other methods (Get*, Create*, Delete*, …) are forwarded as-is.

### Cache keys

```
compute:list_instances
network:list_networks
network:list_subnets
network:list_floating_ips
network:list_routers
network:list_security_groups
storage:list_volumes
```

## Prefetch Map (cross-context)

When the user navigates to a section, `app.go` fires background goroutines for related
sections. The goroutines call the cached client list methods — on cache hit they return
immediately, on miss they populate the cache.

| Section opened | Prefetch in background |
|----------------|----------------------|
| Servers | Networks, Volumes, Floating IPs |
| Networks | Subnets, Routers, Floating IPs |
| Floating IPs | Networks |
| Routers | Networks, Subnets |
| Subnets | Networks |
| Volumes | — |

## Data Flow

1. User selects "Servers" in sidebar
2. `CachedComputeClient.ListInstances()` → cache miss → API call → stores result → returns list
3. `app.go` fires 3 goroutines: `ListNetworks`, `ListVolumes`, `ListFloatingIPs` (fire-and-forget)
4. User enters server detail → graph calls `ListFloatingIPs()` → **cache hit → 0ms**

## Cache Invalidation

After any mutating action (delete, create, stop, reboot, extend), `cache.Delete()` is
called on the relevant key before the list is reloaded. This ensures stale data is never
shown after a user-initiated change.

## TTL

5 minutes (default of `cache.Cache`). Adequate for read-heavy TUI sessions; resource
state rarely changes faster than this during normal navigation.

## Files Modified

- `internal/client/cached_compute.go` — new
- `internal/client/cached_network.go` — new
- `internal/client/cached_storage.go` — new
- `internal/ui/app.go` — create cache, wrap clients, add prefetch goroutines, invalidate on actions
- `internal/cache/cache.go` — no changes
