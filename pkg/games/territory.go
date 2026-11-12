package games

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"gc/pkg/ledger"
)

// RoundStatus describes the progression phases of a Territory multiplayer round.
type RoundStatus string

const (
	StatusWaitingForPlayers RoundStatus = "waitingForPlayers"
	StatusBettingOpen       RoundStatus = "bettingOpen"
	StatusRoundResolving    RoundStatus = "roundResolving"
	StatusWinnerCelebration RoundStatus = "winnerCelebration"
)

// ServerNPCUser defines a simulated bot competitor.
type ServerNPCUser struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	Avatar    string `json:"avatar"`
	Color     uint32 `json:"color"`
}

var DefaultNPCUsers = []ServerNPCUser{
	{ID: "server_alex", Username: "@cryptoking", FirstName: "Alex", Avatar: "assets/npc/avatars/7660412004.jpg", Color: 0xFF2DD4BF},
	{ID: "server_elena", Username: "@starlight", FirstName: "Elena", Avatar: "assets/npc/avatars/1965297855.jpg", Color: 0xFF6366F1},
	{ID: "server_dmitry", Username: "@ton_master", FirstName: "Dmitry", Avatar: "", Color: 0xFF65E538},
	{ID: "server_anna", Username: "@amsexyledy", FirstName: "Anna", Avatar: "assets/npc/avatars/7660412004.jpg", Color: 0xFFF97316},
	{ID: "server_evgeny", Username: "@Jevga22", FirstName: "Evgeny", Avatar: "assets/npc/avatars/1965297855.jpg", Color: 0xFFF43F5E},
	{ID: "server_luna", Username: "@luna_whale", FirstName: "Luna", Avatar: "", Color: 0xFF26B8C5},
	{ID: "server_vlad", Username: "@vlad_ton", FirstName: "Vlad", Avatar: "", Color: 0xFF5850EC},
	{ID: "server_sophia", Username: "@sophia_stars", FirstName: "Sophia", Avatar: "", Color: 0xFF4ADE80},
	{ID: "server_max", Username: "@max_crypto", FirstName: "Max", Avatar: "", Color: 0xFFFB923C},
}

// TerritoryBet represents a player or NPC wager in the active pot.
type TerritoryBet struct {
	UserID         int64   `json:"userId,omitempty"` // 0 for bots
	PlayerID       string  `json:"playerId"`
	Username       string  `json:"username"`
	FirstName      string  `json:"firstName,omitempty"`
	Avatar         string  `json:"avatar"`
	Color          uint32  `json:"color"`
	Currency       string  `json:"currency"`
	AmountAtomic   uint64  `json:"amountAtomic"`
	TotalWeight    float64 `json:"totalWeight"`
	WinProbability float64 `json:"winProbability"`
	IsBot          bool    `json:"isBot"`
	TransferID     string  `json:"-"`
}

// TerritoryRoundState contains client-visible round data.
type TerritoryRoundState struct {
	RoundID            int            `json:"roundId"`
	Status             RoundStatus    `json:"status"`
	Bets               []TerritoryBet `json:"bets"`
	TotalPotTONNano    uint64         `json:"totalPotTonNano"`
	TotalPotStars      uint64         `json:"totalPotStars"`
	TimeRemainingMs    int            `json:"timeRemainingMs"`
	ServerSeedHash     string         `json:"serverSeedHash"`
	RevealedServerSeed *string        `json:"revealedServerSeed"`
	WinnerPlayerID     *string        `json:"winnerPlayerId,omitempty"`
	PoolTier           string         `json:"poolTier"`
	IsBettingClosed    bool           `json:"isBettingClosed"`
}

// TerritoryRoundHistoryRecord archives completed rounds.
type TerritoryRoundHistoryRecord struct {
	RoundID            int            `json:"roundId"`
	Bets               []TerritoryBet `json:"bets"`
	TotalPotTONNano    uint64         `json:"totalPotTonNano"`
	TotalPotStars      uint64         `json:"totalPotStars"`
	ServerSeedHash     string         `json:"serverSeedHash"`
	RevealedServerSeed string         `json:"revealedServerSeed"`
	WinnerPlayerID     string         `json:"winnerPlayerId,omitempty"`
	Timestamp          string         `json:"timestamp"`
}

// TerritoryEngine manages the real-time pot loop, dynamic bot entry, and ledger settlements.
type TerritoryEngine struct {
	ledger        *ledger.Ledger
	roundCounter  int
	currentState  TerritoryRoundState
	secretSeed    string
	history       []TerritoryRoundHistoryRecord
	pendingBots   []TerritoryBet
	bettingTick   int
	subscribers   map[chan []byte]struct{}
	subscribersMu sync.RWMutex
	mu            sync.RWMutex
	stopChan      chan struct{}
}

