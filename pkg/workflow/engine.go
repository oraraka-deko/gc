package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"

	"gc/pkg/ledger"
	"gc/pkg/storage"
	"gc/pkg/telegram"
)

// ActionHandler defines the signature for custom step logic.
type ActionHandler func(ctx context.Context, exec *StepExecutionContext) error

// Engine executes tickless workflows, persists state to Pebble, and evaluates expressions.
type Engine struct {
	store         *storage.Storage
	ledger        *ledger.Ledger
	bot           *telegram.Client
	workflows     map[string]*WorkflowDef
	customActions map[string]ActionHandler
	compiledExprs map[string]*vm.Program
	mu            sync.RWMutex
}

// NewEngine creates a new workflow execution engine.
func NewEngine(store *storage.Storage, led *ledger.Ledger, bot *telegram.Client) *Engine {
	return &Engine{
		store:         store,
		ledger:        led,
		bot:           bot,
		workflows:     make(map[string]*WorkflowDef),
		customActions: make(map[string]ActionHandler),
		compiledExprs: make(map[string]*vm.Program),
	}
}

// RegisterWorkflow adds a workflow definition to the engine.
func (e *Engine) RegisterWorkflow(def *WorkflowDef) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if def.ID == "" {
		return fmt.Errorf("workflow ID cannot be empty")
	}

	// Precompile any step conditions
	for _, step := range def.Steps {
		if step.Condition != "" {
			prog, err := expr.Compile(step.Condition)
			if err != nil {
				return fmt.Errorf("failed to compile condition %q for step %s: %w", step.Condition, step.ID, err)
			}
			e.compiledExprs[step.Condition] = prog
		}
	}

	e.workflows[def.ID] = def
	return nil
}

// RegisterAction registers a custom step handler.
func (e *Engine) RegisterAction(name string, handler ActionHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.customActions[name] = handler
}

// StartInstance creates and immediately begins executing a workflow instance.
func (e *Engine) StartInstance(ctx context.Context, workflowID string, vars map[string]any) (*WorkflowInstance, error) {
	e.mu.RLock()
	def, ok := e.workflows[workflowID]
	e.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrWorkflowNotFound, workflowID)
	}

	if vars == nil {
		vars = make(map[string]any)
	}

	now := time.Now().UnixNano()
	instID := fmt.Sprintf("wf_inst_%s_%d", workflowID, now)

	inst := &WorkflowInstance{
		InstanceID:  instID,
		WorkflowID:  workflowID,
		State:       StateRunning,
		CurrentStep: 0,
		Variables:   vars,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := e.saveInstance(inst); err != nil {
		return nil, err
	}

	// Run inner step evaluation loop
	if err := e.executeSteps(ctx, def, inst); err != nil {
		inst.State = StateFailed
		inst.Error = err.Error()
		_ = e.saveInstance(inst)
		return inst, err
	}

	return inst, nil
}

// ResumeInstance wakes up a suspended workflow instance when its awaited event occurs.
func (e *Engine) ResumeInstance(ctx context.Context, instanceID string, eventName string, eventData any) (*WorkflowInstance, error) {
	inst, err := e.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	if inst.State != StateSuspended {
		return nil, fmt.Errorf("%w: current state is %s", ErrInstanceNotPaused, inst.State)
	}

	if inst.AwaitingEvent != "" && inst.AwaitingEvent != eventName {
		return nil, fmt.Errorf("instance waiting for %s, got %s", inst.AwaitingEvent, eventName)
	}

	e.mu.RLock()
	def, ok := e.workflows[inst.WorkflowID]
	e.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrWorkflowNotFound, inst.WorkflowID)
	}

	// Inject event data into variables
	if inst.Variables == nil {
		inst.Variables = make(map[string]any)
	}
	inst.Variables["last_event"] = eventData
	inst.Variables[eventName] = eventData

	inst.AwaitingEvent = ""
	inst.CurrentStep++ // Advance past the wait_event step
	inst.State = StateRunning
	inst.UpdatedAt = time.Now().UnixNano()

	if err := e.saveInstance(inst); err != nil {
		return nil, err
	}

	// Continue executing subsequent steps
	if err := e.executeSteps(ctx, def, inst); err != nil {
		inst.State = StateFailed
		inst.Error = err.Error()
		_ = e.saveInstance(inst)
		return inst, err
	}

	return inst, nil
}

