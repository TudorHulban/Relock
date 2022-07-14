package relock

import (
	"fmt"
	"sync"
)

// MemoryCache contains cache state.
type MemoryCache struct {
	cache map[string]any

	mu sync.RWMutex
}

// NewCache is constructor for in memory cache.
func NewCache() *MemoryCache {
	return &MemoryCache{
		cache: make(map[string]any),
	}
}

// Set stores KV item in memory.
func (c *MemoryCache) Set(key string, value any) error {
	c.mu.Lock()

	c.cache[key] = value

	c.mu.Unlock()

	return nil
}

// Get returns the corresponding key value if it finds one.
func (c *MemoryCache) Get(key string) (any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	serializedData, exists := c.cache[key]
	if !exists {
		return nil, fmt.Errorf("no cache entry found for key: `%s`", key)
	}

	return serializedData, nil
}

// Delete deletes a key value by passed key.
func (c *MemoryCache) Delete(key string) error {
	c.mu.Lock()

	delete(c.cache, key)

	c.mu.Unlock()
	return nil
}
