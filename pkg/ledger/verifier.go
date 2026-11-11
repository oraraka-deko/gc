package ledger

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gc/pkg/storage"
	"gc/pkg/telegram"
)

var (
	ErrDuplicatePayment = errors.New("verifier: payment already processed")
	ErrInvalidMemo      = errors.New("verifier: unable to parse user ID from deposit memo")
)

// IncomePipeline coordinates multi-asset payment verification and atomic ledger crediting.
type IncomePipeline struct {
	ledger *Ledger
	store  *storage.Storage
}

// NewIncomePipeline creates a new income verification pipeline.
func NewIncomePipeline(ledger *Ledger, store *storage.Storage) *IncomePipeline {
	return &IncomePipeline{
		ledger: ledger,
		store:  store,
	}
}

// ValidatePreCheckout validates an incoming Telegram Stars pre_checkout_query.
// Returns (ok, errorMessage, error).
func (p *IncomePipeline) ValidatePreCheckout(ctx context.Context, query *telegram.PreCheckoutQuery) (bool, string, error) {
	if query == nil {
		return false, "empty query", errors.New("empty pre checkout query")
	}

	// Telegram Stars currency code is always "XTR"
	if query.Currency != AssetStars {
		return false, fmt.Sprintf("unsupported currency %s, expected %s", query.Currency, AssetStars), nil
	}

	if query.TotalAmount <= 0 {
		return false, "invalid amount", nil
	}

	// Check if this payload has already been finalized
	if query.InvoicePayload != "" {
		if _, ok := p.store.Fast.Get("stars:payload:" + query.InvoicePayload); ok {
			return false, "invoice already processed", nil
		}
	}

	return true, "", nil
}

// ProcessStarsPayment processes a confirmed Telegram Stars SuccessfulPayment.
// Atomically credits the user's Stars (XTR) balance in the ledger.
func (p *IncomePipeline) ProcessStarsPayment(ctx context.Context, payment *telegram.SuccessfulPayment, fromUser *telegram.User) (*JournalEntry, error) {
	if payment == nil || fromUser == nil {
		return nil, errors.New("payment or user cannot be nil")
	}

	chargeID := payment.TelegramPaymentChargeID
	if chargeID == "" {
		chargeID = payment.ProviderPaymentChargeID
	}
	if chargeID == "" {
		chargeID = fmt.Sprintf("stars_%d_%d", fromUser.ID, time.Now().UnixNano())
	}

	// Idempotency check via FastCache + Ledger
	cacheKey := "payment:stars:" + chargeID
	if !p.store.Fast.SetIdempotency(cacheKey, []byte("processed"), 48*time.Hour) {
		return nil, ErrDuplicatePayment
	}

	amount := uint64(payment.TotalAmount)
	ref := "stars:" + chargeID
	username := ""
	if fromUser.Username != nil {
		username = *fromUser.Username
	}
	desc := fmt.Sprintf("Telegram Stars payment by %d (%s): %d XTR", fromUser.ID, username, amount)

	entry, err := p.ledger.Deposit(ctx, fromUser.ID, AssetStars, amount, ref, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to credit stars to ledger: %w", err)
	}

	if payment.InvoicePayload != "" {
		p.store.Fast.Set("stars:payload:"+payment.InvoicePayload, []byte(chargeID), 48*time.Hour)
	}

	return entry, nil
}

// ParseUserFromMemo extracts user ID from various memo formats:
// "12345", "user_12345", "user:12345", "uid:12345", "u12345"
var memoRegex = regexp.MustCompile(`(?i)^(?:user[:_]?|uid[:_]?|u)?([0-9]{4,15})$`)

func ParseUserFromMemo(memo string) (int64, error) {
	memo = strings.TrimSpace(memo)
	matches := memoRegex.FindStringSubmatch(memo)
	if len(matches) < 2 {
		return 0, fmt.Errorf("%w: %q", ErrInvalidMemo, memo)
	}

	uid, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user id in memo: %w", err)
	}
	return uid, nil
}

// ProcessTONDeposit handles incoming on-chain TON coin transfers.
func (p *IncomePipeline) ProcessTONDeposit(ctx context.Context, txHash string, fromAddress, toAddress string, amountNano uint64, memo string) (*JournalEntry, error) {
	if txHash == "" {
		return nil, errors.New("txHash cannot be empty")
	}
	if amountNano == 0 {
		return nil, ErrInvalidAmount
	}

	// Idempotency check
	cacheKey := "deposit:ton:" + txHash
	if !p.store.Fast.SetIdempotency(cacheKey, []byte("processed"), 7*24*time.Hour) {
		return nil, ErrDuplicatePayment
	}

	userID, err := ParseUserFromMemo(memo)
	if err != nil {
		return nil, err
	}

	ref := "ton:" + txHash
	desc := fmt.Sprintf("On-chain TON deposit from %s: %s (memo: %s)", fromAddress, FormatAtomic(amountNano, AssetTON), memo)

	entry, err := p.ledger.Deposit(ctx, userID, AssetTON, amountNano, ref, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to deposit TON into ledger: %w", err)
	}

	return entry, nil
}

// ProcessJettonDeposit handles incoming Jetton notifications (e.g. USDT on TON).
func (p *IncomePipeline) ProcessJettonDeposit(ctx context.Context, txHash string, jettonSymbol string, fromAddress string, amountAtomic uint64, memo string) (*JournalEntry, error) {
	if txHash == "" {
		return nil, errors.New("txHash cannot be empty")
	}
	if amountAtomic == 0 {
		return nil, ErrInvalidAmount
	}

	symbol := strings.ToUpper(jettonSymbol)
	cacheKey := fmt.Sprintf("deposit:jetton:%s:%s", symbol, txHash)
	if !p.store.Fast.SetIdempotency(cacheKey, []byte("processed"), 7*24*time.Hour) {
		return nil, ErrDuplicatePayment
	}

	userID, err := ParseUserFromMemo(memo)
	if err != nil {
		return nil, err
	}

	ref := fmt.Sprintf("jetton:%s:%s", symbol, txHash)
	desc := fmt.Sprintf("On-chain Jetton %s deposit from %s: %s (memo: %s)", symbol, fromAddress, FormatAtomic(amountAtomic, symbol), memo)

	entry, err := p.ledger.Deposit(ctx, userID, symbol, amountAtomic, ref, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to deposit %s into ledger: %w", symbol, err)
	}

	return entry, nil
}
