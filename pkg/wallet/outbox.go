package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gc/pkg/ledger"
	"gc/pkg/storage"
)

// OutboxConfig configures the OutboxManager.
type OutboxConfig struct {
	MaxBatchSize int           // Highload Wallet v3 supports up to 254 transfers per cell
	BatchTimeout time.Duration // Max wait before processing a partial batch
}

// OutboxManager manages the lifecycle of outbound transactions.
type OutboxManager struct {
	store        *storage.Storage
	ledger       *ledger.Ledger
	signer       BatchSigner
	maxBatchSize int
	batchTimeout time.Duration
	mu           sync.Mutex
}

// NewOutboxManager creates a new OutboxManager instance.
func NewOutboxManager(store *storage.Storage, led *ledger.Ledger, signer BatchSigner, cfg OutboxConfig) *OutboxManager {
	batchSize := cfg.MaxBatchSize
	if batchSize <= 0 || batchSize > 254 {
		batchSize = 254
	}
	timeout := cfg.BatchTimeout
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}

	return &OutboxManager{
		store:        store,
		ledger:       led,
		signer:       signer,
		maxBatchSize: batchSize,
		batchTimeout: timeout,
	}
}

// Enqueue creates a new outbound transaction and reserves the required funds in the ledger.
func (o *OutboxManager) Enqueue(ctx context.Context, userID int64, asset string, amount uint64, destAddress string, memo string) (*OutboxTx, error) {
	if amount == 0 {
		return nil, ledger.ErrInvalidAmount
	}
	if destAddress == "" {
		return nil, fmt.Errorf("destination address cannot be empty")
	}

	now := time.Now().UnixNano()
	txID := fmt.Sprintf("tx_out_%d_%d", userID, now)
	transferID := fmt.Sprintf("xfr_%s", txID)

	// Two-phase lock: Reserve user funds in the ledger first
	reserveMeta := map[string]string{
		"outbox_tx_id": txID,
		"destination":  destAddress,
		"memo":         memo,
	}

	destAccount := ledger.SystemReserveAccount(asset)
	_, err := o.ledger.Reserve(ctx, transferID, userID, destAccount, asset, amount, reserveMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to reserve funds for outbox: %w", err)
	}

	tx := &OutboxTx{
		ID:                 txID,
		UserID:             userID,
		TransferID:         transferID,
		Asset:              asset,
		Amount:             amount,
		DestinationAddress: destAddress,
		Memo:               memo,
		State:              StateQueued,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Persist OutboxTx into Pebble
	data, err := json.Marshal(tx)
	if err != nil {
		return nil, err
	}

	key := []byte(storage.NSOutbox + txID)
	if err := o.store.Pebble.Set(key, data, true); err != nil {
		return nil, fmt.Errorf("failed to save outbox tx: %w", err)
	}

	return tx, nil
}

// GetTx retrieves an OutboxTx by its ID.
func (o *OutboxManager) GetTx(ctx context.Context, txID string) (*OutboxTx, error) {
	key := []byte(storage.NSOutbox + txID)
	data, err := o.store.Pebble.Get(key)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrTxNotFound
		}
		return nil, err
	}

	var tx OutboxTx
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal outbox tx: %w", err)
	}
	return &tx, nil
}

// ListByState returns transactions in a specific state.
func (o *OutboxManager) ListByState(state OutboxState, limit int) ([]*OutboxTx, error) {
	if limit <= 0 {
		limit = 100
	}

	var results []*OutboxTx
	prefix := []byte(storage.NSOutbox)

	err := o.store.Pebble.PrefixScan(prefix, func(k, v []byte) bool {
		var tx OutboxTx
		if err := json.Unmarshal(v, &tx); err == nil {
			if tx.State == state {
				results = append(results, &tx)
			}
		}
		return len(results) < limit
	})

	return results, err
}

