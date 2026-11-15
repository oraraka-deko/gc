package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWriteBehind_Coalescing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wb_coalesce_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pebbleStore, err := NewPebbleStore(PebbleConfig{Path: filepath.Join(tmpDir, "pebble")})
	if err != nil {
		t.Fatal(err)
	}

	var postCount atomic.Int64
	var flushedUpserts []CFUpsert
	var mu sync.Mutex

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/kv/batch" {
			postCount.Add(1)
			var payload CFBatchPayload
			_ = json.NewDecoder(r.Body).Decode(&payload)
			mu.Lock()
			flushedUpserts = append(flushedUpserts, payload.Upserts...)
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cfg := WriteBehindConfig{
		CloudflareEndpoint: ts.URL,
		FlushIntervalIdle:  10 * time.Minute, // prevent background ticker from firing mid-loop
		MaxDirtyAge:        10 * time.Minute,
		MaxDirtyKeys:       100000,
		MaxBatchSize:       250,
	}

	wb, err := NewWriteBehindStore(pebbleStore, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer wb.Close()

	// Simulate 10,000 rapid balance updates to ONE user key in RAM
	targetKey := []byte("led:acc:user:123:main:XTR")
	for i := 1; i <= 10000; i++ {
		val := []byte(fmt.Sprintf(`{"available":%d}`, i))
		if err := wb.Set(targetKey, val, false); err != nil {
			t.Fatalf("set error: %v", err)
		}
	}

	// Verify RAM reads immediately reflect the latest state in microseconds
	val, err := wb.Get(targetKey)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	expected := `{"available":10000}`
	if string(val) != expected {
		t.Fatalf("expected %s, got %s", expected, string(val))
	}

	// Flush dirty keys
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := wb.Flush(ctx); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Ensure 10,000 writes coalesced into exactly 1 final upsert sent to Cloudflare
	if len(flushedUpserts) != 1 {
		t.Fatalf("expected exactly 1 coalesced upsert, got %d", len(flushedUpserts))
	}
	decoded, _ := base64.StdEncoding.DecodeString(flushedUpserts[0].ValueB64)
	if string(decoded) != expected {
		t.Fatalf("expected flushed value %s, got %s", expected, string(decoded))
	}
}

func TestWriteBehind_VersionMonotonicityProtection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wb_cas_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pebbleStore, err := NewPebbleStore(PebbleConfig{Path: filepath.Join(tmpDir, "pebble")})
	if err != nil {
		t.Fatal(err)
	}

	var writeDelay sync.WaitGroup
	writeDelay.Add(1)

	var requestCount atomic.Int64
	var flushedVersions []uint64
	var mu sync.Mutex

	var wb *WriteBehindStore

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/kv/batch" {
			reqNum := requestCount.Add(1)
			var payload CFBatchPayload
			_ = json.NewDecoder(r.Body).Decode(&payload)

			mu.Lock()
			for _, u := range payload.Upserts {
				flushedVersions = append(flushedVersions, u.Version)
			}
			mu.Unlock()

			if reqNum == 1 {
				// While 1st HTTP request is in-flight, mutate the key again!
				_ = wb.Set([]byte("key1"), []byte("val_v2"), false)
				writeDelay.Done()
			}
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer ts.Close()

	cfg := WriteBehindConfig{
		CloudflareEndpoint: ts.URL,
		FlushIntervalIdle:  200 * time.Millisecond,
		MaxDirtyAge:        5 * time.Second,
		MaxDirtyKeys:       100,
		MaxBatchSize:       250,
	}

	wb, err = NewWriteBehindStore(pebbleStore, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer wb.Close()

	// Initial write
	_ = wb.Set([]byte("key1"), []byte("val_v1"), false)

	// Trigger flush
	ctx := context.Background()
	_ = wb.flusher.flushNextBatch(ctx)

	writeDelay.Wait()

	// Flush again to capture the in-flight mutation
	_ = wb.Flush(ctx)

	mu.Lock()
	defer mu.Unlock()

	// Key should have been sent a second time because version incremented during in-flight!
	if len(flushedVersions) < 2 {
		t.Fatalf("expected at least 2 flushes due to in-flight mutation, got: %v", flushedVersions)
	}

	val, _ := wb.Get([]byte("key1"))
	if string(val) != "val_v2" {
		t.Fatalf("expected val_v2, got: %s", string(val))
	}
}

func TestWriteBehind_CrashRecoveryFromLocalPebble(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wb_recovery_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pebblePath := filepath.Join(tmpDir, "pebble")

	// Phase 1: Run store, write data, simulate sudden crash before flusher runs
	{
		pebble1, err := NewPebbleStore(PebbleConfig{Path: pebblePath})
		if err != nil {
			t.Fatal(err)
		}

		wb1, err := NewWriteBehindStore(pebble1, WriteBehindConfig{
			FlushIntervalIdle: 10 * time.Hour, // don't flush
		})
		if err != nil {
			t.Fatal(err)
		}

		for i := 0; i < 20; i++ {
			k := fmt.Sprintf("rec:item:%d", i)
			v := fmt.Sprintf("val:%d", i)
			_ = wb1.Set([]byte(k), []byte(v), true)
		}

		// Close pebble without flushing to remote
		_ = wb1.pebble.Close()
	}

	// Phase 2: Restart store from same pebble path
	{
		pebble2, err := NewPebbleStore(PebbleConfig{Path: pebblePath})
		if err != nil {
			t.Fatal(err)
		}

		wb2, err := NewWriteBehindStore(pebble2, WriteBehindConfig{
			FlushIntervalIdle: 50 * time.Millisecond,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer wb2.Close()

		// Verify all 20 records hydrated into RAM
		for i := 0; i < 20; i++ {
			k := fmt.Sprintf("rec:item:%d", i)
			v, err := wb2.Get([]byte(k))
			if err != nil {
				t.Fatalf("expected key %s to exist after crash recovery, err: %v", k, err)
			}
			if string(v) != fmt.Sprintf("val:%d", i) {
				t.Fatalf("unexpected value: %s", string(v))
			}
		}

		// Verify PrefixScan in RAM
		count := 0
		_ = wb2.PrefixScan([]byte("rec:item:"), func(k, v []byte) bool {
			count++
			return true
		})
		if count != 20 {
			t.Fatalf("expected 20 items in PrefixScan, got %d", count)
		}
	}
}
