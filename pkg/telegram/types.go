package telegram

import "encoding/json"

// ApiResponse is the standard envelope for Telegram Bot API responses.
type ApiResponse[T any] struct {
	Ok          bool                `json:"ok"`
	Result      T                   `json:"result,omitempty"`
	ErrorCode   int                 `json:"error_code,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  *ResponseParameters `json:"parameters,omitempty"`
}

// ResponseParameters describes why a request was unsuccessful.
type ResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id,omitempty"`
	RetryAfter      int   `json:"retry_after,omitempty"`
}

// User represents a Telegram user or bot.
type User struct {
	ID                      int64   `json:"id"`
	IsBot                   bool    `json:"is_bot"`
	FirstName               string  `json:"first_name"`
	LastName                *string `json:"last_name,omitempty"`
	Username                *string `json:"username,omitempty"`
	LanguageCode            *string `json:"language_code,omitempty"`
	IsPremium               *bool   `json:"is_premium,omitempty"`
	AddedToAttachmentMenu   *bool   `json:"added_to_attachment_menu,omitempty"`
	CanJoinGroups           *bool   `json:"can_join_groups,omitempty"`
	CanReadAllGroupMessages *bool   `json:"can_read_all_group_messages,omitempty"`
	SupportsInlineQueries   *bool   `json:"supports_inline_queries,omitempty"`
	CanConnectToBusiness    *bool   `json:"can_connect_to_business,omitempty"`
	HasMainWebApp           *bool   `json:"has_main_web_app,omitempty"`
}

// Chat represents a chat.
type Chat struct {
	ID               int64             `json:"id"`
	Type             string            `json:"type"` // "private", "group", "supergroup" or "channel"
	Title            *string           `json:"title,omitempty"`
	Username         *string           `json:"username,omitempty"`
	FirstName        *string           `json:"first_name,omitempty"`
	LastName         *string           `json:"last_name,omitempty"`
	IsForum          *bool             `json:"is_forum,omitempty"`
	Photo            *ChatPhoto        `json:"photo,omitempty"`
	ActiveUsernames  []string          `json:"active_usernames,omitempty"`
	Description      *string           `json:"description,omitempty"`
	InviteLink       *string           `json:"invite_link,omitempty"`
	PinnedMessage    *Message          `json:"pinned_message,omitempty"`
	Permissions      *ChatPermissions  `json:"permissions,omitempty"`
	SlowModeDelay    *int              `json:"slow_mode_delay,omitempty"`
	UnrestrictBoostCount *int          `json:"unrestrict_boost_count,omitempty"`
	MessageAutoDeleteTime *int         `json:"message_auto_delete_time,omitempty"`
	HasAggressiveAntiSpamEnabled *bool `json:"has_aggressive_anti_spam_enabled,omitempty"`
	HasHiddenMembers *bool             `json:"has_hidden_members,omitempty"`
	HasProtectedContent *bool          `json:"has_protected_content,omitempty"`
	HasVisibleHistory *bool            `json:"has_visible_history,omitempty"`
	StickerSetName   *string           `json:"sticker_set_name,omitempty"`
	CanSetStickerSet *bool             `json:"can_set_sticker_set,omitempty"`
	CustomEmojiStickerSetName *string  `json:"custom_emoji_sticker_set_name,omitempty"`
	LinkedChatID     *int64            `json:"linked_chat_id,omitempty"`
	Location         *ChatLocation     `json:"location,omitempty"`
}

// ChatPhoto represents a chat photo.
type ChatPhoto struct {
	SmallFileID       string `json:"small_file_id"`
	SmallFileUniqueID string `json:"small_file_unique_id"`
	BigFileID         string `json:"big_file_id"`
	BigFileUniqueID   string `json:"big_file_unique_id"`
}

// ChatLocation represents a location to which a chat is connected.
type ChatLocation struct {
	Location Location `json:"location"`
	Address  string   `json:"address"`
}

// MessageId represents a unique message identifier.
type MessageId struct {
	MessageID int `json:"message_id"`
}

