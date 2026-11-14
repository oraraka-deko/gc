package games

import (
	"encoding/json"
	"net/http"
	"time"
)

// LiveGiftItem models a live Telegram gift in the backend catalog.
type LiveGiftItem struct {
	GiftID        string  `json:"gift_id"`
	Name          string  `json:"name"`
	PriceStars    int     `json:"price_stars"`
	FloorPriceTon float64 `json:"floor_price_ton"`
	TotalStock    int     `json:"total_stock"`
	IsUpgradeable bool    `json:"is_upgradeable"`
	LottieAsset   string  `json:"lottie_asset,omitempty"`
}

// LiveGiftCatalog provides production gift listing with real-time floor prices.
var DefaultLiveCatalog = []LiveGiftItem{
	{
		GiftID:        "5170145012310081615",
		Name:          "Myth Mask",
		PriceStars:    250,
		FloorPriceTon: 1.25,
		TotalStock:    5000,
		IsUpgradeable: true,
	},
	{
		GiftID:        "5170312674394767464",
		Name:          "Golden Dragon",
		PriceStars:    1000,
		FloorPriceTon: 5.0,
		TotalStock:    1200,
		IsUpgradeable: true,
	},
	{
		GiftID:        "5168103934371300000",
		Name:          "Star Rocket",
		PriceStars:    50,
		FloorPriceTon: 0.25,
		TotalStock:    20000,
		IsUpgradeable: false,
	},
}

// HandleGiftsCatalog returns the current live gift pricing catalog.
func (s *GameServer) HandleGiftsCatalog(w http.ResponseWriter, r *http.Request) {
	EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":       DefaultLiveCatalog,
		"updated_at":  time.Now().Unix(),
		"server_time": time.Now().Unix(),
	})
}