// ProcessNextBatch collects queued transactions, signs them with the BatchSigner,
// and transitions their state to StateSigned.
func (o *OutboxManager) ProcessNextBatch(ctx context.Context) (*SignedBatch, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	queued, err := o.ListByState(StateQueued, o.maxBatchSize)
	if err != nil {
		return nil, err
	}
	if len(queued) == 0 {
		return nil, ErrEmptyBatch
	}

	// Sign the batch
	txValues := make([]OutboxTx, len(queued))
	for i, q := range queued {
		txValues[i] = *q
	}

	signedBatch, err := o.signer.SignBatch(ctx, txValues)
	if err != nil {
		return nil, fmt.Errorf("batch signer failed: %w", err)
	}

	// Update items to StateSigned in an atomic Pebble batch
	batch := o.store.Pebble.NewBatch()
	defer batch.Close()

	now := time.Now().UnixNano()
	for _, q := range queued {
		q.State = StateSigned
		q.SignedBOC = signedBatch.BOCBase64
		q.UpdatedAt = now

		data, err := json.Marshal(q)
		if err != nil {
			return nil, err
		}
		if err := batch.Set([]byte(storage.NSOutbox+q.ID), data); err != nil {
			return nil, err
		}
	}

	if err := batch.Commit(true); err != nil {
		return nil, fmt.Errorf("failed to commit signed batch states: %w", err)
	}

	return signedBatch, nil
}

// MarkBroadcasted transitions a transaction to StateBroadcasted with its on-chain hash.
func (o *OutboxManager) MarkBroadcasted(ctx context.Context, txID string, txHash string) error {
	tx, err := o.GetTx(ctx, txID)
	if err != nil {
		return err
	}

	if tx.State != StateSigned {
		return fmt.Errorf("%w: cannot mark broadcasted from state %s", ErrInvalidState, tx.State)
	}

	tx.State = StateBroadcasted
	tx.TxHash = txHash
	tx.UpdatedAt = time.Now().UnixNano()

	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	return o.store.Pebble.Set([]byte(storage.NSOutbox+txID), data, true)
}

// Confirm finalizes a transaction, transitions state to StateConfirmed, and settles the ledger transfer.
func (o *OutboxManager) Confirm(ctx context.Context, txID string, txHash string) error {
	tx, err := o.GetTx(ctx, txID)
	if err != nil {
		return err
	}

	if tx.State == StateConfirmed {
		return nil // idempotent
	}
	if tx.State != StateBroadcasted && tx.State != StateSigned {
		return fmt.Errorf("%w: cannot confirm from state %s", ErrInvalidState, tx.State)
	}

	// Settle ledger two-phase transfer (moves funds from user:locked to system:expense)
	if err := o.ledger.Settle(ctx, tx.TransferID); err != nil {
		return fmt.Errorf("failed to settle ledger transfer %s: %w", tx.TransferID, err)
	}

	tx.State = StateConfirmed
	if txHash != "" {
		tx.TxHash = txHash
	}
	tx.UpdatedAt = time.Now().UnixNano()

	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	return o.store.Pebble.Set([]byte(storage.NSOutbox+txID), data, true)
}

// Fail transitions a transaction to StateFailed, records the reason, and voids the reserved funds in the ledger.
func (o *OutboxManager) Fail(ctx context.Context, txID string, reason string) error {
	tx, err := o.GetTx(ctx, txID)
	if err != nil {
		return err
	}

	if tx.State == StateFailed {
		return nil // idempotent
	}
	if tx.State == StateConfirmed {
		return fmt.Errorf("%w: cannot fail already confirmed tx", ErrInvalidState)
	}

	// Void ledger transfer (refunds funds from user:locked back to user:main)
	if err := o.ledger.Void(ctx, tx.TransferID, reason); err != nil {
		return fmt.Errorf("failed to void ledger transfer %s: %w", tx.TransferID, err)
	}

	tx.State = StateFailed
	tx.LastError = reason
	tx.UpdatedAt = time.Now().UnixNano()

	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	return o.store.Pebble.Set([]byte(storage.NSOutbox+txID), data, true)
}