// Message represents a message.
type Message struct {
	MessageID            int                   `json:"message_id"`
	MessageThreadID      *int                  `json:"message_thread_id,omitempty"`
	From                 *User                 `json:"from,omitempty"`
	SenderChat           *Chat                 `json:"sender_chat,omitempty"`
	SenderBoostCount     *int                  `json:"sender_boost_count,omitempty"`
	SenderBusinessBot    *User                 `json:"sender_business_bot,omitempty"`
	Date                 int64                 `json:"date"`
	BusinessConnectionID *string               `json:"business_connection_id,omitempty"`
	Chat                 Chat                  `json:"chat"`
	ForwardFrom          *User                 `json:"forward_from,omitempty"`
	ForwardFromChat      *Chat                 `json:"forward_from_chat,omitempty"`
	ForwardFromMessageID *int                  `json:"forward_from_message_id,omitempty"`
	ForwardSignature     *string               `json:"forward_signature,omitempty"`
	ForwardSenderName    *string               `json:"forward_sender_name,omitempty"`
	ForwardDate          *int64                `json:"forward_date,omitempty"`
	IsTopicMessage       *bool                 `json:"is_topic_message,omitempty"`
	IsAutomaticForward   *bool                 `json:"is_automatic_forward,omitempty"`
	ReplyToMessage       *Message              `json:"reply_to_message,omitempty"`
	ViaBot               *User                 `json:"via_bot,omitempty"`
	EditDate             *int64                `json:"edit_date,omitempty"`
	HasProtectedContent  *bool                 `json:"has_protected_content,omitempty"`
	IsFromOffline        *bool                 `json:"is_from_offline,omitempty"`
	MediaGroupID         *string               `json:"media_group_id,omitempty"`
	AuthorSignature      *string               `json:"author_signature,omitempty"`
	Text                 *string               `json:"text,omitempty"`
	Entities             []MessageEntity       `json:"entities,omitempty"`
	LinkPreviewOptions   *LinkPreviewOptions   `json:"link_preview_options,omitempty"`
	Animation            *Animation            `json:"animation,omitempty"`
	Audio                *Audio                `json:"audio,omitempty"`
	Document             *Document             `json:"document,omitempty"`
	PaidMedia            *PaidMediaInfo        `json:"paid_media,omitempty"`
	Photo                []PhotoSize           `json:"photo,omitempty"`
	Sticker              *Sticker              `json:"sticker,omitempty"`
	Story                *Story                `json:"story,omitempty"`
	Video                *Video                `json:"video,omitempty"`
	VideoNote            *VideoNote            `json:"video_note,omitempty"`
	Voice                *Voice                `json:"voice,omitempty"`
	Caption              *string               `json:"caption,omitempty"`
	CaptionEntities      []MessageEntity       `json:"caption_entities,omitempty"`
	HasMediaSpoiler      *bool                 `json:"has_media_spoiler,omitempty"`
	Contact              *Contact              `json:"contact,omitempty"`
	Dice                 *Dice                 `json:"dice,omitempty"`
	Game                 *Game                 `json:"game,omitempty"`
	Poll                 *Poll                 `json:"poll,omitempty"`
	Venue                *Venue                `json:"venue,omitempty"`
	Location             *Location             `json:"location,omitempty"`
	NewChatMembers       []User                `json:"new_chat_members,omitempty"`
	LeftChatMember       *User                 `json:"left_chat_member,omitempty"`
	NewChatTitle         *string               `json:"new_chat_title,omitempty"`
	NewChatPhoto         []PhotoSize           `json:"new_chat_photo,omitempty"`
	DeleteChatPhoto      *bool                 `json:"delete_chat_photo,omitempty"`
	GroupChatCreated     *bool                 `json:"group_chat_created,omitempty"`
	SupergroupChatCreated *bool                `json:"supergroup_chat_created,omitempty"`
	ChannelChatCreated   *bool                 `json:"channel_chat_created,omitempty"`
	PinnedMessage        *Message              `json:"pinned_message,omitempty"`
	Invoice              *Invoice              `json:"invoice,omitempty"`
	SuccessfulPayment    *SuccessfulPayment    `json:"successful_payment,omitempty"`
	RefundedPayment      *RefundedPayment      `json:"refunded_payment,omitempty"`
	UserShared           *UserShared           `json:"user_shared,omitempty"`
	ChatShared           *ChatShared           `json:"chat_shared,omitempty"`
	WebAppData           *WebAppData           `json:"web_app_data,omitempty"`
	ReplyMarkup          *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// LinkPreviewOptions describes the options used for link preview generation.
type LinkPreviewOptions struct {
	IsDisabled       *bool   `json:"is_disabled,omitempty"`
	URL              *string `json:"url,omitempty"`
	PreferSmallMedia *bool   `json:"prefer_small_media,omitempty"`
	PreferLargeMedia *bool   `json:"prefer_large_media,omitempty"`
	ShowAboveText    *bool   `json:"show_above_text,omitempty"`
}

// MessageEntity represents one special entity in a text message.
type MessageEntity struct {
	Type          string  `json:"type"` // "mention", "hashtag", "bot_command", "url", "bold", "italic", "code", "pre", "text_link", "custom_emoji"
	Offset        int     `json:"offset"`
	Length        int     `json:"length"`
	URL           *string `json:"url,omitempty"`
	User          *User   `json:"user,omitempty"`
	Language      *string `json:"language,omitempty"`
	CustomEmojiID *string `json:"custom_emoji_id,omitempty"`
}

// PhotoSize represents one size of a photo or a file / sticker thumbnail.
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     *int   `json:"file_size,omitempty"`
}

