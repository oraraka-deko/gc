package miniapp

// Telegram Mini App Event Names
const (
	EventThemeChanged               = "themeChanged"
	EventViewportChanged            = "viewportChanged"
	EventMainButtonClicked          = "mainButtonClicked"
	EventSecondaryButtonClicked      = "secondaryButtonClicked"
	EventBackButtonClicked          = "backButtonClicked"
	EventSettingsButtonClicked      = "settingsButtonClicked"
	EventInvoiceClosed              = "invoiceClosed"
	EventPopupClosed                = "popupClosed"
	EventQrTextReceived             = "qrTextReceived"
	EventScanQrPopupClosed          = "scanQrPopupClosed"
	EventClipboardTextReceived      = "clipboardTextReceived"
	EventWriteAccessRequested       = "writeAccessRequested"
	EventContactRequested           = "contactRequested"
	EventBiometricManagerUpdated    = "biometricManagerUpdated"
	EventBiometricAuthRequested     = "biometricAuthRequested"
	EventBiometricTokenUpdated      = "biometricTokenUpdated"
	EventFullscreenChanged          = "fullscreenChanged"
	EventFullscreenFailed           = "fullscreenFailed"
	EventHomeScreenAdded            = "homeScreenAdded"
	EventHomeScreenChecked          = "homeScreenChecked"
	EventAccelerometerStarted       = "accelerometerStarted"
	EventAccelerometerStopped       = "accelerometerStopped"
	EventAccelerometerChanged       = "accelerometerChanged"
	EventAccelerometerFailed        = "accelerometerFailed"
	EventDeviceOrientationStarted   = "deviceOrientationStarted"
	EventDeviceOrientationStopped   = "deviceOrientationStopped"
	EventDeviceOrientationChanged   = "deviceOrientationChanged"
	EventDeviceOrientationFailed    = "deviceOrientationFailed"
	EventGyroscopeStarted           = "gyroscopeStarted"
	EventGyroscopeStopped           = "gyroscopeStopped"
	EventGyroscopeChanged           = "gyroscopeChanged"
	EventGyroscopeFailed            = "gyroscopeFailed"
	EventLocationManagerUpdated     = "locationManagerUpdated"
	EventLocationRequested          = "locationRequested"
	EventShareMessageSent           = "shareMessageSent"
	EventShareMessageFailed         = "shareMessageFailed"
	EventEmojiStatusSet             = "emojiStatusSet"
	EventEmojiStatusFailed          = "emojiStatusFailed"
	EventEmojiStatusAccessRequested = "emojiStatusAccessRequested"
	EventFileDownloadRequested      = "fileDownloadRequested"
	EventSafeAreaChanged            = "safeAreaChanged"
	EventContentSafeAreaChanged     = "contentSafeAreaChanged"
	EventActivated                  = "activated"
	EventDeactivated                = "deactivated"
)

// ViewportChangedEventPayload holds data from viewportChanged event.
type ViewportChangedEventPayload struct {
	IsStateStable bool `json:"isStateStable"`
}

// InvoiceClosedEventPayload holds data from invoiceClosed event.
type InvoiceClosedEventPayload struct {
	URL    string `json:"url"`
	Status string `json:"status"` // "paid", "cancelled", "failed", "pending"
}

// PopupClosedEventPayload holds data from popupClosed event.
type PopupClosedEventPayload struct {
	ButtonID *string `json:"button_id,omitempty"`
}

// HomeScreenCheckedEventPayload holds status from homeScreenChecked event.
type HomeScreenCheckedEventPayload struct {
	Status string `json:"status"` // "unsupported", "unknown", "added", "missed"
}
