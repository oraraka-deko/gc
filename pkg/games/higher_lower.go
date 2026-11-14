package games

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"gc/pkg/ledger"
)

// ActiveHLRound holds Higher/Lower card state.
type ActiveHLRound struct {
	RoundID       string
	UserID        int64
	Asset         string
	InitialWager  uint64
	CurrentCard   int // 1..13 (Ace..King)
	Streak        int
	Multiplier    float64
	CurrentPayout uint64
}

// HigherLowerEngine manages card predictions, streak multipliers, and cashouts.
type HigherLowerEngine struct {
	mu     sync.RWMutex
	ledger *ledger.Ledger
	rounds map[int64]*ActiveHLRound
}

func NewHigherLowerEngine(led *ledger.Ledger) *HigherLowerEngine {
	return &HigherLowerEngine{
		ledger: led,
		rounds: make(map[int64]*ActiveHLRound),
	}
}

func randomCard() int {
	n, _ := rand.Int(rand.Reader, big.NewInt(13))
	return int(n.Int64()) + 1 // 1..13
}

// StartRound initiates a round and draws the first card.
func (e *HigherLowerEngine) StartRound(userID int64, asset string, wager uint64) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	transferID := fmt.Sprintf("hl_%d_%d", userID, time.Now().UnixNano())
	toAcc := ledger.SystemReserveAccount(asset)
	_, err := e.ledger.Reserve(context.Background(), transferID, userID, toAcc, asset, wager, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to reserve wager: %w", err)
	}

	card := randomCard()
	e.rounds[userID] = &ActiveHLRound{
		RoundID:       transferID,
		UserID:        userID,
		Asset:         asset,
		InitialWager:  wager,
		CurrentCard:   card,
		Streak:        0,
		Multiplier:    1.0,
		CurrentPayout: wager,
	}

	return card, nil
}

// Guess processes Higher or Lower choice.
func (e *HigherLowerEngine) Guess(userID int64, guessHigher bool) (int, bool, float64, uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	round, ok := e.rounds[userID]
	if !ok {
		return 0, false, 0, 0, fmt.Errorf("no active round")
	}

	nextCard := randomCard()
	for nextCard == round.CurrentCard {
		nextCard = randomCard() // Re-draw on equal to avoid ties
	}

	isWin := (guessHigher && nextCard > round.CurrentCard) || (!guessHigher && nextCard < round.CurrentCard)

	if isWin {
		round.Streak++
		round.CurrentCard = nextCard
		round.Multiplier = 1.0 + (float64(round.Streak) * 0.45)
		round.CurrentPayout = uint64(float64(round.InitialWager) * round.Multiplier)
		return nextCard, true, round.Multiplier, round.CurrentPayout, nil
	}

	// Busted
	roundID := round.RoundID
	delete(e.rounds, userID)
	_ = e.ledger.Settle(context.Background(), roundID)

	return nextCard, false, 0, 0, nil
}

// CashOut settles the card round.
func (e *HigherLowerEngine) CashOut(userID int64) (uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	round, ok := e.rounds[userID]
	if !ok || round.Streak == 0 {
		return 0, fmt.Errorf("no streak to cash out")
	}

	payout := round.CurrentPayout
	asset := round.Asset
	roundID := round.RoundID

	delete(e.rounds, userID)
	_ = e.ledger.Settle(context.Background(), roundID)
	if payout > round.InitialWager {
		profit := payout - round.InitialWager
		ref := fmt.Sprintf("hl_win_%d_%d", userID, time.Now().UnixNano())
		_, _ = e.ledger.Deposit(context.Background(), userID, asset, profit, ref, "Higher Lower cashout")
	}

	return payout, nil
}