// Animation represents an animation file.
type Animation struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	Duration     int        `json:"duration"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileName     *string    `json:"file_name,omitempty"`
	MimeType     *string    `json:"mime_type,omitempty"`
	FileSize     *int64     `json:"file_size,omitempty"`
}

// Audio represents an audio file.
type Audio struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Duration     int        `json:"duration"`
	Performer    *string    `json:"performer,omitempty"`
	Title        *string    `json:"title,omitempty"`
	FileName     *string    `json:"file_name,omitempty"`
	MimeType     *string    `json:"mime_type,omitempty"`
	FileSize     *int64     `json:"file_size,omitempty"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
}

// Document represents a general file.
type Document struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileName     *string    `json:"file_name,omitempty"`
	MimeType     *string    `json:"mime_type,omitempty"`
	FileSize     *int64     `json:"file_size,omitempty"`
}

// Video represents a video file.
type Video struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	Duration     int        `json:"duration"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileName     *string    `json:"file_name,omitempty"`
	MimeType     *string    `json:"mime_type,omitempty"`
	FileSize     *int64     `json:"file_size,omitempty"`
}

// VideoNote represents a video message.
type VideoNote struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Length       int        `json:"length"`
	Duration     int        `json:"duration"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileSize     *int64     `json:"file_size,omitempty"`
}

// Voice represents a voice note.
type Voice struct {
	FileID       string  `json:"file_id"`
	FileUniqueID string  `json:"file_unique_id"`
	Duration     int     `json:"duration"`
	MimeType     *string `json:"mime_type,omitempty"`
	FileSize     *int64  `json:"file_size,omitempty"`
}

// Contact represents a phone contact.
type Contact struct {
	PhoneNumber string  `json:"phone_number"`
	FirstName   string  `json:"first_name"`
	LastName    *string `json:"last_name,omitempty"`
	UserID      *int64  `json:"user_id,omitempty"`
	Vcard       *string `json:"vcard,omitempty"`
}

// Dice represents an animated emoji.
type Dice struct {
	Emoji string `json:"emoji"`
	Value int    `json:"value"`
}

// PollOption contains information about one answer option in a poll.
type PollOption struct {
	Text       string          `json:"text"`
	Entities   []MessageEntity `json:"entities,omitempty"`
	VoterCount int             `json:"voter_count"`
}

// PollAnswer represents an answer of a user in a non-anonymous poll.
type PollAnswer struct {
	PollID    string  `json:"poll_id"`
	VoterChat *Chat   `json:"voter_chat,omitempty"`
	User      *User   `json:"user,omitempty"`
	OptionIDs []int   `json:"option_ids"`
}

// Poll contains information about a poll.
type Poll struct {
	ID                    string          `json:"id"`
	Question              string          `json:"question"`
	QuestionEntities      []MessageEntity `json:"question_entities,omitempty"`
	Options               []PollOption    `json:"options"`
	TotalVoterCount       int             `json:"total_voter_count"`
	IsClosed              bool            `json:"is_closed"`
	IsAnonymous           bool            `json:"is_anonymous"`
	Type                  string          `json:"type"` // "regular" or "quiz"
	AllowsMultipleAnswers bool            `json:"allows_multiple_answers"`
	CorrectOptionID       *int            `json:"correct_option_id,omitempty"`
	Explanation           *string         `json:"explanation,omitempty"`
	ExplanationEntities   []MessageEntity `json:"explanation_entities,omitempty"`
	OpenPeriod            *int            `json:"open_period,omitempty"`
	CloseDate             *int64          `json:"close_date,omitempty"`
}

// Location represents a point on the map.
type Location struct {
	Latitude             float64  `json:"latitude"`
	Longitude            float64  `json:"longitude"`
	HorizontalAccuracy   *float64 `json:"horizontal_accuracy,omitempty"`
	LivePeriod           *int     `json:"live_period,omitempty"`
	Heading              *int     `json:"heading,omitempty"`
	ProximityAlertRadius *int     `json:"proximity_alert_radius,omitempty"`
}

// Venue represents a venue.
type Venue struct {
	Location       Location `json:"location"`
	Title          string   `json:"title"`
	Address        string   `json:"address"`
	FoursquareID   *string  `json:"foursquare_id,omitempty"`
	FoursquareType *string  `json:"foursquare_type,omitempty"`
	GooglePlaceID  *string  `json:"google_place_id,omitempty"`
	GooglePlaceType *string `json:"google_place_type,omitempty"`
}

// WebAppData describes data sent from a Web App to the bot.
type WebAppData struct {
	Data       string `json:"data"`
	ButtonText string `json:"button_text"`
}

// WebAppInfo describes a Web App.
type WebAppInfo struct {
	URL string `json:"url"`
}

