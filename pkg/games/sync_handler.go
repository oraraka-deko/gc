package games

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

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

// ClientHub maintains active WebSocket subscriber connections keyed by Telegram UserID.
type ClientHub struct {
	mu      sync.RWMutex
	clients map[int64]map[*websocket.Conn]bool
}

// NewClientHub creates an empty thread-safe connection hub.
func NewClientHub() *ClientHub {
	return &ClientHub{
		clients: make(map[int64]map[*websocket.Conn]bool),
	}
}

// Register adds an active websocket connection under a user id.
func (h *ClientHub) Register(userID int64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[userID]; !ok {
		h.clients[userID] = make(map[*websocket.Conn]bool)
	}
	h.clients[userID][conn] = true
}

// Unregister cleanly removes a websocket connection.
func (h *ClientHub) Unregister(userID int64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.clients[userID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.clients, userID)
		}
	}
}

// BroadcastToUser sends a JSON payload to all active connections for a given user.
func (h *ClientHub) BroadcastToUser(userID int64, payload any) {
	h.mu.RLock()
	conns, ok := h.clients[userID]
	if !ok || len(conns) == 0 {
		h.mu.RUnlock()
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		h.mu.RUnlock()
		return
	}

	// Copy pointers for safe write iteration
	activeList := make([]*websocket.Conn, 0, len(conns))
	for c := range conns {
		activeList = append(activeList, c)
	}
	h.mu.RUnlock()

	for _, conn := range activeList {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
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

	user := miniapp.UserFromContext(r.Context())
	if user == nil {
		// Temporary fallback for local testing if unauthenticated
		user = &miniapp.WebAppUser{
			ID:        99999999,
			FirstName: "TestPlayer",
		}
	}

	ctx := r.Context()
	tonNano := s.getUserBalance(ctx, user.ID, "ton_nano")
	stars := s.getUserBalance(ctx, user.ID, "stars")
	usdtMicro := s.getUserBalance(ctx, user.ID, "usdt_micro")

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
		Gifts:      []map[string]any{},
		ServerTime: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}
