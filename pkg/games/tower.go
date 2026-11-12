package games

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"gc/pkg/ledger"
)

// TowerDifficultyConfig defines column count, trap count, and payout multipliers per row.
type TowerDifficultyConfig struct {
	NumCols     int
	TrapCount   int
	Multipliers [7]float64
}

var TowerDifficultyConfigs = map[string]TowerDifficultyConfig{
	"easy":   {NumCols: 5, TrapCount: 1, Multipliers: [7]float64{1.18, 1.48, 1.85, 2.31, 2.89, 3.61, 4.51}},
	"medium": {NumCols: 5, TrapCount: 2, Multipliers: [7]float64{1.58, 2.64, 4.40, 7.33, 12.20, 20.40, 33.90}},
	"master": {NumCols: 5, TrapCount: 3, Multipliers: [7]float64{2.38, 5.95, 14.88, 37.20, 93.00, 232.50, 581.25}},
}

// TowerBombCols deterministically computes trap columns using SHA-256 for a given row.
func TowerBombCols(serverSeed string, roundID int, row int, numCols int, trapCount int) []int {
	input := fmt.Sprintf("%s:%d:row:%d", serverSeed, roundID, row)
	digest := sha256.Sum256([]byte(input))

	cols := make([]int, numCols)
	for i := 0; i < numCols; i++ {
		cols[i] = i
	}

	for i := 0; i < trapCount && i < numCols-1; i++ {
		val := binary.BigEndian.Uint32(digest[i*4 : i*4+4])
		remaining := numCols - i
		pickIdx := i + int(val%uint32(remaining))
		cols[i], cols[pickIdx] = cols[pickIdx], cols[i]
	}

	result := make([]int, trapCount)
	copy(result, cols[:trapCount])
	sort.Ints(result)
	return result
}

// TowerSession holds the live state of an active tower climb run.
type TowerSession struct {
	RoundID    int
	UserID     int64
	TransferID string
	Difficulty string
	Currency   string
	BetAtomic  uint64
	Step       int // 0..6 (next row to pick)
	ServerSeed string
	SeedHash   string
	BombCols   [7][]int
	CreatedAt  time.Time
}

// TowerStartResult is returned upon beginning a tower climb.
type TowerStartResult struct {
	Type           string `json:"type"`
	Status         string `json:"status"`
	RoundID        int    `json:"roundId"`
	Difficulty     string `json:"difficulty"`
	Step           int    `json:"step"`
	SeedHash       string `json:"seedHash"`
	Currency       string `json:"currency"`
	BetAtomic      uint64 `json:"betAtomic"`
	UpdatedBalance uint64 `json:"updatedBalance"`
}

// TowerStepResult is returned after selecting a tile in a row.
type TowerStepResult struct {
	Type               string   `json:"type"`
	Status             string   `json:"status"` // "playing", "busted", "won"
	RoundID            int      `json:"roundId"`
	Step               int      `json:"step"`
	PickedCol          int      `json:"pickedCol,omitempty"`
	CurrentMultiplier  float64  `json:"currentMultiplier,omitempty"`
	PayoutAtomic       uint64   `json:"payoutAtomic,omitempty"`
	UpdatedBalance     uint64   `json:"updatedBalance,omitempty"`
	BombCols           [][7]int `json:"bombCols,omitempty"`
	RevealedServerSeed *string  `json:"revealedServerSeed,omitempty"`
}

// TowerCashoutResult is returned upon cashing out accumulated winnings.
type TowerCashoutResult struct {
	Type               string   `json:"type"`
	Status             string   `json:"status"` // "cashedOut"
	RoundID            int      `json:"roundId"`
	Step               int      `json:"step"`
	Multiplier         float64  `json:"multiplier"`
	PayoutAtomic       uint64   `json:"payoutAtomic"`
	UpdatedBalance     uint64   `json:"updatedBalance"`
	BombCols           [][7]int `json:"bombCols,omitempty"`
	RevealedServerSeed string   `json:"revealedServerSeed"`
}

// TowerEngine manages Tower Climb game sessions, provably fair seeds, and ledger balances.
type TowerEngine struct {
	ledger       *ledger.Ledger
	sessions     map[int64]*TowerSession
	roundCounter int
	mu           sync.RWMutex
}