// InlineKeyboardButton represents one button of an inline keyboard.
type InlineKeyboardButton struct {
	Text                         string                       `json:"text"`
	URL                          *string                      `json:"url,omitempty"`
	CallbackData                 *string                      `json:"callback_data,omitempty"`
	WebApp                       *WebAppInfo                  `json:"web_app,omitempty"`
	LoginURL                     *LoginUrl                    `json:"login_url,omitempty"`
	SwitchInlineQuery            *string                      `json:"switch_inline_query,omitempty"`
	SwitchInlineQueryCurrentChat *string                      `json:"switch_inline_query_current_chat,omitempty"`
	SwitchInlineQueryChosenChat  *SwitchInlineQueryChosenChat `json:"switch_inline_query_chosen_chat,omitempty"`
	CopyText                     *CopyTextButton              `json:"copy_text,omitempty"`
	CallbackGame                 *CallbackGame                `json:"callback_game,omitempty"`
	Pay                          *bool                        `json:"pay,omitempty"`
}

// CopyTextButton represents an inline button that copies text to clipboard.
type CopyTextButton struct {
	Text string `json:"text"`
}

// LoginUrl represents a parameter of the inline keyboard button used to automatically authorize a user.
type LoginUrl struct {
	URL                string  `json:"url"`
	ForwardText        *string `json:"forward_text,omitempty"`
	BotUsername        *string `json:"bot_username,omitempty"`
	RequestWriteAccess *bool   `json:"request_write_access,omitempty"`
}

// SwitchInlineQueryChosenChat represents an inline button that switches current user to inline mode in a chosen chat.
type SwitchInlineQueryChosenChat struct {
	Query             *string `json:"query,omitempty"`
	AllowUserChats    *bool   `json:"allow_user_chats,omitempty"`
	AllowBotChats     *bool   `json:"allow_bot_chats,omitempty"`
	AllowGroupChats   *bool   `json:"allow_group_chats,omitempty"`
	AllowChannelChats *bool   `json:"allow_channel_chats,omitempty"`
}

// InlineKeyboardMarkup represents an inline keyboard that appears right next to the message it belongs to.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// KeyboardButton represents one button of the reply keyboard.
type KeyboardButton struct {
	Text            string                     `json:"text"`
	RequestUsers    *KeyboardButtonRequestUsers `json:"request_users,omitempty"`
	RequestChat     *KeyboardButtonRequestChat  `json:"request_chat,omitempty"`
	RequestContact  *bool                      `json:"request_contact,omitempty"`
	RequestLocation *bool                      `json:"request_location,omitempty"`
	RequestPoll     *KeyboardButtonPollType    `json:"request_poll,omitempty"`
	WebApp          *WebAppInfo                `json:"web_app,omitempty"`
}

// KeyboardButtonRequestUsers defines criteria used to request users.
type KeyboardButtonRequestUsers struct {
	RequestID     int   `json:"request_id"`
	UserIsBot     *bool `json:"user_is_bot,omitempty"`
	UserIsPremium *bool `json:"user_is_premium,omitempty"`
	MaxQuantity   *int  `json:"max_quantity,omitempty"`
	RequestName   *bool `json:"request_name,omitempty"`
	RequestUsername *bool `json:"request_username,omitempty"`
	RequestPhoto  *bool `json:"request_photo,omitempty"`
}

// KeyboardButtonRequestChat defines criteria used to request a chat.
type KeyboardButtonRequestChat struct {
	RequestID         int                      `json:"request_id"`
	ChatIsChannel     bool                     `json:"chat_is_channel"`
	ChatIsForum       *bool                    `json:"chat_is_forum,omitempty"`
	ChatHasUsername   *bool                    `json:"chat_has_username,omitempty"`
	ChatIsCreated     *bool                    `json:"chat_is_created,omitempty"`
	UserAdministratorRights *ChatAdministratorRights `json:"user_administrator_rights,omitempty"`
	BotAdministratorRights  *ChatAdministratorRights `json:"bot_administrator_rights,omitempty"`
	BotIsMember       *bool                    `json:"bot_is_member,omitempty"`
	RequestTitle      *bool                    `json:"request_title,omitempty"`
	RequestUsername   *bool                    `json:"request_username,omitempty"`
	RequestPhoto      *bool                    `json:"request_photo,omitempty"`
}

// KeyboardButtonPollType represents type of poll allowed to be created.
type KeyboardButtonPollType struct {
	Type *string `json:"type,omitempty"`
}

// ReplyKeyboardMarkup represents a custom keyboard with reply options.
type ReplyKeyboardMarkup struct {
	Keyboard              [][]KeyboardButton `json:"keyboard"`
	IsPersistent          *bool              `json:"is_persistent,omitempty"`
	ResizeKeyboard        *bool              `json:"resize_keyboard,omitempty"`
	OneTimeKeyboard       *bool              `json:"one_time_keyboard,omitempty"`
	InputFieldPlaceholder *string            `json:"input_field_placeholder,omitempty"`
	Selective             *bool              `json:"selective,omitempty"`
}

