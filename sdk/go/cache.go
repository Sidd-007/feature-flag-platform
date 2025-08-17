package featureflags

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Cache provides an in-memory cache for flag evaluation results
type Cache struct {
	maxSize     int
	ttl         time.Duration
	data        map[string]*CacheEntry
	accessOrder []string // LRU tracking
	mutex       sync.RWMutex
	logger      zerolog.Logger

	// Stats
	hits      int64
	misses    int64
	evictions int64
	expiries  int64

	// Cleanup
	stopCleanup chan struct{}
	cleanupDone chan struct{}
}

// NewCache creates a new cache with the specified maximum size and TTL
func NewCache(maxSize int, ttl time.Duration, logger zerolog.Logger) *Cache {
	cache := &Cache{
		maxSize:     maxSize,
		ttl:         ttl,
		data:        make(map[string]*CacheEntry),
		accessOrder: make([]string, 0, maxSize),
		logger:      logger.With().Str("component", "cache").Logger(),
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	cache.logger.Info().
		Int("max_size", maxSize).
		Dur("ttl", ttl).
		Msg("Cache initialized")

	return cache
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.data[key]
	if !exists {
		c.misses++
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		delete(c.data, key)
		c.removeFromAccessOrder(key)
		c.expiries++
		c.misses++
		return nil, false
	}

	// Update access info
	entry.AccessedAt = time.Now()
	entry.AccessCount++

	// Move to front of access order (most recently used)
	c.moveToFront(key)

	c.hits++
	return entry.Value, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	// Check if key already exists
	if existing, exists := c.data[key]; exists {
		// Update existing entry
		existing.Value = value
		existing.ExpiresAt = now.Add(c.ttl)
		existing.AccessedAt = now
		existing.AccessCount++
		c.moveToFront(key)
		return
	}

	// Check if we need to evict entries
	if len(c.data) >= c.maxSize {
		c.evictLRU()
	}

	// Create new entry
	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		ExpiresAt:   now.Add(c.ttl),
		CreatedAt:   now,
		AccessedAt:  now,
		AccessCount: 1,
	}

	c.data[key] = entry
	c.accessOrder = append([]string{key}, c.accessOrder...)

	c.logger.Debug().
		Str("key", key).
		Time("expires_at", entry.ExpiresAt).
		Msg("Cache entry stored")
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.data[key]; exists {
		delete(c.data, key)
		c.removeFromAccessOrder(key)

		c.logger.Debug().
			Str("key", key).
			Msg("Cache entry deleted")
	}
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]*CacheEntry)
	c.accessOrder = c.accessOrder[:0]

	c.logger.Info().Msg("Cache cleared")
}

// Size returns the current number of entries in the cache
func (c *Cache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.data)
}

// Stats returns cache statistics
func (c *Cache) Stats() *CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	total := c.hits + c.misses
	hitRate := 0.0
	missRate := 0.0

	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
		missRate = float64(c.misses) / float64(total)
	}

	return &CacheStats{
		Size:      len(c.data),
		MaxSize:   c.maxSize,
		HitRate:   hitRate,
		MissRate:  missRate,
		Hits:      c.hits,
		Misses:    c.misses,
		Evictions: c.evictions,
		Expiries:  c.expiries,
	}
}

// GetEntry returns the cache entry for a key (for debugging)
func (c *Cache) GetEntry(key string) (*CacheEntry, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid concurrent access
	entryCopy := *entry
	return &entryCopy, true
}

// Keys returns all cache keys
func (c *Cache) Keys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]string, 0, len(c.data))
	for key := range c.data {
		keys = append(keys, key)
	}
	return keys
}

// Close closes the cache and stops background cleanup
func (c *Cache) Close() {
	close(c.stopCleanup)
	<-c.cleanupDone

	c.logger.Info().
		Int64("hits", c.hits).
		Int64("misses", c.misses).
		Int64("evictions", c.evictions).
		Int64("expiries", c.expiries).
		Msg("Cache closed")
}

// Private methods

// evictLRU evicts the least recently used entry
func (c *Cache) evictLRU() {
	if len(c.accessOrder) == 0 {
		return
	}

	// Remove least recently used (last in order)
	lruKey := c.accessOrder[len(c.accessOrder)-1]
	delete(c.data, lruKey)
	c.accessOrder = c.accessOrder[:len(c.accessOrder)-1]
	c.evictions++

	c.logger.Debug().
		Str("key", lruKey).
		Msg("Cache entry evicted (LRU)")
}

