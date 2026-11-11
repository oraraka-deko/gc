package telegram

import (
	"encoding/json"
	"fmt"
)

// UpdateType represents the specific category of an incoming Telegram update.
type UpdateType string

const (
	UpdateTypeUnknown                  UpdateType = "unknown"
	UpdateTypeMessage                  UpdateType = "message"
	UpdateTypeEditedMessage            UpdateType = "edited_message"
	UpdateTypeChannelPost              UpdateType = "channel_post"
	UpdateTypeEditedChannelPost        UpdateType = "edited_channel_post"
	UpdateTypeBusinessConnection       UpdateType = "business_connection"
	UpdateTypeBusinessMessage          UpdateType = "business_message"
	UpdateTypeEditedBusinessMessage    UpdateType = "edited_business_message"
	UpdateTypeDeletedBusinessMessages  UpdateType = "deleted_business_messages"
	UpdateTypeGuestMessage             UpdateType = "guest_message"
	UpdateTypeMessageReaction          UpdateType = "message_reaction"
	UpdateTypeMessageReactionCount     UpdateType = "message_reaction_count"
	UpdateTypeInlineQuery              UpdateType = "inline_query"
	UpdateTypeChosenInlineResult       UpdateType = "chosen_inline_result"
	UpdateTypeCallbackQuery            UpdateType = "callback_query"
	UpdateTypeWidgetQuery              UpdateType = "widget_query"
	UpdateTypeSharedWidgetQuery        UpdateType = "shared_widget_query"
	UpdateTypeShippingQuery            UpdateType = "shipping_query"
	UpdateTypePreCheckoutQuery         UpdateType = "pre_checkout_query"
	UpdateTypePurchasedPaidMedia       UpdateType = "purchased_paid_media"
	UpdateTypePoll                     UpdateType = "poll"
	UpdateTypePollAnswer               UpdateType = "poll_answer"
	UpdateTypeMyChatMember             UpdateType = "my_chat_member"
	UpdateTypeChatMember               UpdateType = "chat_member"
	UpdateTypeChatJoinRequest          UpdateType = "chat_join_request"
	UpdateTypeChatBoost                UpdateType = "chat_boost"
	UpdateTypeRemovedChatBoost         UpdateType = "removed_chat_boost"
	UpdateTypeManagedBot               UpdateType = "managed_bot"
	UpdateTypeSubscription             UpdateType = "subscription"
	UpdateTypeStoppedMessageGeneration UpdateType = "stopped_message_generation"
)

// AllUpdateTypes returns a list of all recognized update type strings for filtering or registration.
func AllUpdateTypes() []string {
	return []string{
		string(UpdateTypeMessage),
		string(UpdateTypeEditedMessage),
		string(UpdateTypeChannelPost),
		string(UpdateTypeEditedChannelPost),
		string(UpdateTypeBusinessConnection),
		string(UpdateTypeBusinessMessage),
		string(UpdateTypeEditedBusinessMessage),
		string(UpdateTypeDeletedBusinessMessages),
		string(UpdateTypeGuestMessage),
		string(UpdateTypeMessageReaction),
		string(UpdateTypeMessageReactionCount),
		string(UpdateTypeInlineQuery),
		string(UpdateTypeChosenInlineResult),
		string(UpdateTypeCallbackQuery),
		string(UpdateTypeWidgetQuery),
		string(UpdateTypeSharedWidgetQuery),
		string(UpdateTypeShippingQuery),
		string(UpdateTypePreCheckoutQuery),
		string(UpdateTypePurchasedPaidMedia),
		string(UpdateTypePoll),
		string(UpdateTypePollAnswer),
		string(UpdateTypeMyChatMember),
		string(UpdateTypeChatMember),
		string(UpdateTypeChatJoinRequest),
		string(UpdateTypeChatBoost),
		string(UpdateTypeRemovedChatBoost),
		string(UpdateTypeManagedBot),
		string(UpdateTypeSubscription),
		string(UpdateTypeStoppedMessageGeneration),
	}
}

