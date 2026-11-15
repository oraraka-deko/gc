package games

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

	"gc/pkg/ledger"
	"gc/pkg/miniapp"
	"gc/pkg/storage"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for Flutter Web & cross-origin mini apps
	},
}

// GameServer multiplexes the HTTP and WebSocket endpoints for all mini-games.
type GameServer struct {
	Ledger      *ledger.Ledger
	Storage     *storage.Storage
	UserManager *miniapp.StoreUserManager
	Territory   *TerritoryEngine
	BallEscape  *BallEscapeEngine
	Tower       *TowerEngine
	Coins       *CoinEngine
	Mines       *MinesEngine
	Plinko      *PlinkoEngine
	HigherLower *HigherLowerEngine
	Upgrade     *UpgradeEngine
	Hub         *ClientHub
	BotToken    string
}

// NewGameServer constructs a fully wired GameServer instance.
func NewGameServer(led *ledger.Ledger, store *storage.Storage, optionalBotToken ...string) *GameServer {
	botToken := ""
	if len(optionalBotToken) > 0 {
		botToken = optionalBotToken[0]
	}
	var userMgr *miniapp.StoreUserManager
	if store != nil && store.DB != nil {
		userMgr = miniapp.NewStoreUserManager(store.DB)
	}
	return &GameServer{
		Ledger:      led,
		Storage:     store,
		UserManager: userMgr,
		Territory:   NewTerritoryEngine(led),
		BallEscape:  NewBallEscapeEngine(led, DefaultBallEscapeSecret),
		Tower:       NewTowerEngine(led),
		Coins:       NewCoinEngine(led),
		Mines:       NewMinesEngine(led),
		Plinko:      NewPlinkoEngine(led),
		HigherLower: NewHigherLowerEngine(led),
		Upgrade:     NewUpgradeEngine(led),
		Hub:         NewClientHub(),
		BotToken:    botToken,
	}
}

// Close terminates active game loops and resources.
func (s *GameServer) Close() {
	if s.Territory != nil {
		s.Territory.Close()
	}
}

// EnableCORS writes standard permissive headers for Flutter Web client access.
func EnableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Telegram-Init-Data")
}

// DecodeJSONBody safely decodes an HTTP request with bounded byte reading against memory exhaustion.
func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	return dec.Decode(dst)
}

// RegisterRoutes binds all game API endpoints to an http.ServeMux.
func (s *GameServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/round/current", s.HandleCurrentRound)
	mux.HandleFunc("/api/bet", s.HandlePlaceBet)
	mux.HandleFunc("/api/history", s.HandleHistory)
	mux.HandleFunc("/api/user/sync", s.HandleUserSync)
	mux.HandleFunc("/api/gifts/catalog", s.HandleGiftsCatalog)
	mux.HandleFunc("/api/ball-escape/start", s.HandleBallEscapeStart)
	mux.HandleFunc("/api/ball-escape/claim", s.HandleBallEscapeClaim)
	mux.HandleFunc("/api/tower/start", s.HandleTowerStart)
	mux.HandleFunc("/api/tower/pick", s.HandleTowerPick)
	mux.HandleFunc("/api/tower/cashout", s.HandleTowerCashout)
	mux.HandleFunc("/api/coins/flip", s.HandleCoinFlip)
	mux.HandleFunc("/api/coins/cashout", s.HandleCoinCashout)
	mux.HandleFunc("/api/mines/start", s.HandleMinesStart)
	mux.HandleFunc("/api/mines/reveal", s.HandleMinesReveal)
	mux.HandleFunc("/api/mines/cashout", s.HandleMinesCashout)
	mux.HandleFunc("/api/plinko/drop", s.HandlePlinkoDrop)
	mux.HandleFunc("/api/higher-lower/start", s.HandleHLStart)
	mux.HandleFunc("/api/higher-lower/guess", s.HandleHLGuess)
	mux.HandleFunc("/api/higher-lower/cashout", s.HandleHLCashout)
	mux.HandleFunc("/api/upgrade/spin", s.HandleUpgradeSpin)
	mux.HandleFunc("/ws", s.HandleWebSocket)
}

