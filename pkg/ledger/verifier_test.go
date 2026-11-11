package ledger

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gc/pkg/storage"
	"gc/pkg/telegram"
)

func setupTestPipeline(t *testing.T) (*IncomePipeline, *Ledger, func()) {
	tmpDir, err := os.MkdirTemp("", "verifier_test_*")
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
	pipeline := NewIncomePipeline(led, store)

	cleanup := func() {
		_ = store.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return pipeline, led, cleanup
}

func TestIncomePipeline_StarsPayment(t *testing.T) {
	pipeline, led, cleanup := setupTestPipeline(t)
	defer cleanup()

	ctx := context.Background()
	user := &telegram.User{
		ID:        777000,
		Username:  telegram.StringPtr("crypto_whale"),
		FirstName: "Whale",
	}

	// 1. Pre checkout query validation
	preQuery := &telegram.PreCheckoutQuery{
		ID:             "pcq:123",
		Currency:       AssetStars,
		TotalAmount:    250,
		InvoicePayload: "order_invoice_99",
	}

	ok, errMsg, err := pipeline.ValidatePreCheckout(ctx, preQuery)
	if err != nil || !ok || errMsg != "" {
		t.Fatalf("expected pre checkout to succeed, got ok=%v, errMsg=%s, err=%v", ok, errMsg, err)
	}

	// Invalid currency
	invalidCurrencyQuery := &telegram.PreCheckoutQuery{
		ID:          "pcq:124",
		Currency:    "EUR",
		TotalAmount: 250,
	}
	ok, errMsg, _ = pipeline.ValidatePreCheckout(ctx, invalidCurrencyQuery)
	if ok {
		t.Fatalf("expected non-XTR currency to fail validation")
	}

	// 2. Successful Payment
	payment := &telegram.SuccessfulPayment{
		Currency:                AssetStars,
		TotalAmount:             250,
		InvoicePayload:          "order_invoice_99",
		TelegramPaymentChargeID: "chg_stars_1001",
	}

	entry, err := pipeline.ProcessStarsPayment(ctx, payment, user)
	if err != nil {
		t.Fatalf("failed to process stars payment: %v", err)
	}
	if entry.Reference != "stars:chg_stars_1001" {
		t.Fatalf("unexpected entry ref: %s", entry.Reference)
	}

	// Verify user balance has 250 Stars
	acc, err := led.GetAccount(ctx, UserMainAccount(user.ID, AssetStars))
	if err != nil || acc.Available != 250 {
		t.Fatalf("expected 250 Stars on user account, got %d", acc.Available)
	}

	// Duplicate payment should be rejected
	_, err = pipeline.ProcessStarsPayment(ctx, payment, user)
	if err != ErrDuplicatePayment {
		t.Fatalf("expected ErrDuplicatePayment on replay, got: %v", err)
	}
}

func TestIncomePipeline_MemoParsing(t *testing.T) {
	testCases := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"123456", 123456, false},
		{"user_998877", 998877, false},
		{"user:554433", 554433, false},
		{"uid:112233", 112233, false},
		{"u778899", 778899, false},
		{"invalid_memo", 0, true},
		{"", 0, true},
		{"user_", 0, true},
		{"12", 0, true}, // too short
	}

	for _, tc := range testCases {
		uid, err := ParseUserFromMemo(tc.input)
		if tc.hasError && err == nil {
			t.Fatalf("expected error for %q, got uid=%d", tc.input, uid)
		}
		if !tc.hasError && (err != nil || uid != tc.expected) {
			t.Fatalf("for %q expected %d, got %d (err: %v)", tc.input, tc.expected, uid, err)
		}
	}
}

func TestIncomePipeline_TONAndJettonDeposits(t *testing.T) {
	pipeline, led, cleanup := setupTestPipeline(t)
	defer cleanup()

	ctx := context.Background()

	// 1. TON Deposit
	txHash := "0xabc123tonhash"
	userID := int64(888111)
	tonAmount := TONToNano(10.5)

	_, err := pipeline.ProcessTONDeposit(ctx, txHash, "EQSender123", "EQDepositVault", tonAmount, "user_888111")
	if err != nil {
		t.Fatalf("TON deposit failed: %v", err)
	}

	accTON, err := led.GetAccount(ctx, UserMainAccount(userID, AssetTON))
	if err != nil || accTON.Available != tonAmount {
		t.Fatalf("expected %d nanoTON on user account, got %d", tonAmount, accTON.Available)
	}

	// Replay TON deposit
	_, err = pipeline.ProcessTONDeposit(ctx, txHash, "EQSender123", "EQDepositVault", tonAmount, "user_888111")
	if err != ErrDuplicatePayment {
		t.Fatalf("expected ErrDuplicatePayment on replay, got: %v", err)
	}

	// 2. Jetton USDT Deposit
	usdtHash := "0xdef456usdthash"
	usdtAmount := USDTToMicro(50.0) // 50 USDT

	_, err = pipeline.ProcessJettonDeposit(ctx, usdtHash, AssetUSDT, "EQSender456", usdtAmount, "888111")
	if err != nil {
		t.Fatalf("Jetton deposit failed: %v", err)
	}

	accUSDT, err := led.GetAccount(ctx, UserMainAccount(userID, AssetUSDT))
	if err != nil || accUSDT.Available != usdtAmount {
		t.Fatalf("expected %d microUSDT on user account, got %d", usdtAmount, accUSDT.Available)
	}
}
