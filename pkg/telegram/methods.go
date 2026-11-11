package telegram

import (
	"context"
)

// Parameter structs for methods

type GetUpdatesParams struct {
	Offset         *int     `json:"offset,omitempty"`
	Limit          *int     `json:"limit,omitempty"`
	Timeout        *int     `json:"timeout,omitempty"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

type SetWebhookParams struct {
	URL                string   `json:"url"`
	IPAddress          *string  `json:"ip_address,omitempty"`
	MaxConnections     *int     `json:"max_connections,omitempty"`
	AllowedUpdates     []string `json:"allowed_updates,omitempty"`
	DropPendingUpdates *bool    `json:"drop_pending_updates,omitempty"`
	SecretToken        *string  `json:"secret_token,omitempty"`
}

type SendMessageParams struct {
	ChatID                   any                   `json:"chat_id"` // int64 or string
	Text                     string                `json:"text"`
	BusinessConnectionID     *string               `json:"business_connection_id,omitempty"`
	MessageThreadID          *int                  `json:"message_thread_id,omitempty"`
	ParseMode                *string               `json:"parse_mode,omitempty"`
	Entities                 []MessageEntity       `json:"entities,omitempty"`
	LinkPreviewOptions       *LinkPreviewOptions   `json:"link_preview_options,omitempty"`
	DisableNotification      *bool                 `json:"disable_notification,omitempty"`
	ProtectContent           *bool                 `json:"protect_content,omitempty"`
	MessageEffectID          *string               `json:"message_effect_id,omitempty"`
	ReplyMarkup              any                   `json:"reply_markup,omitempty"` // InlineKeyboardMarkup, ReplyKeyboardMarkup, etc.
}

type ForwardMessageParams struct {
	ChatID              any     `json:"chat_id"`
	FromChatID          any     `json:"from_chat_id"`
	MessageID           int     `json:"message_id"`
	MessageThreadID     *int    `json:"message_thread_id,omitempty"`
	DisableNotification *bool   `json:"disable_notification,omitempty"`
	ProtectContent      *bool   `json:"protect_content,omitempty"`
}

type ForwardMessagesParams struct {
	ChatID              any    `json:"chat_id"`
	FromChatID          any    `json:"from_chat_id"`
	MessageIDs          []int  `json:"message_ids"`
	MessageThreadID     *int   `json:"message_thread_id,omitempty"`
	DisableNotification *bool  `json:"disable_notification,omitempty"`
	ProtectContent      *bool  `json:"protect_content,omitempty"`
}

type CopyMessageParams struct {
	ChatID              any             `json:"chat_id"`
	FromChatID          any             `json:"from_chat_id"`
	MessageID           int             `json:"message_id"`
	MessageThreadID     *int            `json:"message_thread_id,omitempty"`
	Caption             *string         `json:"caption,omitempty"`
	ParseMode           *string         `json:"parse_mode,omitempty"`
	CaptionEntities     []MessageEntity `json:"caption_entities,omitempty"`
	ShowCaptionAboveMedia *bool         `json:"show_caption_above_media,omitempty"`
	DisableNotification *bool           `json:"disable_notification,omitempty"`
	ProtectContent      *bool           `json:"protect_content,omitempty"`
	ReplyMarkup         any             `json:"reply_markup,omitempty"`
}

type CopyMessagesParams struct {
	ChatID              any   `json:"chat_id"`
	FromChatID          any   `json:"from_chat_id"`
	MessageIDs          []int `json:"message_ids"`
	MessageThreadID     *int  `json:"message_thread_id,omitempty"`
	DisableNotification *bool `json:"disable_notification,omitempty"`
	ProtectContent      *bool `json:"protect_content,omitempty"`
	RemoveCaption       *bool `json:"remove_caption,omitempty"`
}

type SendPhotoParams struct {
	ChatID              any             `json:"chat_id"`
	Photo               string          `json:"photo"` // file_id or url
	Caption             *string         `json:"caption,omitempty"`
	ParseMode           *string         `json:"parse_mode,omitempty"`
	CaptionEntities     []MessageEntity `json:"caption_entities,omitempty"`
	HasSpoiler          *bool           `json:"has_spoiler,omitempty"`
	DisableNotification *bool           `json:"disable_notification,omitempty"`
	ProtectContent      *bool           `json:"protect_content,omitempty"`
	ReplyMarkup         any             `json:"reply_markup,omitempty"`
}

type SendAudioParams struct {
	ChatID              any             `json:"chat_id"`
	Audio               string          `json:"audio"`
	Caption             *string         `json:"caption,omitempty"`
	ParseMode           *string         `json:"parse_mode,omitempty"`
	CaptionEntities     []MessageEntity `json:"caption_entities,omitempty"`
	Duration            *int            `json:"duration,omitempty"`
	Performer           *string         `json:"performer,omitempty"`
	Title               *string         `json:"title,omitempty"`
	DisableNotification *bool           `json:"disable_notification,omitempty"`
	ProtectContent      *bool           `json:"protect_content,omitempty"`
	ReplyMarkup         any             `json:"reply_markup,omitempty"`
}

type SendDocumentParams struct {
	ChatID                      any             `json:"chat_id"`
	Document                    string          `json:"document"`
	Caption                     *string         `json:"caption,omitempty"`
	ParseMode                   *string         `json:"parse_mode,omitempty"`
	CaptionEntities             []MessageEntity `json:"caption_entities,omitempty"`
	DisableContentTypeDetection *bool           `json:"disable_content_type_detection,omitempty"`
	DisableNotification         *bool           `json:"disable_notification,omitempty"`
	ProtectContent              *bool           `json:"protect_content,omitempty"`
	ReplyMarkup                 any             `json:"reply_markup,omitempty"`
}

type SendVideoParams struct {
	ChatID              any             `json:"chat_id"`
	Video               string          `json:"video"`
	Duration            *int            `json:"duration,omitempty"`
	Width               *int            `json:"width,omitempty"`
	Height              *int            `json:"height,omitempty"`
	Caption             *string         `json:"caption,omitempty"`
	ParseMode           *string         `json:"parse_mode,omitempty"`
	CaptionEntities     []MessageEntity `json:"caption_entities,omitempty"`
	HasSpoiler          *bool           `json:"has_spoiler,omitempty"`
	SupportsStreaming   *bool           `json:"supports_streaming,omitempty"`
	DisableNotification *bool           `json:"disable_notification,omitempty"`
	ProtectContent      *bool           `json:"protect_content,omitempty"`
	ReplyMarkup         any             `json:"reply_markup,omitempty"`
}

type SendAnimationParams struct {
	ChatID              any             `json:"chat_id"`
	Animation           string          `json:"animation"`
	Duration            *int            `json:"duration,omitempty"`
	Width               *int            `json:"width,omitempty"`
	Height              *int            `json:"height,omitempty"`
	Caption             *string         `json:"caption,omitempty"`
	ParseMode           *string         `json:"parse_mode,omitempty"`
	CaptionEntities     []MessageEntity `json:"caption_entities,omitempty"`
	HasSpoiler          *bool           `json:"has_spoiler,omitempty"`
	DisableNotification *bool           `json:"disable_notification,omitempty"`
	ProtectContent      *bool           `json:"protect_content,omitempty"`
	ReplyMarkup         any             `json:"reply_markup,omitempty"`
}

type SendVoiceParams struct {
	ChatID              any             `json:"chat_id"`
	Voice               string          `json:"voice"`
	Caption             *string         `json:"caption,omitempty"`
	ParseMode           *string         `json:"parse_mode,omitempty"`
	CaptionEntities     []MessageEntity `json:"caption_entities,omitempty"`
	Duration            *int            `json:"duration,omitempty"`
	DisableNotification *bool           `json:"disable_notification,omitempty"`
	ProtectContent      *bool           `json:"protect_content,omitempty"`
	ReplyMarkup         any             `json:"reply_markup,omitempty"`
}

type SendVideoNoteParams struct {
	ChatID              any     `json:"chat_id"`
	VideoNote           string  `json:"video_note"`
	Duration            *int    `json:"duration,omitempty"`
	Length              *int    `json:"length,omitempty"`
	DisableNotification *bool   `json:"disable_notification,omitempty"`
	ProtectContent      *bool   `json:"protect_content,omitempty"`
	ReplyMarkup         any     `json:"reply_markup,omitempty"`
}

type SendPaidMediaParams struct {
	ChatID              any      `json:"chat_id"`
	StarCount           int      `json:"star_count"`
	Media               []any    `json:"media"`
	Caption             *string  `json:"caption,omitempty"`
	ParseMode           *string  `json:"parse_mode,omitempty"`
	DisableNotification *bool    `json:"disable_notification,omitempty"`
	ProtectContent      *bool    `json:"protect_content,omitempty"`
	ReplyMarkup         any      `json:"reply_markup,omitempty"`
}

type SendLocationParams struct {
	ChatID               any      `json:"chat_id"`
	Latitude             float64  `json:"latitude"`
	Longitude            float64  `json:"longitude"`
	HorizontalAccuracy   *float64 `json:"horizontal_accuracy,omitempty"`
	LivePeriod           *int     `json:"live_period,omitempty"`
	Heading              *int     `json:"heading,omitempty"`
	ProximityAlertRadius *int     `json:"proximity_alert_radius,omitempty"`
	DisableNotification  *bool    `json:"disable_notification,omitempty"`
	ProtectContent       *bool    `json:"protect_content,omitempty"`
	ReplyMarkup          any      `json:"reply_markup,omitempty"`
}

type SendVenueParams struct {
	ChatID              any     `json:"chat_id"`
	Latitude            float64 `json:"latitude"`
	Longitude           float64 `json:"longitude"`
	Title               string  `json:"title"`
	Address             string  `json:"address"`
	FoursquareID        *string `json:"foursquare_id,omitempty"`
	FoursquareType      *string `json:"foursquare_type,omitempty"`
	GooglePlaceID       *string `json:"google_place_id,omitempty"`
	GooglePlaceType     *string `json:"google_place_type,omitempty"`
	DisableNotification *bool   `json:"disable_notification,omitempty"`
	ProtectContent      *bool   `json:"protect_content,omitempty"`
	ReplyMarkup         any     `json:"reply_markup,omitempty"`
}

type SendContactParams struct {
	ChatID              any     `json:"chat_id"`
	PhoneNumber         string  `json:"phone_number"`
	FirstName           string  `json:"first_name"`
	LastName            *string `json:"last_name,omitempty"`
	Vcard               *string `json:"vcard,omitempty"`
	DisableNotification *bool   `json:"disable_notification,omitempty"`
	ProtectContent      *bool   `json:"protect_content,omitempty"`
	ReplyMarkup         any     `json:"reply_markup,omitempty"`
}

type SendPollParams struct {
	ChatID                any             `json:"chat_id"`
	Question              string          `json:"question"`
	Options               []string        `json:"options"`
	IsAnonymous           *bool           `json:"is_anonymous,omitempty"`
	Type                  *string         `json:"type,omitempty"` // "quiz" or "regular"
	AllowsMultipleAnswers *bool           `json:"allows_multiple_answers,omitempty"`
	CorrectOptionID       *int            `json:"correct_option_id,omitempty"`
	Explanation           *string         `json:"explanation,omitempty"`
	OpenPeriod            *int            `json:"open_period,omitempty"`
	CloseDate             *int64          `json:"close_date,omitempty"`
	IsClosed              *bool           `json:"is_closed,omitempty"`
	DisableNotification   *bool           `json:"disable_notification,omitempty"`
	ProtectContent        *bool           `json:"protect_content,omitempty"`
	ReplyMarkup           any             `json:"reply_markup,omitempty"`
}

type SendDiceParams struct {
	ChatID              any     `json:"chat_id"`
	Emoji               *string `json:"emoji,omitempty"` // 🎲, 🎯, 🏀, ⚽, 🎳, 🎰
	DisableNotification *bool   `json:"disable_notification,omitempty"`
	ProtectContent      *bool   `json:"protect_content,omitempty"`
	ReplyMarkup         any     `json:"reply_markup,omitempty"`
}

type SendChatActionParams struct {
	ChatID               any     `json:"chat_id"`
	Action               string  `json:"action"` // "typing", "upload_photo", "record_video", "upload_video", etc.
	BusinessConnectionID *string `json:"business_connection_id,omitempty"`
	MessageThreadID      *int    `json:"message_thread_id,omitempty"`
}

type SetMessageReactionParams struct {
	ChatID    any            `json:"chat_id"`
	MessageID int            `json:"message_id"`
	Reaction  []ReactionType `json:"reaction,omitempty"`
	IsBig     *bool          `json:"is_big,omitempty"`
}

type GetUserProfilePhotosParams struct {
	UserID int64 `json:"user_id"`
	Offset *int  `json:"offset,omitempty"`
	Limit  *int  `json:"limit,omitempty"`
}

type AnswerCallbackQueryParams struct {
	CallbackQueryID string  `json:"callback_query_id"`
	Text            *string `json:"text,omitempty"`
	ShowAlert       *bool   `json:"show_alert,omitempty"`
	URL             *string `json:"url,omitempty"`
	CacheTime       *int    `json:"cache_time,omitempty"`
}

// Mini App specific method params

type AnswerWebAppQueryParams struct {
	WebAppQueryID string `json:"web_app_query_id"`
	Result        any    `json:"result"` // InlineQueryResult
}

type SavePreparedInlineMessageParams struct {
	UserID            int64 `json:"user_id"`
	Result            any   `json:"result"` // InlineQueryResult
	AllowUserChats    *bool `json:"allow_user_chats,omitempty"`
	AllowBotChats     *bool `json:"allow_bot_chats,omitempty"`
	AllowGroupChats   *bool `json:"allow_group_chats,omitempty"`
	AllowChannelChats *bool `json:"allow_channel_chats,omitempty"`
}

type SavePreparedKeyboardButtonParams struct {
	UserID int64          `json:"user_id"`
	Button KeyboardButton `json:"button"`
}

type EditMessageTextParams struct {
	ChatID                any                 `json:"chat_id,omitempty"`
	MessageID             *int                `json:"message_id,omitempty"`
	InlineMessageID       *string             `json:"inline_message_id,omitempty"`
	Text                  string              `json:"text"`
	ParseMode             *string             `json:"parse_mode,omitempty"`
	Entities              []MessageEntity     `json:"entities,omitempty"`
	LinkPreviewOptions    *LinkPreviewOptions `json:"link_preview_options,omitempty"`
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type EditMessageCaptionParams struct {
	ChatID          any                   `json:"chat_id,omitempty"`
	MessageID       *int                  `json:"message_id,omitempty"`
	InlineMessageID *string               `json:"inline_message_id,omitempty"`
	Caption         *string               `json:"caption,omitempty"`
	ParseMode       *string               `json:"parse_mode,omitempty"`
	CaptionEntities []MessageEntity       `json:"caption_entities,omitempty"`
	ReplyMarkup     *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type EditMessageReplyMarkupParams struct {
	ChatID          any                   `json:"chat_id,omitempty"`
	MessageID       *int                  `json:"message_id,omitempty"`
	InlineMessageID *string               `json:"inline_message_id,omitempty"`
	ReplyMarkup     *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type DeleteMessageParams struct {
	ChatID    any `json:"chat_id"`
	MessageID int `json:"message_id"`
}

type DeleteMessagesParams struct {
	ChatID     any   `json:"chat_id"`
	MessageIDs []int `json:"message_ids"`
}

type SetMyCommandsParams struct {
	Commands     []BotCommand     `json:"commands"`
	Scope        *BotCommandScope `json:"scope,omitempty"`
	LanguageCode *string          `json:"language_code,omitempty"`
}

type GetMyCommandsParams struct {
	Scope        *BotCommandScope `json:"scope,omitempty"`
	LanguageCode *string          `json:"language_code,omitempty"`
}

type DeleteMyCommandsParams struct {
	Scope        *BotCommandScope `json:"scope,omitempty"`
	LanguageCode *string          `json:"language_code,omitempty"`
}

type SetMyNameParams struct {
	Name         *string `json:"name,omitempty"`
	LanguageCode *string `json:"language_code,omitempty"`
}

type SetMyDescriptionParams struct {
	Description  *string `json:"description,omitempty"`
	LanguageCode *string `json:"language_code,omitempty"`
}

type SetMyShortDescriptionParams struct {
	ShortDescription *string `json:"short_description,omitempty"`
	LanguageCode     *string `json:"language_code,omitempty"`
}

type SetChatMenuButtonParams struct {
	ChatID     *int64      `json:"chat_id,omitempty"`
	MenuButton *MenuButton `json:"menu_button,omitempty"`
}

type SendInvoiceParams struct {
	ChatID                    any            `json:"chat_id"`
	Title                     string         `json:"title"`
	Description               string         `json:"description"`
	Payload                   string         `json:"payload"`
	ProviderToken             *string        `json:"provider_token,omitempty"`
	Currency                  string         `json:"currency"`
	Prices                    []LabeledPrice `json:"prices"`
	MaxTipAmount              *int           `json:"max_tip_amount,omitempty"`
	SuggestedTipAmounts       []int          `json:"suggested_tip_amounts,omitempty"`
	StartParameter            *string        `json:"start_parameter,omitempty"`
	ProviderData              *string        `json:"provider_data,omitempty"`
	PhotoURL                  *string        `json:"photo_url,omitempty"`
	PhotoSize                 *int           `json:"photo_size,omitempty"`
	PhotoWidth                *int           `json:"photo_width,omitempty"`
	PhotoHeight               *int           `json:"photo_height,omitempty"`
	NeedName                  *bool          `json:"need_name,omitempty"`
	NeedPhoneNumber           *bool          `json:"need_phone_number,omitempty"`
	NeedEmail                 *bool          `json:"need_email,omitempty"`
	NeedShippingAddress       *bool          `json:"need_shipping_address,omitempty"`
	SendPhoneNumberToProvider *bool          `json:"send_phone_number_to_provider,omitempty"`
	SendEmailToProvider       *bool          `json:"send_email_to_provider,omitempty"`
	IsFlexible                *bool          `json:"is_flexible,omitempty"`
	DisableNotification       *bool          `json:"disable_notification,omitempty"`
	ProtectContent            *bool          `json:"protect_content,omitempty"`
	ReplyMarkup               *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type CreateInvoiceLinkParams struct {
	Title               string         `json:"title"`
	Description         string         `json:"description"`
	Payload             string         `json:"payload"`
	ProviderToken       *string        `json:"provider_token,omitempty"`
	Currency            string         `json:"currency"`
	Prices              []LabeledPrice `json:"prices"`
	SubscriptionPeriod  *int           `json:"subscription_period,omitempty"`
	BusinessConnectionID *string       `json:"business_connection_id,omitempty"`
}

type AnswerShippingQueryParams struct {
	ShippingQueryID string           `json:"shipping_query_id"`
	OK              bool             `json:"ok"`
	ShippingOptions []ShippingOption `json:"shipping_options,omitempty"`
	ErrorMessage    *string          `json:"error_message,omitempty"`
}

type AnswerPreCheckoutQueryParams struct {
	PreCheckoutQueryID string  `json:"pre_checkout_query_id"`
	OK                 bool    `json:"ok"`
	ErrorMessage       *string `json:"error_message,omitempty"`
}

type AnswerInlineQueryParams struct {
	InlineQueryID string `json:"inline_query_id"`
	Results       []any  `json:"results"`
	CacheTime     *int   `json:"cache_time,omitempty"`
	IsPersonal    *bool  `json:"is_personal,omitempty"`
	NextOffset    *string `json:"next_offset,omitempty"`
}

// Client Methods

// GetMe tests your bot's authentication token and returns basic information about the bot.
func (c *Client) GetMe(ctx context.Context) (*User, error) {
	resp, err := Do[User](ctx, c, "getMe", nil)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// LogOut logs out from the cloud Bot API server.
func (c *Client) LogOut(ctx context.Context) (bool, error) {
	resp, err := Do[bool](ctx, c, "logOut", nil)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// Close closes the bot instance.
func (c *Client) Close(ctx context.Context) (bool, error) {
	resp, err := Do[bool](ctx, c, "close", nil)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// GetUpdates retrieves incoming updates using long polling.
func (c *Client) GetUpdates(ctx context.Context, params *GetUpdatesParams) ([]Update, error) {
	resp, err := Do[[]Update](ctx, c, "getUpdates", params)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// SetWebhook specifies a URL and receives incoming updates via outgoing webhook.
func (c *Client) SetWebhook(ctx context.Context, params *SetWebhookParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "setWebhook", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// DeleteWebhook removes webhook integration.
func (c *Client) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) (bool, error) {
	payload := map[string]any{"drop_pending_updates": dropPendingUpdates}
	resp, err := Do[bool](ctx, c, "deleteWebhook", payload)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// GetWebhookInfo gets current webhook status.
func (c *Client) GetWebhookInfo(ctx context.Context) (*WebhookInfo, error) {
	resp, err := Do[WebhookInfo](ctx, c, "getWebhookInfo", nil)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendMessage sends text messages.
func (c *Client) SendMessage(ctx context.Context, params *SendMessageParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendMessage", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// ForwardMessage forwards messages of any kind.
func (c *Client) ForwardMessage(ctx context.Context, params *ForwardMessageParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "forwardMessage", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// ForwardMessages forwards multiple messages.
func (c *Client) ForwardMessages(ctx context.Context, params *ForwardMessagesParams) ([]MessageId, error) {
	resp, err := Do[[]MessageId](ctx, c, "forwardMessages", params)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// CopyMessage copies messages of any kind.
func (c *Client) CopyMessage(ctx context.Context, params *CopyMessageParams) (*MessageId, error) {
	resp, err := Do[MessageId](ctx, c, "copyMessage", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// CopyMessages copies multiple messages.
func (c *Client) CopyMessages(ctx context.Context, params *CopyMessagesParams) ([]MessageId, error) {
	resp, err := Do[[]MessageId](ctx, c, "copyMessages", params)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// SendPhoto sends photos.
func (c *Client) SendPhoto(ctx context.Context, params *SendPhotoParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendPhoto", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendAudio sends audio files.
func (c *Client) SendAudio(ctx context.Context, params *SendAudioParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendAudio", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendDocument sends general files.
func (c *Client) SendDocument(ctx context.Context, params *SendDocumentParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendDocument", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendVideo sends video files.
func (c *Client) SendVideo(ctx context.Context, params *SendVideoParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendVideo", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendAnimation sends animation files (GIF or H.264/MPEG-4 AVC video).
func (c *Client) SendAnimation(ctx context.Context, params *SendAnimationParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendAnimation", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendVoice sends voice notes.
func (c *Client) SendVoice(ctx context.Context, params *SendVoiceParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendVoice", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendVideoNote sends video note files.
func (c *Client) SendVideoNote(ctx context.Context, params *SendVideoNoteParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendVideoNote", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendPaidMedia sends paid media.
func (c *Client) SendPaidMedia(ctx context.Context, params *SendPaidMediaParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendPaidMedia", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendLocation sends points on the map.
func (c *Client) SendLocation(ctx context.Context, params *SendLocationParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendLocation", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendVenue sends information about a venue.
func (c *Client) SendVenue(ctx context.Context, params *SendVenueParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendVenue", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendContact sends phone contacts.
func (c *Client) SendContact(ctx context.Context, params *SendContactParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendContact", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendPoll sends native polls.
func (c *Client) SendPoll(ctx context.Context, params *SendPollParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendPoll", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendDice sends animated dice / emoji.
func (c *Client) SendDice(ctx context.Context, params *SendDiceParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendDice", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendChatAction sends chat actions (typing, upload_photo, etc.).
func (c *Client) SendChatAction(ctx context.Context, params *SendChatActionParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "sendChatAction", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// SetMessageReaction changes reactions on a message.
func (c *Client) SetMessageReaction(ctx context.Context, params *SetMessageReactionParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "setMessageReaction", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// GetUserProfilePhotos gets user profile photos.
func (c *Client) GetUserProfilePhotos(ctx context.Context, params *GetUserProfilePhotosParams) (*UserProfilePhotos, error) {
	resp, err := Do[UserProfilePhotos](ctx, c, "getUserProfilePhotos", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// GetFile gets file information for downloading.
func (c *Client) GetFile(ctx context.Context, fileID string) (*File, error) {
	payload := map[string]string{"file_id": fileID}
	resp, err := Do[File](ctx, c, "getFile", payload)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// AnswerCallbackQuery answers callback queries sent from inline keyboards.
func (c *Client) AnswerCallbackQuery(ctx context.Context, params *AnswerCallbackQueryParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "answerCallbackQuery", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// AnswerWebAppQuery sets the result of an interaction with a Web App.
func (c *Client) AnswerWebAppQuery(ctx context.Context, params *AnswerWebAppQueryParams) (*SentWebAppMessage, error) {
	resp, err := Do[SentWebAppMessage](ctx, c, "answerWebAppQuery", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SavePreparedInlineMessage stores a message that can be sent by a user of a Mini App.
func (c *Client) SavePreparedInlineMessage(ctx context.Context, params *SavePreparedInlineMessageParams) (*PreparedInlineMessage, error) {
	resp, err := Do[PreparedInlineMessage](ctx, c, "savePreparedInlineMessage", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SavePreparedKeyboardButton stores a keyboard button that can be used by a user within a Mini App.
func (c *Client) SavePreparedKeyboardButton(ctx context.Context, params *SavePreparedKeyboardButtonParams) (*PreparedKeyboardButton, error) {
	resp, err := Do[PreparedKeyboardButton](ctx, c, "savePreparedKeyboardButton", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// EditMessageText edits text of a message.
func (c *Client) EditMessageText(ctx context.Context, params *EditMessageTextParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "editMessageText", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// EditMessageCaption edits caption of a message.
func (c *Client) EditMessageCaption(ctx context.Context, params *EditMessageCaptionParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "editMessageCaption", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// EditMessageReplyMarkup edits only reply markup of a message.
func (c *Client) EditMessageReplyMarkup(ctx context.Context, params *EditMessageReplyMarkupParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "editMessageReplyMarkup", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// DeleteMessage deletes a message.
func (c *Client) DeleteMessage(ctx context.Context, params *DeleteMessageParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "deleteMessage", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// DeleteMessages deletes multiple messages.
func (c *Client) DeleteMessages(ctx context.Context, params *DeleteMessagesParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "deleteMessages", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// SetMyCommands changes the list of bot commands.
func (c *Client) SetMyCommands(ctx context.Context, params *SetMyCommandsParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "setMyCommands", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// GetMyCommands gets the current list of bot commands.
func (c *Client) GetMyCommands(ctx context.Context, params *GetMyCommandsParams) ([]BotCommand, error) {
	resp, err := Do[[]BotCommand](ctx, c, "getMyCommands", params)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// DeleteMyCommands deletes the bot commands.
func (c *Client) DeleteMyCommands(ctx context.Context, params *DeleteMyCommandsParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "deleteMyCommands", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// SetMyName changes the bot's name.
func (c *Client) SetMyName(ctx context.Context, params *SetMyNameParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "setMyName", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// SetMyDescription changes the bot's description.
func (c *Client) SetMyDescription(ctx context.Context, params *SetMyDescriptionParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "setMyDescription", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// SetMyShortDescription changes the bot's short description.
func (c *Client) SetMyShortDescription(ctx context.Context, params *SetMyShortDescriptionParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "setMyShortDescription", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// SetChatMenuButton changes the bot's menu button.
func (c *Client) SetChatMenuButton(ctx context.Context, params *SetChatMenuButtonParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "setChatMenuButton", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// GetChatMenuButton gets current value of bot's menu button.
func (c *Client) GetChatMenuButton(ctx context.Context, chatID *int64) (*MenuButton, error) {
	payload := map[string]any{}
	if chatID != nil {
		payload["chat_id"] = *chatID
	}
	resp, err := Do[MenuButton](ctx, c, "getChatMenuButton", payload)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// SendInvoice sends an invoice.
func (c *Client) SendInvoice(ctx context.Context, params *SendInvoiceParams) (*Message, error) {
	resp, err := Do[Message](ctx, c, "sendInvoice", params)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// CreateInvoiceLink creates a link for an invoice.
func (c *Client) CreateInvoiceLink(ctx context.Context, params *CreateInvoiceLinkParams) (string, error) {
	resp, err := Do[string](ctx, c, "createInvoiceLink", params)
	if err != nil {
		return "", err
	}
	return resp.Result, nil
}

// AnswerShippingQuery answers a shipping query.
func (c *Client) AnswerShippingQuery(ctx context.Context, params *AnswerShippingQueryParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "answerShippingQuery", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// AnswerPreCheckoutQuery answers a pre-checkout query.
func (c *Client) AnswerPreCheckoutQuery(ctx context.Context, params *AnswerPreCheckoutQueryParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "answerPreCheckoutQuery", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// AnswerInlineQuery answers an inline query.
func (c *Client) AnswerInlineQuery(ctx context.Context, params *AnswerInlineQueryParams) (bool, error) {
	resp, err := Do[bool](ctx, c, "answerInlineQuery", params)
	if err != nil {
		return false, err
	}
	return resp.Result, nil
}

// GetChat gets up-to-date chat information.
func (c *Client) GetChat(ctx context.Context, chatID any) (*Chat, error) {
	payload := map[string]any{"chat_id": chatID}
	resp, err := Do[Chat](ctx, c, "getChat", payload)
	if err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// GetChatMemberCount gets number of members in a chat.
func (c *Client) GetChatMemberCount(ctx context.Context, chatID any) (int, error) {
	payload := map[string]any{"chat_id": chatID}
	resp, err := Do[int](ctx, c, "getChatMemberCount", payload)
	if err != nil {
		return 0, err
	}
	return resp.Result, nil
}