// GetInstance retrieves an instance from storage.
func (e *Engine) GetInstance(ctx context.Context, instanceID string) (*WorkflowInstance, error) {
	key := []byte(storage.NSWorkflow + instanceID)
	data, err := e.store.Pebble.Get(key)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrInstanceNotFound
		}
		return nil, err
	}

	var inst WorkflowInstance
	if err := json.Unmarshal(data, &inst); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow instance: %w", err)
	}
	return &inst, nil
}

func (e *Engine) saveInstance(inst *WorkflowInstance) error {
	inst.UpdatedAt = time.Now().UnixNano()
	data, err := json.Marshal(inst)
	if err != nil {
		return err
	}
	key := []byte(storage.NSWorkflow + inst.InstanceID)
	return e.store.Pebble.Set(key, data, true)
}

// executeSteps is the Inner Loop evaluator executing steps in sequence or branching.
func (e *Engine) executeSteps(ctx context.Context, def *WorkflowDef, inst *WorkflowInstance) error {
	for inst.CurrentStep < len(def.Steps) {
		step := def.Steps[inst.CurrentStep]

		// 1. Evaluate condition if present
		if step.Condition != "" {
			passed, err := e.evalCondition(step.Condition, inst.Variables)
			if err != nil {
				return fmt.Errorf("error evaluating step %s condition: %w", step.ID, err)
			}
			if !passed {
				if step.OnFalseStep != "" {
					idx := findStepIndex(def.Steps, step.OnFalseStep)
					if idx == -1 {
						return fmt.Errorf("on_false_step %s not found in workflow", step.OnFalseStep)
					}
					inst.CurrentStep = idx
					continue
				}
				// Default skip to next step
				inst.CurrentStep++
				continue
			}
		}

		// 2. Handle Suspension: wait_event
		if step.Type == StepActionWaitEvent {
			eventName, _ := step.Params["event"].(string)
			if eventName == "" {
				eventName = step.ID
			}

			inst.State = StateSuspended
			inst.AwaitingEvent = eventName
			if err := e.saveInstance(inst); err != nil {
				return err
			}
			// Suspend goroutine cleanly: returns without error; resume happens on event
			return nil
		}

		// 3. Execute Step Action
		if err := e.executeStepAction(ctx, &step, inst); err != nil {
			return fmt.Errorf("step %s execution failed: %w", step.ID, err)
		}

		// 4. Determine next step
		if step.NextStep != "" {
			idx := findStepIndex(def.Steps, step.NextStep)
			if idx == -1 {
				return fmt.Errorf("next_step %s not found in workflow", step.NextStep)
			}
			inst.CurrentStep = idx
		} else {
			inst.CurrentStep++
		}

		// Persist progress
		if err := e.saveInstance(inst); err != nil {
			return err
		}
	}

	inst.State = StateCompleted
	return e.saveInstance(inst)
}

func (e *Engine) evalCondition(condition string, env map[string]any) (bool, error) {
	e.mu.RLock()
	prog, ok := e.compiledExprs[condition]
	e.mu.RUnlock()

	var err error
	if !ok {
		prog, err = expr.Compile(condition)
		if err != nil {
			return false, err
		}
		e.mu.Lock()
		e.compiledExprs[condition] = prog
		e.mu.Unlock()
	}

	result, err := expr.Run(prog, env)
	if err != nil {
		return false, err
	}

	boolRes, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("condition evaluated to non-boolean: %v", result)
	}
	return boolRes, nil
}

