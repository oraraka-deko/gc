package storage

import (
	"errors"
	"io"
	"os"

	"github.com/cockroachdb/pebble"
)

// Namespace prefixes for organized key separation.
const (
	NSAccount         = "led:acc:" // Account balance & metadata: led:acc:<accountID>
	NSJournal         = "led:jnl:" // Immutable journal entries: led:jnl:<entryID>
	NSAccountHistory  = "led:ach:" // Account history index: led:ach:<accountID>:<timestamp>:<entryID>
	NSPending         = "led:pen:" // Pending two-phase transfers: led:pen:<transferID>
	NSWorkflow        = "wf:cur:"  // Tickless workflow instance cursors: wf:cur:<instanceID>
	NSWorkflowHistory = "wf:his:"  // Workflow run logs: wf:his:<instanceID>:<stepID>
	NSOutbox          = "out:tx:"  // Transactional outbox messages: out:tx:<txID>
	NSIdempotency     = "idm:"     // Durable idempotency keys: idm:<key>
	NSUser            = "usr:pro:" // User profile data: usr:pro:<userID>
	NSUserGifts       = "usr:gft:" // User collectible gifts: usr:gft:<userID>:<assetID>
)

var (
	// ErrNotFound is returned when a key does not exist.
	ErrNotFound = errors.New("storage: key not found")
)

// PebbleStore wraps Pebble DB with transaction and prefix scan utilities.
type PebbleStore struct {
	db   *pebble.DB
	path string
}

// PebbleConfig specifies options for opening Pebble.
type PebbleConfig struct {
	Path     string
	ReadOnly bool
}

// NewPebbleStore creates or opens an LSM-tree database at the given path.
func NewPebbleStore(cfg PebbleConfig) (*PebbleStore, error) {
	if cfg.Path == "" {
		return nil, errors.New("pebble path cannot be empty")
	}

	opts := &pebble.Options{
		ReadOnly: cfg.ReadOnly,
	}

	// Ensure directory exists
	if !cfg.ReadOnly {
		if err := os.MkdirAll(cfg.Path, 0755); err != nil {
			return nil, err
		}
	}

	db, err := pebble.Open(cfg.Path, opts)
	if err != nil {
		return nil, err
	}

	return &PebbleStore{
		db:   db,
		path: cfg.Path,
	}, nil
}

// Get retrieves a key's value. Returns ErrNotFound if not present.
func (p *PebbleStore) Get(key []byte) ([]byte, error) {
	val, closer, err := p.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer closer.Close()

	// Copy value so caller owns the slice safely
	res := make([]byte, len(val))
	copy(res, val)
	return res, nil
}

// Set writes a key-value pair.
func (p *PebbleStore) Set(key, val []byte, sync bool) error {
	opts := pebble.NoSync
	if sync {
		opts = pebble.Sync
	}
	return p.db.Set(key, val, opts)
}

// Delete removes a key.
func (p *PebbleStore) Delete(key []byte, sync bool) error {
	opts := pebble.NoSync
	if sync {
		opts = pebble.Sync
	}
	return p.db.Delete(key, opts)
}

// PebbleBatch wraps pebble.Batch for atomic multi-key updates.
type PebbleBatch struct {
	batch *pebble.Batch
}

// NewBatch creates an atomic batch writer.
func (p *PebbleStore) NewBatch() *PebbleBatch {
	return &PebbleBatch{
		batch: p.db.NewBatch(),
	}
}

// Set adds a Set operation to the batch.
func (b *PebbleBatch) Set(key, val []byte) error {
	return b.batch.Set(key, val, nil)
}

// Delete adds a Delete operation to the batch.
func (b *PebbleBatch) Delete(key []byte) error {
	return b.batch.Delete(key, nil)
}

// Commit commits the batch to the database.
func (b *PebbleBatch) Commit(sync bool) error {
	opts := pebble.NoSync
	if sync {
		opts = pebble.Sync
	}
	return b.batch.Commit(opts)
}

// Close releases the batch without committing if not already committed.
func (b *PebbleBatch) Close() error {
	return b.batch.Close()
}

// PrefixUpperBound computes the upper bound for range scanning a prefix.
func PrefixUpperBound(prefix []byte) []byte {
	upper := make([]byte, len(prefix))
	copy(upper, prefix)
	for i := len(upper) - 1; i >= 0; i-- {
		upper[i]++
		if upper[i] != 0 {
			return upper[:i+1]
		}
	}
	return nil
}

// PrefixScan iterates over all keys matching prefix in lexicographical order.
// Return false from fn to stop iteration early.
func (p *PebbleStore) PrefixScan(prefix []byte, fn func(key, val []byte) bool) error {
	opts := &pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: PrefixUpperBound(prefix),
	}

	iter, err := p.db.NewIter(opts)
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		k := iter.Key()
		v := iter.Value()
		if !fn(k, v) {
			break
		}
	}

	return iter.Error()
}

// Flush forces any buffered memtables to disk.
func (p *PebbleStore) Flush() error {
	return p.db.Flush()
}

// Close closes the underlying Pebble database.
func (p *PebbleStore) Close() error {
	if p.db != nil {
		err := p.db.Close()
		p.db = nil
		return err
	}
	return nil
}

// Closer wraps io.Closer for cleanup helper.
var _ io.Closer = (*PebbleStore)(nil)
