package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryCache_Basic(t *testing.T) {
	cache := NewMemoryCache(CacheConfig{MaxBytes: 1024 * 1024})
	defer cache.Close()

	key := "test:item:1"
	val := []byte("hello world")

	// Set with 1-second TTL
	cache.Set(key, val, 1*time.Second)

	ret, ok := cache.Get(key)
	if !ok || string(ret) != string(val) {
		t.Fatalf("expected %s, got %s (ok=%v)", string(val), string(ret), ok)
	}

	// Sleep until expired
	time.Sleep(1100 * time.Millisecond)
	ret, ok = cache.Get(key)
	if ok || ret != nil {
		t.Fatalf("expected key to be expired, got %s (ok=%v)", string(ret), ok)
	}
}

func TestMemoryCache_IdempotencyAndNonce(t *testing.T) {
	cache := NewMemoryCache(CacheConfig{MaxBytes: 1024 * 1024})
	defer cache.Close()

	key := "idemp:req:123"
	payload := []byte("success")

	// First call should succeed
	first := cache.SetIdempotency(key, payload, 500*time.Millisecond)
	if !first {
		t.Fatalf("expected first idempotency call to return true")
	}

	// Immediate second call should fail
	second := cache.SetIdempotency(key, payload, 500*time.Millisecond)
	if second {
		t.Fatalf("expected duplicate idempotency call to return false")
	}

	// Test nonce
	nonce := "random_nonce_999"
	if !cache.CheckNonce(nonce, 500*time.Millisecond) {
		t.Fatalf("expected first nonce check to succeed")
	}
	if cache.CheckNonce(nonce, 500*time.Millisecond) {
		t.Fatalf("expected second nonce check to fail (replay)")
	}
}

func TestPebbleStore_Operations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pebble_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewPebbleStore(PebbleConfig{Path: filepath.Join(tmpDir, "db")})
	if err != nil {
		t.Fatalf("failed to create pebble store: %v", err)
	}
	defer store.Close()

	// 1. Basic Set & Get
	k1 := []byte("key:1")
	v1 := []byte("value-1")
	if err := store.Set(k1, v1, true); err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	got, err := store.Get(k1)
	if err != nil || string(got) != string(v1) {
		t.Fatalf("expected %s, got %s, err: %v", string(v1), string(got), err)
	}

	// 2. Batch Set
	batch := store.NewBatch()
	for i := 0; i < 5; i++ {
		k := []byte(fmt.Sprintf("%sitem:%d", NSAccount, i))
		v := []byte(fmt.Sprintf("balance_%d", i*100))
		_ = batch.Set(k, v)
	}
	if err := batch.Commit(true); err != nil {
		t.Fatalf("failed to commit batch: %v", err)
	}

	// 3. Prefix Scan
	var scanned []string
	err = store.PrefixScan([]byte(NSAccount), func(k, v []byte) bool {
		scanned = append(scanned, string(k))
		return true
	})
	if err != nil {
		t.Fatalf("prefix scan error: %v", err)
	}
	if len(scanned) != 5 {
		t.Fatalf("expected 5 scanned items, got %d", len(scanned))
	}

	// 4. Delete & NotFound
	if err := store.Delete(k1, true); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}
	_, err = store.Get(k1)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestStorage_Integration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := New(Config{
		PebblePath:     filepath.Join(tmpDir, "pebble"),
		FastCacheBytes: 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}
	defer store.Close()

	// Verify FastCache
	store.Fast.Set("test", []byte("val"), time.Minute)
	v, ok := store.Fast.Get("test")
	if !ok || string(v) != "val" {
		t.Fatalf("fastcache error")
	}

	// Verify Pebble
	if err := store.Pebble.Set([]byte("test_pebble"), []byte("val_pebble"), true); err != nil {
		t.Fatalf("pebble set error: %v", err)
	}
	pv, err := store.Pebble.Get([]byte("test_pebble"))
	if err != nil || string(pv) != "val_pebble" {
		t.Fatalf("pebble get error: %v", err)
	}
}