// moveToFront moves a key to the front of the access order
func (c *Cache) moveToFront(key string) {
	// Find and remove the key from its current position
	for i, k := range c.accessOrder {
		if k == key {
			// Remove from current position
			c.accessOrder = append(c.accessOrder[:i], c.accessOrder[i+1:]...)
			break
		}
	}

	// Add to front
	c.accessOrder = append([]string{key}, c.accessOrder...)
}

// removeFromAccessOrder removes a key from the access order
func (c *Cache) removeFromAccessOrder(key string) {
	for i, k := range c.accessOrder {
		if k == key {
			c.accessOrder = append(c.accessOrder[:i], c.accessOrder[i+1:]...)
			break
		}
	}
}

// cleanupExpired periodically removes expired entries
func (c *Cache) cleanupExpired() {
	defer close(c.cleanupDone)

	ticker := time.NewTicker(time.Minute) // Clean up every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.performCleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// performCleanup removes expired entries
func (c *Cache) performCleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	var expiredKeys []string

	// Find expired entries
	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Remove expired entries
	for _, key := range expiredKeys {
		delete(c.data, key)
		c.removeFromAccessOrder(key)
		c.expiries++
	}

	if len(expiredKeys) > 0 {
		c.logger.Debug().
			Int("expired_count", len(expiredKeys)).
			Msg("Cleaned up expired cache entries")
	}
}

// Warmup pre-populates the cache with commonly used flags
func (c *Cache) Warmup(entries map[string]interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	for key, value := range entries {
		if len(c.data) >= c.maxSize {
			break
		}

		entry := &CacheEntry{
			Key:         key,
			Value:       value,
			ExpiresAt:   now.Add(c.ttl),
			CreatedAt:   now,
			AccessedAt:  now,
			AccessCount: 0,
		}

		c.data[key] = entry
		c.accessOrder = append(c.accessOrder, key)
	}

	c.logger.Info().
		Int("entries_added", len(entries)).
		Msg("Cache warmed up")
}

// SetTTL updates the TTL for the cache (affects new entries)
func (c *Cache) SetTTL(ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.ttl = ttl

	c.logger.Info().
		Dur("new_ttl", ttl).
		Msg("Cache TTL updated")
}

// GetTTL returns the current TTL setting
func (c *Cache) GetTTL() time.Duration {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.ttl
}

// Refresh updates the expiry time for a cache entry
func (c *Cache) Refresh(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.data[key]
	if !exists {
		return false
	}

	entry.ExpiresAt = time.Now().Add(c.ttl)
	entry.AccessedAt = time.Now()
	c.moveToFront(key)

	c.logger.Debug().
		Str("key", key).
		Time("new_expires_at", entry.ExpiresAt).
		Msg("Cache entry refreshed")

	return true
}

// Contains checks if a key exists in the cache (without updating access time)
func (c *Cache) Contains(key string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return false
	}

	// Check if expired
	return time.Now().Before(entry.ExpiresAt)
}

// GetMultiple retrieves multiple values from the cache
func (c *Cache) GetMultiple(keys []string) map[string]interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	results := make(map[string]interface{})
	now := time.Now()

	for _, key := range keys {
		entry, exists := c.data[key]
		if !exists {
			c.misses++
			continue
		}

		// Check if expired
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
			c.removeFromAccessOrder(key)
			c.expiries++
			c.misses++
			continue
		}

		// Update access info
		entry.AccessedAt = now
		entry.AccessCount++
		c.moveToFront(key)

		results[key] = entry.Value
		c.hits++
	}

	return results
}

// SetMultiple stores multiple values in the cache
func (c *Cache) SetMultiple(entries map[string]interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	for key, value := range entries {
		// Check if key already exists
		if existing, exists := c.data[key]; exists {
			// Update existing entry
			existing.Value = value
			existing.ExpiresAt = now.Add(c.ttl)
			existing.AccessedAt = now
			existing.AccessCount++
			c.moveToFront(key)
			continue
		}

		// Check if we need to evict entries
		if len(c.data) >= c.maxSize {
			c.evictLRU()
		}

		// Create new entry
		entry := &CacheEntry{
			Key:         key,
			Value:       value,
			ExpiresAt:   now.Add(c.ttl),
			CreatedAt:   now,
			AccessedAt:  now,
			AccessCount: 1,
		}

		c.data[key] = entry
		c.accessOrder = append([]string{key}, c.accessOrder...)
	}

	c.logger.Debug().
		Int("entries_set", len(entries)).
		Msg("Multiple cache entries stored")
}
