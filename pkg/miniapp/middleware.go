package miniapp

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const (
	UserContextKey contextKey = "telegram_user"
	MaxAuthAge                = 24 * time.Hour
)

// UserFromContext retrieves the authenticated WebAppUser from the request context.
func UserFromContext(ctx context.Context) *WebAppUser {
	if u, ok := ctx.Value(UserContextKey).(*WebAppUser); ok {
		return u
	}
	return nil
}

// TelegramAuthMiddleware cryptographically verifies Telegram initData before granting access.
func TelegramAuthMiddleware(botToken string, maxAge time.Duration) func(http.Handler) http.Handler {
	if maxAge <= 0 {
		maxAge = MaxAuthAge
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			EnableCORS(w)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			// 1. Extract raw initData from HTTP Headers or Query String
			var rawInitData string

			// Check X-Telegram-Init-Data header
			if val := r.Header.Get("X-Telegram-Init-Data"); val != "" {
				rawInitData = val
			} else if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "tma ") {
				rawInitData = strings.TrimPrefix(authHeader, "tma ")
			} else if queryAuth := r.URL.Query().Get("auth"); queryAuth != "" {
				rawInitData = queryAuth
			}

			// If completely empty, reject immediately
			if strings.TrimSpace(rawInitData) == "" {
				http.Error(w, `{"error":"unauthorized","message":"missing telegram initData"}`, http.StatusUnauthorized)
				return
			}

			// 2. Validate cryptographic signature
			valid, err := Validate(rawInitData, botToken, maxAge)
			if !valid || err != nil {
				// Also check third-party Ed25519 validation if available
				validTP, errTP := ValidateThirdParty(rawInitData, "", false, maxAge)
				if !validTP || errTP != nil {
					http.Error(w, `{"error":"unauthorized","message":"invalid cryptographic initData signature"}`, http.StatusUnauthorized)
					return
				}
			}

			// 3. Parse user metadata and inject into Context
			parsedData, err := ParseInitData(rawInitData)
			if err != nil || parsedData.User == nil {
				http.Error(w, `{"error":"unauthorized","message":"missing user identity in initData"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, parsedData.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// EnableCORS configures standard CORS headers for Telegram Web client cross-origin requests.
func EnableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Telegram-Init-Data")
}
