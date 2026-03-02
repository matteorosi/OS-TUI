package client

import (
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"ostui/internal/cache"
)

const cacheResourceStorage = "storage"
const cacheKeyStorageVolumes = "list_volumes"

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
	if v, ok := c.cache.Get(cacheResourceStorage, cacheKeyStorageVolumes); ok {
		if list, ok := v.([]volumes.Volume); ok {
			return list, nil
		}
		// Type mismatch: evict and fall through to live fetch.
		c.cache.Delete(cacheResourceStorage, cacheKeyStorageVolumes)
	}
	list, err := c.inner.ListVolumes()
	if err != nil {
		return nil, err
	}
	c.cache.Set(cacheResourceStorage, cacheKeyStorageVolumes, list)
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
