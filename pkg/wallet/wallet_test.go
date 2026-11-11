package wallet

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gc/pkg/ledger"
	"gc/pkg/storage"
)

func setupTestWallet(t *testing.T) (*OutboxManager, *ledger.Ledger, *MockSigner, func()) {
	tmpDir, err := os.MkdirTemp("", "wallet_test_*")
	if err != nil {
		t.Fatal(err)
	}

	store, err := storage.New(storage.Config{
		PebblePath:     filepath.Join(tmpDir, "pebble"),
		FastCacheBytes: 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	led := ledger.New(store)
	signer, err := NewMockSigner()
	if err != nil {
		t.Fatalf("failed to create mock signer: %v", err)
	}

	outbox := NewOutboxManager(store, led, signer, OutboxConfig{
		MaxBatchSize: 10,
		BatchTimeout: 100 * time.Millisecond,
	})

	cleanup := func() {
		_ = store.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return outbox, led, signer, cleanup
}

func TestMockSigner_Basic(t *testing.T) {
	signer, err := NewMockSigner()
	if err != nil {
		t.Fatalf("failed to init mock signer: %v", err)
	}
	defer signer.Close()

	if signer.PublicKey() == "" {
		t.Fatalf("expected non-empty public key")
	}

	transfers := []OutboxTx{
		{
			ID:                 "tx_1",
			Asset:              ledger.AssetTON,
			Amount:             ledger.TONToNano(1.0),
			DestinationAddress: "EQTestDestination1",
		},
		{
			ID:                 "tx_2",
			Asset:              ledger.AssetTON,
			Amount:             ledger.TONToNano(2.5),
			DestinationAddress: "EQTestDestination2",
		},
	}

	batch, err := signer.SignBatch(context.Background(), transfers)
	if err != nil {
		t.Fatalf("failed to sign batch: %v", err)
	}

	if len(batch.TxIDs) != 2 || batch.BOCBase64 == "" || batch.BatchID == "" {
		t.Fatalf("invalid signed batch output: %+v", batch)
	}
}

func TestIPCSignerClient_WithServer(t *testing.T) {
	// Start daemon server on random port
	server, err := NewSignerServer("127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("failed to start signer server: %v", err)
	}
	defer server.Stop()

	// Connect client
	client, err := NewIPCSignerClient(server.Addr())
	if err != nil {
		t.Fatalf("failed to connect IPC client: %v", err)
	}
	defer client.Close()

	if client.PublicKey() != server.PublicKey() {
		t.Fatalf("public key mismatch: server=%s, client=%s", server.PublicKey(), client.PublicKey())
	}

	transfers := []OutboxTx{
		{
			ID:                 "ipc_tx_1",
			Asset:              ledger.AssetUSDT,
			Amount:             ledger.USDTToMicro(15.0),
			DestinationAddress: "EQUSDTRecipient",
		},
	}

	batch, err := client.SignBatch(context.Background(), transfers)
	if err != nil {
		t.Fatalf("failed to sign batch via IPC: %v", err)
	}

	if len(batch.TxIDs) != 1 || batch.BOCBase64 == "" {
		t.Fatalf("invalid IPC batch: %+v", batch)
	}
}

func TestOutboxManager_FullLifecycle_Confirm(t *testing.T) {
	outbox, led, _, cleanup := setupTestWallet(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(9001)

	// 1. Deposit 10 TON to user
	_, err := led.Deposit(ctx, userID, ledger.AssetTON, ledger.TONToNano(10.0), "dep:wallet:1", "Deposit")
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// 2. Enqueue outbound transaction for 4 TON
	transferAmount := ledger.TONToNano(4.0)
	tx, err := outbox.Enqueue(ctx, userID, ledger.AssetTON, transferAmount, "EQDestWallet999", "withdrawal memo")
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	if tx.State != StateQueued {
		t.Fatalf("expected state QUEUED, got %s", tx.State)
	}

	// User main balance should now be 6 TON, locked 4 TON
	acc, _ := led.GetAccount(ctx, ledger.UserMainAccount(userID, ledger.AssetTON))
	if acc.Available != ledger.TONToNano(6.0) {
		t.Fatalf("expected 6 TON available, got %d", acc.Available)
	}
	accLocked, _ := led.GetAccount(ctx, ledger.UserLockedAccount(userID, ledger.AssetTON))
	if accLocked.Available != transferAmount {
		t.Fatalf("expected 4 TON locked, got %d", accLocked.Available)
	}

	// 3. Process batch
	batch, err := outbox.ProcessNextBatch(ctx)
	if err != nil {
		t.Fatalf("process next batch failed: %v", err)
	}
	if len(batch.TxIDs) != 1 || batch.TxIDs[0] != tx.ID {
		t.Fatalf("batch does not contain txID")
	}

	// Check tx state is now SIGNED
	updatedTx, err := outbox.GetTx(ctx, tx.ID)
	if err != nil || updatedTx.State != StateSigned || updatedTx.SignedBOC == "" {
		t.Fatalf("expected state SIGNED with BOC, got: %+v", updatedTx)
	}

	// 4. Mark Broadcasted
	fakeTxHash := "0xhash999ton"
	if err := outbox.MarkBroadcasted(ctx, tx.ID, fakeTxHash); err != nil {
		t.Fatalf("mark broadcasted failed: %v", err)
	}

	// 5. Confirm
	if err := outbox.Confirm(ctx, tx.ID, fakeTxHash); err != nil {
		t.Fatalf("confirm failed: %v", err)
	}

	// User locked should now be 0
	accLocked, _ = led.GetAccount(ctx, ledger.UserLockedAccount(userID, ledger.AssetTON))
	if accLocked.Available != 0 {
		t.Fatalf("expected 0 TON locked after settle, got %d", accLocked.Available)
	}

	confirmedTx, err := outbox.GetTx(ctx, tx.ID)
	if err != nil || confirmedTx.State != StateConfirmed {
		t.Fatalf("expected state CONFIRMED, got: %+v", confirmedTx)
	}
}

func TestOutboxManager_FailureRefundFlow(t *testing.T) {
	outbox, led, _, cleanup := setupTestWallet(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(9002)

	// Deposit 500 Stars
	_, err := led.Deposit(ctx, userID, ledger.AssetStars, 500, "dep:stars:9002", "Deposit")
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// Enqueue 300 Stars
	tx, err := outbox.Enqueue(ctx, userID, ledger.AssetStars, 300, "stars_refund_target", "gift")
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	// Available balance is 200 Stars
	acc, _ := led.GetAccount(ctx, ledger.UserMainAccount(userID, ledger.AssetStars))
	if acc.Available != 200 {
		t.Fatalf("expected 200 Stars available, got %d", acc.Available)
	}

	// Fail the transaction (e.g. target address rejected or network timeout)
	if err := outbox.Fail(ctx, tx.ID, "destination recipient blocked"); err != nil {
		t.Fatalf("fail failed: %v", err)
	}

	// Check that funds are refunded back to main account (500 Stars)
	acc, _ = led.GetAccount(ctx, ledger.UserMainAccount(userID, ledger.AssetStars))
	if acc.Available != 500 {
		t.Fatalf("expected 500 Stars refunded, got %d", acc.Available)
	}

	failedTx, err := outbox.GetTx(ctx, tx.ID)
	if err != nil || failedTx.State != StateFailed || failedTx.LastError == "" {
		t.Fatalf("expected state FAILED with error, got: %+v", failedTx)
	}
}
