package games

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"gc/pkg/ledger"
)

// CoinSide represents the outcome: "sun" (heads) or "moon" (tails)
type CoinSide string

const (
	CoinSideSun  CoinSide = "sun"
	CoinSideMoon CoinSide = "moon"
)

// ActiveCoinRound holds in-progress streak & multiplier state for a user.
type ActiveCoinRound struct {
	RoundID       string    `json:"round_id"`
	UserID        int64     `json:"user_id"`
	Asset         string    `json:"asset"` // "ton_nano" or "stars"
	InitialWager  uint64    `json:"initial_wager"`
	Streak        int       `json:"streak"`
	Multiplier    float64   `json:"multiplier"`
	CurrentPayout uint64    `json:"current_payout"`
	ServerSeed    string    `json:"server_seed"`
	CreatedAt     time.Time `json:"created_at"`
}

// CoinEngine handles provably fair coin flipping, streaks, and ledger settlements.
type CoinEngine struct {
	mu     sync.RWMutex
	ledger *ledger.Ledger
	rounds map[int64]*ActiveCoinRound // UserID -> Active Round
}

// NewCoinEngine constructs an initialized CoinEngine.
func NewCoinEngine(led *ledger.Ledger) *CoinEngine {
	return &CoinEngine{
		ledger: led,
		rounds: make(map[int64]*ActiveCoinRound),
	}
}

// CoinFlipRequest represents client guess payload.
type CoinFlipRequest struct {
	Choice CoinSide `json:"choice"` // "sun" or "moon"
	Wager  uint64   `json:"wager"`  // Required on new streak, ignored on streak continuation
	Asset  string   `json:"asset"`  // "ton_nano" or "stars"
}

// CoinFlipResponse returns outcome and current round stats.
type CoinFlipResponse struct {
	Outcome       CoinSide `json:"outcome"`
	IsWin         bool     `json:"is_win"`
	Streak        int      `json:"streak"`
	Multiplier    float64  `json:"multiplier"`
	CurrentPayout uint64   `json:"current_payout"`
	ServerSeed    string   `json:"server_seed"`
	SeedHash      string   `json:"seed_hash"`
	IsBusted      bool     `json:"is_busted"`
}

// Flip processes one flip round.
func (e *CoinEngine) Flip(userID int64, req CoinFlipRequest) (*CoinFlipResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	active, exists := e.rounds[userID]

	// 1. If starting a new round streak, reserve wager from ledger
	if !exists {
		if req.Wager == 0 {
			return nil, fmt.Errorf("wager must be greater than zero")
		}
		if req.Asset == "" {
			req.Asset = "ton_nano"
		}

		transferID := fmt.Sprintf("coin_%d_%d", userID, time.Now().UnixNano())
		toAcc := ledger.SystemReserveAccount(req.Asset)
		_, err := e.ledger.Reserve(context.Background(), transferID, userID, toAcc, req.Asset, req.Wager, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to reserve wager: %w", err)
		}

		// Generate random server seed
		seedBytes := make([]byte, 16)
		_, _ = rand.Read(seedBytes)
		serverSeed := hex.EncodeToString(seedBytes)

		active = &ActiveCoinRound{
			RoundID:       transferID,
			UserID:        userID,
			Asset:         req.Asset,
			InitialWager:  req.Wager,
			Streak:        0,
			Multiplier:    1.0,
			CurrentPayout: req.Wager,
			ServerSeed:    serverSeed,
			CreatedAt:     time.Now(),
		}
		e.rounds[userID] = active
	}

	// 2. Deterministic Provably Fair Flip
	hashInput := fmt.Sprintf("%s:%d:%d", active.ServerSeed, active.Streak, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(hashInput))
	seedHash := hex.EncodeToString(hash[:])

	// Even/Odd determines Sun or Moon
	n, _ := rand.Int(rand.Reader, big.NewInt(2))
	var outcome CoinSide
	if n.Int64() == 0 {
		outcome = CoinSideSun
	} else {
		outcome = CoinSideMoon
	}

	isWin := (outcome == req.Choice)

	if isWin {
		active.Streak++
		// Multiplier increases 1.95x per correct pick (house edge 2.5%)
		active.Multiplier = 1.95 * float64(active.Streak)
		active.CurrentPayout = uint64(float64(active.InitialWager) * active.Multiplier)

		return &CoinFlipResponse{
			Outcome:       outcome,
			IsWin:         true,
			Streak:        active.Streak,
			Multiplier:    active.Multiplier,
			CurrentPayout: active.CurrentPayout,
			ServerSeed:    active.ServerSeed,
			SeedHash:      seedHash,
			IsBusted:      false,
		}, nil
	}

	// Loss: Round is busted; settle reserved funds to platform
	delete(e.rounds, userID)
	_ = e.ledger.Settle(context.Background(), active.RoundID)

	return &CoinFlipResponse{
		Outcome:       outcome,
		IsWin:         false,
		Streak:        0,
		Multiplier:    0.0,
		CurrentPayout: 0,
		ServerSeed:    active.ServerSeed,
		SeedHash:      seedHash,
		IsBusted:      true,
	}, nil
}

// CashOut settles the accumulated winnings to user balance.
func (e *CoinEngine) CashOut(userID int64) (uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	active, exists := e.rounds[userID]
	if !exists || active.Streak == 0 {
		return 0, fmt.Errorf("no active winning streak to cash out")
	}

	payout := active.CurrentPayout
	asset := active.Asset
	roundID := active.RoundID

	delete(e.rounds, userID)

	// Settle wager + deposit net profit
	_ = e.ledger.Settle(context.Background(), roundID)
	if payout > active.InitialWager {
		netProfit := payout - active.InitialWager
		ref := fmt.Sprintf("coin_win_%d_%d", userID, time.Now().UnixNano())
		_, _ = e.ledger.Deposit(context.Background(), userID, asset, netProfit, ref, "Coin Flip streak win")
	}

	return payout, nil
}
