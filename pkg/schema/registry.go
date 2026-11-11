package schema

import (
	"sync"
)

// Registry stores all registered schemas.
type Registry struct {
	mu      sync.RWMutex
	schemas map[string]*Schema
}

// GlobalRegistry is the default instance populated with all Bot and Mini App schemas.
var GlobalRegistry = NewRegistry()

func NewRegistry() *Registry {
	r := &Registry{
		schemas: make(map[string]*Schema),
	}
	r.registerDefaultSchemas()
	return r
}

func (r *Registry) Register(s *Schema) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schemas[s.Name] = s
}

func (r *Registry) Get(name string) (*Schema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.schemas[name]
	return s, ok
}

func (r *Registry) List(kinds ...SchemaKind) []*Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]*Schema, 0, len(r.schemas))
	kindMap := make(map[SchemaKind]bool)
	for _, k := range kinds {
		kindMap[k] = true
	}
	for _, s := range r.schemas {
		if len(kinds) == 0 || kindMap[s.Kind] {
			res = append(res, s)
		}
	}
	return res
}

func (r *Registry) registerDefaultSchemas() {
	// ==========================================
	// 1. Telegram Bot API Methods
	// ==========================================

	// getMe
	r.Register(NewSchema("getMe", KindBotAPIMethod).
		Description("A simple method for testing your bot's authentication token.").
		Method("GET", "/getMe", "User").
		DocURL("https://core.telegram.org/bots/api#getme").
		Build())

	// logOut
	r.Register(NewSchema("logOut", KindBotAPIMethod).
		Description("Log out from the cloud Bot API server before launching locally.").
		Method("POST", "/logOut", "bool").
		DocURL("https://core.telegram.org/bots/api#logout").
		Build())

	// close
	r.Register(NewSchema("close", KindBotAPIMethod).
		Description("Close the bot instance before moving it to another local server.").
		Method("POST", "/close", "bool").
		DocURL("https://core.telegram.org/bots/api#close").
		Build())

	// getUpdates
	r.Register(NewSchema("getUpdates", KindBotAPIMethod).
		Description("Receive incoming updates using long polling.").
		Method("POST", "/getUpdates", "[]Update").
		Field(FieldSpec{Name: "offset", Type: TypeInteger, Description: "Identifier of the first update to be returned"}).
		Field(FieldSpec{Name: "limit", Type: TypeInteger, Description: "Limits the number of updates to be retrieved (1-100)", Minimum: FloatPtr(1), Maximum: FloatPtr(100)}).
		Field(FieldSpec{Name: "timeout", Type: TypeInteger, Description: "Timeout in seconds for long polling"}).
		Field(FieldSpec{Name: "allowed_updates", Type: TypeArray, Description: "List of the update types you want your bot to receive"}).
		DocURL("https://core.telegram.org/bots/api#getupdates").
		Build())

	// setWebhook
	r.Register(NewSchema("setWebhook", KindBotAPIMethod).
		Description("Specify a URL and receive incoming updates via an outgoing webhook.").
		Method("POST", "/setWebhook", "bool").
		Field(FieldSpec{Name: "url", Type: TypeString, Description: "HTTPS URL to send updates to", Required: true}).
		Field(FieldSpec{Name: "ip_address", Type: TypeString, Description: "Target IP address to bypass DNS"}).
		Field(FieldSpec{Name: "max_connections", Type: TypeInteger, Description: "Maximum allowed number of simultaneous HTTPS connections"}).
		Field(FieldSpec{Name: "allowed_updates", Type: TypeArray, Description: "List of the update types you want your bot to receive"}).
		Field(FieldSpec{Name: "drop_pending_updates", Type: TypeBoolean, Description: "Pass True to drop all pending updates"}).
		Field(FieldSpec{Name: "secret_token", Type: TypeString, Description: "Secret token to be sent in header X-Telegram-Bot-Api-Secret-Token"}).
		DocURL("https://core.telegram.org/bots/api#setwebhook").
		Build())

	// deleteWebhook
	r.Register(NewSchema("deleteWebhook", KindBotAPIMethod).
		Description("Remove webhook integration.").
		Method("POST", "/deleteWebhook", "bool").
		Field(FieldSpec{Name: "drop_pending_updates", Type: TypeBoolean, Description: "Pass True to drop all pending updates"}).
		DocURL("https://core.telegram.org/bots/api#deletewebhook").
		Build())

	// getWebhookInfo
	r.Register(NewSchema("getWebhookInfo", KindBotAPIMethod).
		Description("Get current webhook status.").
		Method("GET", "/getWebhookInfo", "WebhookInfo").
		DocURL("https://core.telegram.org/bots/api#getwebhookinfo").
		Build())

	// sendMessage
	r.Register(NewSchema("sendMessage", KindBotAPIMethod).
		Description("Send text messages.").
		Method("POST", "/sendMessage", "Message").
		Field(FieldSpec{Name: "chat_id", Type: TypeAny, Description: "Unique identifier for the target chat or username", Required: true}).
		Field(FieldSpec{Name: "text", Type: TypeString, Description: "Text of the message to be sent", Required: true, MinLength: IntPtr(1), MaxLength: IntPtr(4096)}).
		Field(FieldSpec{Name: "business_connection_id", Type: TypeString, Description: "Unique identifier of the business connection"}).
		Field(FieldSpec{Name: "message_thread_id", Type: TypeInteger, Description: "Unique identifier of forum topic"}).
		Field(FieldSpec{Name: "parse_mode", Type: TypeString, Description: "Parse mode: MarkdownV2, HTML, or Markdown", AllowedValues: []string{"MarkdownV2", "HTML", "Markdown"}}).
		Field(FieldSpec{Name: "disable_notification", Type: TypeBoolean, Description: "Sends the message silently"}).
		Field(FieldSpec{Name: "protect_content", Type: TypeBoolean, Description: "Protects the contents from forwarding and saving"}).
		Field(FieldSpec{Name: "reply_markup", Type: TypeObject, Description: "InlineKeyboardMarkup, ReplyKeyboardMarkup, etc."}).
		DocURL("https://core.telegram.org/bots/api#sendmessage").
		Build())

	// sendPhoto
	r.Register(NewSchema("sendPhoto", KindBotAPIMethod).
		Description("Send photos.").
		Method("POST", "/sendPhoto", "Message").
		Field(FieldSpec{Name: "chat_id", Type: TypeAny, Description: "Target chat id", Required: true}).
		Field(FieldSpec{Name: "photo", Type: TypeString, Description: "Photo to send (file_id or HTTP URL)", Required: true}).
		Field(FieldSpec{Name: "caption", Type: TypeString, Description: "Photo caption, 0-1024 characters", MaxLength: IntPtr(1024)}).
		Field(FieldSpec{Name: "parse_mode", Type: TypeString, Description: "Parse mode", AllowedValues: []string{"MarkdownV2", "HTML", "Markdown"}}).
		Field(FieldSpec{Name: "has_spoiler", Type: TypeBoolean, Description: "Cover photo with a spoiler animation"}).
		Field(FieldSpec{Name: "reply_markup", Type: TypeObject, Description: "Reply markup"}).
		DocURL("https://core.telegram.org/bots/api#sendphoto").
		Build())

	// sendDocument
	r.Register(NewSchema("sendDocument", KindBotAPIMethod).
		Description("Send general files.").
		Method("POST", "/sendDocument", "Message").
		Field(FieldSpec{Name: "chat_id", Type: TypeAny, Description: "Target chat id", Required: true}).
		Field(FieldSpec{Name: "document", Type: TypeString, Description: "File to send", Required: true}).
		Field(FieldSpec{Name: "caption", Type: TypeString, Description: "Document caption", MaxLength: IntPtr(1024)}).
		Field(FieldSpec{Name: "reply_markup", Type: TypeObject, Description: "Reply markup"}).
		DocURL("https://core.telegram.org/bots/api#senddocument").
		Build())

	// answerCallbackQuery
	r.Register(NewSchema("answerCallbackQuery", KindBotAPIMethod).
		Description("Send answers to callback queries sent from inline keyboards.").
		Method("POST", "/answerCallbackQuery", "bool").
		Field(FieldSpec{Name: "callback_query_id", Type: TypeString, Description: "Unique identifier for query", Required: true}).
		Field(FieldSpec{Name: "text", Type: TypeString, Description: "Notification text to be shown", MaxLength: IntPtr(200)}).
		Field(FieldSpec{Name: "show_alert", Type: TypeBoolean, Description: "Show an alert instead of notification"}).
		Field(FieldSpec{Name: "url", Type: TypeString, Description: "URL to open"}).
		Field(FieldSpec{Name: "cache_time", Type: TypeInteger, Description: "Max cache duration in seconds"}).
		DocURL("https://core.telegram.org/bots/api#answercallbackquery").
		Build())

	// answerWebAppQuery (Mini App method)
	r.Register(NewSchema("answerWebAppQuery", KindBotAPIMethod).
		Description("Set the result of an interaction with a Web App and send message on behalf of user.").
		Method("POST", "/answerWebAppQuery", "SentWebAppMessage").
		Field(FieldSpec{Name: "web_app_query_id", Type: TypeString, Description: "Unique identifier for the query", Required: true}).
		Field(FieldSpec{Name: "result", Type: TypeObject, Description: "InlineQueryResult to be sent", Required: true}).
		DocURL("https://core.telegram.org/bots/api#answerwebappquery").
		Build())

	// savePreparedInlineMessage (Mini App method)
	r.Register(NewSchema("savePreparedInlineMessage", KindBotAPIMethod).
		Description("Stores a message that can be sent by a user of a Mini App.").
		Method("POST", "/savePreparedInlineMessage", "PreparedInlineMessage").
		Field(FieldSpec{Name: "user_id", Type: TypeInteger, Description: "Identifier of the target user", Required: true}).
		Field(FieldSpec{Name: "result", Type: TypeObject, Description: "InlineQueryResult to be sent", Required: true}).
		Field(FieldSpec{Name: "allow_user_chats", Type: TypeBoolean, Description: "Pass True if message can be sent to private chats"}).
		Field(FieldSpec{Name: "allow_bot_chats", Type: TypeBoolean, Description: "Pass True if message can be sent to bot chats"}).
		Field(FieldSpec{Name: "allow_group_chats", Type: TypeBoolean, Description: "Pass True if message can be sent to group chats"}).
		Field(FieldSpec{Name: "allow_channel_chats", Type: TypeBoolean, Description: "Pass True if message can be sent to channel chats"}).
		DocURL("https://core.telegram.org/bots/api#savepreparedinlinemessage").
		Build())

	// savePreparedKeyboardButton (Mini App method)
	r.Register(NewSchema("savePreparedKeyboardButton", KindBotAPIMethod).
		Description("Stores a keyboard button that can be used by a user within a Mini App.").
		Method("POST", "/savePreparedKeyboardButton", "PreparedKeyboardButton").
		Field(FieldSpec{Name: "user_id", Type: TypeInteger, Description: "Identifier of the target user", Required: true}).
		Field(FieldSpec{Name: "button", Type: TypeObject, Description: "KeyboardButton to be stored", Required: true}).
		DocURL("https://core.telegram.org/bots/api#savepreparedkeyboardbutton").
		Build())

	// editMessageText
	r.Register(NewSchema("editMessageText", KindBotAPIMethod).
		Description("Edit text and game messages.").
		Method("POST", "/editMessageText", "Message").
		Field(FieldSpec{Name: "chat_id", Type: TypeAny, Description: "Target chat id"}).
		Field(FieldSpec{Name: "message_id", Type: TypeInteger, Description: "Target message id"}).
		Field(FieldSpec{Name: "inline_message_id", Type: TypeString, Description: "Target inline message id"}).
		Field(FieldSpec{Name: "text", Type: TypeString, Description: "New text of the message", Required: true, MaxLength: IntPtr(4096)}).
		DocURL("https://core.telegram.org/bots/api#editmessagetext").
		Build())

	// deleteMessage
	r.Register(NewSchema("deleteMessage", KindBotAPIMethod).
		Description("Delete a message.").
		Method("POST", "/deleteMessage", "bool").
		Field(FieldSpec{Name: "chat_id", Type: TypeAny, Description: "Target chat id", Required: true}).
		Field(FieldSpec{Name: "message_id", Type: TypeInteger, Description: "Identifier of the message to delete", Required: true}).
		DocURL("https://core.telegram.org/bots/api#deletemessage").
		Build())

	// setMyCommands
	r.Register(NewSchema("setMyCommands", KindBotAPIMethod).
		Description("Change the list of the bot's commands.").
		Method("POST", "/setMyCommands", "bool").
		Field(FieldSpec{Name: "commands", Type: TypeArray, Description: "List of bot commands", Required: true}).
		Field(FieldSpec{Name: "scope", Type: TypeObject, Description: "Scope of users for which the commands are relevant"}).
		Field(FieldSpec{Name: "language_code", Type: TypeString, Description: "Two-letter ISO 639-1 language code"}).
		DocURL("https://core.telegram.org/bots/api#setmycommands").
		Build())

	// setChatMenuButton
	r.Register(NewSchema("setChatMenuButton", KindBotAPIMethod).
		Description("Change the bot's menu button in a private chat, or the default menu button.").
		Method("POST", "/setChatMenuButton", "bool").
		Field(FieldSpec{Name: "chat_id", Type: TypeInteger, Description: "Target chat id"}).
		Field(FieldSpec{Name: "menu_button", Type: TypeObject, Description: "New menu button"}).
		DocURL("https://core.telegram.org/bots/api#setchatmenubutton").
		Build())

	// ==========================================
	// 2. Telegram Mini App Components & Sub-objects
	// ==========================================

	// BackButton
	r.Register(NewSchema("BackButton", KindMiniAppComponent).
		Description("Controls the back button in the header of the Mini App.").
		Field(FieldSpec{Name: "isVisible", Type: TypeBoolean, Description: "Shows whether button is visible"}).
		DocURL("https://core.telegram.org/bots/webapps#backbutton").
		Build())

	// BottomButton (MainButton and SecondaryButton)
	r.Register(NewSchema("BottomButton", KindMiniAppComponent).
		Description("Controls the bottom buttons (MainButton and SecondaryButton) of the Mini App.").
		Field(FieldSpec{Name: "type", Type: TypeString, Description: "main or secondary", AllowedValues: []string{"main", "secondary"}}).
		Field(FieldSpec{Name: "text", Type: TypeString, Description: "Current button text"}).
		Field(FieldSpec{Name: "color", Type: TypeString, Description: "Button color in #RRGGBB format"}).
		Field(FieldSpec{Name: "textColor", Type: TypeString, Description: "Button text color in #RRGGBB format"}).
		Field(FieldSpec{Name: "isVisible", Type: TypeBoolean, Description: "Whether button is visible"}).
		Field(FieldSpec{Name: "isActive", Type: TypeBoolean, Description: "Whether button is active"}).
		Field(FieldSpec{Name: "hasShineEffect", Type: TypeBoolean, Description: "Bot API 7.10+ shine effect"}).
		Field(FieldSpec{Name: "position", Type: TypeString, Description: "Position: left, right, top, bottom", AllowedValues: []string{"left", "right", "top", "bottom"}}).
		Field(FieldSpec{Name: "isProgressVisible", Type: TypeBoolean, Description: "Shows loading indicator"}).
		Field(FieldSpec{Name: "iconCustomEmojiId", Type: TypeString, Description: "Custom emoji shown before text"}).
		DocURL("https://core.telegram.org/bots/webapps#bottombutton").
		Build())

	// SettingsButton
	r.Register(NewSchema("SettingsButton", KindMiniAppComponent).
		Description("Controls the Settings item in the context menu of the Mini App.").
		Field(FieldSpec{Name: "isVisible", Type: TypeBoolean, Description: "Whether context menu item is visible"}).
		DocURL("https://core.telegram.org/bots/webapps#settingsbutton").
		Build())

	// HapticFeedback
	r.Register(NewSchema("HapticFeedback", KindMiniAppComponent).
		Description("Controls haptic feedback styles and triggers.").
		Field(FieldSpec{Name: "impact_style", Type: TypeString, Description: "light, medium, heavy, rigid, soft", AllowedValues: []string{"light", "medium", "heavy", "rigid", "soft"}}).
		Field(FieldSpec{Name: "notification_type", Type: TypeString, Description: "error, success, warning", AllowedValues: []string{"error", "success", "warning"}}).
		DocURL("https://core.telegram.org/bots/webapps#hapticfeedback").
		Build())

	// CloudStorage
	r.Register(NewSchema("CloudStorage", KindMiniAppComponent).
		Description("Controls cloud storage: up to 1024 items per user (keys 1-128 chars, values 0-4096 chars).").
		Field(FieldSpec{Name: "key", Type: TypeString, Description: "Key 1-128 chars (A-Z, a-z, 0-9, _, -)", MinLength: IntPtr(1), MaxLength: IntPtr(128)}).
		Field(FieldSpec{Name: "value", Type: TypeString, Description: "Value 0-4096 characters", MaxLength: IntPtr(4096)}).
		DocURL("https://core.telegram.org/bots/webapps#cloudstorage").
		Build())

	// BiometricManager
	r.Register(NewSchema("BiometricManager", KindMiniAppComponent).
		Description("Controls biometrics on the device (finger, face, unknown).").
		Field(FieldSpec{Name: "isInited", Type: TypeBoolean, Description: "Shows whether biometrics object is initialized"}).
		Field(FieldSpec{Name: "isBiometricAvailable", Type: TypeBoolean, Description: "Shows whether biometrics is available"}).
		Field(FieldSpec{Name: "biometricType", Type: TypeString, Description: "finger, face, or unknown", AllowedValues: []string{"finger", "face", "unknown"}}).
		Field(FieldSpec{Name: "isAccessGranted", Type: TypeBoolean, Description: "Shows whether permission granted"}).
		Field(FieldSpec{Name: "deviceId", Type: TypeString, Description: "Unique device identifier"}).
		DocURL("https://core.telegram.org/bots/webapps#biometricmanager").
		Build())

	// Accelerometer
	r.Register(NewSchema("Accelerometer", KindMiniAppComponent).
		Description("Provides access to accelerometer data on device (Bot API 8.0+).").
		Field(FieldSpec{Name: "isStarted", Type: TypeBoolean, Description: "Indicates tracking status"}).
		Field(FieldSpec{Name: "x", Type: TypeFloat, Description: "Acceleration in X-axis (m/s²)"}).
		Field(FieldSpec{Name: "y", Type: TypeFloat, Description: "Acceleration in Y-axis (m/s²)"}).
		Field(FieldSpec{Name: "z", Type: TypeFloat, Description: "Acceleration in Z-axis (m/s²)"}).
		Field(FieldSpec{Name: "refresh_rate", Type: TypeInteger, Description: "Refresh rate in ms (20-1000)", Minimum: FloatPtr(20), Maximum: FloatPtr(1000)}).
		DocURL("https://core.telegram.org/bots/webapps#accelerometer").
		Build())

	// DeviceOrientation
	r.Register(NewSchema("DeviceOrientation", KindMiniAppComponent).
		Description("Provides access to orientation data on device (Bot API 8.0+).").
		Field(FieldSpec{Name: "isStarted", Type: TypeBoolean, Description: "Indicates tracking status"}).
		Field(FieldSpec{Name: "absolute", Type: TypeBoolean, Description: "True if absolute values relative to magnetic north"}).
		Field(FieldSpec{Name: "alpha", Type: TypeFloat, Description: "Rotation around Z-axis in radians"}).
		Field(FieldSpec{Name: "beta", Type: TypeFloat, Description: "Rotation around X-axis in radians"}).
		Field(FieldSpec{Name: "gamma", Type: TypeFloat, Description: "Rotation around Y-axis in radians"}).
		DocURL("https://core.telegram.org/bots/webapps#deviceorientation").
		Build())

	// Gyroscope
	r.Register(NewSchema("Gyroscope", KindMiniAppComponent).
		Description("Provides access to gyroscope rotation data on device (Bot API 8.0+).").
		Field(FieldSpec{Name: "isStarted", Type: TypeBoolean, Description: "Indicates tracking status"}).
		Field(FieldSpec{Name: "x", Type: TypeFloat, Description: "Rotation rate around X-axis (rad/s)"}).
		Field(FieldSpec{Name: "y", Type: TypeFloat, Description: "Rotation rate around Y-axis (rad/s)"}).
		Field(FieldSpec{Name: "z", Type: TypeFloat, Description: "Rotation rate around Z-axis (rad/s)"}).
		DocURL("https://core.telegram.org/bots/webapps#gyroscope").
		Build())

	// LocationManager
	r.Register(NewSchema("LocationManager", KindMiniAppComponent).
		Description("Controls device location access (Bot API 8.0+).").
		Field(FieldSpec{Name: "isInited", Type: TypeBoolean, Description: "Whether LocationManager initialized"}).
		Field(FieldSpec{Name: "isLocationAvailable", Type: TypeBoolean, Description: "Whether location services available"}).
		Field(FieldSpec{Name: "isAccessGranted", Type: TypeBoolean, Description: "Whether access permission granted"}).
		DocURL("https://core.telegram.org/bots/webapps#locationmanager").
		Build())

	// DeviceStorage
	r.Register(NewSchema("DeviceStorage", KindMiniAppComponent).
		Description("Persistent local storage on user device up to 5 MB per user (Bot API 9.0+).").
		Field(FieldSpec{Name: "key", Type: TypeString, Description: "Storage key", Required: true}).
		Field(FieldSpec{Name: "value", Type: TypeString, Description: "Storage value"}).
		DocURL("https://core.telegram.org/bots/webapps#devicestorage").
		Build())

	// SecureStorage
	r.Register(NewSchema("SecureStorage", KindMiniAppComponent).
		Description("Secure encrypted storage on user device (iOS Keychain / Android Keystore, up to 10 items) (Bot API 9.0+).").
		Field(FieldSpec{Name: "key", Type: TypeString, Description: "Secure storage key", Required: true}).
		Field(FieldSpec{Name: "value", Type: TypeString, Description: "Encrypted value"}).
		DocURL("https://core.telegram.org/bots/webapps#securestorage").
		Build())

	// Serverless
	r.Register(NewSchema("Serverless", KindMiniAppComponent).
		Description("Endpoints execution on Telegram Serverless platform with authenticated initData.").
		Field(FieldSpec{Name: "name", Type: TypeString, Description: "Endpoint name", Required: true}).
		Field(FieldSpec{Name: "input", Type: TypeObject, Description: "Input JSON-serializable parameters"}).
		DocURL("https://core.telegram.org/bots/webapps#serverless").
		Build())

	// ==========================================
	// 3. Telegram Mini App Types
	// ==========================================

	// WebAppInitData
	r.Register(NewSchema("WebAppInitData", KindMiniAppType).
		Description("Data transferred to Mini App when opened.").
		Field(FieldSpec{Name: "query_id", Type: TypeString, Description: "Unique identifier for Mini App session"}).
		Field(FieldSpec{Name: "chat_join_request_query_id", Type: TypeString, Description: "Unique identifier for chat join request query (Bot API 10.1+)"}).
		Field(FieldSpec{Name: "user", Type: TypeObject, Description: "Current user data"}).
		Field(FieldSpec{Name: "receiver", Type: TypeObject, Description: "Chat partner in private attachment menu launch"}).
		Field(FieldSpec{Name: "chat", Type: TypeObject, Description: "Chat data where bot was opened"}).
		Field(FieldSpec{Name: "chat_type", Type: TypeString, Description: "sender, private, group, supergroup, or channel", AllowedValues: []string{"sender", "private", "group", "supergroup", "channel"}}).
		Field(FieldSpec{Name: "chat_instance", Type: TypeString, Description: "Global identifier corresponding to chat"}).
		Field(FieldSpec{Name: "start_param", Type: TypeString, Description: "startattach parameter value"}).
		Field(FieldSpec{Name: "can_send_after", Type: TypeInteger, Description: "Time in seconds after which message can be sent"}).
		Field(FieldSpec{Name: "auth_date", Type: TypeInteger, Description: "Unix time when opened", Required: true}).
		Field(FieldSpec{Name: "hash", Type: TypeString, Description: "HMAC-SHA256 signature hash", Required: true}).
		Field(FieldSpec{Name: "signature", Type: TypeString, Description: "Ed25519 signature for third-party validation"}).
		DocURL("https://core.telegram.org/bots/webapps#webappinitdata").
		Build())

	// ThemeParams
	r.Register(NewSchema("ThemeParams", KindMiniAppType).
		Description("Theme settings in Telegram app matching user color schemes.").
		Field(FieldSpec{Name: "bg_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "text_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "hint_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "link_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "button_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "button_text_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "secondary_bg_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "header_bg_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "bottom_bar_bg_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "accent_text_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "section_bg_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "section_header_text_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "section_separator_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "subtitle_text_color", Type: TypeString, Description: "#RRGGBB format"}).
		Field(FieldSpec{Name: "destructive_text_color", Type: TypeString, Description: "#RRGGBB format"}).
		DocURL("https://core.telegram.org/bots/webapps#themeparams").
		Build())

	// PopupParams
	r.Register(NewSchema("PopupParams", KindMiniAppType).
		Description("Native dialog popup configuration.").
		Field(FieldSpec{Name: "title", Type: TypeString, Description: "Title, 0-64 chars", MaxLength: IntPtr(64)}).
		Field(FieldSpec{Name: "message", Type: TypeString, Description: "Body message, 1-256 chars", Required: true, MinLength: IntPtr(1), MaxLength: IntPtr(256)}).
		Field(FieldSpec{Name: "buttons", Type: TypeArray, Description: "List of 1-3 buttons"}).
		DocURL("https://core.telegram.org/bots/webapps#popupparams").
		Build())

	// SafeAreaInset
	r.Register(NewSchema("SafeAreaInset", KindMiniAppType).
		Description("System safe area padding in pixels.").
		Field(FieldSpec{Name: "top", Type: TypeInteger, Description: "Top inset pixels", Required: true}).
		Field(FieldSpec{Name: "bottom", Type: TypeInteger, Description: "Bottom inset pixels", Required: true}).
		Field(FieldSpec{Name: "left", Type: TypeInteger, Description: "Left inset pixels", Required: true}).
		Field(FieldSpec{Name: "right", Type: TypeInteger, Description: "Right inset pixels", Required: true}).
		DocURL("https://core.telegram.org/bots/webapps#safeareainset").
		Build())

	// ==========================================
	// 4. Telegram Mini App Events
	// ==========================================

	events := []string{
		"themeChanged", "viewportChanged", "mainButtonClicked", "secondaryButtonClicked",
		"backButtonClicked", "settingsButtonClicked", "invoiceClosed", "popupClosed",
		"qrTextReceived", "scanQrPopupClosed", "clipboardTextReceived", "writeAccessRequested",
		"contactRequested", "biometricManagerUpdated", "biometricAuthRequested",
		"biometricTokenUpdated", "fullscreenChanged", "fullscreenFailed", "homeScreenAdded",
		"homeScreenChecked", "accelerometerStarted", "accelerometerStopped",
		"accelerometerChanged", "accelerometerFailed", "deviceOrientationStarted",
		"deviceOrientationStopped", "deviceOrientationChanged", "deviceOrientationFailed",
		"gyroscopeStarted", "gyroscopeStopped", "gyroscopeChanged", "gyroscopeFailed",
		"locationManagerUpdated", "locationRequested", "shareMessageSent", "shareMessageFailed",
		"emojiStatusSet", "emojiStatusFailed", "emojiStatusAccessRequested",
		"fileDownloadRequested", "safeAreaChanged", "contentSafeAreaChanged",
		"activated", "deactivated",
	}

	for _, ev := range events {
		r.Register(NewSchema(ev, KindMiniAppEvent).
			Description("Mini App client event: " + ev).
			DocURL("https://core.telegram.org/bots/webapps#events-available-for-mini-apps").
			Build())
	}
}
