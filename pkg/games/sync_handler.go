package games

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"gc/pkg/ledger"
	"gc/pkg/miniapp"
)

// UserSyncResponse formats user balance & gift collection for Flutter frontend bootstrap.
type UserSyncResponse struct {
	UserID     int64            `json:"user_id"`
	FirstName  string           `json:"first_name"`
	Username   string           `json:"username,omitempty"`
	Balances   map[string]any   `json:"balances"`
	Gifts      []map[string]any `json:"gifts"`
	ServerTime int64            `json:"server_time"`
}

// ClientHub maintains active WebSocket subscriber channels keyed by Telegram UserID.
type ClientHub struct {
	mu      sync.RWMutex
	clients map[int64]map[chan []byte]bool
}

// NewClientHub creates an empty thread-safe connection hub.
func NewClientHub() *ClientHub {
	return &ClientHub{
		clients: make(map[int64]map[chan []byte]bool),
	}
}

// Register adds an active websocket writer channel under a user id.
func (h *ClientHub) Register(userID int64, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[userID]; !ok {
		h.clients[userID] = make(map[chan []byte]bool)
	}
	h.clients[userID][ch] = true
}

// Unregister cleanly removes a websocket writer channel.
func (h *ClientHub) Unregister(userID int64, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if chans, ok := h.clients[userID]; ok {
		delete(chans, ch)
		if len(chans) == 0 {
			delete(h.clients, userID)
		}
	}
}

// BroadcastToUser sends a JSON payload to all active connections for a given user.
func (h *ClientHub) BroadcastToUser(userID int64, payload any) {
	h.mu.RLock()
	chans, ok := h.clients[userID]
	if !ok || len(chans) == 0 {
		h.mu.RUnlock()
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		h.mu.RUnlock()
		return
	}

	activeList := make([]chan []byte, 0, len(chans))
	for c := range chans {
		activeList = append(activeList, c)
	}
	h.mu.RUnlock()

	for _, ch := range activeList {
		select {
		case ch <- data:
		default: // non-blocking write to prevent stalling
		}
	}
}

// BroadcastBalance queries current user balances from Ledger and pushes to all active user sessions.
func (s *GameServer) BroadcastBalance(ctx context.Context, userID int64) {
	tonNano := s.getUserBalance(ctx, userID, "ton_nano")
	stars := s.getUserBalance(ctx, userID, "stars")
	usdtMicro := s.getUserBalance(ctx, userID, "usdt_micro")

	payload := map[string]any{
		"type": "balance_updated",
		"balances": map[string]any{
			"ton_nano":   strconv.FormatUint(tonNano, 10),
			"stars":      stars,
			"usdt_micro": strconv.FormatUint(usdtMicro, 10),
		},
		"user_id": userID,
	}
	s.Hub.BroadcastToUser(userID, payload)
}

// getUserBalance helper queries the double-entry account or initializes with sandbox funds if brand new
func (s *GameServer) getUserBalance(ctx context.Context, userID int64, asset string) uint64 {
	accID := ledger.UserMainAccount(userID, asset)
	acc, err := s.Ledger.GetAccount(ctx, accID)
	if err == nil {
		return acc.Available
	}

	// Initialize new user account with default sandbox funds for testing
	var initAmount uint64
	switch asset {
	case "ton_nano":
		initAmount = 1500000000 // 1.5 TON
	case "stars":
		initAmount = 500
	case "usdt_micro":
		initAmount = 10000000 // 10 USDT
	}

	if initAmount > 0 {
		ref := fmt.Sprintf("welcome:%d:%s", userID, asset)
		_, _ = s.Ledger.Deposit(ctx, userID, asset, initAmount, ref, "Sandbox welcome funds")
		return initAmount
	}
	return 0
}

// HandleUserSync responds with the current balances and gifts of the authenticated user.
func (s *GameServer) HandleUserSync(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := s.ExtractUser(r)
	if user == nil {
		user = &miniapp.WebAppUser{
			ID:        99999999,
			FirstName: "TestPlayer",
		}
	}
	log.Printf("[API] /api/user/sync called for user=%d (%s)", user.ID, user.FirstName)

	ctx := r.Context()
	tonNano := s.getUserBalance(ctx, user.ID, "ton_nano")
	stars := s.getUserBalance(ctx, user.ID, "stars")
	usdtMicro := s.getUserBalance(ctx, user.ID, "usdt_micro")

	// Ensure profile is persisted into WriteBehindStore / RAM
	if s.UserManager != nil {
		_, _ = s.UserManager.EnsureUser(ctx, user)
	}

	// Fetch any user gifts from RAM / WriteBehindStore
	var giftList []map[string]any
	if s.UserManager != nil {
		if gifts, err := s.UserManager.GetUserGifts(ctx, user.ID); err == nil && len(gifts) > 0 {
			for _, g := range gifts {
				giftList = append(giftList, map[string]any{
					"asset_id":    g.AssetID,
					"item_type":   g.ItemType,
					"acquired_at": g.AcquiredAt,
					"metadata":    g.Metadata,
				})
			}
		}
	}
	if giftList == nil {
		giftList = []map[string]any{}
	}

	usernameStr := ""
	if user.Username != nil {
		usernameStr = *user.Username
	}

	res := UserSyncResponse{
		UserID:    user.ID,
		FirstName: user.FirstName,
		Username:  usernameStr,
		Balances: map[string]any{
			"ton_nano":   strconv.FormatUint(tonNano, 10),
			"stars":      stars,
			"usdt_micro": strconv.FormatUint(usdtMicro, 10),
		},
		Gifts:      giftList,
		ServerTime: time.Now().Unix(),
	}

	log.Printf("[API] /api/user/sync returning: user=%d, tonNano=%d, stars=%d, usdtMicro=%d, gifts=%d",
		user.ID, tonNano, stars, usdtMicro, len(giftList))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}
