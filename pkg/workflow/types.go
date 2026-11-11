package workflow

import (
	"errors"
	"time"
)

// InstanceState represents the state of a workflow execution instance.
type InstanceState string

const (
	StateCreated   InstanceState = "CREATED"
	StateRunning   InstanceState = "RUNNING"
	StateSuspended InstanceState = "SUSPENDED" // Persisted to Pebble waiting for external event/timer
	StateCompleted InstanceState = "COMPLETED"
	StateFailed    InstanceState = "FAILED"
)

// StepType describes supported workflow action types.
const (
	StepActionSendMessage   = "telegram:send_message"
	StepActionEditMessage   = "telegram:edit_message"
	StepActionSendInvoice   = "telegram:send_invoice"
	StepActionLedgerReserve = "ledger:reserve"
	StepActionLedgerSettle  = "ledger:settle"
	StepActionCondition     = "flow:condition"
	StepActionWaitEvent     = "flow:wait_event"
	StepActionCustom        = "flow:custom"
)

var (
	ErrWorkflowNotFound = errors.New("workflow: definition not found")
	ErrInstanceNotFound = errors.New("workflow: instance not found")
	ErrInstanceNotPaused = errors.New("workflow: instance is not suspended")
)

// StepDef defines a single step/action in a workflow.
type StepDef struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Condition   string         `json:"condition,omitempty"`    // expr-lang expression
	Params      map[string]any `json:"params,omitempty"`       // Step parameters
	NextStep    string         `json:"next_step,omitempty"`    // Explicit step ID jump
	OnFalseStep string         `json:"on_false_step,omitempty"`// Step ID if Condition is false
}

// CycleConfig configures the outer loop of a workflow.
type CycleConfig struct {
	Interval      time.Duration `json:"interval"`       // Tickless interval between cycles
	MaxIterations int           `json:"max_iterations"` // 0 for infinite until cancelled
}

// WorkflowDef defines an executable workflow schema.
type WorkflowDef struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Steps       []StepDef   `json:"steps"`
	CycleConfig CycleConfig `json:"cycle_config,omitempty"`
}

// WorkflowInstance tracks the durable execution state of a workflow run.
type WorkflowInstance struct {
	InstanceID    string         `json:"instance_id"`
	WorkflowID    string         `json:"workflow_id"`
	State         InstanceState  `json:"state"`
	CurrentStep   int            `json:"current_step"` // Index into WorkflowDef.Steps
	Variables     map[string]any `json:"variables"`
	AwaitingEvent string         `json:"awaiting_event,omitempty"` // Event name waiting for
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
	Error         string         `json:"error,omitempty"`
}

// StepExecutionContext is provided to custom action handlers.
type StepExecutionContext struct {
	Instance  *WorkflowInstance
	Step      *StepDef
	Variables map[string]any
}
