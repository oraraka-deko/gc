package miniapp

import (
	"time"
)

// AccountStatus represents the operational status of a user's account.
type AccountStatus string

const (
	AccountStatusActive       AccountStatus = "active"
	AccountStatusSuspended    AccountStatus = "suspended"
	AccountStatusFlaggedSybil AccountStatus = "flagged_sybil"
	AccountStatusShadowBanned AccountStatus = "shadow_banned"
)

// PeerInteractionType defines the type of interaction between two users.
type PeerInteractionType string

const (
	InteractionPvPMatch      PeerInteractionType = "pvp_match"
	InteractionAssetTransfer PeerInteractionType = "asset_transfer"
	InteractionCoOp          PeerInteractionType = "co_op"
)

// CollectibleItemType defines standard collectible categories.
type CollectibleItemType string

const (
	CollectibleNFTGift   CollectibleItemType = "nft_gift"
	CollectibleSkin      CollectibleItemType = "skin"
	CollectibleBadge     CollectibleItemType = "badge"
	CollectibleEquipment CollectibleItemType = "equipment"
)

// PublicGiftItem represents a gift displayed on the user's public profile.
type PublicGiftItem struct {
	GiftID     string `json:"gift_id"`
	SenderID   *int64 `json:"sender_id,omitempty"`
	Title      string `json:"title"`
	ReceivedAt int64  `json:"received_at"`
}

// TelegramAccountProfile captures enriched profile attributes & advertising signals.
type TelegramAccountProfile struct {
	AccountAgeApproxDays *int             `json:"account_age_approx_days,omitempty"`
	Bio                  *string          `json:"bio,omitempty"`
	CurrentMusicTrack    *string          `json:"current_music_track,omitempty"`
	StarsBalanceEstimate *int             `json:"stars_balance_estimate,omitempty"`
	PublicGiftsCount     *int             `json:"public_gifts_count,omitempty"`
	PublicGifts          []PublicGiftItem `json:"public_gifts,omitempty"`
	StoriesCount         *int             `json:"stories_count,omitempty"`
	ChannelPostsCount    *int             `json:"channel_posts_count,omitempty"`
	AdAudienceTags       []string         `json:"ad_audience_tags,omitempty"`
	LastScrapedAt        *int64           `json:"last_scraped_at,omitempty"`
}

// PeerInteraction tracks mutual interactive sessions for social graph and anti-collusion scoring.
type PeerInteraction struct {
	PeerUserID       int64               `json:"peer_user_id"`
	InteractionType  PeerInteractionType `json:"interaction_type"`
	Frequency        int                 `json:"frequency"`
	LastInteractedAt int64               `json:"last_interacted_at"`
	RiskScore        float64             `json:"risk_score"` // High frequency + mutual IP = circular collusive farming flag
}

// UserSocialGraph tracks multi-layer connectivity, referral ancestry, and interaction history.
type UserSocialGraph struct {
	ReferrerID           *int64            `json:"referrer_id,omitempty"`
	ReferralPath         []int64           `json:"referral_path,omitempty"` // [rootId, L3Id, L2Id, L1Id]
	DirectReferralsCount int               `json:"direct_referrals_count"`
	NetworkSizeByDepth   map[int]int       `json:"network_size_by_depth,omitempty"` // { 1: 15, 2: 84, 3: 310 }
	PeerInteractions     []PeerInteraction `json:"peer_interactions,omitempty"`
}

// DeviceIdentity contains hardware & network fingerprints for sybil resistance.
type DeviceIdentity struct {
	FingerprintHash         string  `json:"fingerprint_hash,omitempty"` // Canonical hash of WebGL, Canvas, AudioContext
	IPAddress               string  `json:"ip_address,omitempty"`
	UserAgent               string  `json:"user_agent,omitempty"`
	ScreenResolution        string  `json:"screen_resolution,omitempty"` // e.g. "390x844@3x"
	WebGLRenderer           string  `json:"webgl_renderer,omitempty"`    // e.g. "Apple GPU" / "Mali-G77"
	IsVPNOrProxy            *bool   `json:"is_vpn_or_proxy,omitempty"`
	AssociatedAccountsCount int     `json:"associated_accounts_count,omitempty"`
	FirstSeenAt             int64   `json:"first_seen_at,omitempty"`
	LastSeenAt              int64   `json:"last_seen_at,omitempty"`
}

// CollectibleItem represents non-fungible or discrete items in user inventory.
type CollectibleItem struct {
	AssetID    string              `json:"asset_id"`
	ItemType   CollectibleItemType `json:"item_type"`
	Metadata   map[string]any      `json:"metadata,omitempty"`
	AcquiredAt int64               `json:"acquired_at"`
}

// InventoryBalances holds key-value fungible token balances with extensible custom token support.
type InventoryBalances struct {
	TONNano      *string        `json:"ton_nano,omitempty"` // Big numbers stored as string to prevent precision loss
	Stars        *int64         `json:"stars,omitempty"`
	Coins        *int64         `json:"coins,omitempty"`
	Gems         *int64         `json:"gems,omitempty"`
	CustomTokens map[string]any `json:"custom_tokens,omitempty"`
}

