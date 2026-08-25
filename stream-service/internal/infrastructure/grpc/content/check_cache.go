package content

import (
	"sync"
	"time"
)

type cacheEntry struct {
	allowed bool
	expires time.Time
}

type CheckCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

func NewCheckCache(ttl time.Duration) *CheckCache {
	c := &CheckCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
	go c.cleanup()
	return c
}

func cacheKey(ownerID, songID string) string {
	return ownerID + ":" + songID
}

func (c *CheckCache) Get(ownerID, songID string) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[cacheKey(ownerID, songID)]
	if !ok || time.Now().After(e.expires) {
		return false, false
	}
	return e.allowed, true
}

func (c *CheckCache) Set(ownerID, songID string, allowed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[cacheKey(ownerID, songID)] = cacheEntry{
		allowed: allowed,
		expires: time.Now().Add(c.ttl),
	}
}

func (c *CheckCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, e := range c.entries {
			if now.After(e.expires) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}