// HandleCurrentRound responds with current Territory pot round state.
func (s *GameServer) HandleCurrentRound(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	st := s.Territory.GetCurrentState()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

// PlaceBetRequest models the HTTP payload for placing a pot bet.
type PlaceBetRequest struct {
	UserID       int64   `json:"userId"`
	Username     string  `json:"username"`
	FirstName    string  `json:"firstName,omitempty"`
	Avatar       string  `json:"avatar"`
	Currency     string  `json:"currency"`
	AmountAtomic uint64  `json:"amountAtomic"`
	Amount       float64 `json:"amount,omitempty"` // Legacy float fallback
}

// HandlePlaceBet places a wager in Territory.
func (s *GameServer) HandlePlaceBet(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PlaceBetRequest
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, "Invalid bet payload", http.StatusBadRequest)
		return
	}

	// Normalize legacy amount if provided
	amountAtomic := req.AmountAtomic
	currency := NormalizeCurrency(req.Currency)
	if amountAtomic == 0 && req.Amount > 0 {
		if currency == ledger.AssetTON {
			amountAtomic = ledger.TONToNano(req.Amount)
		} else {
			amountAtomic = uint64(req.Amount)
		}
	}

	if req.UserID <= 0 {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}

	err := s.Territory.PlaceBet(r.Context(), req.UserID, req.Username, req.FirstName, req.Avatar, currency, amountAtomic)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.Territory.broadcast()

	st := s.Territory.GetCurrentState()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"roundId": st.RoundID,
		"state":   st,
	})
}

// HandleHistory returns past completed Territory rounds.
func (s *GameServer) HandleHistory(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hist := s.Territory.GetHistory()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(hist)
}