func getInt64(val any) int64 {
	switch v := val.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case uint64:
		return int64(v)
	default:
		return 0
	}
}

func getUint64(val any) uint64 {
	switch v := val.(type) {
	case int:
		return uint64(v)
	case int64:
		return uint64(v)
	case float64:
		return uint64(v)
	case float32:
		return uint64(v)
	case uint64:
		return v
	default:
		return 0
	}
}

func (e *Engine) executeStepAction(ctx context.Context, step *StepDef, inst *WorkflowInstance) error {
	switch step.Type {
	case StepActionSendMessage:
		if e.bot == nil {
			// In tests or headless without bot, simulate ok
			return nil
		}
		chatID := getInt64(step.Params["chat_id"])
		text, _ := step.Params["text"].(string)
		if text == "" {
			text = fmt.Sprintf("Workflow notification: %s", step.ID)
		}
		_, err := e.bot.SendMessage(ctx, &telegram.SendMessageParams{
			ChatID: chatID,
			Text:   text,
		})
		return err

	case StepActionLedgerReserve:
		fromUser := getInt64(step.Params["from_user"])
		toAcc, _ := step.Params["to_account"].(string)
		asset, _ := step.Params["asset"].(string)
		amount := getUint64(step.Params["amount"])
		txID := fmt.Sprintf("wf_xfr_%s_%d", inst.InstanceID, time.Now().UnixNano())

		pt, err := e.ledger.Reserve(ctx, txID, fromUser, toAcc, asset, amount, map[string]string{
			"workflow_id": inst.WorkflowID,
			"instance_id": inst.InstanceID,
		})
		if err != nil {
			return err
		}
		inst.Variables["last_transfer_id"] = pt.TransferID
		return nil

	case StepActionLedgerSettle:
		transferID, _ := step.Params["transfer_id"].(string)
		if transferID == "" {
			if lastID, ok := inst.Variables["last_transfer_id"].(string); ok {
				transferID = lastID
			}
		}
		if transferID == "" {
			return fmt.Errorf("no transfer_id specified for settle")
		}
		return e.ledger.Settle(ctx, transferID)

	case StepActionCondition:
		// Pure branching step; logic handled in evalCondition
		return nil

	case StepActionCustom:
		actionName, _ := step.Params["action"].(string)
		e.mu.RLock()
		handler, ok := e.customActions[actionName]
		e.mu.RUnlock()
		if !ok {
			return fmt.Errorf("custom action handler %q not registered", actionName)
		}

		exec := &StepExecutionContext{
			Instance:  inst,
			Step:      step,
			Variables: inst.Variables,
		}
		return handler(ctx, exec)

	default:
		return fmt.Errorf("unsupported step type: %s", step.Type)
	}
}

func findStepIndex(steps []StepDef, id string) int {
	for i, s := range steps {
		if s.ID == id {
			return i
		}
	}
	return -1
}

// StartCycleLoop runs the Outer Loop driving recurring cycle runs ticklessly.
func (e *Engine) StartCycleLoop(ctx context.Context, workflowID string, initialVars map[string]any) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		e.mu.RLock()
		def, ok := e.workflows[workflowID]
		e.mu.RUnlock()
		if !ok {
			return
		}

		interval := def.CycleConfig.Interval
		if interval <= 0 {
			interval = 1 * time.Second
		}

		maxIter := def.CycleConfig.MaxIterations
		iteration := 0

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Clone vars for this cycle
			cycleVars := make(map[string]any)
			for k, v := range initialVars {
				cycleVars[k] = v
			}
			cycleVars["iteration"] = iteration

			_, _ = e.StartInstance(ctx, workflowID, cycleVars)

			iteration++
			if maxIter > 0 && iteration >= maxIter {
				return
			}

			// Tickless sleep
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}()

	return done
}