// ReplyKeyboardRemove requests clients to remove the custom keyboard.
type ReplyKeyboardRemove struct {
	RemoveKeyboard bool  `json:"remove_keyboard"`
	Selective      *bool `json:"selective,omitempty"`
}

// ForceReply forces Telegram clients to display a reply interface to the user.
type ForceReply struct {
	ForceReply            bool    `json:"force_reply"`
	InputFieldPlaceholder *string `json:"input_field_placeholder,omitempty"`
	Selective             *bool   `json:"selective,omitempty"`
}

// MenuButton describes the bot's menu button in a chat.
type MenuButton struct {
	Type   string      `json:"type"` // "commands", "web_app", "default"
	Text   *string     `json:"text,omitempty"`
	WebApp *WebAppInfo `json:"web_app,omitempty"`
}

// ChatPermissions describes actions that a non-administrator user is allowed to take in a chat.
type ChatPermissions struct {
	CanSendMessages       *bool `json:"can_send_messages,omitempty"`
	CanSendAudios         *bool `json:"can_send_audios,omitempty"`
	CanSendDocuments      *bool `json:"can_send_documents,omitempty"`
	CanSendPhotos         *bool `json:"can_send_photos,omitempty"`
	CanSendVideos         *bool `json:"can_send_videos,omitempty"`
	CanSendVideoNotes     *bool `json:"can_send_video_notes,omitempty"`
	CanSendVoiceNotes     *bool `json:"can_send_voice_notes,omitempty"`
	CanSendPolls          *bool `json:"can_send_polls,omitempty"`
	CanSendOtherMessages  *bool `json:"can_send_other_messages,omitempty"`
	CanAddWebPagePreviews *bool `json:"can_add_web_page_previews,omitempty"`
	CanChangeInfo         *bool `json:"can_change_info,omitempty"`
	CanInviteUsers        *bool `json:"can_invite_users,omitempty"`
	CanPinMessages        *bool `json:"can_pin_messages,omitempty"`
	CanManageTopics       *bool `json:"can_manage_topics,omitempty"`
}

// ChatAdministratorRights represents rights of an administrator in a chat.
type ChatAdministratorRights struct {
	IsAnonymous         bool  `json:"is_anonymous"`
	CanManageChat       bool  `json:"can_manage_chat"`
	CanDeleteMessages   bool  `json:"can_delete_messages"`
	CanManageVideoChats bool  `json:"can_manage_video_chats"`
	CanRestrictMembers  bool  `json:"can_restrict_members"`
	CanPromoteMembers   bool  `json:"can_promote_members"`
	CanChangeInfo       bool  `json:"can_change_info"`
	CanInviteUsers      bool  `json:"can_invite_users"`
	CanPostStories      *bool `json:"can_post_stories,omitempty"`
	CanEditStories      *bool `json:"can_edit_stories,omitempty"`
	CanDeleteStories    *bool `json:"can_delete_stories,omitempty"`
	CanPostMessages     *bool `json:"can_post_messages,omitempty"`
	CanEditMessages     *bool `json:"can_edit_messages,omitempty"`
	CanPinMessages      *bool `json:"can_pin_messages,omitempty"`
	CanManageTopics     *bool `json:"can_manage_topics,omitempty"`
}

// ChatMember contains information about one member of a chat.
type ChatMember struct {
	Status      string `json:"status"` // "creator", "administrator", "member", "restricted", "left", "kicked"
	User        User   `json:"user"`
	IsAnonymous *bool  `json:"is_anonymous,omitempty"`
	CustomTitle *string`json:"custom_title,omitempty"`
}

// ChatInviteLink represents an invite link for a chat.
type ChatInviteLink struct {
	InviteLink              string `json:"invite_link"`
	Creator                 User   `json:"creator"`
	CreatesJoinRequest      bool   `json:"creates_join_request"`
	IsPrimary               bool   `json:"is_primary"`
	IsRevoked               bool   `json:"is_revoked"`
	Name                    *string`json:"name,omitempty"`
	ExpireDate              *int64 `json:"expire_date,omitempty"`
	MemberLimit             *int   `json:"member_limit,omitempty"`
	PendingMemberCount      *int   `json:"pending_member_count,omitempty"`
	SubscriptionPeriod      *int   `json:"subscription_period,omitempty"`
	SubscriptionPrice       *int   `json:"subscription_price,omitempty"`
}

// ChatJoinRequest represents a join request sent to a chat.
type ChatJoinRequest struct {
	Chat       Chat            `json:"chat"`
	From       User            `json:"from"`
	UserChatID int64           `json:"user_chat_id"`
	Date       int64           `json:"date"`
	Bio        *string         `json:"bio,omitempty"`
	InviteLink *ChatInviteLink `json:"invite_link,omitempty"`
}

