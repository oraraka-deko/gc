package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// CFBatchPayload matches Cloudflare Worker POST /v1/kv/batch contract.
type CFBatchPayload struct {
	Upserts []CFUpsert `json:"upserts"`
	Deletes []string   `json:"deletes"`
}

type CFUpsert struct {
	Key      string `json:"key"`
	ValueB64 string `json:"value_b64"`
	Version  uint64 `json:"version"`
}

// LoadAwareFlusher continuously monitors server pressure and flushes coalesced batches.
type LoadAwareFlusher struct {
	store          *WriteBehindStore
	cfg            WriteBehindConfig
	activeRequests atomic.Int64
	gameTicksCount atomic.Uint64

	stopChan chan struct{}
	doneChan chan struct{}
	drainMu  sync.Mutex
}

func newLoadAwareFlusher(store *WriteBehindStore, cfg WriteBehindConfig) *LoadAwareFlusher {
	return &LoadAwareFlusher{
		store:    store,
		cfg:      cfg,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

func (f *LoadAwareFlusher) start() {
	go f.runLoop()
}

func (f *LoadAwareFlusher) stop() {
	close(f.stopChan)
	<-f.doneChan
}

func (f *LoadAwareFlusher) runLoop() {
	defer close(f.doneChan)

	ticker := time.NewTicker(f.cfg.FlushIntervalIdle)
	defer ticker.Stop()

	for {
		select {
		case <-f.stopChan:
			return
		case <-ticker.C:
			f.evalAndFlush()
		}
	}
}

// isHighLoad determines whether VPS is under heavy load.
func (f *LoadAwareFlusher) isHighLoad() bool {
	// 1. Check in-flight HTTP/WebSocket requests
	if f.cfg.LoadThreshold > 0 && f.activeRequests.Load() > f.cfg.LoadThreshold {
		return true
	}

	// 2. Check goroutine runaway spike
	if runtime.NumGoroutine() > 10000 {
		return true
	}

	return false
}

func (f *LoadAwareFlusher) evalAndFlush() {
	f.store.dirtyMu.RLock()
	dirtyCount := len(f.store.dirtyOps)
	var oldestAge time.Duration
	now := time.Now()
	for _, op := range f.store.dirtyOps {
		age := now.Sub(op.FirstDirtyAt)
		if age > oldestAge {
			oldestAge = age
		}
	}
	f.store.dirtyMu.RUnlock()

	if dirtyCount == 0 {
		return
	}

	// Safety Bounds: Force flush if dirtyCount >= MaxDirtyKeys or max staleness exceeded
	forceFlush := dirtyCount >= f.cfg.MaxDirtyKeys || oldestAge >= f.cfg.MaxDirtyAge

	if f.isHighLoad() && !forceFlush {
		// High load & within safety bounds: defer flush to prioritize live websocket players
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := f.flushNextBatch(ctx); err != nil {
		log.Printf("[Flusher ERROR] WriteBehind flush to Cloudflare failed: %v", err)
	}
}

// flushNextBatch extracts up to MaxBatchSize coalesced ops and posts them to Cloudflare.
func (f *LoadAwareFlusher) flushNextBatch(ctx context.Context) error {
	f.drainMu.Lock()
	defer f.drainMu.Unlock()

	// 1. Snapshot a chunk of dirty ops
	f.store.dirtyMu.Lock()
	if len(f.store.dirtyOps) == 0 {
		f.store.dirtyMu.Unlock()
		return nil
	}

	batchSize := f.cfg.MaxBatchSize
	if batchSize <= 0 {
		batchSize = 250
	}

	payload := CFBatchPayload{
		Upserts: make([]CFUpsert, 0, batchSize),
		Deletes: make([]string, 0),
	}
	inFlightVersions := make(map[string]uint64)

	for k, op := range f.store.dirtyOps {
		inFlightVersions[k] = op.Version
		if op.OpType == OpUpsert {
			payload.Upserts = append(payload.Upserts, CFUpsert{
				Key:      k,
				ValueB64: base64.StdEncoding.EncodeToString(op.Value),
				Version:  op.Version,
			})
		} else {
			payload.Deletes = append(payload.Deletes, k)
		}

		if len(payload.Upserts)+len(payload.Deletes) >= batchSize {
			break
		}
	}
	f.store.dirtyMu.Unlock()

	// If no Cloudflare endpoint configured (e.g. offline/local test), simply acknowledge mutations
	if f.cfg.CloudflareEndpoint == "" {
		f.acknowledgeBatch(inFlightVersions)
		return nil
	}

	// 2. Send HTTP batch to Cloudflare Worker
	if err := f.sendToCloudflare(ctx, payload); err != nil {
		return err
	}

	// 3. Acknowledge with CAS version-check: only remove dirty keys if no new mutations occurred in-flight
	f.acknowledgeBatch(inFlightVersions)
	return nil
}

func (f *LoadAwareFlusher) sendToCloudflare(ctx context.Context, payload CFBatchPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v1/kv/batch", f.cfg.CloudflareEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TMA-Backend-StorageGateway/1.0")
	if f.cfg.CloudflareToken != "" {
		req.Header.Set("Authorization", "Bearer "+f.cfg.CloudflareToken)
	}

	resp, err := f.cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("writebehind flush failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("writebehind cloudflare returned status: %d", resp.StatusCode)
	}

	log.Printf("[Flusher] Flushed %d upserts, %d deletes to Cloudflare D1/DO (%s)",
		len(payload.Upserts), len(payload.Deletes), f.cfg.CloudflareEndpoint)
	return nil
}

// acknowledgeBatch clears dirty ops only if their version hasn't incremented while in-flight.
func (f *LoadAwareFlusher) acknowledgeBatch(inFlightVersions map[string]uint64) {
	f.store.dirtyMu.Lock()
	defer f.store.dirtyMu.Unlock()

	for k, inFlightVer := range inFlightVersions {
		current, exists := f.store.dirtyOps[k]
		if exists && current.Version == inFlightVer {
			// Version unchanged: clean up dirty entry
			delete(f.store.dirtyOps, k)
		}
		// If current.Version > inFlightVer: keep dirty! A newer write occurred while flushing.
	}
}

// drainAll drains all dirty operations in chunks until dirtyOps map is empty.
func (f *LoadAwareFlusher) drainAll(ctx context.Context) error {
	for {
		f.store.dirtyMu.RLock()
		remaining := len(f.store.dirtyOps)
		f.store.dirtyMu.RUnlock()

		if remaining == 0 {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := f.flushNextBatch(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}
