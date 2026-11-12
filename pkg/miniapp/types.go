package miniapp

// ThemeParams contains the user's current theme settings.
type ThemeParams struct {
	BgColor                *string `json:"bg_color,omitempty"`
	TextColor              *string `json:"text_color,omitempty"`
	HintColor              *string `json:"hint_color,omitempty"`
	LinkColor              *string `json:"link_color,omitempty"`
	ButtonColor            *string `json:"button_color,omitempty"`
	ButtonTextColor        *string `json:"button_text_color,omitempty"`
	SecondaryBgColor       *string `json:"secondary_bg_color,omitempty"`
	HeaderBgColor          *string `json:"header_bg_color,omitempty"`
	BottomBarBgColor       *string `json:"bottom_bar_bg_color,omitempty"`
	AccentTextColor        *string `json:"accent_text_color,omitempty"`
	SectionBgColor         *string `json:"section_bg_color,omitempty"`
	SectionHeaderTextColor *string `json:"section_header_text_color,omitempty"`
	SectionSeparatorColor  *string `json:"section_separator_color,omitempty"`
	SubtitleTextColor      *string `json:"subtitle_text_color,omitempty"`
	DestructiveTextColor   *string `json:"destructive_text_color,omitempty"`
}

// StoryShareParams describes additional sharing settings for the native story editor.
type StoryShareParams struct {
	Text       *string          `json:"text,omitempty"`
	WidgetLink *StoryWidgetLink `json:"widget_link,omitempty"`
}

// StoryWidgetLink describes a widget link to be included in a story.
type StoryWidgetLink struct {
	URL  string  `json:"url"`
	Name *string `json:"name,omitempty"`
}

// ScanQrPopupParams describes the native popup for scanning QR codes.
type ScanQrPopupParams struct {
	Text *string `json:"text,omitempty"`
}

// PopupParams describes the native popup.
type PopupParams struct {
	Title   *string       `json:"title,omitempty"`
	Message string        `json:"message"`
	Buttons []PopupButton `json:"buttons,omitempty"`
}

// PopupButton describes a native popup button.
type PopupButton struct {
	ID   *string `json:"id,omitempty"`
	Type *string `json:"type,omitempty"` // "default", "ok", "close", "cancel", "destructive"
	Text *string `json:"text,omitempty"`
}

// EmojiStatusParams describes additional settings for setting an emoji status.
type EmojiStatusParams struct {
	Duration *int `json:"duration,omitempty"`
}

// DownloadFileParams describes the parameters for the file download request.
type DownloadFileParams struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
}

// SafeAreaInset represents the system-defined safe area insets in pixels.
type SafeAreaInset struct {
	Top    int `json:"top"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
	Right  int `json:"right"`
}

// ContentSafeAreaInset represents the content-defined safe area insets in pixels.
type ContentSafeAreaInset struct {
	Top    int `json:"top"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
	Right  int `json:"right"`
}

// LocationData contains data about the current device location.
type LocationData struct {
	Latitude           float64  `json:"latitude"`
	Longitude          float64  `json:"longitude"`
	Altitude           *float64 `json:"altitude,omitempty"`
	Course             *float64 `json:"course,omitempty"`
	Speed              *float64 `json:"speed,omitempty"`
	HorizontalAccuracy *float64 `json:"horizontal_accuracy,omitempty"`
	VerticalAccuracy   *float64 `json:"vertical_accuracy,omitempty"`
	CourseAccuracy     *float64 `json:"course_accuracy,omitempty"`
	SpeedAccuracy      *float64 `json:"speed_accuracy,omitempty"`
}

// AccelerometerStartParams defines parameters for starting accelerometer tracking.
type AccelerometerStartParams struct {
	RefreshRate *int `json:"refresh_rate,omitempty"` // 20 to 1000 ms, default 1000
}

// AccelerometerData contains accelerometer measurements in m/s^2.
type AccelerometerData struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// DeviceOrientationStartParams defines parameters for device orientation tracking.
type DeviceOrientationStartParams struct {
	RefreshRate  *int  `json:"refresh_rate,omitempty"`
	NeedAbsolute *bool `json:"need_absolute,omitempty"`
}