// NewTerritoryEngine initializes the multiplayer Territory game engine.
func NewTerritoryEngine(led *ledger.Ledger) *TerritoryEngine {
	e := &TerritoryEngine{
		ledger:       led,
		roundCounter: 10850,
		history:      make([]TerritoryRoundHistoryRecord, 0, 50),
		subscribers:  make(map[chan []byte]struct{}),
		stopChan:     make(chan struct{}),
	}
	e.startNewRoundLocked()
	go e.runLoop()
	return e
}

// Close gracefully stops the background game loop.
func (e *TerritoryEngine) Close() {
	select {
	case <-e.stopChan:
		return
	default:
		close(e.stopChan)
	}
}

// Subscribe adds a channel to receive real-time state broadcasts.
func (e *TerritoryEngine) Subscribe(ch chan []byte) {
	e.subscribersMu.Lock()
	defer e.subscribersMu.Unlock()
	e.subscribers[ch] = struct{}{}

	// Send current state immediately
	e.mu.RLock()
	st := e.clientStateLocked()
	e.mu.RUnlock()

	msg, _ := json.Marshal(map[string]any{
		"type":  "round_state",
		"state": st,
	})
	select {
	case ch <- msg:
	default:
	}
}

// Unsubscribe removes a channel from broadcasts.
func (e *TerritoryEngine) Unsubscribe(ch chan []byte) {
	e.subscribersMu.Lock()
	defer e.subscribersMu.Unlock()
	delete(e.subscribers, ch)
}

func (e *TerritoryEngine) broadcast() {
	e.mu.RLock()
	clientState := e.clientStateLocked()
	e.mu.RUnlock()

	msg, err := json.Marshal(map[string]any{
		"type":  "round_state",
		"state": clientState,
	})
	if err != nil {
		return
	}

	e.subscribersMu.RLock()
	defer e.subscribersMu.RUnlock()
	for ch := range e.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (e *TerritoryEngine) startNewRoundLocked() {
	e.roundCounter++
	seed, seedHash := GenerateServerSeed("territory", e.roundCounter)
	e.secretSeed = seed

	// Spawn 3 to 8 planned bots
	botCount := 3 + int(CryptoRandInt(6))
	shuffled := make([]ServerNPCUser, len(DefaultNPCUsers))
	copy(shuffled, DefaultNPCUsers)
	for i := len(shuffled) - 1; i > 0; i-- {
		j := int(CryptoRandInt(int64(i + 1)))
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	e.pendingBots = make([]TerritoryBet, 0, botCount)
	for i := 0; i < botCount && i < len(shuffled); i++ {
		bot := shuffled[i]
		stars := uint64(50 + CryptoRandInt(151)) // 50 to 200 Stars
		weight := float64(stars) / 10.0

		e.pendingBots = append(e.pendingBots, TerritoryBet{
			UserID:         0,
			PlayerID:       bot.ID,
			Username:       bot.Username,
			FirstName:      bot.FirstName,
			Avatar:         bot.Avatar,
			Color:          bot.Color,
			Currency:       ledger.AssetStars,
			AmountAtomic:   stars,
			TotalWeight:    weight,
			WinProbability: 0,
			IsBot:          true,
		})
	}

	e.bettingTick = 0
	e.currentState = TerritoryRoundState{
		RoundID:            e.roundCounter,
		Status:             StatusBettingOpen,
		Bets:               []TerritoryBet{},
		TotalPotTONNano:    0,
		TotalPotStars:      0,
		TimeRemainingMs:    15000,
		ServerSeedHash:     seedHash,
		RevealedServerSeed: nil,
		WinnerPlayerID:     nil,
		PoolTier:           "STANDARD",
		IsBettingClosed:    false,
	}
}

func (e *TerritoryEngine) recalculateProbabilitiesLocked() {
	var totalWeight float64
	for i := range e.currentState.Bets {
		totalWeight += e.currentState.Bets[i].TotalWeight
	}
	if totalWeight <= 0 {
		for i := range e.currentState.Bets {
			e.currentState.Bets[i].WinProbability = 0
		}
		return
	}
	for i := range e.currentState.Bets {
		prob := (e.currentState.Bets[i].TotalWeight / totalWeight) * 100.0
		e.currentState.Bets[i].WinProbability = math.Round(prob*100) / 100
	}
}

func (e *TerritoryEngine) pickWinnerLocked() *TerritoryBet {
	if len(e.currentState.Bets) == 0 {
		return nil
	}
	var totalWeight float64
	for _, b := range e.currentState.Bets {
		totalWeight += b.TotalWeight
	}
	if totalWeight <= 0 {
		return &e.currentState.Bets[0]
	}

	hash := sha256.Sum256([]byte(e.secretSeed))
	val := binary.BigEndian.Uint64(hash[:8])
	unit := float64(val>>11) / float64(1<<53)
	target := unit * totalWeight

	var cumulative float64
	for i := range e.currentState.Bets {
		cumulative += e.currentState.Bets[i].TotalWeight
		if target <= cumulative {
			return &e.currentState.Bets[i]
		}
	}
	return &e.currentState.Bets[len(e.currentState.Bets)-1]
}

func (e *TerritoryEngine) clientStateLocked() TerritoryRoundState {
	st := e.currentState
	betsCopy := make([]TerritoryBet, len(e.currentState.Bets))
	copy(betsCopy, e.currentState.Bets)
	st.Bets = betsCopy

	if st.Status == StatusBettingOpen {
		st.RevealedServerSeed = nil
		st.WinnerPlayerID = nil
	} else {
		seedCopy := e.secretSeed
		st.RevealedServerSeed = &seedCopy
		if e.currentState.WinnerPlayerID != nil {
			winCopy := *e.currentState.WinnerPlayerID
			st.WinnerPlayerID = &winCopy
		}
	}
	return st
}

// GetCurrentState returns a copy of the current round state.
func (e *TerritoryEngine) GetCurrentState() TerritoryRoundState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.clientStateLocked()
}

// GetHistory returns past completed round records.
func (e *TerritoryEngine) GetHistory() []TerritoryRoundHistoryRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()
	histCopy := make([]TerritoryRoundHistoryRecord, len(e.history))
	copy(histCopy, e.history)
	return histCopy
}

