package miniapp

import (
	"fmt"
	"strings"
	"time"
)

// EligibilityAudienceRule configures targeting restrictions for promos, chests, and tasks.
type EligibilityAudienceRule struct {
	MinAccountAgeDays      *int     `json:"min_account_age_days,omitempty"`
	MinTotalDepositedStars *int64   `json:"min_total_deposited_stars,omitempty"`
	MinAppOpenCount        *int     `json:"min_app_open_count,omitempty"`
	RequireTelegramPremium *bool    `json:"require_telegram_premium,omitempty"`
	ExcludedRiskStatuses   []string `json:"excluded_risk_statuses,omitempty"` // e.g. ['flagged_sybil', 'suspended']
	AllowedCountries       []string `json:"allowed_countries,omitempty"`
}

// IsEligible evaluates if a MiniAppUser satisfies the audience rule.
func (r *EligibilityAudienceRule) IsEligible(user *MiniAppUser) (bool, string) {
	if user == nil {
		return false, "user is nil"
	}

	// 1. Excluded Statuses
	for _, excluded := range r.ExcludedRiskStatuses {
		if string(user.AccountStatus) == excluded {
			return false, fmt.Sprintf("account status %s is excluded", user.AccountStatus)
		}
	}

	// 2. Telegram Premium Check
	if r.RequireTelegramPremium != nil && *r.RequireTelegramPremium {
		if user.InitUser == nil || user.InitUser.IsPremium == nil || !*user.InitUser.IsPremium {
			return false, "requires telegram premium"
		}
	}

	// 3. Minimum Account Age
	if r.MinAccountAgeDays != nil && *r.MinAccountAgeDays > 0 {
		if user.Profile == nil || user.Profile.AccountAgeApproxDays == nil || *user.Profile.AccountAgeApproxDays < *r.MinAccountAgeDays {
			return false, fmt.Sprintf("account age below %d days", *r.MinAccountAgeDays)
		}
	}

	// 4. Minimum Deposited Stars
	if r.MinTotalDepositedStars != nil && *r.MinTotalDepositedStars > 0 {
		if user.Financials == nil || user.Financials.TotalDepositedStars == nil || *user.Financials.TotalDepositedStars < *r.MinTotalDepositedStars {
			return false, fmt.Sprintf("requires minimum %d deposited stars", *r.MinTotalDepositedStars)
		}
	}

	// 5. Minimum App Opens
	if r.MinAppOpenCount != nil && *r.MinAppOpenCount > 0 {
		if user.Activity == nil || user.Activity.AppOpenCountLifetime == nil || *user.Activity.AppOpenCountLifetime < *r.MinAppOpenCount {
			return false, fmt.Sprintf("requires minimum %d app opens", *r.MinAppOpenCount)
		}
	}

	return true, ""
}

// PromoCodeRewardItem describes a non-currency item awarded via promo codes.
type PromoCodeRewardItem struct {
	AssetID  string  `json:"asset_id"`
	ItemType string  `json:"item_type"`
	Quantity float64 `json:"quantity"`
}

// PromoCodeRewardPayload defines the granted currencies or items upon promo redemption.
type PromoCodeRewardPayload struct {
	Currencies                     map[string]float64    `json:"currencies,omitempty"`
	Items                          []PromoCodeRewardItem `json:"items,omitempty"`
	IsDynamicMultiplierBasedOnTier *bool                 `json:"is_dynamic_multiplier_based_on_tier,omitempty"`
}

// PromoCodeConfig defines the configuration and rules of a promotional campaign.
type PromoCodeConfig struct {
	Code                string                  `json:"code"` // Normalized uppercase (e.g. "SUMMER2026")
	Description         string                  `json:"description"`
	MaxTotalRedemptions int                     `json:"max_total_redemptions"`
	CurrentRedemptions  int                     `json:"current_redemptions"`
	MaxPerUser          int                     `json:"max_per_user"`
	StartsAt            int64                   `json:"starts_at"`
	ExpiresAt           int64                   `json:"expires_at"`
	Eligibility         EligibilityAudienceRule `json:"eligibility"`
	RewardPayload       PromoCodeRewardPayload  `json:"reward_payload"`
}

// PromoCodeRedemptionRecord logs an individual user's promo code redemption.
type PromoCodeRedemptionRecord struct {
	ID                    string         `json:"id"`
	Code                  string         `json:"code"`
	UserID                int64          `json:"user_id"`
	RedeemedAt            int64          `json:"redeemed_at"`
	GrantedPayload        map[string]any `json:"granted_payload"`
	IPAddress             string         `json:"ip_address,omitempty"`
	DeviceFingerprintHash string         `json:"device_fingerprint_hash,omitempty"`
}

// DailyCaseDropItem specifies an item slot and weighted probability in a daily chest drop table.
type DailyCaseDropItem struct {
	ItemID    string  `json:"item_id"`
	ItemType  string  `json:"item_type"` // "currency", "collectible", "nft_gift"
	AmountMin float64 `json:"amount_min"`
	AmountMax float64 `json:"amount_max"`
	Weight    int     `json:"weight"` // Relative drop rate probability
}

// DailyCaseConfig defines daily reward chest presets.
type DailyCaseConfig struct {
	CaseID          string                  `json:"case_id"` // e.g. "vip_daily_chest"
	Title           string                  `json:"title"`
	TierRequired    string                  `json:"tier_required"` // "free", "silver", "gold", "whale"
	CooldownSeconds int                     `json:"cooldown_seconds"`
	Eligibility     EligibilityAudienceRule `json:"eligibility"`
	DropTable       []DailyCaseDropItem     `json:"drop_table"`
}

// UserDailyCaseState tracks chest availability, streaks, and cooldowns for a user.
type UserDailyCaseState struct {
	UserID          int64  `json:"user_id"`
	CaseID          string `json:"case_id"`
	LastOpenedAt    int64  `json:"last_opened_at"`
	CurrentStreak   int    `json:"current_streak"`
	NextAvailableAt int64  `json:"next_available_at"`
}

// CanOpen evaluates whether the cooldown has elapsed.
func (s *UserDailyCaseState) CanOpen() bool {
	return time.Now().UnixMilli() >= s.NextAvailableAt
}

// NormalizePromoCode formats codes to trimmed uppercase.
func NormalizePromoCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