// DeviceOrientationData contains orientation angles in radians.
type DeviceOrientationData struct {
	Absolute bool    `json:"absolute"`
	Alpha    float64 `json:"alpha"`
	Beta     float64 `json:"beta"`
	Gamma    float64 `json:"gamma"`
}

// GyroscopeStartParams defines parameters for gyroscope tracking.
type GyroscopeStartParams struct {
	RefreshRate *int `json:"refresh_rate,omitempty"`
}

// GyroscopeData contains rotation rates in rad/s.
type GyroscopeData struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// BiometricRequestAccessParams describes request access parameters.
type BiometricRequestAccessParams struct {
	Reason *string `json:"reason,omitempty"`
}

// BiometricAuthenticateParams describes authentication parameters.
type BiometricAuthenticateParams struct {
	Reason *string `json:"reason,omitempty"`
}

// BiometricManagerStatus represents the current biometric state on device.
type BiometricManagerStatus struct {
	IsInited              bool   `json:"is_inited"`
	IsBiometricAvailable  bool   `json:"is_biometric_available"`
	BiometricType         string `json:"biometric_type"` // "finger", "face", "unknown"
	IsAccessRequested     bool   `json:"is_access_requested"`
	IsAccessGranted       bool   `json:"is_access_granted"`
	IsBiometricTokenSaved bool   `json:"is_biometric_token_saved"`
	DeviceID              string `json:"device_id"`
}

// ServerlessCall represents a call to serverless function endpoints.
type ServerlessCall struct {
	Name  string `json:"name"`
	Input any    `json:"input,omitempty"`
}

// ServerlessError represents an error returned by Telegram Serverless platform.
type ServerlessError struct {
	Message    string `json:"message"`
	Type       string `json:"type,omitempty"` // "ENDPOINT_ERROR", "UNAUTHORIZED", etc.
	Status     int    `json:"status"`
	Parameters any    `json:"parameters,omitempty"`
}

// WebAppUser contains the data of the Mini App user.
type WebAppUser struct {
	ID                    int64   `json:"id"`
	IsBot                 *bool   `json:"is_bot,omitempty"`
	FirstName             string  `json:"first_name"`
	LastName              *string `json:"last_name,omitempty"`
	Username              *string `json:"username,omitempty"`
	LanguageCode          *string `json:"language_code,omitempty"`
	IsPremium             *bool   `json:"is_premium,omitempty"`
	AddedToAttachmentMenu *bool   `json:"added_to_attachment_menu,omitempty"`
	AllowsWriteToPM       *bool   `json:"allows_write_to_pm,omitempty"`
	PhotoURL              *string `json:"photo_url,omitempty"`
}

// WebAppChat represents the chat from which the Mini App was launched.
type WebAppChat struct {
	ID       int64   `json:"id"`
	Type     string  `json:"type"` // "group", "supergroup", "channel"
	Title    string  `json:"title"`
	Username *string `json:"username,omitempty"`
	PhotoURL *string `json:"photo_url,omitempty"`
}

// WebAppInitData contains data transferred to the Mini App when opened.
type WebAppInitData struct {
	QueryID                 *string     `json:"query_id,omitempty"`
	ChatJoinRequestQueryID *string     `json:"chat_join_request_query_id,omitempty"`
	User                    *WebAppUser `json:"user,omitempty"`
	Receiver                *WebAppUser `json:"receiver,omitempty"`
	Chat                    *WebAppChat `json:"chat,omitempty"`
	ChatType                *string     `json:"chat_type,omitempty"` // "sender", "private", "group", "supergroup", "channel"
	ChatInstance            *string     `json:"chat_instance,omitempty"`
	StartParam              *string     `json:"start_param,omitempty"`
	CanSendAfter            *int        `json:"can_send_after,omitempty"`
	AuthDate                int64       `json:"auth_date"`
	Hash                    string      `json:"hash"`
	Signature               *string     `json:"signature,omitempty"`
}

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// StringPtr returns a pointer to a string.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int.
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr returns a pointer to an int64.
func Int64Ptr(i int64) *int64 {
	return &i
}

// BoolPtr returns a pointer to a bool.
func BoolPtr(b bool) *bool {
	return &b
}

// Float64Ptr returns a pointer to a float64.
func Float64Ptr(f float64) *float64 {
	return &f
}
