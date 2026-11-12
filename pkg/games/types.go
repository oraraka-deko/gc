package games

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"gc/pkg/ledger"
)

const (
	MaxRequestBodyBytes = 64 * 1024 // 64 KB limit against memory exhaustion
	MaxBetAmountStars   = 1_000_000
	MinBetAmountStars   = 1
	MaxBetAmountTONNano = 1000 * 1e9       // 1000 TON
	MinBetAmountTONNano = uint64(0.01 * 1e9) // 0.01 TON
)

// CryptoRandInt returns a secure random integer in [0, max).
func CryptoRandInt(max int64) int64 {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return time.Now().UnixNano() % max
	}
	return n.Int64()
}

// GenerateServerSeed produces a cryptographically secure seed string and its SHA-256 hash.
func GenerateServerSeed(prefix string, roundID int) (seed string, seedHash string) {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	rndNum := CryptoRandInt(9999999)
	seed = fmt.Sprintf("%s_seed_%d_%s_%d", prefix, roundID, hex.EncodeToString(bytes[:8]), rndNum)
	hash := sha256.Sum256([]byte(seed))
	seedHash = hex.EncodeToString(hash[:])
	return seed, seedHash
}

// NormalizeCurrency standardizes currency string to AssetStars ("XTR") or AssetTON ("TON").
func NormalizeCurrency(c string) string {
	switch strings.ToUpper(strings.TrimSpace(c)) {
	case "TON", "GRAM":
		return ledger.AssetTON
	case "STARS", "STAR", "XTR":
		return ledger.AssetStars
	case "USDT":
		return ledger.AssetUSDT
	default:
		return ledger.AssetStars
	}
}

// FormatGameCurrencyAmount formats an atomic integer amount into readable display text.
func FormatGameCurrencyAmount(amount uint64, asset string) string {
	return ledger.FormatAtomic(amount, asset)
}
