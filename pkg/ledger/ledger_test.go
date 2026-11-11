package ledger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"gc/pkg/storage"
)

func setupTestLedger(t *testing.T) (*Ledger, func()) {
	tmpDir, err := os.MkdirTemp("", "ledger_test_*")
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

	led := New(store)

	cleanup := func() {
		_ = store.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return led, cleanup
}

func TestLedger_InvariantEnforcement(t *testing.T) {
	led, cleanup := setupTestLedger(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Unbalanced entry (debit 100 != credit 90)
	unbalanced := NewJournalEntry("jnl:1", "ref:1", "Unbalanced", []Posting{
		{AccountID: SystemReserveAccount(AssetTON), Asset: AssetTON, Amount: 100, Direction: Debit},
		{AccountID: UserMainAccount(1, AssetTON), Asset: AssetTON, Amount: 90, Direction: Credit},
	})
	err := led.RecordJournalEntry(ctx, unbalanced)
	if err == nil {
		t.Fatalf("expected ErrUnbalancedEntry, got nil")
	}

	// 2. Multi-asset unbalance (TON balanced, USDT unbalanced)
	multiAssetUnbalanced := NewJournalEntry("jnl:2", "ref:2", "Multi Unbalanced", []Posting{
		{AccountID: SystemReserveAccount(AssetTON), Asset: AssetTON, Amount: 100, Direction: Debit},
		{AccountID: UserMainAccount(1, AssetTON), Asset: AssetTON, Amount: 100, Direction: Credit},
		{AccountID: SystemReserveAccount(AssetUSDT), Asset: AssetUSDT, Amount: 50, Direction: Debit},
		{AccountID: UserMainAccount(1, AssetUSDT), Asset: AssetUSDT, Amount: 40, Direction: Credit},
	})
	err = led.RecordJournalEntry(ctx, multiAssetUnbalanced)
	if err == nil {
		t.Fatalf("expected ErrUnbalancedEntry on multi-asset, got nil")
	}

	// 3. Zero amount posting
	zeroAmount := NewJournalEntry("jnl:3", "ref:3", "Zero Amount", []Posting{
		{AccountID: SystemReserveAccount(AssetTON), Asset: AssetTON, Amount: 0, Direction: Debit},
		{AccountID: UserMainAccount(1, AssetTON), Asset: AssetTON, Amount: 0, Direction: Credit},
	})
	err = led.RecordJournalEntry(ctx, zeroAmount)
	if err == nil {
		t.Fatalf("expected ErrInvalidAmount, got nil")
	}
}

func TestLedger_DepositWithdrawOverdraft(t *testing.T) {
	led, cleanup := setupTestLedger(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(1001)

	// 1. Deposit 5 TON (5 * 10^9 nanoTON)
	depositAmount := TONToNano(5.0)
	_, err := led.Deposit(ctx, userID, AssetTON, depositAmount, "dep:101", "Initial deposit")
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// Verify balance
	acc, err := led.GetAccount(ctx, UserMainAccount(userID, AssetTON))
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}
	if acc.Available != depositAmount {
		t.Fatalf("expected available %d, got %d", depositAmount, acc.Available)
	}

	// Verify system reserve
	resAcc, err := led.GetAccount(ctx, SystemReserveAccount(AssetTON))
	if err != nil {
		t.Fatalf("failed to get reserve account: %v", err)
	}
	if resAcc.Available != depositAmount {
		t.Fatalf("expected reserve %d, got %d", depositAmount, resAcc.Available)
	}

	// 2. Overdraft attempt: try to withdraw 6 TON
	overdraftAmount := TONToNano(6.0)
	_, err = led.Withdraw(ctx, userID, AssetTON, overdraftAmount, "wdr:102", "Overdraft withdrawal")
	if err == nil {
		t.Fatalf("expected overdraft error, got nil")
	}

	// 3. Valid withdrawal: withdraw 2 TON
	validWithdraw := TONToNano(2.0)
	_, err = led.Withdraw(ctx, userID, AssetTON, validWithdraw, "wdr:103", "Valid withdrawal")
	if err != nil {
		t.Fatalf("withdraw failed: %v", err)
	}

	// Balance should now be 3 TON
	acc, err = led.GetAccount(ctx, UserMainAccount(userID, AssetTON))
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}
	expectedRem := TONToNano(3.0)
	if acc.Available != expectedRem {
		t.Fatalf("expected remaining %d, got %d", expectedRem, acc.Available)
	}
}

func TestLedger_TwoPhaseTransfer_Settle(t *testing.T) {
	led, cleanup := setupTestLedger(t)
	defer cleanup()

	ctx := context.Background()
	userA := int64(2001)
	userB := int64(2002)

	// Deposit 100 USDT (100 * 10^6 microUSDT)
	initialUSDT := USDTToMicro(100.0)
	_, err := led.Deposit(ctx, userA, AssetUSDT, initialUSDT, "dep:201", "User A deposit")
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// 1. Reserve 40 USDT for transfer to User B
	transferID := "xfr:tx:555"
	reserveAmount := USDTToMicro(40.0)
	pt, err := led.Reserve(ctx, transferID, userA, UserMainAccount(userB, AssetUSDT), AssetUSDT, reserveAmount, map[string]string{
		"purpose": "p2p_payment",
	})
	if err != nil {
		t.Fatalf("reserve failed: %v", err)
	}
	if pt.Status != TransferPending {
		t.Fatalf("expected status PENDING, got %s", pt.Status)
	}

	// Check User A main balance: should be 60 USDT
	accA, err := led.GetAccount(ctx, UserMainAccount(userA, AssetUSDT))
	if err != nil || accA.Available != USDTToMicro(60.0) {
		t.Fatalf("expected 60 USDT available, got %d", accA.Available)
	}

	// Check User A locked balance: should be 40 USDT
	accALocked, err := led.GetAccount(ctx, UserLockedAccount(userA, AssetUSDT))
	if err != nil || accALocked.Available != reserveAmount {
		t.Fatalf("expected 40 USDT locked, got %d", accALocked.Available)
	}

	// 2. Settle transfer
	if err := led.Settle(ctx, transferID); err != nil {
		t.Fatalf("settle failed: %v", err)
	}

	// Check User A locked balance: should be 0
	accALocked, err = led.GetAccount(ctx, UserLockedAccount(userA, AssetUSDT))
	if err != nil || accALocked.Available != 0 {
		t.Fatalf("expected 0 USDT locked, got %d", accALocked.Available)
	}

	// Check User B main balance: should be 40 USDT
	accB, err := led.GetAccount(ctx, UserMainAccount(userB, AssetUSDT))
	if err != nil || accB.Available != reserveAmount {
		t.Fatalf("expected 40 USDT on User B, got %d", accB.Available)
	}

	// Settle again should fail (not pending)
	if err := led.Settle(ctx, transferID); err == nil {
		t.Fatalf("expected re-settle to fail")
	}
}

func TestLedger_TwoPhaseTransfer_Void(t *testing.T) {
	led, cleanup := setupTestLedger(t)
	defer cleanup()

	ctx := context.Background()
	userA := int64(3001)

	// Deposit 500 Stars (XTR)
	starsAmount := uint64(500)
	_, err := led.Deposit(ctx, userA, AssetStars, starsAmount, "dep:301", "Stars deposit")
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// 1. Reserve 200 Stars
	transferID := "xfr:tx:777"
	reserveAmount := uint64(200)
	_, err = led.Reserve(ctx, transferID, userA, SystemFeeAccount(AssetStars), AssetStars, reserveAmount, nil)
	if err != nil {
		t.Fatalf("reserve failed: %v", err)
	}

	// User main should have 300 Stars
	accA, _ := led.GetAccount(ctx, UserMainAccount(userA, AssetStars))
	if accA.Available != 300 {
		t.Fatalf("expected 300 Stars available, got %d", accA.Available)
	}

	// 2. Void transfer (simulate outbox timeout or abort)
	if err := led.Void(ctx, transferID, "user cancelled order"); err != nil {
		t.Fatalf("void failed: %v", err)
	}

	// User main should be restored to 500 Stars
	accA, _ = led.GetAccount(ctx, UserMainAccount(userA, AssetStars))
	if accA.Available != 500 {
		t.Fatalf("expected 500 Stars restored, got %d", accA.Available)
	}

	// User locked should be 0
	accALocked, _ := led.GetAccount(ctx, UserLockedAccount(userA, AssetStars))
	if accALocked.Available != 0 {
		t.Fatalf("expected 0 Stars locked, got %d", accALocked.Available)
	}
}

func TestLedger_AuditHistory(t *testing.T) {
	led, cleanup := setupTestLedger(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(4001)

	// Make 3 deposits
	for i := 1; i <= 3; i++ {
		ref := fmt.Sprintf("dep:hist:%d", i)
		_, err := led.Deposit(ctx, userID, AssetTON, TONToNano(float64(i)), ref, fmt.Sprintf("Deposit #%d", i))
		if err != nil {
			t.Fatalf("deposit %d failed: %v", i, err)
		}
	}

	history, err := led.GetAccountHistory(ctx, UserMainAccount(userID, AssetTON), 10)
	if err != nil {
		t.Fatalf("get history failed: %v", err)
	}

	if len(history) != 3 {
		t.Fatalf("expected 3 history entries, got %d", len(history))
	}
}

func TestLedger_ConcurrentTransfers(t *testing.T) {
	led, cleanup := setupTestLedger(t)
	defer cleanup()

	ctx := context.Background()
	userA := int64(5001)
	userB := int64(5002)

	// Deposit 1,000 Stars into User A
	initialAmount := uint64(1000)
	_, err := led.Deposit(ctx, userA, AssetStars, initialAmount, "dep:conc:init", "Initial Stars")
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// Concurrently attempt 10 transfers of 100 Stars each (exact total = 1000)
	const workers = 10
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ref := fmt.Sprintf("conc:xfr:%d", idx)
			_, err := led.InternalTransfer(ctx, userA, userB, AssetStars, 100, ref, fmt.Sprintf("Transfer %d", idx))
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent transfer error: %v", err)
	}

	// User A should now have 0 Stars
	accA, _ := led.GetAccount(ctx, UserMainAccount(userA, AssetStars))
	if accA.Available != 0 {
		t.Fatalf("expected 0 Stars on User A, got %d", accA.Available)
	}

	// User B should now have 1000 Stars
	accB, _ := led.GetAccount(ctx, UserMainAccount(userB, AssetStars))
	if accB.Available != 1000 {
		t.Fatalf("expected 1000 Stars on User B, got %d", accB.Available)
	}
}