// ForumTopic represents a forum topic.
type ForumTopic struct {
	MessageThreadID   int     `json:"message_thread_id"`
	Name              string  `json:"name"`
	IconColor         int     `json:"icon_color"`
	IconCustomEmojiID *string `json:"icon_custom_emoji_id,omitempty"`
}

// BotCommand represents a bot command.
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// BotCommandScope represents the scope to which bot commands are applied.
type BotCommandScope struct {
	Type   string `json:"type"` // "default", "all_private_chats", "all_group_chats", "all_chat_administrators", "chat", "chat_administrators", "chat_member"
	ChatID *any   `json:"chat_id,omitempty"`
	UserID *int64 `json:"user_id,omitempty"`
}

// CallbackQuery represents an incoming callback query from a callback button.
type CallbackQuery struct {
	ID              string   `json:"id"`
	From            User     `json:"from"`
	Message         *Message `json:"message,omitempty"`
	InlineMessageID *string  `json:"inline_message_id,omitempty"`
	ChatInstance    string   `json:"chat_instance"`
	Data            *string  `json:"data,omitempty"`
	GameShortName   *string  `json:"game_short_name,omitempty"`
}

// InlineQuery represents an incoming inline query.
type InlineQuery struct {
	ID       string    `json:"id"`
	From     User      `json:"from"`
	Query    string    `json:"query"`
	Offset   string    `json:"offset"`
	ChatType *string   `json:"chat_type,omitempty"`
	Location *Location `json:"location,omitempty"`
}

// ChosenInlineResult represents a result of an inline query that was chosen by the user and sent to their chat partner.
type ChosenInlineResult struct {
	ResultID        string    `json:"result_id"`
	From            User      `json:"from"`
	Location        *Location `json:"location,omitempty"`
	InlineMessageID *string   `json:"inline_message_id,omitempty"`
	Query           string    `json:"query"`
}

// Update represents an incoming update.
type Update struct {
	UpdateID           int                 `json:"update_id"`
	Message            *Message            `json:"message,omitempty"`
	EditedMessage      *Message            `json:"edited_message,omitempty"`
	ChannelPost        *Message            `json:"channel_post,omitempty"`
	EditedChannelPost  *Message            `json:"edited_channel_post,omitempty"`
	BusinessConnection *BusinessConnection `json:"business_connection,omitempty"`
	BusinessMessage    *Message            `json:"business_message,omitempty"`
	EditedBusinessMessage *Message         `json:"edited_business_message,omitempty"`
	DeletedBusinessMessages *BusinessMessagesDeleted `json:"deleted_business_messages,omitempty"`
	MessageReaction    *MessageReactionUpdated `json:"message_reaction,omitempty"`
	MessageReactionCount *MessageReactionCountUpdated `json:"message_reaction_count,omitempty"`
	InlineQuery        *InlineQuery        `json:"inline_query,omitempty"`
	ChosenInlineResult *ChosenInlineResult `json:"chosen_inline_result,omitempty"`
	CallbackQuery      *CallbackQuery      `json:"callback_query,omitempty"`
	ShippingQuery      *ShippingQuery      `json:"shipping_query,omitempty"`
	PreCheckoutQuery   *PreCheckoutQuery   `json:"pre_checkout_query,omitempty"`
	Poll               *Poll               `json:"poll,omitempty"`
	PollAnswer         *PollAnswer         `json:"poll_answer,omitempty"`
	MyChatMember       *ChatMemberUpdated  `json:"my_chat_member,omitempty"`
	ChatMember         *ChatMemberUpdated  `json:"chat_member,omitempty"`
	ChatJoinRequest    *ChatJoinRequest    `json:"chat_join_request,omitempty"`
	ChatBoost          *ChatBoostUpdated   `json:"chat_boost,omitempty"`
	RemovedChatBoost   *ChatBoostRemoved   `json:"removed_chat_boost,omitempty"`
}

// BusinessConnection describes the connection of the bot with a business account.
type BusinessConnection struct {
	ID         string `json:"id"`
	User       User   `json:"user"`
	UserChatID int64  `json:"user_chat_id"`
	Date       int64  `json:"date"`
	CanReply   bool   `json:"can_reply"`
	IsEnabled  bool   `json:"is_enabled"`
}

// BusinessMessagesDeleted received when messages are deleted on behalf of a business account.
type BusinessMessagesDeleted struct {
	BusinessConnectionID string `json:"business_connection_id"`
	Chat                 Chat   `json:"chat"`
	MessageIDs           []int  `json:"message_ids"`
}

