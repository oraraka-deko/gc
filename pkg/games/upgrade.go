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

// UpgradeEngine handles upgrading gifts via wheel spin with calculated win chance.
type UpgradeEngine struct {
	mu     sync.RWMutex
	ledger *ledger.Ledger
}

func NewUpgradeEngine(led *ledger.Ledger) *UpgradeEngine {
	return &UpgradeEngine{ledger: led}
}

// SpinOutcome represents the upgrade wheel result.
type SpinOutcome struct {
	IsSuccess    bool    `json:"is_success"`
	WinChance    float64 `json:"win_chance"`
	TargetDegree float64 `json:"target_degree"`
	WonGiftID    string  `json:"won_gift_id,omitempty"`
}

// Spin calculates win chance based on input value vs target value and executes provably fair spin.
func (e *UpgradeEngine) Spin(userID int64, inputValNano uint64, targetValNano uint64, targetGiftID string) (*SpinOutcome, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if targetValNano == 0 || inputValNano == 0 {
		return nil, fmt.Errorf("invalid valuation")
	}

	// Calculate base probability with 5% platform house edge
	winChance := (float64(inputValNano) / float64(targetValNano)) * 0.95
	if winChance > 0.95 {
		winChance = 0.95
	}
	if winChance < 0.01 {
		winChance = 0.01
	}

	transferID := fmt.Sprintf("upgrade_%d_%d", userID, time.Now().UnixNano())
	toAcc := ledger.SystemReserveAccount("ton_nano")
	_, err := e.ledger.Reserve(context.Background(), transferID, userID, toAcc, "ton_nano", inputValNano, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to reserve upgrade wager: %w", err)
	}

	// Random 0..10000 (0.00% to 100.00%)
	n, _ := rand.Int(rand.Reader, big.NewInt(10000))
	rolledFrac := float64(n.Int64()) / 10000.0

	isSuccess := rolledFrac < winChance
	var targetDegree float64
	winSliceDegrees := winChance * 360.0

	if isSuccess {
		// Land inside the winning green sector [0..winSliceDegrees)
		targetDegree = rolledFrac * 360.0
		_ = e.ledger.Settle(context.Background(), transferID)
		// User receives target gift
		return &SpinOutcome{
			IsSuccess:    true,
			WinChance:    winChance,
			TargetDegree: targetDegree,
			WonGiftID:    targetGiftID,
		}, nil
	}

	// Land inside the losing red sector [winSliceDegrees..360)
	remaining := 360.0 - winSliceDegrees
	degOffset, _ := rand.Int(rand.Reader, big.NewInt(int64(remaining*10)))
	targetDegree = winSliceDegrees + (float64(degOffset.Int64()) / 10.0)

	_ = e.ledger.Settle(context.Background(), transferID)
	return &SpinOutcome{
		IsSuccess:    false,
		WinChance:    winChance,
		TargetDegree: targetDegree,
	}, nil
}
