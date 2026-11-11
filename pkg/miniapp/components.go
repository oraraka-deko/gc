package miniapp

// BackButton controls the back button displayed in the Mini App header.
type BackButton struct {
	IsVisible bool `json:"is_visible"`
}

func NewBackButton() *BackButton {
	return &BackButton{IsVisible: false}
}

func (b *BackButton) Show() *BackButton {
	b.IsVisible = true
	return b
}

func (b *BackButton) Hide() *BackButton {
	b.IsVisible = false
	return b
}

// BottomButton controls the main or secondary button displayed at the bottom of the Mini App.
type BottomButton struct {
	Type              string  `json:"type"` // "main" or "secondary"
	Text              string  `json:"text"`
	Color             string  `json:"color,omitempty"`
	TextColor         string  `json:"text_color,omitempty"`
	IsVisible         bool    `json:"is_visible"`
	IsActive          bool    `json:"is_active"`
	HasShineEffect    bool    `json:"has_shine_effect,omitempty"`
	Position          string  `json:"position,omitempty"` // "left", "right", "top", "bottom"
	IsProgressVisible bool    `json:"is_progress_visible"`
	IconCustomEmojiID *string `json:"icon_custom_emoji_id,omitempty"`
}

func NewMainButton(text string) *BottomButton {
	return &BottomButton{
		Type:      "main",
		Text:      text,
		IsVisible: true,
		IsActive:  true,
	}
}

func NewSecondaryButton(text string) *BottomButton {
	return &BottomButton{
		Type:      "secondary",
		Text:      text,
		Position:  "left",
		IsVisible: true,
		IsActive:  true,
	}
}

func (b *BottomButton) SetText(text string) *BottomButton {
	b.Text = text
	return b
}

func (b *BottomButton) SetColors(bg, text string) *BottomButton {
	b.Color = bg
	b.TextColor = text
	return b
}

func (b *BottomButton) SetShine(hasShine bool) *BottomButton {
	b.HasShineEffect = hasShine
	return b
}

func (b *BottomButton) SetPosition(pos string) *BottomButton {
	b.Position = pos
	return b
}

func (b *BottomButton) SetIconCustomEmojiID(emojiID string) *BottomButton {
	b.IconCustomEmojiID = &emojiID
	return b
}

func (b *BottomButton) Show() *BottomButton {
	b.IsVisible = true
	return b
}

func (b *BottomButton) Hide() *BottomButton {
	b.IsVisible = false
	return b
}

func (b *BottomButton) Enable() *BottomButton {
	b.IsActive = true
	return b
}

func (b *BottomButton) Disable() *BottomButton {
	b.IsActive = false
	return b
}

func (b *BottomButton) ShowProgress(leaveActive bool) *BottomButton {
	b.IsProgressVisible = true
	b.IsActive = leaveActive
	return b
}

func (b *BottomButton) HideProgress() *BottomButton {
	b.IsProgressVisible = false
	b.IsActive = true
	return b
}

// SettingsButton controls the Settings menu item.
type SettingsButton struct {
	IsVisible bool `json:"is_visible"`
}

func NewSettingsButton() *SettingsButton {
	return &SettingsButton{IsVisible: false}
}

func (s *SettingsButton) Show() *SettingsButton {
	s.IsVisible = true
	return s
}

func (s *SettingsButton) Hide() *SettingsButton {
	s.IsVisible = false
	return s
}

// HapticFeedback models the haptic feedback component.
type HapticFeedback struct{}

func (h *HapticFeedback) ImpactOccurred(style string) map[string]any {
	return map[string]any{
		"action": "impactOccurred",
		"style":  style, // "light", "medium", "heavy", "rigid", "soft"
	}
}

func (h *HapticFeedback) NotificationOccurred(notificationType string) map[string]any {
	return map[string]any{
		"action": "notificationOccurred",
		"type":   notificationType, // "error", "success", "warning"
	}
}

func (h *HapticFeedback) SelectionChanged() map[string]any {
	return map[string]any{
		"action": "selectionChanged",
	}
}

// CloudStorage models the cloud storage component.
type CloudStorage struct{}

func (c *CloudStorage) SetItem(key, value string) map[string]any {
	return map[string]any{"method": "setItem", "key": key, "value": value}
}

func (c *CloudStorage) GetItem(key string) map[string]any {
	return map[string]any{"method": "getItem", "key": key}
}

func (c *CloudStorage) GetItems(keys []string) map[string]any {
	return map[string]any{"method": "getItems", "keys": keys}
}

func (c *CloudStorage) RemoveItem(key string) map[string]any {
	return map[string]any{"method": "removeItem", "key": key}
}

func (c *CloudStorage) RemoveItems(keys []string) map[string]any {
	return map[string]any{"method": "removeItems", "keys": keys}
}

func (c *CloudStorage) GetKeys() map[string]any {
	return map[string]any{"method": "getKeys"}
}

// BiometricManager models device biometrics management.
type BiometricManager struct {
	IsInited             bool   `json:"is_inited"`
	IsBiometricAvailable bool   `json:"is_biometric_available"`
	BiometricType        string `json:"biometric_type"` // "finger", "face", "unknown"
	IsAccessRequested    bool   `json:"is_access_requested"`
	IsAccessGranted      bool   `json:"is_access_granted"`
}

// LocationManager models location services control.
type LocationManager struct {
	IsInited            bool `json:"is_inited"`
	IsLocationAvailable bool `json:"is_location_available"`
	IsAccessRequested   bool `json:"is_access_requested"`
	IsAccessGranted     bool `json:"is_access_granted"`
}

// DeviceStorage models local device storage (up to 5MB).
type DeviceStorage struct{}

func (d *DeviceStorage) SetItem(key, value string) map[string]any {
	return map[string]any{"method": "setItem", "key": key, "value": value}
}

func (d *DeviceStorage) GetItem(key string) map[string]any {
	return map[string]any{"method": "getItem", "key": key}
}

func (d *DeviceStorage) RemoveItem(key string) map[string]any {
	return map[string]any{"method": "removeItem", "key": key}
}

func (d *DeviceStorage) Clear() map[string]any {
	return map[string]any{"method": "clear"}
}

// SecureStorage models secure encrypted keychain/keystore storage (up to 10 items).
type SecureStorage struct{}

func (s *SecureStorage) SetItem(key, value string) map[string]any {
	return map[string]any{"method": "setItem", "key": key, "value": value}
}

func (s *SecureStorage) GetItem(key string) map[string]any {
	return map[string]any{"method": "getItem", "key": key}
}

func (s *SecureStorage) RestoreItem(key string) map[string]any {
	return map[string]any{"method": "restoreItem", "key": key}
}

func (s *SecureStorage) RemoveItem(key string) map[string]any {
	return map[string]any{"method": "removeItem", "key": key}
}

func (s *SecureStorage) Clear() map[string]any {
	return map[string]any{"method": "clear"}
}
