package miniapp

import (
	"time"
)

// RafflePrizeType categorizes prizes awarded in raffles.
type RafflePrizeType string

const (
	PrizeStars         RafflePrizeType = "stars"
	PrizeTON           RafflePrizeType = "ton"
	PrizeNFTGift       RafflePrizeType = "nft_gift"
	PrizeInAppCurrency RafflePrizeType = "in_app_currency"
	PrizePhysical      RafflePrizeType = "physical"
	PrizePromoCode     RafflePrizeType = "promo_code"
)

// RafflePrize defines a tier and quantity of rewards in a prize pool.
type RafflePrize struct {
	PrizeID                 string          `json:"prize_id"`
	Rank                    int             `json:"rank"` // 1 = First place, 2 = Second, etc.
	Title                   string          `json:"title"`
	PrizeType               RafflePrizeType `json:"prize_type"`
	Value                   any             `json:"value"` // e.g. "1000000000" (nanoTON string) or 500 (Stars number)
	WinnerCountForThisPrize int             `json:"winner_count_for_this_prize"`
	DeliveredState          *string         `json:"delivered_state,omitempty"` // "escrowed", "manual_claim", "auto_credited"
}

// EntryTaskType defines requirements for ticket qualification.
type EntryTaskType string

const (
	EntryTaskJoinChannel   EntryTaskType = "join_telegram_channel"
	EntryTaskJoinGroup     EntryTaskType = "join_telegram_group"
	EntryTaskViewPost      EntryTaskType = "view_telegram_post"
	EntryTaskVisitLink     EntryTaskType = "visit_external_link"
	EntryTaskInviteFriends EntryTaskType = "invite_friends"
)

// EntryTaskRequirement defines a mandatory or optional task to earn raffle tickets.
type EntryTaskRequirement struct {
	TaskID                   string        `json:"task_id"`
	TaskType                 EntryTaskType `json:"task_type"`
	Title                    string        `json:"title"`
	TargetURLOrChatID        string        `json:"target_url_or_chat_id"` // e.g. "@mychannel" or "-100123..."
	PostID                   *int          `json:"post_id,omitempty"`
	MinimumReferralsRequired *int          `json:"minimum_referrals_required,omitempty"`
	IsMandatory              bool          `json:"is_mandatory"`
}

// WinnerSelectionPreset defines the mathematical drawing algorithm.
type WinnerSelectionPreset string

const (
	PresetRandomProvablyFair       WinnerSelectionPreset = "random_provably_fair"       // SHA-256 seed commitment
	PresetTopTicketHolders         WinnerSelectionPreset = "top_ticket_holders"         // Leaderboard-based
	PresetWeightedRandomByTickets WinnerSelectionPreset = "weighted_random_by_tickets" // More tickets = higher probability
	PresetFirstComeFirstServed     WinnerSelectionPreset = "first_come_first_served"    // Speed run
	PresetManualHostSelection      WinnerSelectionPreset = "manual_host_selection"     // Manual review
)

// WinnerSelectionConfig holds entropy seeds and rules for drawing winners.
type WinnerSelectionConfig struct {
	Preset                   WinnerSelectionPreset `json:"preset"`
	AllowMultipleWinsPerUser bool                  `json:"allow_multiple_wins_per_user"`
	ServerSeedHash           *string               `json:"server_seed_hash,omitempty"` // SHA-256 commitment published at creation
	PublicSeed               *string               `json:"public_seed,omitempty"`      // Future TON block hash / external entropy
}

// CreationFee describes the platform creation cost for launching a raffle.
type CreationFee struct {
	Currency        string  `json:"currency"` // "stars", "ton", "platform_coin"
	Amount          any     `json:"amount"`   // string for TON, number for Stars
	IsPaid          bool    `json:"is_paid"`
	PaymentChargeID *string `json:"payment_charge_id,omitempty"`
	CollectedAt     *int64  `json:"collected_at,omitempty"`
}