// UserInventory represents the user's fungible and non-fungible assets.
type UserInventory struct {
	Balances     *InventoryBalances `json:"balances,omitempty"`
	Collectibles []CollectibleItem  `json:"collectibles,omitempty"`
}

// UserFinancialMetrics aggregates user transactional, wagering, and revenue statistics.
type UserFinancialMetrics struct {
	TotalDepositedStars   *int64   `json:"total_deposited_stars,omitempty"`
	TotalWithdrawnStars   *int64   `json:"total_withdrawn_stars,omitempty"`
	TotalTONDepositedNano *string  `json:"total_ton_deposited_nano,omitempty"`
	TotalTONWithdrawnNano *string  `json:"total_ton_withdrawn_nano,omitempty"`
	NetSpendVolume        *float64 `json:"net_spend_volume,omitempty"`
	OrderCount            *int     `json:"order_count,omitempty"`
	HighestWinAmount      *float64 `json:"highest_win_amount,omitempty"`
	LowestWinAmount       *float64 `json:"lowest_win_amount,omitempty"`
	TotalBonusClaimed     *float64 `json:"total_bonus_claimed,omitempty"`
	AdsWatchedLifetime    *int     `json:"ads_watched_lifetime,omitempty"`
	AdsWatchedToday       *int     `json:"ads_watched_today,omitempty"`
}

// UserActivityHeatmap tracks engagement times, module preferences, and daily streaks.
type UserActivityHeatmap struct {
	AppOpenCountLifetime    *int           `json:"app_open_count_lifetime,omitempty"`
	FavoriteModules         map[string]int `json:"favorite_modules,omitempty"` // e.g. { "pvp_arena": 140, "lootbox": 28 }
	ActiveHoursDistribution []int          `json:"active_hours_distribution,omitempty"` // 24 slots (0..23)
	DaysActiveStreak        *int           `json:"days_active_streak,omitempty"`
	LastSessionStart        *int64         `json:"last_session_start,omitempty"`
	LastSessionEnd          *int64         `json:"last_session_end,omitempty"`
}

// MiniAppUser (UserAccount) is the root aggregate model for an enriched Telegram Mini App user.
// Almost all extra sub-models are pointers (nullable) so apps only serialize/populate what they use.
type MiniAppUser struct {
	UserID        int64                   `json:"user_id"` // Telegram User ID (Primary Key)
	InitUser      *WebAppUser             `json:"init_user,omitempty"`
	Profile       *TelegramAccountProfile `json:"profile,omitempty"`
	Social        *UserSocialGraph        `json:"social,omitempty"`
	Device        *DeviceIdentity         `json:"device,omitempty"`
	Inventory     *UserInventory          `json:"inventory,omitempty"`
	Financials    *UserFinancialMetrics   `json:"financials,omitempty"`
	Activity      *UserActivityHeatmap    `json:"activity,omitempty"`
	AccountStatus AccountStatus           `json:"account_status"`
	CreatedAt     int64                   `json:"created_at"`
	UpdatedAt     int64                   `json:"updated_at"`
}

// UserAccount is an alias to MiniAppUser for convenient naming.
type UserAccount = MiniAppUser

// NewUserAccount creates a default MiniAppUser initialized from a Telegram WebAppUser.
func NewUserAccount(initUser *WebAppUser) *MiniAppUser {
	now := time.Now().UnixMilli()
	var userID int64
	if initUser != nil {
		userID = initUser.ID
	}

	return &MiniAppUser{
		UserID:        userID,
		InitUser:      initUser,
		AccountStatus: AccountStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// NewUserAccountFromInitData creates a default MiniAppUser from a parsed WebAppInitData.
func NewUserAccountFromInitData(initData *WebAppInitData) *MiniAppUser {
	if initData == nil {
		return NewUserAccount(nil)
	}
	return NewUserAccount(initData.User)
}

// SetStatus updates account status and timestamp.
func (u *MiniAppUser) SetStatus(status AccountStatus) {
	u.AccountStatus = status
	u.UpdatedAt = time.Now().UnixMilli()
}

// RecordSession updates activity streak and last session timestamps.
func (u *MiniAppUser) RecordSession(sessionStart int64) {
	if u.Activity == nil {
		u.Activity = &UserActivityHeatmap{
			FavoriteModules:         make(map[string]int),
			ActiveHoursDistribution: make([]int, 24),
		}
	}

	now := time.Now().UTC()
	hour := now.Hour()
	if len(u.Activity.ActiveHoursDistribution) == 24 {
		u.Activity.ActiveHoursDistribution[hour]++
	}

	count := 1
	if u.Activity.AppOpenCountLifetime != nil {
		count = *u.Activity.AppOpenCountLifetime + 1
	}
	u.Activity.AppOpenCountLifetime = &count
	u.Activity.LastSessionStart = &sessionStart
	u.UpdatedAt = time.Now().UnixMilli()
}
