package cache

import (
	"sync"
	"time"
)

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

type Cache struct {
	mu         sync.RWMutex
	items      map[string]cacheEntry
	ttl        map[string]time.Duration
	defaultTTL time.Duration
}

// NewCache creates a new Cache with the given default TTL (or 5m if zero).
func NewCache(defaultTTL time.Duration) *Cache {
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}
	return &Cache{
		items:      make(map[string]cacheEntry),
		ttl:        make(map[string]time.Duration),
		defaultTTL: defaultTTL,
	}
}

// SetTTL sets a custom TTL for a resource type.
func (c *Cache) SetTTL(resource string, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	c.mu.Lock()
	c.ttl[resource] = ttl
	c.mu.Unlock()
}

// getTTL returns TTL for resource or default.
func (c *Cache) getTTL(resource string) time.Duration {
	c.mu.RLock()
	ttl, ok := c.ttl[resource]
	c.mu.RUnlock()
	if ok {
		return ttl
	}
	return c.defaultTTL
}

// makeKey builds map key.
func (c *Cache) makeKey(resource, key string) string {
	return resource + ":" + key
}

// Set stores a value with expiration.
func (c *Cache) Set(resource, key string, value interface{}) {
	expires := time.Now().Add(c.getTTL(resource))
	c.mu.Lock()
	c.items[c.makeKey(resource, key)] = cacheEntry{value: value, expiresAt: expires}
	c.mu.Unlock()
}

// Get retrieves a value if present and not expired.
func (c *Cache) Get(resource, key string) (interface{}, bool) {
	c.mu.RLock()
	entry, ok := c.items[c.makeKey(resource, key)]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.items, c.makeKey(resource, key))
		c.mu.Unlock()
		return nil, false
	}
	return entry.value, true
}

// Delete removes an entry.
func (c *Cache) Delete(resource, key string) {
	c.mu.Lock()
	delete(c.items, c.makeKey(resource, key))
	c.mu.Unlock()
}

// Clear removes all entries.
func (c *Cache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]cacheEntry)
	c.mu.Unlock()
}

// PurgeExpired removes all expired entries.
func (c *Cache) PurgeExpired() {
	now := time.Now()
	c.mu.Lock()
	for k, e := range c.items {
		if now.After(e.expiresAt) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

// Ensure package compiles even if unused.
var _ = time.Now
