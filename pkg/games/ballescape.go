package games

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"gc/pkg/ledger"
)

const (
	MaxBallEscapeMultiplier = 100.0
	DefaultBallEscapeSecret = "ball_escape_cryptographic_secret_2026"
)

// ActiveBallEscapeRound tracks an in-flight round awaiting client outcome submission.
type ActiveBallEscapeRound struct {
	RoundID    int
	UserID     int64
	TransferID string
	Currency   string
	BetAtomic  uint64
	Difficulty string
	ServerSeed string
	Sig        string
	CreatedAt  time.Time
}

// BallEscapeStartResult is returned upon successfully starting a round.
type BallEscapeStartResult struct {
	RoundID        int    `json:"roundId"`
	ServerSeed     string `json:"serverSeed"`
	Sig            string `json:"sig"`
	Currency       string `json:"currency"`
	BetAtomic      uint64 `json:"betAtomic"`
	UpdatedBalance uint64 `json:"updatedBalance"`
}

// BallEscapeClaimResult is returned upon successfully claiming a round.
type BallEscapeClaimResult struct {
	Status         string  `json:"status"`
	RoundID        int     `json:"roundId"`
	Currency       string  `json:"currency"`
	PayoutAtomic   uint64  `json:"payoutAtomic"`
	Multiplier     float64 `json:"multiplier"`
	UpdatedBalance uint64  `json:"updatedBalance"`
}

// BallEscapeEngine coordinates provably fair rounds and double-entry settlements.
type BallEscapeEngine struct {
	ledger       *ledger.Ledger
	hmacSecret   []byte
	roundCounter int
	activeRounds map[int]ActiveBallEscapeRound
	mu           sync.RWMutex
}

// NewBallEscapeEngine initializes the Ball Escape game engine.
func NewBallEscapeEngine(led *ledger.Ledger, secret string) *BallEscapeEngine {
	if secret == "" {
		secret = DefaultBallEscapeSecret
	}
	return &BallEscapeEngine{
		ledger:       led,
		hmacSecret:   []byte(secret),
		roundCounter: 6000,
		activeRounds: make(map[int]ActiveBallEscapeRound),
	}
}

// generateSig computes HMAC-SHA256(roundId:userId:serverSeed:currency:betAtomic)
func (e *BallEscapeEngine) generateSig(roundID int, userID int64, serverSeed string, currency string, betAtomic uint64) string {
	mac := hmac.New(sha256.New, e.hmacSecret)
	mac.Write([]byte(fmt.Sprintf("%d:%d:%s:%s:%d", roundID, userID, serverSeed, currency, betAtomic)))
	return hex.EncodeToString(mac.Sum(nil))
}

