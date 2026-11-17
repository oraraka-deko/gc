package storage

import (
	"context"
	"fmt"
	"time"
)

// Storage integrates both FastCache (in-memory zero-GC cache), Pebble (durable LSM-tree),
// and DB (WriteBehindStore implementing DurableStore with microsecond RAM speed and Cloudflare D1/DO sync).
type Storage struct {
	Fast   *MemoryCache
	Pebble *PebbleStore
	DB     DurableStore
}

// Config provides initialization parameters for Storage.
type Config struct {
	PebblePath         string
	FastCacheBytes     int
	CloudflareEndpoint string
	CloudflareToken    string
	FlushIntervalIdle  time.Duration
	MaxDirtyAge        time.Duration
	MaxDirtyKeys       int
	MaxBatchSize       int
	LoadThreshold      int64
}

// New creates a new Storage instance with WriteBehindStore wired as DB.
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

	wbCfg := WriteBehindConfig{
		CloudflareEndpoint: cfg.CloudflareEndpoint,
		CloudflareToken:    cfg.CloudflareToken,
		FlushIntervalIdle:  cfg.FlushIntervalIdle,
		MaxDirtyAge:        cfg.MaxDirtyAge,
		MaxDirtyKeys:       cfg.MaxDirtyKeys,
		MaxBatchSize:       cfg.MaxBatchSize,
		LoadThreshold:      cfg.LoadThreshold,
	}

	writeBehindStore, err := NewWriteBehindStore(pebbleStore, wbCfg)
	if err != nil {
		_ = pebbleStore.Close()
		return nil, fmt.Errorf("failed to init writebehind store: %w", err)
	}

	return &Storage{
		Fast:   fastCache,
		Pebble: pebbleStore,
		DB:     writeBehindStore,
	}, nil
}

// Store returns the primary DurableStore (WriteBehindStore with RAM shards and Cloudflare sync),
// or falls back to PebbleStore if DB is not configured (e.g. in isolated unit tests).
func (s *Storage) Store() DurableStore {
	if s != nil && s.DB != nil {
		return s.DB
	}
	if s != nil && s.Pebble != nil {
		return &pebbleDurableAdapter{s.Pebble}
	}
	return nil
}

// pebbleDurableAdapter wraps PebbleStore to satisfy the DurableStore interface.
type pebbleDurableAdapter struct {
	pebble *PebbleStore
}

func (a *pebbleDurableAdapter) Get(key []byte) ([]byte, error) {
	return a.pebble.Get(key)
}

func (a *pebbleDurableAdapter) Set(key, val []byte, sync bool) error {
	return a.pebble.Set(key, val, sync)
}

func (a *pebbleDurableAdapter) Delete(key []byte, sync bool) error {
	return a.pebble.Delete(key, sync)
}

func (a *pebbleDurableAdapter) PrefixScan(prefix []byte, fn func(key, val []byte) bool) error {
	return a.pebble.PrefixScan(prefix, fn)
}

func (a *pebbleDurableAdapter) NewBatch() DurableBatch {
	return a.pebble.NewBatch()
}

func (a *pebbleDurableAdapter) Flush(ctx context.Context) error {
	return a.pebble.Flush()
}

func (a *pebbleDurableAdapter) Close() error {
	return a.pebble.Close()
}

// Flush forces an immediate drain of pending dirty records to Cloudflare.
func (s *Storage) Flush(ctx context.Context) error {
	if s.DB != nil {
		return s.DB.Flush(ctx)
	}
	return nil
}

// Close gracefully closes WriteBehindStore, Pebble, and FastCache.
func (s *Storage) Close() error {
	if s.Fast != nil {
		_ = s.Fast.Close()
	}
	if s.DB != nil {
		return s.DB.Close()
	} else if s.Pebble != nil {
		return s.Pebble.Close()
	}
	return nil
}
