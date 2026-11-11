package main

import (
	"regexp"
	"strconv"
	"strings"
)

// ConversionQuery holds parsed conversion request details.
type ConversionQuery struct {
	Amount       float64
	FromCurrency string // Normalized: TON, TRX, USDT, BTC, ETH
	ToCurrency   string // Normalized: TON, TRX, USDT, BTC, ETH, USD
	IsSingleCoin bool
	RawInput     string
}

var (
	tokenRegex = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?|[a-zA-Z$]+)`)

	currencyAliases = map[string]string{
		// TON / Gram aliases
		"ton":      "TON",
		"gram":     "TON",
		"tons":     "TON",
		"grams":    "TON",
		"gr":       "TON",
		"the-open": "TON",

		// TRON / TRX aliases
		"tron": "TRX",
		"trx":  "TRX",

		// USDT / USD aliases
		"usdt":    "USDT",
		"usd":     "USDT",
		"tether":  "USDT",
		"$":       "USDT",
		"dollar":  "USDT",
		"dollars": "USDT",

		// BTC
		"btc":     "BTC",
		"bitcoin": "BTC",

		// ETH
		"eth":      "ETH",
		"ether":    "ETH",
		"ethereum": "ETH",
	}

	fillerWords = map[string]bool{
		"to":      true,
		"in":      true,
		"into":    true,
		"for":     true,
		"of":      true,
		"convert": true,
		"price":   true,
		"rate":    true,
		"rates":   true,
		"how":     true,
		"much":    true,
		"is":      true,
		"the":     true,
		"check":   true,
	}
)

// ParseConversionQuery analyzes a user string (group message, DM, or inline query)
// and extracts amount, from-currency, and to-currency.
func ParseConversionQuery(input string) (*ConversionQuery, bool) {
	clean := strings.TrimSpace(input)
	if clean == "" {
		return nil, false
	}

	// Remove leading slash if user typed /price 1 ton or /ton
	if strings.HasPrefix(clean, "/") {
		parts := strings.SplitN(clean, " ", 2)
		cmd := strings.ToLower(parts[0])
		if cmd == "/price" || cmd == "/convert" || cmd == "/rate" {
			if len(parts) > 1 {
				clean = parts[1]
			} else {
				return nil, false
			}
		} else {
			// e.g. /ton -> ton
			clean = strings.TrimPrefix(cmd, "/")
			if len(parts) > 1 {
				clean += " " + parts[1]
			}
		}
	}

	matches := tokenRegex.FindAllString(strings.ToLower(clean), -1)
	if len(matches) == 0 {
		return nil, false
	}

	var amount *float64
	var foundCurrencies []string

	for _, token := range matches {
		if fillerWords[token] {
			continue
		}

		// 1. Check if token is a number (e.g. "1", "500", "0.25")
		if num, err := strconv.ParseFloat(token, 64); err == nil && num > 0 {
			if amount == nil {
				amount = &num
			}
			continue
		}

		// 2. Check if token is a currency alias
		if canon, ok := currencyAliases[token]; ok {
			foundCurrencies = append(foundCurrencies, canon)
			continue
		}
	}

	if len(foundCurrencies) == 0 {
		return nil, false
	}

	finalAmount := 1.0
	if amount != nil {
		finalAmount = *amount
	}

	query := &ConversionQuery{
		Amount:   finalAmount,
		RawInput: input,
	}

	if len(foundCurrencies) == 1 {
		query.FromCurrency = foundCurrencies[0]
		query.ToCurrency = "USD"
		query.IsSingleCoin = true
	} else {
		query.FromCurrency = foundCurrencies[0]
		query.ToCurrency = foundCurrencies[1]
		query.IsSingleCoin = false
	}

	return query, true
}
