package storage

import (
	"encoding/binary"
	"hash/fnv"
	"sync"
	"time"

	"github.com/VictoriaMetrics/fastcache"
)

const (
	defaultCacheBytes = 64 * 1024 * 1024 // 64 MB default
	numShards         = 256
)

// MemoryCache provides zero-GC in-memory caching with TTL and idempotency protection.
type MemoryCache struct {
	cache   *fastcache.Cache
	shards  [numShards]sync.RWMutex
	maxSize int
}

// CacheConfig configures the MemoryCache.
type CacheConfig struct {
	MaxBytes int
}

// NewMemoryCache creates a new fastcache-backed memory cache.
func NewMemoryCache(cfg CacheConfig) *MemoryCache {
	size := cfg.MaxBytes
	if size <= 0 {
		size = defaultCacheBytes
	}
	return &MemoryCache{
		cache:   fastcache.New(size),
		maxSize: size,
	}
}

// getShardIndex computes shard index for a key to serialize mutations per key.
func getShardIndex(key string) uint8 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return uint8(h.Sum32() % numShards)
}

// Set stores a key-value pair with a TTL. If ttl <= 0, the key never expires.
func (c *MemoryCache) Set(key string, val []byte, ttl time.Duration) {
	shard := getShardIndex(key)
	c.shards[shard].Lock()
	defer c.shards[shard].Unlock()

	var expireNano int64
	if ttl > 0 {
		expireNano = time.Now().Add(ttl).UnixNano()
	}

	// Payload layout: [8 bytes expireNano][payload]
	buf := make([]byte, 8+len(val))
	binary.BigEndian.PutUint64(buf[0:8], uint64(expireNano))
	copy(buf[8:], val)

	c.cache.Set([]byte(key), buf)
}

// Get retrieves a key's value. Returns false if not found or expired.
func (c *MemoryCache) Get(key string) ([]byte, bool) {
	shard := getShardIndex(key)
	c.shards[shard].RLock()
	raw := c.cache.Get(nil, []byte(key))
	c.shards[shard].RUnlock()

	if len(raw) < 8 {
		return nil, false
	}

	expireNano := int64(binary.BigEndian.Uint64(raw[0:8]))
	if expireNano > 0 && time.Now().UnixNano() > expireNano {
		// Key expired: clean it up
		c.shards[shard].Lock()
		c.cache.Del([]byte(key))
		c.shards[shard].Unlock()
		return nil, false
	}

	val := make([]byte, len(raw)-8)
	copy(val, raw[8:])
	return val, true
}

// Delete removes a key from cache.
func (c *MemoryCache) Delete(key string) {
	shard := getShardIndex(key)
	c.shards[shard].Lock()
	defer c.shards[shard].Unlock()

	c.cache.Del([]byte(key))
}

// SetIdempotency sets a key if it does not exist or has expired.
// Returns true if successfully set (first caller), false if already set and unexpired.
func (c *MemoryCache) SetIdempotency(key string, val []byte, ttl time.Duration) bool {
	shard := getShardIndex(key)
	c.shards[shard].Lock()
	defer c.shards[shard].Unlock()

	raw := c.cache.Get(nil, []byte(key))
	if len(raw) >= 8 {
		expireNano := int64(binary.BigEndian.Uint64(raw[0:8]))
		if expireNano == 0 || time.Now().UnixNano() <= expireNano {
			// Key exists and is valid
			return false
		}
	}

	var expireNano int64
	if ttl > 0 {
		expireNano = time.Now().Add(ttl).UnixNano()
	}

	buf := make([]byte, 8+len(val))
	binary.BigEndian.PutUint64(buf[0:8], uint64(expireNano))
	copy(buf[8:], val)

	c.cache.Set([]byte(key), buf)
	return true
}

// CheckNonce validates and records a nonce.
// Returns true if nonce is valid and consumed, false if already consumed (replay).
func (c *MemoryCache) CheckNonce(nonce string, ttl time.Duration) bool {
	return c.SetIdempotency("nonce:"+nonce, []byte{1}, ttl)
}

// Reset clears all cache entries.
func (c *MemoryCache) Reset() {
	for i := range c.shards {
		c.shards[i].Lock()
	}
	c.cache.Reset()
	for i := range c.shards {
		c.shards[i].Unlock()
	}
}

// Close releases any cache resources.
func (c *MemoryCache) Close() error {
	c.Reset()
	return nil
}