// NewTowerEngine creates a new Tower Climb game engine.
func NewTowerEngine(led *ledger.Ledger) *TowerEngine {
	return &TowerEngine{
		ledger:       led,
		sessions:     make(map[int64]*TowerSession),
		roundCounter: 20000,
	}
}

// StartRound initializes a new tower climb session, auto-cashing out any prior unfinished winning session.
func (e *TowerEngine) StartRound(ctx context.Context, userID int64, currency string, betAtomic uint64, difficulty string) (*TowerStartResult, error) {
	currency = NormalizeCurrency(currency)
	if betAtomic == 0 {
		return nil, ledger.ErrInvalidAmount
	}

	cfg, ok := TowerDifficultyConfigs[difficulty]
	if !ok {
		difficulty = "medium"
		cfg = TowerDifficultyConfigs[difficulty]
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. Auto-cashout prior unfinished session with progress
	if existing, hasActive := e.sessions[userID]; hasActive {
		if existing.Step >= 1 {
			oldCfg := TowerDifficultyConfigs[existing.Difficulty]
			mult := oldCfg.Multipliers[existing.Step-1]
			payoutFloat := math.Round(float64(existing.BetAtomic) * mult)
			payoutAtomic := uint64(payoutFloat)

			_ = e.ledger.Settle(ctx, existing.TransferID)
			ref := fmt.Sprintf("tw_autocash_%d", existing.RoundID)
			desc := fmt.Sprintf("Tower Climb auto-cashout Round %d (%.2fx)", existing.RoundID, mult)
			_, _ = e.ledger.Deposit(ctx, userID, existing.Currency, payoutAtomic, ref, desc)
		} else {
			// Step 0: Void to restore funds
			_ = e.ledger.Void(ctx, existing.TransferID, "new_round_started")
		}
		delete(e.sessions, userID)
	}

	e.roundCounter++
	roundID := e.roundCounter
	now := time.Now()

	// 2. Reserve wager in ledger
	transferID := fmt.Sprintf("tw_xfr_%d_%d", roundID, now.UnixNano())
	destAcc := ledger.SystemFeeAccount(currency)
	_, err := e.ledger.Reserve(ctx, transferID, userID, destAcc, currency, betAtomic, map[string]string{
		"game":     "tower_climb",
		"round_id": fmt.Sprintf("%d", roundID),
	})
	if err != nil {
		return nil, fmt.Errorf("insufficient balance or reserve failed: %w", err)
	}

	// 3. Generate seed and bomb traps
	seedBytes := make([]byte, 16)
	_, _ = rand.Read(seedBytes)
	serverSeed := fmt.Sprintf("tw_%d_%s", roundID, hex.EncodeToString(seedBytes))
	seedHashBytes := sha256.Sum256([]byte(serverSeed))
	seedHash := hex.EncodeToString(seedHashBytes[:])

	var bombCols [7][]int
	for row := 0; row < 7; row++ {
		bombCols[row] = TowerBombCols(serverSeed, roundID, row, cfg.NumCols, cfg.TrapCount)
	}

	session := &TowerSession{
		RoundID:    roundID,
		UserID:     userID,
		TransferID: transferID,
		Difficulty: difficulty,
		Currency:   currency,
		BetAtomic:  betAtomic,
		Step:       0,
		ServerSeed: serverSeed,
		SeedHash:   seedHash,
		BombCols:   bombCols,
		CreatedAt:  now,
	}
	e.sessions[userID] = session

	userAcc, _ := e.ledger.GetAccount(ctx, ledger.UserMainAccount(userID, currency))
	var avail uint64
	if userAcc != nil {
		avail = userAcc.Available
	}

	return &TowerStartResult{
		Type:           "tower_state",
		Status:         "playing",
		RoundID:        roundID,
		Difficulty:     difficulty,
		Step:           0,
		SeedHash:       seedHash,
		Currency:       currency,
		BetAtomic:      betAtomic,
		UpdatedBalance: avail,
	}, nil
}

// PickTile processes a tile pick at the current step row.
func (e *TowerEngine) PickTile(ctx context.Context, userID int64, roundID int, step int, col int) (*TowerStepResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	session, exists := e.sessions[userID]
	if !exists || session.RoundID != roundID {
		return nil, errors.New("no active tower session")
	}

	cfg := TowerDifficultyConfigs[session.Difficulty]
	if col < 0 || col >= cfg.NumCols {
		return nil, errors.New("invalid column index")
	}
	if step != session.Step {
		return nil, errors.New("step mismatch")
	}

	// Check if picked column hit a bomb trap
	isBomb := false
	for _, bCol := range session.BombCols[step] {
		if col == bCol {
			isBomb = true
			break
		}
	}

	if isBomb {
		revealedSeed := session.ServerSeed
		transferID := session.TransferID
		delete(e.sessions, userID)

		// Loss: Settle reserved funds as house revenue
		_ = e.ledger.Settle(ctx, transferID)

		userAcc, _ := e.ledger.GetAccount(ctx, ledger.UserMainAccount(userID, session.Currency))
		var avail uint64
		if userAcc != nil {
			avail = userAcc.Available
		}

		return &TowerStepResult{
			Type:               "tower_state",
			Status:             "busted",
			RoundID:            session.RoundID,
			Step:               step,
			PickedCol:          col,
			UpdatedBalance:     avail,
			RevealedServerSeed: &revealedSeed,
		}, nil
	}

	// Safe pick: advance step
	session.Step++

	// Cleared all 7 rows: Grand Win!
	if session.Step >= 7 {
		multiplier := cfg.Multipliers[6]
		payoutFloat := math.Round(float64(session.BetAtomic) * multiplier)
		payoutAtomic := uint64(payoutFloat)
		revealedSeed := session.ServerSeed
		transferID := session.TransferID
		curr := session.Currency
		rID := session.RoundID
		delete(e.sessions, userID)

		// Settle original bet and credit win
		_ = e.ledger.Settle(ctx, transferID)
		ref := fmt.Sprintf("tw_win_%d", rID)
		desc := fmt.Sprintf("Tower Climb 7-Row Grand Win: Round %d (%.2fx)", rID, multiplier)
		_, _ = e.ledger.Deposit(ctx, userID, curr, payoutAtomic, ref, desc)

		userAcc, _ := e.ledger.GetAccount(ctx, ledger.UserMainAccount(userID, curr))
		var avail uint64
		if userAcc != nil {
			avail = userAcc.Available
		}

		return &TowerStepResult{
			Type:               "tower_state",
			Status:             "won",
			RoundID:            rID,
			Step:               6,
			CurrentMultiplier:  multiplier,
			PayoutAtomic:       payoutAtomic,
			UpdatedBalance:     avail,
			RevealedServerSeed: &revealedSeed,
		}, nil
	}

	// Intermediate safe pick
	multiplier := cfg.Multipliers[session.Step-1]
	return &TowerStepResult{
		Type:              "tower_state",
		Status:            "playing",
		RoundID:           session.RoundID,
		Step:              session.Step,
		CurrentMultiplier: multiplier,
	}, nil
}

// Cashout finalizes the round at the current multiplier and credits winnings.
func (e *TowerEngine) Cashout(ctx context.Context, userID int64, roundID int) (*TowerCashoutResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	session, exists := e.sessions[userID]
	if !exists || session.RoundID != roundID {
		return nil, errors.New("no active tower session")
	}

	if session.Step < 1 {
		return nil, errors.New("must clear at least one row before cashing out")
	}

	cfg := TowerDifficultyConfigs[session.Difficulty]
	multiplier := cfg.Multipliers[session.Step-1]
	payoutFloat := math.Round(float64(session.BetAtomic) * multiplier)
	payoutAtomic := uint64(payoutFloat)
	revealedSeed := session.ServerSeed
	transferID := session.TransferID
	curr := session.Currency
	rID := session.RoundID
	step := session.Step
	delete(e.sessions, userID)

	// Settle wager and deposit winnings
	_ = e.ledger.Settle(ctx, transferID)
	ref := fmt.Sprintf("tw_cash_%d", rID)
	desc := fmt.Sprintf("Tower Climb cashout: Round %d (%.2fx)", rID, multiplier)
	_, _ = e.ledger.Deposit(ctx, userID, curr, payoutAtomic, ref, desc)

	userAcc, _ := e.ledger.GetAccount(ctx, ledger.UserMainAccount(userID, curr))
	var avail uint64
	if userAcc != nil {
		avail = userAcc.Available
	}

	return &TowerCashoutResult{
		Type:               "tower_state",
		Status:             "cashedOut",
		RoundID:            rID,
		Step:               step,
		Multiplier:         multiplier,
		PayoutAtomic:       payoutAtomic,
		UpdatedBalance:     avail,
		RevealedServerSeed: revealedSeed,
	}, nil
}