// HandleBallEscapeStart initializes a Ball Escape round.
func (s *GameServer) HandleBallEscapeStart(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID       int64   `json:"userId"`
		Currency     string  `json:"currency"`
		BetAtomic    uint64  `json:"betAtomic"`
		Bet          float64 `json:"bet,omitempty"` // Legacy float fallback
		Difficulty   string  `json:"difficulty"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	currency := NormalizeCurrency(req.Currency)
	betAtomic := req.BetAtomic
	if betAtomic == 0 && req.Bet > 0 {
		if currency == ledger.AssetTON {
			betAtomic = ledger.TONToNano(req.Bet)
		} else {
			betAtomic = uint64(req.Bet)
		}
	}

	res, err := s.BallEscape.StartRound(r.Context(), req.UserID, currency, betAtomic, req.Difficulty)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// HandleBallEscapeClaim finalizes a Ball Escape round and pays out winnings.
func (s *GameServer) HandleBallEscapeClaim(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID     int64   `json:"userId"`
		RoundID    int     `json:"roundId"`
		ServerSeed string  `json:"serverSeed"`
		Sig        string  `json:"sig"`
		Multiplier float64 `json:"multiplier"`
		IsWin      bool    `json:"isWin"`
		BetAtomic  uint64  `json:"betAtomic"`
		Bet        float64 `json:"bet,omitempty"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	betAtomic := req.BetAtomic
	if betAtomic == 0 && req.Bet > 0 {
		betAtomic = uint64(req.Bet)
	}

	res, err := s.BallEscape.ClaimRound(r.Context(), req.UserID, req.RoundID, req.ServerSeed, req.Sig, req.Multiplier, req.IsWin, betAtomic)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// HandleTowerStart begins a Tower Climb round.
func (s *GameServer) HandleTowerStart(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID     int64   `json:"userId"`
		Difficulty string  `json:"difficulty"`
		Currency   string  `json:"currency"`
		BetAtomic  uint64  `json:"betAtomic"`
		BetAmount  float64 `json:"betAmount,omitempty"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	currency := NormalizeCurrency(req.Currency)
	betAtomic := req.BetAtomic
	if betAtomic == 0 && req.BetAmount > 0 {
		if currency == ledger.AssetTON {
			betAtomic = ledger.TONToNano(req.BetAmount)
		} else {
			betAtomic = uint64(req.BetAmount)
		}
	}

	res, err := s.Tower.StartRound(r.Context(), req.UserID, currency, betAtomic, req.Difficulty)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// HandleTowerPick processes a row tile selection in Tower Climb.
func (s *GameServer) HandleTowerPick(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID  int64 `json:"userId"`
		RoundID int   `json:"roundId"`
		Step    int   `json:"step"`
		Col     int   `json:"col"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	res, err := s.Tower.PickTile(r.Context(), req.UserID, req.RoundID, req.Step, req.Col)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// HandleTowerCashout completes a Tower Climb round and pays accumulated multiplier.
func (s *GameServer) HandleTowerCashout(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID  int64 `json:"userId"`
		RoundID int   `json:"roundId"`
	}
	if err := DecodeJSONBody(w, r, &req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	res, err := s.Tower.Cashout(r.Context(), req.UserID, req.RoundID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// HandleWebSocket manages real-time two-way client communication with single-writer safety.
func (s *GameServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	conn.SetReadLimit(MaxRequestBodyBytes)
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	ch := make(chan []byte, 64)
	done := make(chan struct{})

	// Subscribe to Territory engine broadcasts
	s.Territory.Subscribe(ch)
	defer func() {
		s.Territory.Unsubscribe(ch)
		close(done)
		_ = conn.Close()
	}()

	// Dedicated single-writer goroutine (prevents concurrent write panics)
	go func() {
		pingTicker := time.NewTicker(30 * time.Second)
		defer pingTicker.Stop()

		for {
			select {
			case <-done:
				return
			case data, ok := <-ch:
				if !ok {
					return
				}
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					return
				}
			case <-pingTicker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))

		var incoming map[string]any
		if err := json.Unmarshal(raw, &incoming); err != nil {
			continue
		}

		action, _ := incoming["action"].(string)
		switch action {
		case "bet":
			userID := parseUserID(incoming["userId"])
			username, _ := incoming["username"].(string)
			firstName, _ := incoming["firstName"].(string)
			avatar, _ := incoming["avatar"].(string)
			currency, _ := incoming["currency"].(string)
			amountFloat, _ := incoming["amount"].(float64)

			if userID > 0 && amountFloat > 0 {
				currency = NormalizeCurrency(currency)
				var amountAtomic uint64
				if currency == ledger.AssetTON {
					amountAtomic = ledger.TONToNano(amountFloat)
				} else {
					amountAtomic = uint64(amountFloat)
				}
				_ = s.Territory.PlaceBet(r.Context(), userID, username, firstName, avatar, currency, amountAtomic)
				s.Territory.broadcast()
			}

		case "ball_escape_start":
			userID := parseUserID(incoming["userId"])
			currency, _ := incoming["currency"].(string)
			diff, _ := incoming["difficulty"].(string)
			betFloat, _ := incoming["bet"].(float64)

			currency = NormalizeCurrency(currency)
			var betAtomic uint64
			if currency == ledger.AssetTON {
				betAtomic = ledger.TONToNano(betFloat)
			} else {
				betAtomic = uint64(betFloat)
			}

			res, err := s.BallEscape.StartRound(r.Context(), userID, currency, betAtomic, diff)
			if err == nil {
				data, _ := json.Marshal(map[string]any{
					"type":   "ball_escape_started",
					"result": res,
				})
				ch <- data
			} else {
				data, _ := json.Marshal(map[string]any{
					"type":  "ball_escape_error",
					"error": err.Error(),
				})
				ch <- data
			}

		case "ball_escape_claim":
			userID := parseUserID(incoming["userId"])
			roundID := int(parseUserID(incoming["roundId"]))
			serverSeed, _ := incoming["serverSeed"].(string)
			sig, _ := incoming["sig"].(string)
			mult, _ := incoming["multiplier"].(float64)
			isWin, _ := incoming["isWin"].(bool)
			betFloat, _ := incoming["bet"].(float64)

			betAtomic := uint64(betFloat)
			res, err := s.BallEscape.ClaimRound(r.Context(), userID, roundID, serverSeed, sig, mult, isWin, betAtomic)
			if err == nil {
				data, _ := json.Marshal(map[string]any{
					"type":   "ball_escape_claimed",
					"result": res,
				})
				ch <- data
			} else {
				data, _ := json.Marshal(map[string]any{
					"type":  "ball_escape_error",
					"error": err.Error(),
				})
				ch <- data
			}

		case "ping":
			ch <- []byte(`{"type":"pong"}`)
		}
	}
}

func parseUserID(val any) int64 {
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		id, _ := strconv.ParseInt(v, 10, 64)
		return id
	default:
		return 0
	}
}
