// Copyright 2025 Alessandro Pitocchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