// RaffleMonetizationType distinguishes between free entry and paid ticket events.
type RaffleMonetizationType string

const (
	MonetizationFree RaffleMonetizationType = "free"
	MonetizationPaid RaffleMonetizationType = "paid"
)

// RaffleEntryCost defines paid ticket pricing.
type RaffleEntryCost struct {
	Currency           string `json:"currency"` // "stars", "ton", "in_app_currency"
	AmountPerTicket    any    `json:"amount_per_ticket"`
	MaxTicketsPerUser int    `json:"max_tickets_per_user"`
}

// RaffleStatus describes the lifecycle phase of a raffle.
type RaffleStatus string

const (
	RaffleStatusDraft     RaffleStatus = "draft"
	RaffleStatusActive    RaffleStatus = "active"
	RaffleStatusDrawing   RaffleStatus = "drawing"
	RaffleStatusCompleted RaffleStatus = "completed"
	RaffleStatusCancelled RaffleStatus = "cancelled"
)

// RaffleStats holds participant and pool aggregation numbers.
type RaffleStats struct {
	TotalParticipants   int `json:"total_participants"`
	TotalTicketsIssued  int `json:"total_tickets_issued"`
	PoolCollectedAmount any `json:"pool_collected_amount,omitempty"`
}

// RaffleObject is the aggregate entity representing a giveaway or lottery.
type RaffleObject struct {
	RaffleID           string                 `json:"raffle_id"` // UUID / ULID
	CreatorUserID      int64                  `json:"creator_user_id"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	BannerImageURL     *string                `json:"banner_image_url,omitempty"`
	MonetizationType   RaffleMonetizationType `json:"monetization_type"`
	EntryCost          *RaffleEntryCost       `json:"entry_cost,omitempty"`
	CreationFee        *CreationFee           `json:"creation_fee,omitempty"`
	StartsAt           int64                  `json:"starts_at"`
	EndsAt             int64                  `json:"ends_at"`
	DrawnAt            *int64                 `json:"drawn_at,omitempty"`
	Status             RaffleStatus           `json:"status"`
	TotalWinnersCount  int                    `json:"total_winners_count"`
	Prizes             []RafflePrize          `json:"prizes"`
	SelectionConfig    WinnerSelectionConfig  `json:"selection_config"`
	Tasks              []EntryTaskRequirement `json:"tasks,omitempty"`
	Stats              RaffleStats            `json:"stats"`
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
}

// RaffleTaskCompleted logs a verified requirement for an entry.
type RaffleTaskCompleted struct {
	TaskID      string `json:"task_id"`
	CompletedAt int64  `json:"completed_at"`
	VerifiedBy  string `json:"verified_by"` // "bot_check", "client_ack", "webhook"
}

// RafflePaymentDetails records ticket purchase information.
type RafflePaymentDetails struct {
	TxID      string `json:"tx_id"`
	Currency  string `json:"currency"`
	TotalPaid any    `json:"total_paid"`
}

// RaffleEntry models a participant's ticket bundle in a raffle.
type RaffleEntry struct {
	EntryID         string                `json:"entry_id"`
	RaffleID        string                `json:"raffle_id"`
	UserID          int64                 `json:"user_id"`
	TicketsCount    int                   `json:"tickets_count"`
	TasksCompleted  []RaffleTaskCompleted `json:"tasks_completed,omitempty"`
	IsEligible      bool                  `json:"is_eligible"`
	PaymentDetails  *RafflePaymentDetails `json:"payment_details,omitempty"`
	WonPrizeIDs     []string              `json:"won_prize_ids,omitempty"`
	IsClaimed       *bool                 `json:"is_claimed,omitempty"`
}

// IsActive returns whether the raffle is currently open for entries.
func (r *RaffleObject) IsActive() bool {
	now := time.Now().UnixMilli()
	return r.Status == RaffleStatusActive && now >= r.StartsAt && now <= r.EndsAt
}
