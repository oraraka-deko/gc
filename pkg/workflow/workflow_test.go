package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gc/pkg/ledger"
	"gc/pkg/storage"
)

func setupTestWorkflow(t *testing.T) (*Engine, *ledger.Ledger, func()) {
	tmpDir, err := os.MkdirTemp("", "workflow_test_*")
	if err != nil {
		t.Fatal(err)
	}

	store, err := storage.New(storage.Config{
		PebblePath:     filepath.Join(tmpDir, "pebble"),
		FastCacheBytes: 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	led := ledger.New(store)
	eng := NewEngine(store, led, nil)

	cleanup := func() {
		_ = store.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return eng, led, cleanup
}

func TestWorkflow_SequentialCustomActions(t *testing.T) {
	eng, _, cleanup := setupTestWorkflow(t)
	defer cleanup()

	ctx := context.Background()

	// Register custom actions
	eng.RegisterAction("step_one", func(ctx context.Context, exec *StepExecutionContext) error {
		exec.Variables["counter"] = 10
		return nil
	})
	eng.RegisterAction("step_two", func(ctx context.Context, exec *StepExecutionContext) error {
		c := exec.Variables["counter"].(int)
		exec.Variables["counter"] = c + 5
		return nil
	})

	wf := &WorkflowDef{
		ID:   "wf_counter",
		Name: "Counter Pipeline",
		Steps: []StepDef{
			{
				ID:   "s1",
				Type: StepActionCustom,
				Params: map[string]any{
					"action": "step_one",
				},
			},
			{
				ID:   "s2",
				Type: StepActionCustom,
				Params: map[string]any{
					"action": "step_two",
				},
			},
		},
	}

	if err := eng.RegisterWorkflow(wf); err != nil {
		t.Fatalf("register workflow failed: %v", err)
	}

	inst, err := eng.StartInstance(ctx, "wf_counter", nil)
	if err != nil {
		t.Fatalf("start instance failed: %v", err)
	}

	if inst.State != StateCompleted {
		t.Fatalf("expected StateCompleted, got %s", inst.State)
	}

	if inst.Variables["counter"] != 15 {
		t.Fatalf("expected counter = 15, got %v", inst.Variables["counter"])
	}
}

func TestWorkflow_ConditionBranching(t *testing.T) {
	eng, _, cleanup := setupTestWorkflow(t)
	defer cleanup()

	ctx := context.Background()

	eng.RegisterAction("mark_vip", func(ctx context.Context, exec *StepExecutionContext) error {
		exec.Variables["tier"] = "VIP"
		return nil
	})
	eng.RegisterAction("mark_regular", func(ctx context.Context, exec *StepExecutionContext) error {
		exec.Variables["tier"] = "REGULAR"
		return nil
	})

	wf := &WorkflowDef{
		ID:   "wf_tier_eval",
		Name: "Tier Evaluation",
		Steps: []StepDef{
			{
				ID:          "cond_vip",
				Type:        StepActionCondition,
				Condition:   "amount >= 1000",
				NextStep:    "step_vip",
				OnFalseStep: "step_regular",
			},
			{
				ID:   "step_vip",
				Type: StepActionCustom,
				Params: map[string]any{
					"action": "mark_vip",
				},
				NextStep: "end_step",
			},
			{
				ID:   "step_regular",
				Type: StepActionCustom,
				Params: map[string]any{
					"action": "mark_regular",
				},
				NextStep: "end_step",
			},
			{
				ID:   "end_step",
				Type: StepActionCondition,
			},
		},
	}

	if err := eng.RegisterWorkflow(wf); err != nil {
		t.Fatalf("register workflow failed: %v", err)
	}

	// Case 1: amount = 1500 (VIP)
	instVIP, err := eng.StartInstance(ctx, "wf_tier_eval", map[string]any{"amount": 1500})
	if err != nil {
		t.Fatalf("failed VIP instance: %v", err)
	}
	if instVIP.Variables["tier"] != "VIP" {
		t.Fatalf("expected VIP, got %v", instVIP.Variables["tier"])
	}

	// Case 2: amount = 500 (Regular)
	instReg, err := eng.StartInstance(ctx, "wf_tier_eval", map[string]any{"amount": 500})
	if err != nil {
		t.Fatalf("failed Regular instance: %v", err)
	}
	if instReg.Variables["tier"] != "REGULAR" {
		t.Fatalf("expected REGULAR, got %v", instReg.Variables["tier"])
	}
}

func TestWorkflow_TicklessSuspensionAndResume(t *testing.T) {
	eng, _, cleanup := setupTestWorkflow(t)
	defer cleanup()

	ctx := context.Background()

	eng.RegisterAction("action_before", func(ctx context.Context, exec *StepExecutionContext) error {
		exec.Variables["stage"] = "pre_wait"
		return nil
	})
	eng.RegisterAction("action_after", func(ctx context.Context, exec *StepExecutionContext) error {
		eventVal := exec.Variables["last_event"].(string)
		exec.Variables["stage"] = "approved:" + eventVal
		return nil
	})

	wf := &WorkflowDef{
		ID:   "wf_approval",
		Name: "Approval Workflow",
		Steps: []StepDef{
			{
				ID:   "s_before",
				Type: StepActionCustom,
				Params: map[string]any{
					"action": "action_before",
				},
			},
			{
				ID:   "wait_for_user",
				Type: StepActionWaitEvent,
				Params: map[string]any{
					"event": "user_approved",
				},
			},
			{
				ID:   "s_after",
				Type: StepActionCustom,
				Params: map[string]any{
					"action": "action_after",
				},
			},
		},
	}

	if err := eng.RegisterWorkflow(wf); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// 1. Start instance: should suspend at wait_for_user
	inst, err := eng.StartInstance(ctx, "wf_approval", nil)
	if err != nil {
		t.Fatalf("start instance failed: %v", err)
	}

	if inst.State != StateSuspended {
		t.Fatalf("expected StateSuspended, got %s", inst.State)
	}
	if inst.AwaitingEvent != "user_approved" {
		t.Fatalf("expected waiting for user_approved, got %s", inst.AwaitingEvent)
	}
	if inst.Variables["stage"] != "pre_wait" {
		t.Fatalf("expected stage pre_wait, got %v", inst.Variables["stage"])
	}

	// Verify durable storage holds the suspended instance
	saved, err := eng.GetInstance(ctx, inst.InstanceID)
	if err != nil || saved.State != StateSuspended {
		t.Fatalf("storage did not persist suspended instance: %v", err)
	}

	// 2. Simulate external event arrival and resume
	resumed, err := eng.ResumeInstance(ctx, inst.InstanceID, "user_approved", "approved_by_admin")
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}

	if resumed.State != StateCompleted {
		t.Fatalf("expected StateCompleted, got %s", resumed.State)
	}
	if resumed.Variables["stage"] != "approved:approved_by_admin" {
		t.Fatalf("expected stage approved:approved_by_admin, got %v", resumed.Variables["stage"])
	}
}

func TestWorkflow_LedgerIntegration(t *testing.T) {
	eng, led, cleanup := setupTestWorkflow(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(6001)

	// Deposit 100 USDT into user account
	depositAmount := ledger.USDTToMicro(100.0)
	_, err := led.Deposit(ctx, userID, ledger.AssetUSDT, depositAmount, "dep:wf:6001", "Initial deposit")
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	wf := &WorkflowDef{
		ID:   "wf_payment",
		Name: "Payment Execution",
		Steps: []StepDef{
			{
				ID:   "reserve_funds",
				Type: StepActionLedgerReserve,
				Params: map[string]any{
					"from_user":  float64(userID), // JSON number unmarshaled as float64
					"to_account": ledger.SystemReserveAccount(ledger.AssetUSDT),
					"asset":      ledger.AssetUSDT,
					"amount":     float64(ledger.USDTToMicro(30.0)),
				},
			},
			{
				ID:   "settle_funds",
				Type: StepActionLedgerSettle,
			},
		},
	}

	if err := eng.RegisterWorkflow(wf); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	inst, err := eng.StartInstance(ctx, "wf_payment", nil)
	if err != nil {
		t.Fatalf("start workflow failed: %v", err)
	}

	if inst.State != StateCompleted {
		t.Fatalf("expected StateCompleted, got %s", inst.State)
	}

	// Verify user balance is now 70 USDT
	acc, _ := led.GetAccount(ctx, ledger.UserMainAccount(userID, ledger.AssetUSDT))
	expectedRemaining := ledger.USDTToMicro(70.0)
	if acc.Available != expectedRemaining {
		t.Fatalf("expected 70 USDT available, got %d", acc.Available)
	}
}

func TestWorkflow_OuterCycleLoop(t *testing.T) {
	eng, _, cleanup := setupTestWorkflow(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runCount := 0
	eng.RegisterAction("inc_count", func(ctx context.Context, exec *StepExecutionContext) error {
		runCount++
		return nil
	})

	wf := &WorkflowDef{
		ID:   "wf_cyclic",
		Name: "Cyclic Job",
		Steps: []StepDef{
			{
				ID:   "s1",
				Type: StepActionCustom,
				Params: map[string]any{
					"action": "inc_count",
				},
			},
		},
		CycleConfig: CycleConfig{
			Interval:      50 * time.Millisecond,
			MaxIterations: 3,
		},
	}

	if err := eng.RegisterWorkflow(wf); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	done := eng.StartCycleLoop(ctx, "wf_cyclic", nil)

	select {
	case <-done:
		// Succeeded
	case <-time.After(2 * time.Second):
		t.Fatalf("cycle loop timed out")
	}

	if runCount != 3 {
		t.Fatalf("expected exactly 3 cycle runs, got %d", runCount)
	}
}
