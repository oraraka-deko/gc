package miniapp

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Known Telegram public keys for third-party InitData validation (Ed25519 hex).
const (
	TelegramPublicKeyTestHex = "40055058a4ee38156a06562e52eece92a771bcd8346a8c4615cb7376eddf72ec"
	TelegramPublicKeyProdHex = "e7bf03a2fa4602af4580703d88dda5bb59f32ed8b02a56c187fe7d34caed242d"
)

var (
	ErrMissingHash       = errors.New("missing hash in initData")
	ErrMissingAuthDate   = errors.New("missing auth_date in initData")
	ErrExpired           = errors.New("initData is expired")
	ErrInvalidSignature = errors.New("invalid signature or hash")
)

// ParseInitData parses a raw query string into a WebAppInitData struct.
func ParseInitData(rawQuery string) (*WebAppInitData, error) {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse initData query string: %w", err)
	}

	data := &WebAppInitData{}

	if val := values.Get("query_id"); val != "" {
		data.QueryID = &val
	}
	if val := values.Get("chat_join_request_query_id"); val != "" {
		data.ChatJoinRequestQueryID = &val
	}
	if val := values.Get("chat_type"); val != "" {
		data.ChatType = &val
	}
	if val := values.Get("chat_instance"); val != "" {
		data.ChatInstance = &val
	}
	if val := values.Get("start_param"); val != "" {
		data.StartParam = &val
	}
	if val := values.Get("hash"); val != "" {
		data.Hash = val
	}
	if val := values.Get("signature"); val != "" {
		data.Signature = &val
	}

	if val := values.Get("can_send_after"); val != "" {
		if c, err := strconv.Atoi(val); err == nil {
			data.CanSendAfter = &c
		}
	}

	if val := values.Get("auth_date"); val != "" {
		if ad, err := strconv.ParseInt(val, 10, 64); err == nil {
			data.AuthDate = ad
		}
	}

	if val := values.Get("user"); val != "" {
		var user WebAppUser
		if err := json.Unmarshal([]byte(val), &user); err == nil {
			data.User = &user
		}
	}

	if val := values.Get("receiver"); val != "" {
		var receiver WebAppUser
		if err := json.Unmarshal([]byte(val), &receiver); err == nil {
			data.Receiver = &receiver
		}
	}

	if val := values.Get("chat"); val != "" {
		var chat WebAppChat
		if err := json.Unmarshal([]byte(val), &chat); err == nil {
			data.Chat = &chat
		}
	}

	return data, nil
}

// BuildDataCheckString builds the sorted key=value string separated by \n.
func BuildDataCheckString(params map[string]string, excludeHashAndSig bool, prefix string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if excludeHashAndSig && (k == "hash" || k == "signature") {
			continue
		}
		if !excludeHashAndSig && k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys)+1)
	if prefix != "" {
		lines = append(lines, prefix)
	}
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", k, params[k]))
	}

	return strings.Join(lines, "\n")
}

// Validate verifies the Telegram Bot API HMAC-SHA256 signature of initData.
func Validate(rawQuery string, botToken string, maxAge time.Duration) (bool, error) {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return false, fmt.Errorf("invalid query string: %w", err)
	}

	expectedHash := values.Get("hash")
	if expectedHash == "" {
		return false, ErrMissingHash
	}

	authDateStr := values.Get("auth_date")
	if authDateStr == "" {
		return false, ErrMissingAuthDate
	}

	authDateUnix, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid auth_date: %w", err)
	}

	if maxAge > 0 {
		authTime := time.Unix(authDateUnix, 0)
		if time.Since(authTime) > maxAge {
			return false, ErrExpired
		}
	}

	flatParams := make(map[string]string, len(values))
	for k := range values {
		flatParams[k] = values.Get(k)
	}

	dataCheckString := BuildDataCheckString(flatParams, false, "")

	// secret_key = HMAC_SHA256("WebAppData", bot_token)
	macSecret := hmac.New(sha256.New, []byte("WebAppData"))
	macSecret.Write([]byte(botToken))
	secretKey := macSecret.Sum(nil)

	// calculated_hash = hex(HMAC_SHA256(secret_key, data_check_string))
	macData := hmac.New(sha256.New, secretKey)
	macData.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(macData.Sum(nil))

	if !strings.EqualFold(calculatedHash, expectedHash) {
		return false, ErrInvalidSignature
	}

	return true, nil
}

// ValidateThirdParty verifies the Ed25519 signature for third-party use.
func ValidateThirdParty(rawQuery string, botID string, isTestEnv bool, maxAge time.Duration) (bool, error) {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return false, fmt.Errorf("invalid query string: %w", err)
	}

	sigBase64 := values.Get("signature")
	if sigBase64 == "" {
		return false, errors.New("missing signature parameter")
	}

	authDateStr := values.Get("auth_date")
	if authDateStr != "" && maxAge > 0 {
		if authDateUnix, err := strconv.ParseInt(authDateStr, 10, 64); err == nil {
			if time.Since(time.Unix(authDateUnix, 0)) > maxAge {
				return false, ErrExpired
			}
		}
	}

	flatParams := make(map[string]string, len(values))
	for k := range values {
		flatParams[k] = values.Get(k)
	}

	prefix := fmt.Sprintf("%s:WebAppData", botID)
	dataCheckString := BuildDataCheckString(flatParams, true, prefix)

	// Decode signature from base64url or standard base64
	sigBytes, err := base64.RawURLEncoding.DecodeString(sigBase64)
	if err != nil {
		sigBytes, err = base64.URLEncoding.DecodeString(sigBase64)
		if err != nil {
			sigBytes, err = base64.StdEncoding.DecodeString(sigBase64)
			if err != nil {
				return false, fmt.Errorf("failed to decode signature: %w", err)
			}
		}
	}

	pubKeyHex := TelegramPublicKeyProdHex
	if isTestEnv {
		pubKeyHex = TelegramPublicKeyTestHex
	}
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return false, fmt.Errorf("failed to decode public key hex: %w", err)
	}

	if !ed25519.Verify(pubKeyBytes, []byte(dataCheckString), sigBytes) {
		return false, ErrInvalidSignature
	}

	return true, nil
}

// GenerateSignedInitData creates a valid signed query string for testing purposes.
func GenerateSignedInitData(botToken string, params map[string]string) string {
	if _, ok := params["auth_date"]; !ok {
		params["auth_date"] = strconv.FormatInt(time.Now().Unix(), 10)
	}

	dataCheckString := BuildDataCheckString(params, false, "")

	macSecret := hmac.New(sha256.New, []byte("WebAppData"))
	macSecret.Write([]byte(botToken))
	secretKey := macSecret.Sum(nil)

	macData := hmac.New(sha256.New, secretKey)
	macData.Write([]byte(dataCheckString))
	calcHash := hex.EncodeToString(macData.Sum(nil))

	params["hash"] = calcHash

	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values.Encode()
}
