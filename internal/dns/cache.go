package dns

import (
	"sync"
	"time"
)

// cacheEntry represents a cached DNS entry
type cacheEntry struct {
	ip      string
	expires time.Time
}

// cache is a simple in-memory cache for DNS records
type cache struct {
	entries map[string]*cacheEntry
	ttl     time.Duration
	maxSize int
	mu      sync.RWMutex
}

// newCache creates a new cache
func newCache(ttl time.Duration, maxSize int) *cache {
	return &cache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// get retrieves a value from the cache
func (c *cache) get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return ""
	}

	if time.Now().After(entry.expires) {
		// Entry expired
		return ""
	}

	return entry.ip
}

// set stores a value in the cache
func (c *cache) set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check cache size
	if len(c.entries) >= c.maxSize {
		// Remove oldest entry (simple LRU approximation)
		oldestKey := ""
		oldestTime := time.Now()
		for k, v := range c.entries {
			if v.expires.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.expires
			}
		}
		if oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	c.entries[key] = &cacheEntry{
		ip:      value,
		expires: time.Now().Add(c.ttl),
	}
}