// PlaceBet places a player wager, reserving user funds in the double-entry ledger.
func (e *TerritoryEngine) PlaceBet(ctx context.Context, userID int64, username string, firstName string, avatar string, currency string, amountAtomic uint64) error {
	currency = NormalizeCurrency(currency)
	if amountAtomic == 0 {
		return ledger.ErrInvalidAmount
	}

	playerID := fmt.Sprintf("user_%d", userID)

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.currentState.Status != StatusBettingOpen || e.currentState.IsBettingClosed {
		return fmt.Errorf("betting is closed for round %d", e.currentState.RoundID)
	}

	// 1. Reserve wager in ledger
	now := time.Now().UnixNano()
	transferID := fmt.Sprintf("pot_xfr_%d_%d_%d", e.currentState.RoundID, userID, now)
	destAcc := ledger.SystemReserveAccount(currency)
	_, err := e.ledger.Reserve(ctx, transferID, userID, destAcc, currency, amountAtomic, map[string]string{
		"game":     "territory",
		"round_id": fmt.Sprintf("%d", e.currentState.RoundID),
	})
	if err != nil {
		return fmt.Errorf("failed to reserve bet in ledger: %w", err)
	}

	// Calculate normalized weight: Stars / 10.0, TON nano / 1e8
	var weight float64
	if currency == ledger.AssetStars {
		weight = float64(amountAtomic) / 10.0
		e.currentState.TotalPotStars += amountAtomic
	} else {
		weight = float64(amountAtomic) / 1e8
		e.currentState.TotalPotTONNano += amountAtomic
	}

	// Check if player already bet with this currency
	found := false
	for i := range e.currentState.Bets {
		if e.currentState.Bets[i].UserID == userID && e.currentState.Bets[i].Currency == currency {
			e.currentState.Bets[i].AmountAtomic += amountAtomic
			e.currentState.Bets[i].TotalWeight += weight
			found = true
			break
		}
	}

	if !found {
		color := uint32(0xFFF97316) // Active highlight color
		e.currentState.Bets = append(e.currentState.Bets, TerritoryBet{
			UserID:         userID,
			PlayerID:       playerID,
			Username:       username,
			FirstName:      firstName,
			Avatar:         avatar,
			Color:          color,
			Currency:       currency,
			AmountAtomic:   amountAtomic,
			TotalWeight:    weight,
			WinProbability: 0,
			IsBot:          false,
			TransferID:     transferID,
		})
	}

	e.recalculateProbabilitiesLocked()
	return nil
}

func (e *TerritoryEngine) runLoop() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.onTick()
			e.broadcast()
		}
	}
}

