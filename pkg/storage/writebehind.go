package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	NumShards = 256
)

// OpType represents the mutation type for write-behind tracking.
type OpType string

const (
	OpUpsert OpType = "UPSERT"
	OpDelete OpType = "DELETE"
)

// DirtyOp records the latest state and monotonic sequence for write coalescing.
type DirtyOp struct {
	OpType       OpType    `json:"op_type"`
	Key          string    `json:"key"`
	Value        []byte    `json:"value,omitempty"`
	Version      uint64    `json:"version"`
	FirstDirtyAt time.Time `json:"first_dirty_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// memoryShard manages an in-memory key-value map and sorted index with RWMutex.
type memoryShard struct {
	mu     sync.RWMutex
	data   map[string][]byte
	sorted []string
	dirty  bool
}

func newMemoryShard() *memoryShard {
	return &memoryShard{
		data:   make(map[string][]byte),
		sorted: make([]string, 0),
	}
}

func (s *memoryShard) get(key string) ([]byte, bool) {
	s.mu.RLock()
	val, ok := s.data[key]
	if !ok {
		s.mu.RUnlock()
		return nil, false
	}
	res := make([]byte, len(val))
	copy(res, val)
	s.mu.RUnlock()
	return res, true
}

func (s *memoryShard) set(key string, val []byte) {
	s.mu.Lock()
	_, exists := s.data[key]
	buf := make([]byte, len(val))
	copy(buf, val)
	s.data[key] = buf
	if !exists {
		s.sorted = append(s.sorted, key)
		s.dirty = true
	}
	s.mu.Unlock()
}

func (s *memoryShard) delete(key string) {
	s.mu.Lock()
	if _, exists := s.data[key]; exists {
		delete(s.data, key)
		s.dirty = true
	}
	s.mu.Unlock()
}

func (s *memoryShard) ensureSortedLocked() {
	if s.dirty {
		// Clean up any deleted keys from sorted list
		active := make([]string, 0, len(s.data))
		for k := range s.data {
			active = append(active, k)
		}
		sort.Strings(active)
		s.sorted = active
		s.dirty = false
	}
}

// WriteBehindConfig configures WriteBehindStore.
type WriteBehindConfig struct {
	CloudflareEndpoint string
	CloudflareToken    string
	FlushIntervalIdle  time.Duration
	MaxDirtyAge        time.Duration
	MaxDirtyKeys       int
	MaxBatchSize       int
	LoadThreshold      int64 // In-flight request count threshold
	HTTPClient         *http.Client
}

// DefaultWriteBehindConfig provides production defaults.
func DefaultWriteBehindConfig() WriteBehindConfig {
	return WriteBehindConfig{
		FlushIntervalIdle: 500 * time.Millisecond,
		MaxDirtyAge:       15 * time.Second,
		MaxDirtyKeys:      2000,
		MaxBatchSize:      250,
		LoadThreshold:     50,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WriteBehindStore implements DurableStore with zero-network microsecond in-memory operations,
// local crash-safe Pebble WAL, and background adaptive write-behind syncing to Cloudflare.
type WriteBehindStore struct {
	shards    [NumShards]*memoryShard
	pebble    *PebbleStore
	cfg       WriteBehindConfig
	flusher   *LoadAwareFlusher
	versionSeq atomic.Uint64

	// Dirty tracking: shard-locked map of pending ops
	dirtyMu    sync.RWMutex
	dirtyOps   map[string]*DirtyOp
	inFlight   map[string]uint64 // key -> in-flight version sent to Cloudflare

	// Lifecycle
	closed   atomic.Bool
	stopOnce sync.Once
}

// NewWriteBehindStore initializes the in-memory shards, opens Pebble, hydrates state, and launches the flusher.
func NewWriteBehindStore(pebbleStore *PebbleStore, cfg WriteBehindConfig) (*WriteBehindStore, error) {
	if pebbleStore == nil {
		return nil, errors.New("writebehind: pebbleStore cannot be nil")
	}
	if cfg.FlushIntervalIdle <= 0 {
		cfg.FlushIntervalIdle = 500 * time.Millisecond
	}
	if cfg.MaxDirtyAge <= 0 {
		cfg.MaxDirtyAge = 15 * time.Second
	}
	if cfg.MaxDirtyKeys <= 0 {
		cfg.MaxDirtyKeys = 2000
	}
	if cfg.MaxBatchSize <= 0 {
		cfg.MaxBatchSize = 250
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}

	w := &WriteBehindStore{
		pebble:   pebbleStore,
		cfg:      cfg,
		dirtyOps: make(map[string]*DirtyOp),
		inFlight: make(map[string]uint64),
	}

	for i := 0; i < NumShards; i++ {
		w.shards[i] = newMemoryShard()
	}

	// 1. Hydrate in-memory state from local Pebble first
	if err := w.hydrateFromLocalPebble(); err != nil {
		return nil, fmt.Errorf("failed to hydrate from local pebble: %w", err)
	}

	// 2. If local Pebble was empty and Cloudflare is configured, hydrate from Cloudflare
	if w.totalMemoryKeys() == 0 && cfg.CloudflareEndpoint != "" {
		_ = w.hydrateFromCloudflare(context.Background(), "")
	}

	// 3. Start background load-aware flusher
	w.flusher = newLoadAwareFlusher(w, cfg)
	w.flusher.start()

	return w, nil
}

func shardIndex(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() % NumShards)
}

func (w *WriteBehindStore) totalMemoryKeys() int {
	total := 0
	for i := 0; i < NumShards; i++ {
		w.shards[i].mu.RLock()
		total += len(w.shards[i].data)
		w.shards[i].mu.RUnlock()
	}
	return total
}

// hydrateFromLocalPebble scans local Pebble and populates memory shards.
func (w *WriteBehindStore) hydrateFromLocalPebble() error {
	var count int
	err := w.pebble.PrefixScan(nil, func(key, val []byte) bool {
		k := string(key)
		idx := shardIndex(k)
		w.shards[idx].set(k, val)
		count++
		return true
	})
	return err
}

// hydrateFromCloudflare calls POST /v1/kv/scan on the Cloudflare worker gateway to pre-warm RAM.
func (w *WriteBehindStore) hydrateFromCloudflare(ctx context.Context, prefix string) error {
	if w.cfg.CloudflareEndpoint == "" {
		return nil
	}

	url := fmt.Sprintf("%s/v1/kv/scan", w.cfg.CloudflareEndpoint)
	body, _ := json.Marshal(map[string]any{
		"prefix": prefix,
		"limit":  10000,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.cfg.CloudflareToken != "" {
		req.Header.Set("Authorization", "Bearer "+w.cfg.CloudflareToken)
	}

	resp, err := w.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloudflare scan returned status: %d", resp.StatusCode)
	}

	var res struct {
		Items []struct {
			Key      string `json:"key"`
			ValueB64 string `json:"value_b64"`
			Version  uint64 `json:"version"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	batch := w.pebble.NewBatch()
	defer batch.Close()

	for _, item := range res.Items {
		val, err := base64.StdEncoding.DecodeString(item.ValueB64)
		if err != nil {
			continue
		}
		idx := shardIndex(item.Key)
		w.shards[idx].set(item.Key, val)
		_ = batch.Set([]byte(item.Key), val)
	}

	_ = batch.Commit(false)
	return nil
}

