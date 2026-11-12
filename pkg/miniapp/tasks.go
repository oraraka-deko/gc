package miniapp

import (
	"time"
)

// TaskType describes the verification mechanism for a task.
type TaskType string

const (
	TaskTypeInstantTelegramChannel TaskType = "instant_telegram_channel" // Checked via Telegram Bot getChatMember
	TaskTypeCounterAccumulator      TaskType = "counter_accumulator"      // Win X games, invite Y friends
	TaskTypeExternalCallbackVerify TaskType = "external_callback_verify" // Ad networks, payment webhooks
	TaskTypeStoryShare             TaskType = "story_share"              // Checked via Telegram story share link
)

// TaskCategory categorizes user tasks.
type TaskCategory string

const (
	TaskCategoryDaily       TaskCategory = "daily"
	TaskCategoryOnboarding  TaskCategory = "onboarding"
	TaskCategoryEvent       TaskCategory = "event"
	TaskCategoryAchievement TaskCategory = "achievement"
)

// ActiveWindowConfig defines the lifespan and associated event for timed tasks.
type ActiveWindowConfig struct {
	StartsAt          int64   `json:"starts_at"`
	EndsAt            int64   `json:"ends_at"`
	AssociatedEventID *string `json:"associated_event_id,omitempty"` // e.g. "halloween_2026"
}

// TaskVerificationConfig specifies criteria and callback protocols.
type TaskVerificationConfig struct {
	TargetMetric            *string `json:"target_metric,omitempty"`       // e.g. "pvp_wins", "ads_watched_count"
	RequiredTargetCount     int     `json:"required_target_count"`         // e.g. 5
	RequiresWebhookCallback bool    `json:"requires_webhook_callback"`
	CallbackProvider        *string `json:"callback_provider,omitempty"`   // "telegram_bot", "ad_network", "payment_provider"
	CallbackSecretKey       *string `json:"callback_secret_key,omitempty"`
}

// TaskCollectibleReward describes an awarded non-currency asset.
type TaskCollectibleReward struct {
	AssetID  string `json:"asset_id"`
	Quantity int    `json:"quantity"`
}

// TaskRewards specifies prizes awarded upon task completion.
type TaskRewards struct {
	Currencies         map[string]float64      `json:"currencies,omitempty"`
	Collectibles       []TaskCollectibleReward `json:"collectibles,omitempty"`
	AchievementBadgeID *string                 `json:"achievement_badge_id,omitempty"`
}

// TaskDefinition defines a quest, challenge, or achievement.
type TaskDefinition struct {
	TaskID             string                  `json:"task_id"` // e.g. "task_ad_watch_daily", "task_pvp_5_wins"
	Category           TaskCategory            `json:"category"`
	Title              string                  `json:"title"`
	Description        string                  `json:"description"`
	IconURL            *string                 `json:"icon_url,omitempty"`
	ActiveWindow       *ActiveWindowConfig     `json:"active_window,omitempty"`
	TargetingRule      EligibilityAudienceRule `json:"targeting_rule"`
	TaskType           TaskType                `json:"task_type"`
	VerificationConfig TaskVerificationConfig  `json:"verification_config"`
	Rewards            TaskRewards             `json:"rewards"`
}

// TaskVerificationContext holds proof metadata for webhook-verified completions.
type TaskVerificationContext struct {
	CallbackEventID *string `json:"callback_event_id,omitempty"` // External tx/session ID
	VerifiedAt      *int64  `json:"verified_at,omitempty"`
	Signature       *string `json:"signature,omitempty"`
}

// UserTaskProgress tracks individual player milestones and claims.
type UserTaskProgress struct {
	UserID              int64                    `json:"user_id"`
	TaskID              string                   `json:"task_id"`
	CurrentProgress     int                      `json:"current_progress"` // e.g. 3 out of 5 wins
	IsCompleted         bool                     `json:"is_completed"`
	IsRewardClaimed     bool                     `json:"is_reward_claimed"`
	LastProgressUpdate  int64                    `json:"last_progress_update"`
	CompletedAt         *int64                   `json:"completed_at,omitempty"`
	ClaimedAt           *int64                   `json:"claimed_at,omitempty"`
	VerificationContext *TaskVerificationContext `json:"verification_context,omitempty"`
}

// IncrementProgress increases progress and checks for completion.
func (p *UserTaskProgress) IncrementProgress(amount int, requiredCount int) bool {
	if p.IsCompleted {
		return true
	}
	p.CurrentProgress += amount
	p.LastProgressUpdate = time.Now().UnixMilli()

	if p.CurrentProgress >= requiredCount {
		now := time.Now().UnixMilli()
		p.IsCompleted = true
		p.CompletedAt = &now
		return true
	}
	return false
}

// Claim marks the reward as collected.
func (p *UserTaskProgress) Claim() bool {
	if !p.IsCompleted || p.IsRewardClaimed {
		return false
	}
	now := time.Now().UnixMilli()
	p.IsRewardClaimed = true
	p.ClaimedAt = &now
	return true
}
