package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CryptoRate represents the price and 24h change of a cryptocurrency.
type CryptoRate struct {
	Symbol    string    `json:"symbol"`
	PriceUSD  float64   `json:"price_usd"`
	Change24h string    `json:"change_24h"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RateManager fetches and caches realtime cryptocurrency rates from multiple sources.
type RateManager struct {
	httpClient *http.Client
	cacheTTL   time.Duration
	mu         sync.RWMutex
	cache      map[string]*CryptoRate
}

// NewRateManager creates a new RateManager.
func NewRateManager() *RateManager {
	return &RateManager{
		httpClient: &http.Client{Timeout: 6 * time.Second},
		cacheTTL:   20 * time.Second, // Cache for 20 seconds to keep data fresh yet avoid rate limits
		cache:      make(map[string]*CryptoRate),
	}
}

// GetRate retrieves current rate for a symbol (TON, TRX, USDT, BTC, ETH) using fallback sources.
func (rm *RateManager) GetRate(ctx context.Context, symbol string) (*CryptoRate, error) {
	normSymbol := strings.ToUpper(strings.TrimSpace(symbol))
	if normSymbol == "GRAM" {
		normSymbol = "TON"
	}
	if normSymbol == "TRON" {
		normSymbol = "TRX"
	}
	if normSymbol == "USD" || normSymbol == "TETHER" {
		normSymbol = "USDT"
	}

	if normSymbol == "USDT" {
		return &CryptoRate{
			Symbol:    "USDT",
			PriceUSD:  1.0,
			Change24h: "0.00%",
			Source:    "peg (1:1 USD)",
			UpdatedAt: time.Now(),
		}, nil
	}

	rm.mu.RLock()
	cached, ok := rm.cache[normSymbol]
	if ok && time.Since(cached.UpdatedAt) < rm.cacheTTL {
		rm.mu.RUnlock()
		return cached, nil
	}
	rm.mu.RUnlock()

	var rate *CryptoRate
	var err error

	// Source 1: TonAPI (best for TON, TRX, USDT)
	if normSymbol == "TON" || normSymbol == "TRX" {
		rate, err = rm.fetchTonAPI(ctx, normSymbol)
		if err == nil && rate != nil && rate.PriceUSD > 0 {
			rm.saveCache(normSymbol, rate)
			return rate, nil
		}
	}

	// Source 2: Binance
	rate, err = rm.fetchBinance(ctx, normSymbol)
	if err == nil && rate != nil && rate.PriceUSD > 0 {
		rm.saveCache(normSymbol, rate)
		return rate, nil
	}

	// Source 3: CoinGecko
	rate, err = rm.fetchCoinGecko(ctx, normSymbol)
	if err == nil && rate != nil && rate.PriceUSD > 0 {
		rm.saveCache(normSymbol, rate)
		return rate, nil
	}

	// Source 4: OKX
	rate, err = rm.fetchOKX(ctx, normSymbol)
	if err == nil && rate != nil && rate.PriceUSD > 0 {
		rm.saveCache(normSymbol, rate)
		return rate, nil
	}

	// If failed, return cached even if expired
	rm.mu.RLock()
	stale, hasStale := rm.cache[normSymbol]
	rm.mu.RUnlock()
	if hasStale {
		return stale, nil
	}

	return nil, fmt.Errorf("failed to fetch rate for %s from all sources", normSymbol)
}

func (rm *RateManager) saveCache(symbol string, rate *CryptoRate) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.cache[symbol] = rate
}

// Source 1: TonAPI
func (rm *RateManager) fetchTonAPI(ctx context.Context, symbol string) (*CryptoRate, error) {
	url := fmt.Sprintf("https://tonapi.io/v2/rates?tokens=%s&currencies=usd", strings.ToLower(symbol))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := rm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tonapi status %d", resp.StatusCode)
	}

	var data struct {
		Rates map[string]struct {
			Prices struct {
				USD float64 `json:"USD"`
			} `json:"prices"`
			Diff24h struct {
				USD string `json:"USD"`
			} `json:"diff_24h"`
		} `json:"rates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	tokenData, ok := data.Rates[symbol]
	if !ok || tokenData.Prices.USD <= 0 {
		return nil, fmt.Errorf("empty rate for %s", symbol)
	}

	return &CryptoRate{
		Symbol:    symbol,
		PriceUSD:  tokenData.Prices.USD,
		Change24h: tokenData.Diff24h.USD,
		Source:    "tonapi.io",
		UpdatedAt: time.Now(),
	}, nil
}