// markDirty records a mutation in the dirty coalescing map.
func (w *WriteBehindStore) markDirty(key string, opType OpType, val []byte) {
	ver := w.versionSeq.Add(1)
	now := time.Now()

	w.dirtyMu.Lock()
	defer w.dirtyMu.Unlock()

	existing, ok := w.dirtyOps[key]
	if ok {
		existing.OpType = opType
		existing.Value = val
		existing.Version = ver
		existing.UpdatedAt = now
	} else {
		w.dirtyOps[key] = &DirtyOp{
			OpType:       opType,
			Key:          key,
			Value:        val,
			Version:      ver,
			FirstDirtyAt: now,
			UpdatedAt:    now,
		}
	}
}

// Get fetches from fast RAM shards in sub-microsecond time.
func (w *WriteBehindStore) Get(key []byte) ([]byte, error) {
	k := string(key)
	idx := shardIndex(k)
	val, ok := w.shards[idx].get(k)
	if !ok {
		return nil, ErrNotFound
	}
	return val, nil
}

// Set mutates RAM, writes to local Pebble WAL immediately, and marks dirty for Cloudflare sync.
func (w *WriteBehindStore) Set(key, val []byte, syncWAL bool) error {
	if w.closed.Load() {
		return errors.New("writebehind: store is closed")
	}

	k := string(key)
	idx := shardIndex(k)

	// 1. Mutate RAM
	w.shards[idx].set(k, val)

	// 2. Write to local Pebble WAL
	if err := w.pebble.Set(key, val, syncWAL); err != nil {
		return fmt.Errorf("writebehind pebble set: %w", err)
	}

	// 3. Mark dirty for coalesced Cloudflare push
	w.markDirty(k, OpUpsert, val)
	return nil
}