// Type dynamically determines which update field is present according to Telegram Bot API specification.
// At most one optional field can be present in any given update.
func (u *Update) Type() UpdateType {
	if u == nil {
		return UpdateTypeUnknown
	}

	switch {
	case u.Message != nil:
		return UpdateTypeMessage
	case u.EditedMessage != nil:
		return UpdateTypeEditedMessage
	case u.ChannelPost != nil:
		return UpdateTypeChannelPost
	case u.EditedChannelPost != nil:
		return UpdateTypeEditedChannelPost
	case u.BusinessConnection != nil:
		return UpdateTypeBusinessConnection
	case u.BusinessMessage != nil:
		return UpdateTypeBusinessMessage
	case u.EditedBusinessMessage != nil:
		return UpdateTypeEditedBusinessMessage
	case u.DeletedBusinessMessages != nil:
		return UpdateTypeDeletedBusinessMessages
	case u.GuestMessage != nil:
		return UpdateTypeGuestMessage
	case u.MessageReaction != nil:
		return UpdateTypeMessageReaction
	case u.MessageReactionCount != nil:
		return UpdateTypeMessageReactionCount
	case u.InlineQuery != nil:
		return UpdateTypeInlineQuery
	case u.ChosenInlineResult != nil:
		return UpdateTypeChosenInlineResult
	case u.CallbackQuery != nil:
		return UpdateTypeCallbackQuery
	case u.WidgetQuery != nil:
		return UpdateTypeWidgetQuery
	case u.SharedWidgetQuery != nil:
		return UpdateTypeSharedWidgetQuery
	case u.ShippingQuery != nil:
		return UpdateTypeShippingQuery
	case u.PreCheckoutQuery != nil:
		return UpdateTypePreCheckoutQuery
	case u.PurchasedPaidMedia != nil:
		return UpdateTypePurchasedPaidMedia
	case u.Poll != nil:
		return UpdateTypePoll
	case u.PollAnswer != nil:
		return UpdateTypePollAnswer
	case u.MyChatMember != nil:
		return UpdateTypeMyChatMember
	case u.ChatMember != nil:
		return UpdateTypeChatMember
	case u.ChatJoinRequest != nil:
		return UpdateTypeChatJoinRequest
	case u.ChatBoost != nil:
		return UpdateTypeChatBoost
	case u.RemovedChatBoost != nil:
		return UpdateTypeRemovedChatBoost
	case u.ManagedBot != nil:
		return UpdateTypeManagedBot
	case u.Subscription != nil:
		return UpdateTypeSubscription
	case u.StoppedMessageGeneration != nil:
		return UpdateTypeStoppedMessageGeneration
	default:
		return UpdateTypeUnknown
	}
}

// Sender returns the user who initiated the update action, if available.
func (u *Update) Sender() *User {
	if u == nil {
		return nil
	}
	switch {
	case u.Message != nil:
		return u.Message.From
	case u.EditedMessage != nil:
		return u.EditedMessage.From
	case u.BusinessMessage != nil:
		return u.BusinessMessage.From
	case u.GuestMessage != nil:
		return u.GuestMessage.From
	case u.CallbackQuery != nil:
		return &u.CallbackQuery.From
	case u.InlineQuery != nil:
		return &u.InlineQuery.From
	case u.ChosenInlineResult != nil:
		return &u.ChosenInlineResult.From
	case u.ShippingQuery != nil:
		return &u.ShippingQuery.From
	case u.PreCheckoutQuery != nil:
		return &u.PreCheckoutQuery.From
	case u.PurchasedPaidMedia != nil:
		return &u.PurchasedPaidMedia.From
	case u.PollAnswer != nil:
		return u.PollAnswer.User
	case u.MyChatMember != nil:
		return &u.MyChatMember.From
	case u.ChatMember != nil:
		return &u.ChatMember.From
	case u.ChatJoinRequest != nil:
		return &u.ChatJoinRequest.From
	case u.BusinessConnection != nil:
		return &u.BusinessConnection.User
	case u.MessageReaction != nil:
		return u.MessageReaction.User
	case u.WidgetQuery != nil:
		return &u.WidgetQuery.From
	case u.SharedWidgetQuery != nil:
		return &u.SharedWidgetQuery.From
	case u.Subscription != nil:
		return u.Subscription.User
	case u.StoppedMessageGeneration != nil:
		return u.StoppedMessageGeneration.User
	default:
		return nil
	}
}