// MessageReactionUpdated represents a change of a reaction on a message.
type MessageReactionUpdated struct {
	Chat        Chat             `json:"chat"`
	MessageID   int              `json:"message_id"`
	User        *User            `json:"user,omitempty"`
	ActorChat   *Chat            `json:"actor_chat,omitempty"`
	Date        int64            `json:"date"`
	OldReaction []ReactionType   `json:"old_reaction"`
	NewReaction []ReactionType   `json:"new_reaction"`
}

// MessageReactionCountUpdated represents reaction changes on a message with anonymous reactions.
type MessageReactionCountUpdated struct {
	Chat      Chat            `json:"chat"`
	MessageID int             `json:"message_id"`
	Date      int64           `json:"date"`
	Reactions []ReactionCount `json:"reactions"`
}

// ReactionType describes the type of a reaction.
type ReactionType struct {
	Type          string  `json:"type"` // "emoji" or "custom_emoji"
	Emoji         *string `json:"emoji,omitempty"`
	CustomEmojiID *string `json:"custom_emoji_id,omitempty"`
}

// ReactionCount represents reaction count for a specific reaction type.
type ReactionCount struct {
	Type       ReactionType `json:"type"`
	TotalCount int          `json:"total_count"`
}

// ChatMemberUpdated represents changes in the status of a chat member.
type ChatMemberUpdated struct {
	Chat          Chat            `json:"chat"`
	From          User            `json:"from"`
	Date          int64           `json:"date"`
	OldChatMember ChatMember      `json:"old_chat_member"`
	NewChatMember ChatMember      `json:"new_chat_member"`
	InviteLink    *ChatInviteLink `json:"invite_link,omitempty"`
	ViaJoinRequest *bool          `json:"via_join_request,omitempty"`
	ViaChatFolderInviteLink *bool `json:"via_chat_folder_invite_link,omitempty"`
}

// ChatBoostUpdated represents a boost added to a chat.
type ChatBoostUpdated struct {
	Chat  Chat      `json:"chat"`
	Boost ChatBoost `json:"boost"`
}

// ChatBoostRemoved represents a boost removed from a chat.
type ChatBoostRemoved struct {
	Chat       Chat   `json:"chat"`
	BoostID    string `json:"boost_id"`
	RemoveDate int64  `json:"remove_date"`
}

// ChatBoost contains information about a chat boost.
type ChatBoost struct {
	BoostID   string `json:"boost_id"`
	AddDate   int64  `json:"add_date"`
	ExpirationDate int64 `json:"expiration_date"`
}

// ShippingQuery contains information about an incoming shipping query.
type ShippingQuery struct {
	ID              string          `json:"id"`
	From            User            `json:"from"`
	InvoicePayload  string          `json:"invoice_payload"`
	ShippingAddress ShippingAddress `json:"shipping_address"`
}

// PreCheckoutQuery contains information about an incoming pre-checkout query.
type PreCheckoutQuery struct {
	ID               string     `json:"id"`
	From             User       `json:"from"`
	Currency         string     `json:"currency"`
	TotalAmount      int        `json:"total_amount"`
	InvoicePayload   string     `json:"invoice_payload"`
	ShippingOptionID *string    `json:"shipping_option_id,omitempty"`
	OrderInfo        *OrderInfo `json:"order_info,omitempty"`
}

// SuccessfulPayment contains basic information about a successful payment.
type SuccessfulPayment struct {
	Currency                string     `json:"currency"`
	TotalAmount             int        `json:"total_amount"`
	InvoicePayload          string     `json:"invoice_payload"`
	ShippingOptionID        *string    `json:"shipping_option_id,omitempty"`
	OrderInfo               *OrderInfo `json:"order_info,omitempty"`
	TelegramPaymentChargeID string     `json:"telegram_payment_charge_id"`
	ProviderPaymentChargeID string     `json:"provider_payment_charge_id"`
}

// RefundedPayment contains basic information about a refunded payment.
type RefundedPayment struct {
	Currency                string  `json:"currency"`
	TotalAmount             int     `json:"total_amount"`
	InvoicePayload          string  `json:"invoice_payload"`
	TelegramPaymentChargeID string  `json:"telegram_payment_charge_id"`
	ProviderPaymentChargeID *string `json:"provider_payment_charge_id,omitempty"`
}

// Invoice contains basic information about an invoice.
type Invoice struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	StartParameter string `json:"start_parameter"`
	Currency       string `json:"currency"`
	TotalAmount    int    `json:"total_amount"`
}

// ShippingAddress represents a shipping address.
type ShippingAddress struct {
	CountryCode string `json:"country_code"`
	State       string `json:"state"`
	City        string `json:"city"`
	StreetLine1 string `json:"street_line1"`
	StreetLine2 string `json:"street_line2"`
	PostCode    string `json:"post_code"`
}

// OrderInfo represents information about an order.
type OrderInfo struct {
	Name            *string          `json:"name,omitempty"`
	PhoneNumber     *string          `json:"phone_number,omitempty"`
	Email           *string          `json:"email,omitempty"`
	ShippingAddress *ShippingAddress `json:"shipping_address,omitempty"`
}

