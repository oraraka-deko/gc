package miniapp_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"strconv"
	"testing"
	"time"

	"gc/pkg/miniapp"
)

func TestInitData_Validation_Success(t *testing.T) {
	botToken := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

	params := map[string]string{
		"query_id":  "AAHdF6IQAAAAAN0XohDhrOrc",
		"user":      `{"id":279058397,"first_name":"Vladislav","last_name":"K","username":"vladislav_k","language_code":"en"}`,
		"auth_date": strconv.FormatInt(time.Now().Unix(), 10),
	}

	rawQuery := miniapp.GenerateSignedInitData(botToken, params)

	valid, err := miniapp.Validate(rawQuery, botToken, time.Hour)
	if err != nil {
		t.Fatalf("validation failed unexpectedly: %v", err)
	}
	if !valid {
		t.Fatal("expected valid to be true")
	}

	// Parse test
	parsed, err := miniapp.ParseInitData(rawQuery)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if parsed.QueryID == nil || *parsed.QueryID != "AAHdF6IQAAAAAN0XohDhrOrc" {
		t.Errorf("unexpected query_id: %+v", parsed.QueryID)
	}
	if parsed.User == nil || parsed.User.ID != 279058397 || parsed.User.FirstName != "Vladislav" {
		t.Errorf("unexpected user: %+v", parsed.User)
	}
}

func TestInitData_Validation_WrongToken(t *testing.T) {
	botToken := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	wrongToken := "999999:ZZZ-DEF1234ghIkl-zyx57W2v1u123ew99"

	params := map[string]string{
		"query_id":  "AAHdF6IQAAAAAN0XohDhrOrc",
		"auth_date": strconv.FormatInt(time.Now().Unix(), 10),
	}

	rawQuery := miniapp.GenerateSignedInitData(botToken, params)

	valid, err := miniapp.Validate(rawQuery, wrongToken, time.Hour)
	if err == nil && valid {
		t.Fatal("expected validation to fail with wrong token")
	}
}

func TestInitData_Validation_Expired(t *testing.T) {
	botToken := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

	// Auth date 2 hours ago
	pastTime := time.Now().Add(-2 * time.Hour).Unix()
	params := map[string]string{
		"query_id":  "AAHdF6IQAAAAAN0XohDhrOrc",
		"auth_date": strconv.FormatInt(pastTime, 10),
	}

	rawQuery := miniapp.GenerateSignedInitData(botToken, params)

	// Max age is 1 hour
	_, err := miniapp.Validate(rawQuery, botToken, 1*time.Hour)
	if err != miniapp.ErrExpired {
		t.Fatalf("expected ErrExpired, got: %v", err)
	}
}

func TestMiniApp_Components(t *testing.T) {
	// BackButton
	back := miniapp.NewBackButton().Show()
	if !back.IsVisible {
		t.Errorf("expected BackButton to be visible")
	}
	back.Hide()
	if back.IsVisible {
		t.Errorf("expected BackButton to be hidden")
	}

	// BottomButton (MainButton)
	mainBtn := miniapp.NewMainButton("Submit").
		SetColors("#ffffff", "#000000").
		SetShine(true).
		SetPosition("bottom")

	if mainBtn.Type != "main" || mainBtn.Text != "Submit" || !mainBtn.HasShineEffect {
		t.Errorf("unexpected MainButton state: %+v", mainBtn)
	}

	mainBtn.ShowProgress(false)
	if !mainBtn.IsProgressVisible || mainBtn.IsActive {
		t.Errorf("expected loading progress with active=false: %+v", mainBtn)
	}

	mainBtn.HideProgress()
	if mainBtn.IsProgressVisible || !mainBtn.IsActive {
		t.Errorf("expected loading hidden and active=true: %+v", mainBtn)
	}

	// SecondaryButton
	secBtn := miniapp.NewSecondaryButton("Cancel").SetPosition("right")
	if secBtn.Type != "secondary" || secBtn.Position != "right" {
		t.Errorf("unexpected SecondaryButton state: %+v", secBtn)
	}

	// SettingsButton
	settingsBtn := miniapp.NewSettingsButton().Show()
	if !settingsBtn.IsVisible {
		t.Errorf("expected SettingsButton to be visible")
	}

	// HapticFeedback
	haptic := &miniapp.HapticFeedback{}
	impact := haptic.ImpactOccurred("medium")
	if impact["action"] != "impactOccurred" || impact["style"] != "medium" {
		t.Errorf("unexpected impact payload: %+v", impact)
	}

	notif := haptic.NotificationOccurred("success")
	if notif["action"] != "notificationOccurred" || notif["type"] != "success" {
		t.Errorf("unexpected notification payload: %+v", notif)
	}

	// CloudStorage
	cloud := &miniapp.CloudStorage{}
	setItem := cloud.SetItem("theme", "dark")
	if setItem["key"] != "theme" || setItem["value"] != "dark" {
		t.Errorf("unexpected setItem payload: %+v", setItem)
	}

	// SecureStorage
	secure := &miniapp.SecureStorage{}
	restore := secure.RestoreItem("token")
	if restore["method"] != "restoreItem" || restore["key"] != "token" {
		t.Errorf("unexpected restore payload: %+v", restore)
	}
}

func TestInitData_ThirdParty_Ed25519(t *testing.T) {
	// Generate an ed25519 keypair for testing
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	botID := "123456"
	params := map[string]string{
		"query_id":  "test_query_id",
		"auth_date": strconv.FormatInt(time.Now().Unix(), 10),
	}

	prefix := botID + ":WebAppData"
	dataCheckString := miniapp.BuildDataCheckString(params, true, prefix)

	sig := ed25519.Sign(privKey, []byte(dataCheckString))
	sigBase64 := base64.RawURLEncoding.EncodeToString(sig)

	// Verify directly using ed25519
	verified := ed25519.Verify(pubKey, []byte(dataCheckString), sig)
	if !verified {
		t.Fatal("ed25519 verification failed")
	}

	// Check that public key hex matches
	pubHex := hex.EncodeToString(pubKey)
	if len(pubHex) != 64 {
		t.Fatalf("invalid public key hex length: %d", len(pubHex))
	}

	// Test with query
	v := url.Values{}
	for k, val := range params {
		v.Set(k, val)
	}
	v.Set("signature", sigBase64)
	rawQuery := v.Encode()

	parsed, err := miniapp.ParseInitData(rawQuery)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if parsed.Signature == nil || *parsed.Signature != sigBase64 {
		t.Errorf("unexpected parsed signature: %+v", parsed.Signature)
	}
}