// Delete removes from RAM, writes tombstone to Pebble, and marks dirty.
func (w *WriteBehindStore) Delete(key []byte, syncWAL bool) error {
	if w.closed.Load() {
		return errors.New("writebehind: store is closed")
	}

	k := string(key)
	idx := shardIndex(k)

	// 1. Remove from RAM
	w.shards[idx].delete(k)

	// 2. Remove in local Pebble
	if err := w.pebble.Delete(key, syncWAL); err != nil {
		return fmt.Errorf("writebehind pebble delete: %w", err)
	}

	// 3. Mark dirty
	w.markDirty(k, OpDelete, nil)
	return nil
}

// PrefixScan scans memory shards with prefix matching in sorted order.
func (w *WriteBehindStore) PrefixScan(prefix []byte, fn func(key, val []byte) bool) error {
	pfx := string(prefix)

	// Collect matching keys from all shards
	var matchedKeys []string
	for i := 0; i < NumShards; i++ {
		shard := w.shards[i]
		shard.mu.Lock()
		shard.ensureSortedLocked()
		// Scan through sorted slice
		for _, k := range shard.sorted {
			if len(k) >= len(pfx) && k[:len(pfx)] == pfx {
				matchedKeys = append(matchedKeys, k)
			}
		}
		shard.mu.Unlock()
	}

	// Sort globally
	sort.Strings(matchedKeys)

	// Execute callback
	for _, k := range matchedKeys {
		idx := shardIndex(k)
		v, ok := w.shards[idx].get(k)
		if ok {
			if !fn([]byte(k), v) {
				break
			}
		}
	}

	return nil
}

// NewBatch returns an atomic write-behind batch.
func (w *WriteBehindStore) NewBatch() DurableBatch {
	return &writeBehindBatch{
		store: w,
		ops:   make(map[string]*DirtyOp),
	}
}

// writeBehindBatch collects atomic multi-key updates.
type writeBehindBatch struct {
	store *WriteBehindStore
	ops   map[string]*DirtyOp
	mu    sync.Mutex
}

func (b *writeBehindBatch) Set(key, val []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	buf := make([]byte, len(val))
	copy(buf, val)
	b.ops[string(key)] = &DirtyOp{
		OpType: OpUpsert,
		Key:    string(key),
		Value:  buf,
	}
	return nil
}

func (b *writeBehindBatch) Delete(key []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ops[string(key)] = &DirtyOp{
		OpType: OpDelete,
		Key:    string(key),
	}
	return nil
}

func (b *writeBehindBatch) Commit(syncWAL bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.ops) == 0 {
		return nil
	}

	pebbleBatch := b.store.pebble.NewBatch()
	defer pebbleBatch.Close()

	for k, op := range b.ops {
		keyBytes := []byte(k)
		idx := shardIndex(k)

		if op.OpType == OpUpsert {
			b.store.shards[idx].set(k, op.Value)
			_ = pebbleBatch.Set(keyBytes, op.Value)
		} else {
			b.store.shards[idx].delete(k)
			_ = pebbleBatch.Delete(keyBytes)
		}
		b.store.markDirty(k, op.OpType, op.Value)
	}

	return pebbleBatch.Commit(syncWAL)
}

func (b *writeBehindBatch) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ops = nil
	return nil
}

// Flush synchronously drains any pending dirty operations to Cloudflare.
func (w *WriteBehindStore) Flush(ctx context.Context) error {
	if w.flusher != nil {
		return w.flusher.drainAll(ctx)
	}
	return nil
}

// Close drains all unflushed dirty operations and closes Pebble cleanly.
func (w *WriteBehindStore) Close() error {
	var err error
	w.stopOnce.Do(func() {
		w.closed.Store(true)

		if w.flusher != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = w.flusher.drainAll(ctx)
			w.flusher.stop()
		}

		if w.pebble != nil {
			err = w.pebble.Close()
		}
	})
	return err
}

// In-flight active request measurement hook for load detection
func (w *WriteBehindStore) IncrementActiveRequests() {
	if w.flusher != nil {
		w.flusher.activeRequests.Add(1)
	}
}

func (w *WriteBehindStore) DecrementActiveRequests() {
	if w.flusher != nil {
		w.flusher.activeRequests.Add(-1)
	}
}

// PebbleStore accessor for underlying low-level ops if needed
func (w *WriteBehindStore) Pebble() *PebbleStore {
	return w.pebble
}
