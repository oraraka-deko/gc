package miniapp

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserAccount_InitDataIntegration(t *testing.T) {
	// 1. Simulate parsed WebAppInitData
	isPrem := true
	initData := &WebAppInitData{
		User: &WebAppUser{
			ID:        123456789,
			FirstName: "Pavel",
			IsPremium: &isPrem,
		},
		AuthDate: time.Now().Unix(),
		Hash:     "mock_hash",
	}

	userAcc := NewUserAccountFromInitData(initData)
	if userAcc.UserID != 123456789 {
		t.Fatalf("expected user ID 123456789, got %d", userAcc.UserID)
	}
	if userAcc.AccountStatus != AccountStatusActive {
		t.Fatalf("expected active status, got %s", userAcc.AccountStatus)
	}
	if userAcc.InitUser == nil || *userAcc.InitUser.IsPremium != true {
		t.Fatalf("expected premium init user")
	}

	// 2. Extra data is nil by default
	if userAcc.Profile != nil || userAcc.Inventory != nil || userAcc.Financials != nil {
		t.Fatalf("expected optional sub-models to be nil by default")
	}

	// 3. Populate optional inventory with Stars and TON
	stars := int64(750)
	tonStr := "5000000000"
	userAcc.Inventory = &UserInventory{
		Balances: &InventoryBalances{
			Stars:   &stars,
			TONNano: &tonStr,
		},
		Collectibles: []CollectibleItem{
			{
				AssetID:    "nft_gift_001",
				ItemType:   CollectibleNFTGift,
				AcquiredAt: time.Now().UnixMilli(),
			},
		},
	}

	// 4. Marshal and unmarshal test
	data, err := json.Marshal(userAcc)
	if err != nil {
		t.Fatalf("failed to marshal UserAccount: %v", err)
	}

	var parsed MiniAppUser
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal UserAccount: %v", err)
	}

	if *parsed.Inventory.Balances.Stars != 750 {
		t.Fatalf("expected 750 stars, got %d", *parsed.Inventory.Balances.Stars)
	}
	if *parsed.Inventory.Balances.TONNano != "5000000000" {
		t.Fatalf("expected 5000000000 tonNano, got %s", *parsed.Inventory.Balances.TONNano)
	}
	if len(parsed.Inventory.Collectibles) != 1 {
		t.Fatalf("expected 1 collectible, got %d", len(parsed.Inventory.Collectibles))
	}
}

func TestEligibilityAudienceRule_Evaluation(t *testing.T) {
	isPrem := true
	ageDays := 45
	minAge := 30
	minStars := int64(100)
	userStars := int64(500)

	user := &MiniAppUser{
		UserID: 998877,
		InitUser: &WebAppUser{
			ID:        998877,
			FirstName: "Alice",
			IsPremium: &isPrem,
		},
		Profile: &TelegramAccountProfile{
			AccountAgeApproxDays: &ageDays,
		},
		Financials: &UserFinancialMetrics{
			TotalDepositedStars: &userStars,
		},
		AccountStatus: AccountStatusActive,
	}

	rule := EligibilityAudienceRule{
		MinAccountAgeDays:      &minAge,
		MinTotalDepositedStars: &minStars,
		RequireTelegramPremium: &isPrem,
		ExcludedRiskStatuses:   []string{"flagged_sybil", "suspended"},
	}

	ok, reason := rule.IsEligible(user)
	if !ok {
		t.Fatalf("expected user to be eligible, got false (reason: %s)", reason)
	}

	// Case: User is flagged as sybil
	user.AccountStatus = AccountStatusFlaggedSybil
	ok, _ = rule.IsEligible(user)
	if ok {
		t.Fatalf("expected flagged_sybil user to fail eligibility")
	}
}

