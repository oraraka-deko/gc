package games

import (
	"encoding/json"
	"net/http"

	"gc/pkg/miniapp"
)

func getAuthUserID(r *http.Request) int64 {
	user := miniapp.UserFromContext(r.Context())
	if user != nil {
		return user.ID
	}
	return 99999999 // Fallback ID for local sandbox testing
}

func (s *GameServer) HandleCoinFlip(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req CoinFlipRequest
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getAuthUserID(r)
	resp, err := s.Coins.Flip(userID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *GameServer) HandleCoinCashout(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	userID := getAuthUserID(r)
	payout, err := s.Coins.CashOut(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"payout": payout})
}

func (s *GameServer) HandleMinesStart(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req struct {
		Dimension int    `json:"dimension"`
		MineCount int    `json:"mine_count"`
		Asset     string `json:"asset"`
		Wager     uint64 `json:"wager"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getAuthUserID(r)
	seedHash, err := s.Mines.StartGame(userID, req.Dimension, req.MineCount, req.Asset, req.Wager)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"seed_hash": seedHash})
}

func (s *GameServer) HandleMinesReveal(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req struct {
		TileIndex int `json:"tile_index"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getAuthUserID(r)
	exploded, mult, payout, allMines, err := s.Mines.RevealTile(userID, req.TileIndex)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"exploded":   exploded,
		"multiplier": mult,
		"payout":     payout,
		"mines":      allMines,
	})
}

func (s *GameServer) HandleMinesCashout(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	userID := getAuthUserID(r)
	payout, allMines, err := s.Mines.CashOut(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"payout": payout,
		"mines":  allMines,
	})
}

func (s *GameServer) HandlePlinkoDrop(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req struct {
		Asset string `json:"asset"`
		Wager uint64 `json:"wager"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getAuthUserID(r)
	path, mult, payout, err := s.Plinko.DropBall(userID, req.Asset, req.Wager)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"path":       path,
		"multiplier": mult,
		"payout":     payout,
	})
}

func (s *GameServer) HandleHLStart(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req struct {
		Asset string `json:"asset"`
		Wager uint64 `json:"wager"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getAuthUserID(r)
	card, err := s.HigherLower.StartRound(userID, req.Asset, req.Wager)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"card": card})
}

func (s *GameServer) HandleHLGuess(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req struct {
		GuessHigher bool `json:"guess_higher"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getAuthUserID(r)
	nextCard, isWin, mult, payout, err := s.HigherLower.Guess(userID, req.GuessHigher)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"card":       nextCard,
		"is_win":     isWin,
		"multiplier": mult,
		"payout":     payout,
	})
}

func (s *GameServer) HandleHLCashout(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	userID := getAuthUserID(r)
	payout, err := s.HigherLower.CashOut(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"payout": payout})
}

func (s *GameServer) HandleUpgradeSpin(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req struct {
		InputValNano  uint64 `json:"input_val_nano"`
		TargetValNano uint64 `json:"target_val_nano"`
		TargetGiftID  string `json:"target_gift_id"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getAuthUserID(r)
	outcome, err := s.Upgrade.Spin(userID, req.InputValNano, req.TargetValNano, req.TargetGiftID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(outcome)
}