// Source 2: Binance
func (rm *RateManager) fetchBinance(ctx context.Context, symbol string) (*CryptoRate, error) {
	pair := symbol + "USDT"
	url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/24hr?symbol=%s", pair)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := rm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance status %d", resp.StatusCode)
	}

	var data struct {
		LastPrice          string `json:"lastPrice"`
		PriceChangePercent string `json:"priceChangePercent"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	price, err := strconv.ParseFloat(data.LastPrice, 64)
	if err != nil || price <= 0 {
		return nil, fmt.Errorf("invalid binance price: %s", data.LastPrice)
	}

	change := data.PriceChangePercent
	if !strings.HasPrefix(change, "-") && !strings.HasPrefix(change, "+") {
		change = "+" + change
	}
	change += "%"

	return &CryptoRate{
		Symbol:    symbol,
		PriceUSD:  price,
		Change24h: change,
		Source:    "binance",
		UpdatedAt: time.Now(),
	}, nil
}

// Source 3: CoinGecko
func (rm *RateManager) fetchCoinGecko(ctx context.Context, symbol string) (*CryptoRate, error) {
	idMap := map[string]string{
		"TON":  "the-open-network",
		"TRX":  "tron",
		"USDT": "tether",
		"BTC":  "bitcoin",
		"ETH":  "ethereum",
	}
	id, ok := idMap[symbol]
	if !ok {
		return nil, fmt.Errorf("unsupported symbol for coingecko: %s", symbol)
	}

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd&include_24hr_change=true", id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := rm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko status %d", resp.StatusCode)
	}

	var data map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	coinData, ok := data[id]
	if !ok {
		return nil, fmt.Errorf("coingecko data missing for %s", id)
	}

	price := coinData["usd"]
	changeVal := coinData["usd_24h_change"]

	changeStr := fmt.Sprintf("%+.2f%%", changeVal)

	return &CryptoRate{
		Symbol:    symbol,
		PriceUSD:  price,
		Change24h: changeStr,
		Source:    "coingecko",
		UpdatedAt: time.Now(),
	}, nil
}

// Source 4: OKX
func (rm *RateManager) fetchOKX(ctx context.Context, symbol string) (*CryptoRate, error) {
	instID := fmt.Sprintf("%s-USDT", symbol)
	url := fmt.Sprintf("https://www.okx.com/api/v5/market/ticker?instId=%s", instID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := rm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("okx status %d", resp.StatusCode)
	}

	var data struct {
		Data []struct {
			Last string `json:"last"`
			Open24h string `json:"open24h"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if len(data.Data) == 0 {
		return nil, fmt.Errorf("empty okx data")
	}

	price, err := strconv.ParseFloat(data.Data[0].Last, 64)
	if err != nil || price <= 0 {
		return nil, fmt.Errorf("invalid okx price")
	}

	changeStr := "N/A"
	if openPrice, err := strconv.ParseFloat(data.Data[0].Open24h, 64); err == nil && openPrice > 0 {
		pct := ((price - openPrice) / openPrice) * 100.0
		changeStr = fmt.Sprintf("%+.2f%%", pct)
	}

	return &CryptoRate{
		Symbol:    symbol,
		PriceUSD:  price,
		Change24h: changeStr,
		Source:    "okx",
		UpdatedAt: time.Now(),
	}, nil
}
