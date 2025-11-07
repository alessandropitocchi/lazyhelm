package helm

import (
	"fmt"
	"sync"
	"time"
)

type CacheEntry struct {
	values    string
	timestamp time.Time
}

type Cache struct {
	entries map[string]CacheEntry
	ttl     time.Duration
	mu      sync.RWMutex
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]CacheEntry),
		ttl:     ttl,
	}
}

func (c *Cache) Get(chartName, version string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.buildKey(chartName, version)
	entry, exists := c.entries[key]

	if !exists {
		return "", false
	}

	if time.Since(entry.timestamp) > c.ttl {
		return "", false
	}

	return entry.values, true
}

func (c *Cache) Set(chartName, version, values string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.buildKey(chartName, version)
	c.entries[key] = CacheEntry{
		values:    values,
		timestamp: time.Now(),
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]CacheEntry)
}

func (c *Cache) buildKey(chartName, version string) string {
	if version == "" {
		return chartName
	}
	return fmt.Sprintf("%s@%s", chartName, version)
}