// StartRound locks the user's wager in the ledger and issues cryptographic seed & signature.
func (e *BallEscapeEngine) StartRound(ctx context.Context, userID int64, currency string, betAtomic uint64, difficulty string) (*BallEscapeStartResult, error) {
	currency = NormalizeCurrency(currency)
	if betAtomic == 0 {
		return nil, ledger.ErrInvalidAmount
	}

	if currency == ledger.AssetStars {
		if betAtomic < MinBetAmountStars || betAtomic > MaxBetAmountStars {
			return nil, fmt.Errorf("stars bet must be between %d and %d", MinBetAmountStars, MaxBetAmountStars)
		}
	} else if currency == ledger.AssetTON {
		if betAtomic < MinBetAmountTONNano || betAtomic > MaxBetAmountTONNano {
			return nil, fmt.Errorf("TON bet must be between %s and %s",
				ledger.FormatAtomic(MinBetAmountTONNano, ledger.AssetTON),
				ledger.FormatAtomic(MaxBetAmountTONNano, ledger.AssetTON))
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Clean up stale rounds older than 1 hour
	now := time.Now()
	for id, r := range e.activeRounds {
		if now.Sub(r.CreatedAt) > time.Hour {
			// Void stale reservation to refund user
			_ = e.ledger.Void(ctx, r.TransferID, "stale_round_timeout")
			delete(e.activeRounds, id)
		}
	}

	e.roundCounter++
	roundID := e.roundCounter

	// Reserve wager in double-entry ledger (user available -> user locked)
	transferID := fmt.Sprintf("be_xfr_%d_%d", roundID, now.UnixNano())
	destAcc := ledger.SystemFeeAccount(currency)
	_, err := e.ledger.Reserve(ctx, transferID, userID, destAcc, currency, betAtomic, map[string]string{
		"game":     "ball_escape",
		"round_id": fmt.Sprintf("%d", roundID),
	})
	if err != nil {
		return nil, fmt.Errorf("insufficient balance or reserve failed: %w", err)
	}

	// Generate cryptographic seed
	randBytes := make([]byte, 16)
	_, _ = rand.Read(randBytes)
	serverSeed := fmt.Sprintf("be_seed_%d_%s", roundID, hex.EncodeToString(randBytes))

	sig := e.generateSig(roundID, userID, serverSeed, currency, betAtomic)

	e.activeRounds[roundID] = ActiveBallEscapeRound{
		RoundID:    roundID,
		UserID:     userID,
		TransferID: transferID,
		Currency:   currency,
		BetAtomic:  betAtomic,
		Difficulty: difficulty,
		ServerSeed: serverSeed,
		Sig:        sig,
		CreatedAt:  now,
	}

	// Query updated balance
	userAcc, _ := e.ledger.GetAccount(ctx, ledger.UserMainAccount(userID, currency))
	var avail uint64
	if userAcc != nil {
		avail = userAcc.Available
	}

	return &BallEscapeStartResult{
		RoundID:        roundID,
		ServerSeed:     serverSeed,
		Sig:            sig,
		Currency:       currency,
		BetAtomic:      betAtomic,
		UpdatedBalance: avail,
	}, nil
}

// ClaimRound verifies the HMAC signature and atomically settles or credits winnings.
func (e *BallEscapeEngine) ClaimRound(ctx context.Context, userID int64, roundID int, serverSeed string, sig string, multiplier float64, isWin bool, betAtomic uint64) (*BallEscapeClaimResult, error) {
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier < 0 || multiplier > MaxBallEscapeMultiplier {
		return nil, errors.New("invalid multiplier")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	active, exists := e.activeRounds[roundID]
	if !exists {
		return nil, errors.New("round not found or already claimed")
	}

	if active.UserID != userID || active.ServerSeed != serverSeed || active.BetAtomic != betAtomic {
		return nil, errors.New("round parameter mismatch")
	}

	// Constant-time HMAC signature verification
	expectedSig := e.generateSig(roundID, userID, serverSeed, active.Currency, active.BetAtomic)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil, errors.New("cryptographic signature mismatch")
	}

	// Consume round immediately to guarantee zero-replay safety
	delete(e.activeRounds, roundID)

	var payoutAtomic uint64
	if isWin && multiplier > 0 {
		// Calculate payout: atomic amount * multiplier
		payoutFloat := math.Round(float64(active.BetAtomic) * multiplier)
		payoutAtomic = uint64(payoutFloat)

		// 1. Settle original wager (clears user locked liability)
		if err := e.ledger.Settle(ctx, active.TransferID); err != nil {
			return nil, fmt.Errorf("failed to settle round wager: %w", err)
		}

		// 2. Deposit total win payout to user account
		ref := fmt.Sprintf("be_win_%d", roundID)
		desc := fmt.Sprintf("Ball Escape win: Round %d (%.2fx)", roundID, multiplier)
		if _, err := e.ledger.Deposit(ctx, userID, active.Currency, payoutAtomic, ref, desc); err != nil {
			return nil, fmt.Errorf("failed to deposit win payout: %w", err)
		}
	} else {
		// Loss: Settle original wager to platform reserve/fee account
		if err := e.ledger.Settle(ctx, active.TransferID); err != nil {
			return nil, fmt.Errorf("failed to settle loss wager: %w", err)
		}
	}

	// Query updated balance
	userAcc, _ := e.ledger.GetAccount(ctx, ledger.UserMainAccount(userID, active.Currency))
	var avail uint64
	if userAcc != nil {
		avail = userAcc.Available
	}

	return &BallEscapeClaimResult{
		Status:         "claimed",
		RoundID:        roundID,
		Currency:       active.Currency,
		PayoutAtomic:   payoutAtomic,
		Multiplier:     multiplier,
		UpdatedBalance: avail,
	}, nil
}