// LabeledPrice represents a portion of the price for goods or services.
type LabeledPrice struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"`
}

// ShippingOption represents one shipping option.
type ShippingOption struct {
	ID     string         `json:"id"`
	Title  string         `json:"title"`
	Prices []LabeledPrice `json:"prices"`
}

// WebhookInfo contains information about the current status of a webhook.
type WebhookInfo struct {
	URL                          string   `json:"url"`
	HasCustomCertificate         bool     `json:"has_custom_certificate"`
	PendingUpdateCount           int      `json:"pending_update_count"`
	IPAddress                    *string  `json:"ip_address,omitempty"`
	LastErrorDate                *int64   `json:"last_error_date,omitempty"`
	LastErrorMessage             *string  `json:"last_error_message,omitempty"`
	LastSynchronizationErrorDate *int64   `json:"last_synchronization_error_date,omitempty"`
	MaxConnections               *int     `json:"max_connections,omitempty"`
	AllowedUpdates               []string `json:"allowed_updates,omitempty"`
}

// File represents a file ready to be downloaded.
type File struct {
	FileID       string  `json:"file_id"`
	FileUniqueID string  `json:"file_unique_id"`
	FileSize     *int64  `json:"file_size,omitempty"`
	FilePath     *string `json:"file_path,omitempty"`
}

// UserProfilePhotos represents a user's profile pictures.
type UserProfilePhotos struct {
	TotalCount int           `json:"total_count"`
	Photos     [][]PhotoSize `json:"photos"`
}

// SentWebAppMessage describes an inline message sent by a Web App on behalf of a user.
type SentWebAppMessage struct {
	InlineMessageID *string `json:"inline_message_id,omitempty"`
}

// PreparedInlineMessage describes an inline message prepared to be sent by a Mini App user.
type PreparedInlineMessage struct {
	ID         string `json:"id"`
	ExpirationDate int64 `json:"expiration_date"`
}

// PreparedKeyboardButton describes a keyboard button prepared to be used by a Mini App user.
type PreparedKeyboardButton struct {
	ID string `json:"id"`
}

// Sticker represents a sticker.
type Sticker struct {
	FileID           string     `json:"file_id"`
	FileUniqueID     string     `json:"file_unique_id"`
	Type             string     `json:"type"` // "regular", "mask", "custom_emoji"
	Width            int        `json:"width"`
	Height           int        `json:"height"`
	IsAnimated       bool       `json:"is_animated"`
	IsVideo          bool       `json:"is_video"`
	Thumbnail        *PhotoSize `json:"thumbnail,omitempty"`
	Emoji            *string    `json:"emoji,omitempty"`
	SetName          *string    `json:"set_name,omitempty"`
	CustomEmojiID    *string    `json:"custom_emoji_id,omitempty"`
	FileSize         *int64     `json:"file_size,omitempty"`
}

// StickerSet represents a sticker set.
type StickerSet struct {
	Name        string     `json:"name"`
	Title       string     `json:"title"`
	StickerType string     `json:"sticker_type"` // "regular", "mask", "custom_emoji"
	Stickers    []Sticker  `json:"stickers"`
	Thumbnail   *PhotoSize `json:"thumbnail,omitempty"`
}

// Story represents a story.
type Story struct {
	Chat Chat `json:"chat"`
	ID   int  `json:"id"`
}

// Game represents a game.
type Game struct {
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Photo        []PhotoSize     `json:"photo"`
	Text         *string         `json:"text,omitempty"`
	TextEntities []MessageEntity `json:"text_entities,omitempty"`
	Animation    *Animation      `json:"animation,omitempty"`
}

// CallbackGame is a placeholder for games.
type CallbackGame struct{}

// PaidMediaInfo describes the paid media added to a message.
type PaidMediaInfo struct {
	StarCount int              `json:"star_count"`
	PaidMedia []json.RawMessage `json:"paid_media"`
}

// UserShared contains information about the user whose identifier was shared with the bot using a KeyboardButtonRequestUsers button.
type UserShared struct {
	RequestID int   `json:"request_id"`
	UserID    int64 `json:"user_id"`
}

// ChatShared contains information about the chat whose identifier was shared with the bot using a KeyboardButtonRequestChat button.
type ChatShared struct {
	RequestID int   `json:"request_id"`
	ChatID    int64 `json:"chat_id"`
}

// StarTransactions contains a list of Telegram Star transactions.
type StarTransactions struct {
	Transactions []StarTransaction `json:"transactions"`
}

// StarTransaction describes a Telegram Star transaction.
type StarTransaction struct {
	ID     string `json:"id"`
	Amount int    `json:"amount"`
	Date   int64  `json:"date"`
}