// Chat returns the chat associated with the update, if applicable.
func (u *Update) Chat() *Chat {
	if u == nil {
		return nil
	}
	switch {
	case u.Message != nil:
		return &u.Message.Chat
	case u.EditedMessage != nil:
		return &u.EditedMessage.Chat
	case u.ChannelPost != nil:
		return &u.ChannelPost.Chat
	case u.EditedChannelPost != nil:
		return &u.EditedChannelPost.Chat
	case u.BusinessMessage != nil:
		return &u.BusinessMessage.Chat
	case u.DeletedBusinessMessages != nil:
		return &u.DeletedBusinessMessages.Chat
	case u.GuestMessage != nil:
		return &u.GuestMessage.Chat
	case u.CallbackQuery != nil && u.CallbackQuery.Message != nil:
		return &u.CallbackQuery.Message.Chat
	case u.ChatJoinRequest != nil:
		return &u.ChatJoinRequest.Chat
	case u.ChatBoost != nil:
		return &u.ChatBoost.Chat
	case u.RemovedChatBoost != nil:
		return &u.RemovedChatBoost.Chat
	case u.MyChatMember != nil:
		return &u.MyChatMember.Chat
	case u.ChatMember != nil:
		return &u.ChatMember.Chat
	case u.MessageReaction != nil:
		return &u.MessageReaction.Chat
	case u.MessageReactionCount != nil:
		return &u.MessageReactionCount.Chat
	case u.WidgetQuery != nil:
		return u.WidgetQuery.Chat
	case u.SharedWidgetQuery != nil:
		return u.SharedWidgetQuery.Chat
	case u.Subscription != nil:
		return &u.Subscription.Chat
	case u.StoppedMessageGeneration != nil:
		return &u.StoppedMessageGeneration.Chat
	default:
		return nil
	}
}

// Text extracts textual content or query data from the update.
func (u *Update) Text() string {
	if u == nil {
		return ""
	}
	switch {
	case u.Message != nil && u.Message.Text != nil:
		return *u.Message.Text
	case u.Message != nil && u.Message.Caption != nil:
		return *u.Message.Caption
	case u.EditedMessage != nil && u.EditedMessage.Text != nil:
		return *u.EditedMessage.Text
	case u.ChannelPost != nil && u.ChannelPost.Text != nil:
		return *u.ChannelPost.Text
	case u.BusinessMessage != nil && u.BusinessMessage.Text != nil:
		return *u.BusinessMessage.Text
	case u.CallbackQuery != nil && u.CallbackQuery.Data != nil:
		return *u.CallbackQuery.Data
	case u.InlineQuery != nil:
		return u.InlineQuery.Query
	default:
		return ""
	}
}

// UnmarshalUpdate decodes a raw JSON payload into an Update struct.
func UnmarshalUpdate(data []byte) (*Update, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot unmarshal empty update payload")
	}
	var u Update
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON update: %w", err)
	}
	return &u, nil
}

// MarshalUpdate encodes an Update struct into standard JSON bytes.
func MarshalUpdate(u *Update) ([]byte, error) {
	if u == nil {
		return nil, fmt.Errorf("cannot marshal nil update")
	}
	return json.Marshal(u)
}
