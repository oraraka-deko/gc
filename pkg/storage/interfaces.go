package storage

import (
	"context"
	"io"
)

// DurableStore defines the uniform storage contract for low-latency in-memory
// reads/writes with durable crash-safe local WAL and asynchronous off-site replication.
type DurableStore interface {
	io.Closer

	// Get retrieves a key's value. Returns ErrNotFound if not present.
	Get(key []byte) ([]byte, error)

	// Set stores a key-value pair.
	Set(key, val []byte, sync bool) error

	// Delete removes a key.
	Delete(key []byte, sync bool) error

	// PrefixScan iterates over all keys matching prefix in lexicographical order.
	// Returning false from fn halts iteration early.
	PrefixScan(prefix []byte, fn func(key, val []byte) bool) error

	// NewBatch returns an atomic multi-key batch writer.
	NewBatch() DurableBatch

	// Flush forces an immediate drain of pending dirty keys.
	Flush(ctx context.Context) error
}

// DurableBatch abstracts atomic multi-key operations.
type DurableBatch interface {
	Set(key, val []byte) error
	Delete(key []byte) error
	Commit(sync bool) error
	Close() error
}
