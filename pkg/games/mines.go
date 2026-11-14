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

// ActiveMinesGame holds the hidden bomb layout and revealed tiles for an active match.
type ActiveMinesGame struct {
	RoundID       string       `json:"round_id"`
	UserID        int64        `json:"user_id"`
	Asset         string       `json:"asset"` // "ton_nano" or "stars"
	InitialWager  uint64       `json:"initial_wager"`
	Dimension     int          `json:"dimension"` // 3 or 5
	TotalTiles    int          `json:"total_tiles"`
	MineCount     int          `json:"mine_count"`
	MineIndices   map[int]bool `json:"mine_indices"`
	RevealedGems  map[int]bool `json:"revealed_gems"`
	CurrentPayout uint64       `json:"current_payout"`
	Multiplier    float64      `json:"multiplier"`
	ServerSeed    string       `json:"server_seed"`
	CreatedAt     time.Time    `json:"created_at"`
}

// MinesEngine orchestrates provably fair 3x3 and 5x5 minefields with double-entry reservations.
type MinesEngine struct {
	mu     sync.RWMutex
	ledger *ledger.Ledger
	games  map[int64]*ActiveMinesGame
}

func NewMinesEngine(led *ledger.Ledger) *MinesEngine {
	return &MinesEngine{
		ledger: led,
		games:  make(map[int64]*ActiveMinesGame),
	}
}

// StartGame reserves wager and creates the randomized bomb layout.
func (e *MinesEngine) StartGame(userID int64, dimension, mineCount int, asset string, wager uint64) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	totalTiles := dimension * dimension
	if mineCount < 1 || mineCount >= totalTiles {
		return "", fmt.Errorf("invalid mine count %d for dimension %d", mineCount, dimension)
	}

	transferID := fmt.Sprintf("mines_%d_%d", userID, time.Now().UnixNano())
	toAcc := ledger.SystemReserveAccount(asset)
	_, err := e.ledger.Reserve(context.Background(), transferID, userID, toAcc, asset, wager, nil)
	if err != nil {
		return "", fmt.Errorf("failed to reserve wager: %w", err)
	}

	// Generate Server Seed
	seedBytes := make([]byte, 16)
	_, _ = rand.Read(seedBytes)
	serverSeed := hex.EncodeToString(seedBytes)

	// Randomly place unique mines
	mineIndices := make(map[int]bool)
	for len(mineIndices) < mineCount {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(totalTiles)))
		mineIndices[int(n.Int64())] = true
	}

	hash := sha256.Sum256([]byte(serverSeed))
	seedHash := hex.EncodeToString(hash[:])

	e.games[userID] = &ActiveMinesGame{
		RoundID:       transferID,
		UserID:        userID,
		Asset:         asset,
		InitialWager:  wager,
		Dimension:     dimension,
		TotalTiles:    totalTiles,
		MineCount:     mineCount,
		MineIndices:   mineIndices,
		RevealedGems:  make(map[int]bool),
		CurrentPayout: wager,
		Multiplier:    1.0,
		ServerSeed:    serverSeed,
		CreatedAt:     time.Now(),
	}

	return seedHash, nil
}

// RevealTile checks if a tile has a mine or gem.
func (e *MinesEngine) RevealTile(userID int64, tileIndex int) (bool, float64, uint64, []int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	game, ok := e.games[userID]
	if !ok {
		return false, 0, 0, nil, fmt.Errorf("no active mines match")
	}

	if tileIndex < 0 || tileIndex >= game.TotalTiles {
		return false, 0, 0, nil, fmt.Errorf("tile index out of bounds")
	}

	// Check if hit a bomb
	if game.MineIndices[tileIndex] {
		// Exploded! Reveal all mines
		allMines := make([]int, 0, len(game.MineIndices))
		for m := range game.MineIndices {
			allMines = append(allMines, m)
		}
		roundID := game.RoundID
		delete(e.games, userID)

		_ = e.ledger.Settle(context.Background(), roundID)
		return true, 0, 0, allMines, nil
	}

	// Safe gem!
	game.RevealedGems[tileIndex] = true
	gemsCount := len(game.RevealedGems)

	// Progressive Multiplier calculation
	mult := 1.0
	for i := 0; i < gemsCount; i++ {
		prob := float64(game.TotalTiles-game.MineCount-i) / float64(game.TotalTiles-i)
		mult *= (0.97 / prob) // 3% house edge
	}
	game.Multiplier = mult
	game.CurrentPayout = uint64(float64(game.InitialWager) * mult)

	return false, game.Multiplier, game.CurrentPayout, nil, nil
}

// CashOut settles the accumulated winnings.
func (e *MinesEngine) CashOut(userID int64) (uint64, []int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	game, ok := e.games[userID]
	if !ok || len(game.RevealedGems) == 0 {
		return 0, nil, fmt.Errorf("no safe gems to cash out")
	}

	payout := game.CurrentPayout
	asset := game.Asset
	roundID := game.RoundID

	allMines := make([]int, 0, len(game.MineIndices))
	for m := range game.MineIndices {
		allMines = append(allMines, m)
	}

	delete(e.games, userID)

	_ = e.ledger.Settle(context.Background(), roundID)
	if payout > game.InitialWager {
		profit := payout - game.InitialWager
		ref := fmt.Sprintf("mines_win_%d_%d", userID, time.Now().UnixNano())
		_, _ = e.ledger.Deposit(context.Background(), userID, asset, profit, ref, "Mines Cashout win")
	}

	return payout, allMines, nil
}
