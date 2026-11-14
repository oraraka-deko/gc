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

// PlinkoEngine models a standard 16-row pyramid plinko physics board.
type PlinkoEngine struct {
	mu     sync.RWMutex
	ledger *ledger.Ledger
}

func NewPlinkoEngine(led *ledger.Ledger) *PlinkoEngine {
	return &PlinkoEngine{ledger: led}
}

// PlinkoMultipliers for standard medium-risk 16-row configuration
var PlinkoMultipliers = []float64{
	16.0, 9.0, 2.0, 1.4, 1.1, 1.0, 0.5, 0.4, 0.4, 0.5, 1.0, 1.1, 1.4, 2.0, 9.0, 16.0, 33.0,
}

// DropBall simulates a ball dropping through rows with 50/50 left/right peg deflections.
func (e *PlinkoEngine) DropBall(userID int64, asset string, wager uint64) ([]int, float64, uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	transferID := fmt.Sprintf("plinko_%d_%d", userID, time.Now().UnixNano())
	toAcc := ledger.SystemReserveAccount(asset)
	_, err := e.ledger.Reserve(context.Background(), transferID, userID, toAcc, asset, wager, nil)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to reserve wager: %w", err)
	}

	const rows = 16
	path := make([]int, rows)
	slotIndex := 0

	for r := 0; r < rows; r++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(2))
		dir := int(n.Int64()) // 0: left, 1: right
		path[r] = dir
		slotIndex += dir
	}

	if slotIndex >= len(PlinkoMultipliers) {
		slotIndex = len(PlinkoMultipliers) - 1
	}

	mult := PlinkoMultipliers[slotIndex]
	payout := uint64(float64(wager) * mult)

	_ = e.ledger.Settle(context.Background(), transferID)
	if payout > 0 {
		ref := fmt.Sprintf("plinko_win_%d_%d", userID, time.Now().UnixNano())
		_, _ = e.ledger.Deposit(context.Background(), userID, asset, payout, ref, "Plinko ball payout")
	}

	return path, mult, payout, nil
}