func (e *TerritoryEngine) onTick() {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch e.currentState.Status {
	case StatusBettingOpen:
		e.bettingTick++

		// Dynamically add planned bots to simulate active lobby
		if len(e.pendingBots) > 0 {
			shouldAddBot := false
			activeCount := len(e.currentState.Bets)

			if activeCount == 0 && e.bettingTick >= 10 {
				shouldAddBot = true
			} else if activeCount == 1 && e.bettingTick >= 22 {
				shouldAddBot = true
			} else if activeCount >= 2 && e.bettingTick >= 22+(activeCount-1)*10 && !e.currentState.IsBettingClosed {
				shouldAddBot = true
			}

			if shouldAddBot {
				bot := e.pendingBots[0]
				e.pendingBots = e.pendingBots[1:]
				e.currentState.Bets = append(e.currentState.Bets, bot)
				e.currentState.TotalPotStars += bot.AmountAtomic
				e.recalculateProbabilitiesLocked()
			}
		}

		if len(e.currentState.Bets) < 2 {
			e.currentState.TimeRemainingMs = 15000
			e.currentState.IsBettingClosed = false
		} else {
			e.currentState.TimeRemainingMs -= 250
			if e.currentState.TimeRemainingMs <= 2500 {
				e.currentState.IsBettingClosed = true
			}
			if e.currentState.TimeRemainingMs <= 0 {
				e.currentState.Status = StatusRoundResolving
				e.currentState.TimeRemainingMs = 13400 // 1.4s aim + 12s trajectory

				seedCopy := e.secretSeed
				e.currentState.RevealedServerSeed = &seedCopy
				winner := e.pickWinnerLocked()
				if winner != nil {
					winID := winner.PlayerID
					e.currentState.WinnerPlayerID = &winID
				}
			}
		}

	case StatusRoundResolving:
		e.currentState.TimeRemainingMs -= 250
		if e.currentState.TimeRemainingMs <= 0 {
			e.currentState.Status = StatusWinnerCelebration
			e.currentState.TimeRemainingMs = 5000 // 5s celebration modal

			// Settle bets and payout winner via double-entry ledger
			e.settleRoundLocked()

			betsArchive := make([]TerritoryBet, len(e.currentState.Bets))
			copy(betsArchive, e.currentState.Bets)
			winnerID := ""
			if e.currentState.WinnerPlayerID != nil {
				winnerID = *e.currentState.WinnerPlayerID
			}

			record := TerritoryRoundHistoryRecord{
				RoundID:            e.currentState.RoundID,
				Bets:               betsArchive,
				TotalPotTONNano:    e.currentState.TotalPotTONNano,
				TotalPotStars:      e.currentState.TotalPotStars,
				ServerSeedHash:     e.currentState.ServerSeedHash,
				RevealedServerSeed: e.secretSeed,
				WinnerPlayerID:     winnerID,
				Timestamp:          time.Now().UTC().Format(time.RFC3339),
			}

			if len(e.history) >= 50 {
				copy(e.history[1:], e.history[:49])
				e.history[0] = record
			} else {
				e.history = append(e.history, TerritoryRoundHistoryRecord{})
				copy(e.history[1:], e.history[:len(e.history)-1])
				e.history[0] = record
			}
		}

	case StatusWinnerCelebration:
		e.currentState.TimeRemainingMs -= 250
		if e.currentState.TimeRemainingMs <= 0 {
			intermissionMs := 5000 + int(CryptoRandInt(10001))
			e.currentState.Status = StatusWaitingForPlayers
			e.currentState.TimeRemainingMs = intermissionMs
			e.currentState.IsBettingClosed = true
		}

	case StatusWaitingForPlayers:
		e.currentState.TimeRemainingMs -= 250
		if e.currentState.TimeRemainingMs <= 0 {
			e.startNewRoundLocked()
		}
	}
}

func (e *TerritoryEngine) settleRoundLocked() {
	ctx := context.Background()
	var winner *TerritoryBet
	if e.currentState.WinnerPlayerID != nil {
		for i := range e.currentState.Bets {
			if e.currentState.Bets[i].PlayerID == *e.currentState.WinnerPlayerID {
				winner = &e.currentState.Bets[i]
				break
			}
		}
	}

	// 1. Settle all real user wagers
	for _, bet := range e.currentState.Bets {
		if !bet.IsBot && bet.TransferID != "" {
			_ = e.ledger.Settle(ctx, bet.TransferID)
		}
	}

	// 2. If winner is a real player, credit total pot
	if winner != nil && !winner.IsBot && winner.UserID > 0 {
		ref := fmt.Sprintf("pot_win_%d", e.currentState.RoundID)
		desc := fmt.Sprintf("Territory Round %d Winner Payout", e.currentState.RoundID)

		if e.currentState.TotalPotStars > 0 {
			_, _ = e.ledger.Deposit(ctx, winner.UserID, ledger.AssetStars, e.currentState.TotalPotStars, ref+":stars", desc)
		}
		if e.currentState.TotalPotTONNano > 0 {
			_, _ = e.ledger.Deposit(ctx, winner.UserID, ledger.AssetTON, e.currentState.TotalPotTONNano, ref+":ton", desc)
		}
		log.Printf("[Territory] Payout to real winner %d: %d Stars, %s TON",
			winner.UserID, e.currentState.TotalPotStars, ledger.FormatAtomic(e.currentState.TotalPotTONNano, ledger.AssetTON))
	}
}