func TestReferralRelationship_ProgressAndQualification(t *testing.T) {
	rel := &ReferralRelationship{
		ID:         "ref_1_2",
		ReferrerID: 1001,
		InviteeID:  2002,
		LayerDepth: 1,
		Status:     ReferralPendingRequirements,
		Requirements: []ReferralRequirementProgress{
			{
				ActionType:    ActionReachLevel,
				RequiredValue: 5,
				CurrentValue:  1,
				IsCompleted:   false,
			},
			{
				ActionType:    ActionSpendStars,
				RequiredValue: 50,
				CurrentValue:  0,
				IsCompleted:   false,
			},
		},
	}

	// Step 1: Reach level 5
	qualified := rel.UpdateProgress(ActionReachLevel, 5)
	if qualified || rel.Status != ReferralPendingRequirements {
		t.Fatalf("expected pending until all requirements met")
	}

	// Step 2: Spend 50 Stars -> both met
	qualified = rel.UpdateProgress(ActionSpendStars, 50)
	if !qualified || rel.Status != ReferralQualifiedCounted {
		t.Fatalf("expected relationship to be qualified_counted")
	}
	if rel.QualifiedAt == nil {
		t.Fatalf("expected qualified_at timestamp to be set")
	}
}

func TestDailyCaseAndPromoCode(t *testing.T) {
	// Promo code normalization
	if NormalizePromoCode("  summer2026  ") != "SUMMER2026" {
		t.Fatalf("promo code normalization failed")
	}

	// Daily case state
	now := time.Now().UnixMilli()
	caseState := &UserDailyCaseState{
		UserID:          1001,
		CaseID:          "gold_chest",
		NextAvailableAt: now - 1000, // already passed
	}
	if !caseState.CanOpen() {
		t.Fatalf("expected case to be available to open")
	}

	caseState.NextAvailableAt = now + 60000 // future
	if caseState.CanOpen() {
		t.Fatalf("expected case to be on cooldown")
	}
}

func TestTaskProgress_IncrementAndClaim(t *testing.T) {
	progress := &UserTaskProgress{
		UserID: 1001,
		TaskID: "task_pvp_5_wins",
	}

	// Increment 3 wins
	completed := progress.IncrementProgress(3, 5)
	if completed || progress.IsCompleted {
		t.Fatalf("expected incomplete with 3/5 wins")
	}

	// Increment 2 wins -> completed
	completed = progress.IncrementProgress(2, 5)
	if !completed || !progress.IsCompleted || progress.CompletedAt == nil {
		t.Fatalf("expected completed with 5/5 wins")
	}

	// Claim reward
	if !progress.Claim() {
		t.Fatalf("expected claim to succeed")
	}
	// Re-claim should fail
	if progress.Claim() {
		t.Fatalf("expected duplicate claim to fail")
	}
}

func TestRaffleObject_Lifecycle(t *testing.T) {
	now := time.Now().UnixMilli()
	raffle := &RaffleObject{
		RaffleID:         "raf_test_001",
		CreatorUserID:    555000,
		Title:            "Weekly TON Giveaway",
		MonetizationType: MonetizationFree,
		StartsAt:         now - 1000,
		EndsAt:           now + 10000,
		Status:           RaffleStatusActive,
		Prizes: []RafflePrize{
			{
				PrizeID:                 "prz_1",
				Rank:                    1,
				Title:                   "10 TON",
				PrizeType:               PrizeTON,
				Value:                   "10000000000",
				WinnerCountForThisPrize: 1,
			},
		},
		SelectionConfig: WinnerSelectionConfig{
			Preset:                   PresetRandomProvablyFair,
			AllowMultipleWinsPerUser: false,
		},
	}

	if !raffle.IsActive() {
		t.Fatalf("expected raffle to be active")
	}

	data, err := json.Marshal(raffle)
	if err != nil {
		t.Fatalf("failed to marshal raffle: %v", err)
	}

	var decoded RaffleObject
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal raffle: %v", err)
	}
	if decoded.Prizes[0].Title != "10 TON" {
		t.Fatalf("unexpected prize title: %s", decoded.Prizes[0].Title)
	}
}
