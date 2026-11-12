package miniapp

import (
	"time"
)

// ReferralAttributionStatus defines the qualification state of an invitation.
type ReferralAttributionStatus string

const (
	ReferralPendingRequirements ReferralAttributionStatus = "pending_requirements" // Invitee joined, but hasn't met criteria
	ReferralQualifiedCounted    ReferralAttributionStatus = "qualified_counted"    // Criteria met, reward credited
	ReferralFlaggedSybil        ReferralAttributionStatus = "flagged_sybil"        // Collusion/same IP/same WebGL detected
	ReferralDisqualified        ReferralAttributionStatus = "disqualified"         // Invitee deleted/banned before criteria
)

// ReferralRequirementAction describes measurable milestones needed to qualify.
type ReferralRequirementAction string

const (
	ActionReachLevel              ReferralRequirementAction = "reach_level"
	ActionSpendStars              ReferralRequirementAction = "spend_stars"
	ActionVerifyTelegramPremium   ReferralRequirementAction = "verify_telegram_premium"
	ActionPlayPvPMatches          ReferralRequirementAction = "play_pvp_matches"
)

// ReferralRequirementProgress tracks a single qualification milestone.
type ReferralRequirementProgress struct {
	ActionType    ReferralRequirementAction `json:"action_type"`
	RequiredValue float64                   `json:"required_value"`
	CurrentValue  float64                   `json:"current_value"`
	IsCompleted   bool                      `json:"is_completed"`
}

// ReferralRewardConfig specifies the compensation for successful conversion.
type ReferralRewardConfig struct {
	InviterCurrency     string  `json:"inviter_currency"` // e.g. "coins", "stars", "lootbox_key"
	InviterAmount       float64 `json:"inviter_amount"`
	InviteeBonusGranted bool    `json:"invitee_bonus_granted"` // Did the invited user get their welcome incentive?
	PayoutTxID          *string `json:"payout_tx_id,omitempty"` // Ledger transaction hash
}

// ReferralRelationship models an edge in the viral referral graph.
type ReferralRelationship struct {
	ID           string                        `json:"id"`           // e.g. "ref_100293_849201"
	ReferrerID   int64                         `json:"referrer_id"`  // Inviter user ID
	InviteeID    int64                         `json:"invitee_id"`   // Invited user ID
	LayerDepth   int                           `json:"layer_depth"`  // 1 = Direct, 2 = Indirect, 3+
	Status       ReferralAttributionStatus     `json:"status"`
	RiskScore    float64                       `json:"risk_score"` // 0 to 100
	RiskFlags    []string                      `json:"risk_flags,omitempty"`
	Requirements []ReferralRequirementProgress `json:"requirements,omitempty"`
	RewardConfig ReferralRewardConfig          `json:"reward_config"`
	CreatedAt    int64                         `json:"created_at"`
	QualifiedAt  *int64                        `json:"qualified_at,omitempty"`
}

// AreRequirementsMet checks whether all qualification milestones have been completed.
func (r *ReferralRelationship) AreRequirementsMet() bool {
	if len(r.Requirements) == 0 {
		return true
	}
	for _, req := range r.Requirements {
		if !req.IsCompleted {
			return false
		}
	}
	return true
}

// UpdateProgress updates the progress of a specific requirement action.
func (r *ReferralRelationship) UpdateProgress(action ReferralRequirementAction, val float64) bool {
	allMet := true
	for i := range r.Requirements {
		req := &r.Requirements[i]
		if req.ActionType == action {
			req.CurrentValue = val
			if req.CurrentValue >= req.RequiredValue {
				req.IsCompleted = true
			}
		}
		if !req.IsCompleted {
			allMet = false
		}
	}

	if allMet && r.Status == ReferralPendingRequirements {
		now := time.Now().UnixMilli()
		r.Status = ReferralQualifiedCounted
		r.QualifiedAt = &now
		return true
	}
	return false
}
