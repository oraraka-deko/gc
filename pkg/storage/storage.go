package storage

import (
	"fmt"
)

// Storage integrates both FastCache (in-memory zero-GC cache) and Pebble (durable LSM-tree).
type Storage struct {
	Fast   *MemoryCache
	Pebble *PebbleStore
}

// Config provides initialization parameters for Storage.
type Config struct {
	PebblePath     string
	FastCacheBytes int
}

// New creates a new Storage instance.
func New(cfg Config) (*Storage, error) {
	pebbleStore, err := NewPebbleStore(PebbleConfig{
		Path: cfg.PebblePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble store: %w", err)
	}

	fastCache := NewMemoryCache(CacheConfig{
		MaxBytes: cfg.FastCacheBytes,
	})

	return &Storage{
		Fast:   fastCache,
		Pebble: pebbleStore,
	}, nil
}

// Close gracefully closes Pebble and clears cache.
func (s *Storage) Close() error {
	if s.Fast != nil {
		_ = s.Fast.Close()
	}
	if s.Pebble != nil {
		return s.Pebble.Close()
	}
	return nil
}
