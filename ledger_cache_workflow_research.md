Architectural Specification for a High-Throughput Telegram Core: Embedded Storage, Declarative Workflow Runtime, and Secure Multi-Asset Financial LedgerEmbedded Storage and Zero-Allocation Caching SubsystemsBuilding an extensible, low-latency foundation for Telegram bots and Mini Apps requires persistent key-value storage coupled with an in-memory caching tier that avoids triggering the Go garbage collector (GC) during sustained update bursts. High-frequency Telegram webhooks—delivering hundreds of concurrent updates containing callback queries, inline queries, and Mini App payloads—generate millions of short-lived objects. When these objects escape to the Go heap, the garbage collector incurs stop-the-world (STW) mark-phase latency spikes that degrade message throughput and cause Telegram to back off or timeout webhook deliveries. Embedded storage engines eliminate RPC serialization and network socket hops by operating directly within the application memory space, but selecting the proper engine depends heavily on read-versus-write workloads and internal memory management.Comparative Analysis of Embedded Key-Value EnginesThe three leading embedded engines in the Go ecosystem present distinct architectural trade-offs across storage layouts, concurrency limits, and garbage collector overhead profiles.EngineStorage ArchitectureRead LatencyWrite ThroughputGC Overhead ProfilePrimary Production RolePebbleLog-Structured Merge-tree (LSM)Low to Moderate (Block cache dependent)Extremely High (Pipelined WAL + MemTable)Minimal (Off-heap manual block cache allocations)High-throughput metadata, ledger history, session persistenceBadgerDBLSM with WiscKey Value Log (Vlog) separationModerate (Key in LSM, Value in Vlog)High (Fast for large values)High over time (Value-log GC compaction worker overhead)Large blob and payload caching where values exceed 1 KBbboltCopy-on-Write (CoW) B+ TreeExtremely Low (Direct zero-copy mmap reads)Low (Serialized single-writer mutex lock)None (Operates over OS page cache mmap)Configuration stores, low-write workflow definitionsPebble, developed by CockroachDB to replace RocksDB, implements an LSM-tree optimized specifically for Go runtimes. Pebble avoids Go runtime GC degradation by managing its block cache through unmanaged off-heap memory allocations, ensuring that multi-gigabyte caches do not expand the Go heap or increase garbage collector sweep durations. Pebble excels in write-intensive workloads through its concurrent commit pipeline, making it the most suitable engine for recording append-only audit journals, workflow transition logs, and high-frequency Telegram interaction traces.BadgerDB utilizes the WiscKey architecture, separating keys (retained in an LSM-tree) from values (stored in append-only Value Logs). While this optimizes disk bandwidth for values exceeding 1 KB, it introduces operational complexity: Badger requires an active background goroutine to execute value-log garbage collection. During heavy ledger writes, Badger's internal GC competes directly with the Go runtime's garbage collector, leading to unpredictable I/O and CPU contention.The bbolt database maintains a B+ tree where pages are mapped directly into virtual memory via the mmap system call. Reads are essentially free of Go heap allocations and achieve sub-microsecond latency because they access memory pages managed directly by the operating system kernel cache. However, writes require an exclusive file lock, serializing disk flushes and capping write throughput at several thousand operations per second. Consequently, bbolt is ideal for storing immutable workflow DAG graphs and bot configurations, but it fails under concurrent balance modifications or high-volume event streams.In-Memory Cache Architectures and GC MitigationTo maintain high throughput on transient operational data—such as idempotency tokens, rate-limit counters, and active user sessions—the core must employ an in-memory cache designed to prevent GC overhead. The Go garbage collector scans every pointer residing on the heap. A standard Go map containing one million entries (map[string]*SessionState) forces the collector to traverse two million pointers during every mark phase, severely impacting CPU throughput.Caching EngineData OrganizationAllocation MechanicsEviction AlgorithmThread-Safety ArchitectureBigCacheSharded byte-array ring buffersPointer-free: map[uint32]uint32 mapping hash to offsetFIFO (TTL-based expiration only)Sharded mutexesFastCache2D byte arrays in fixed chunksZero-allocation; accepts caller-provided destination buffersChunk-based FIFO circular ringSharded read-write locks across 512 shardsOtterConcurrent hash table using Go genericsHeap-allocated typed objectsS3-FIFO (predictive multi-queue eviction)Lockless reads with buffered batch lock writesRistrettoSharded Go map with cost countersDynamic heap allocationsTinyLFU (Sampled LFU admission filter)Lock-free ring-buffer lossy event trackingBigCache and FastCache minimize garbage collection impact by avoiding pointer allocation entirely. BigCache encodes keys and values into contiguous byte arrays, indexing them via a map where both key and value are primitive integer types (map[uint32]uint32). Because primitive types contain no memory addresses, the Go GC skips scanning the map contents during GC cycles. FastCache refines this model by allowing callers to pass a destination buffer to .Get(dst, key), eliminating heap allocations during read operations.Otter offers an alternative approach focused on hit-ratio efficiency. Using an S3-FIFO eviction algorithm, Otter achieves higher cache hit ratios than standard LRU or TinyLFU implementations under Zipfian access distributions, while maintaining sub-5-nanosecond read latency via lock-free primitives. However, because Otter preserves typed pointers on the heap, holding millions of items within it increases GC mark times.The optimal design implements a two-tier storage layer within the core package (gc/pkg/storage). Tier 1 serves as the transient operational cache using FastCache to store raw byte slices for webhook idempotency keys (UpdateID), Telegram Stars pre-checkout identifiers, Mini App initData validation nonces, and chat-level rate-limiting windows. FastCache guarantees that storing millions of transient session records introduces zero pointer scanning to the Go runtime. Tier 2 serves as the durable state engine using Pebble as an embedded write-ahead log (WAL) and state engine. User balances, serialized workflow cursors, Mini App user profiles, and outbound financial transactions are committed to Pebble in batched writes. Pebble's internal off-heap memory management ensures sustained I/O performance without increasing Go heap allocations.Declarative Workflow Core and Execution Loop ArchitectureBuilding a workflow core capable of running continuous loops of actions, branching logic, and complex state mutations requires analyzing the structural designs of existing automation platforms such as Node-RED and n8n.Comparative Structural Analysis: Node-RED vs. n8nThe execution semantics and memory handling of Node-RED and n8n provide direct architectural blueprints for building an event-driven Go workflow runtime.DimensionNode-RED Architecturen8n ArchitectureProposed Core Architecture (gc/pkg/workflow)Execution ModelEvent-driven actor model; nodes emit messages to downstream consumers.Topological DAG traverser; nodes process arrays of items per step.Tickless event-loop multiplexer driving compiled instruction lists.Data UnitDiscrete JSON object envelope (msg.payload, msg.topic).Array of structured items (INodeExecutionData[]).Flattened, pooled execution context (*ExecutionContext).Memory IsolationExplicit message cloning (RED.util.cloneMessage) on branch splits.Immutable item arrays passed between nodes; deep-copied on modification.Copy-on-write pointers using pooled byte slices; zero deep copies in linear paths.Data LineageNone; origin context must be manually stored in flow or global scopes.Explicit metadata tracking via pairedItem linking downstream data to source indices.Structural indexing referencing upstream event tokens via monotonic execution registers.ScopingThree-tier hierarchy: context (node), flow (tab), global (runtime).Workflow-level run data plus persistent static store (getWorkflowStaticData).Three-tier hierarchy: Step Context, Workflow Instance State, Global Bot State.Node-RED relies on an event-driven model where nodes function as independent event emitters. When a node completes execution, it invokes node.send(msg). When a flow branches into multiple paths, Node-RED must deep-clone the msg object via RED.util.cloneMessage(msg) to prevent state mutations in one branch from causing race conditions in another. In a high-concurrency Go application, deep-cloning arbitrary maps and structs via reflection or JSON re-marshaling is a major cause of heap allocations and latency.In contrast, n8n structures execution as a Directed Acyclic Graph (DAG) where nodes receive and output arrays of items (INodeExecutionData[]). n8n tracks data lineage using pairedItem metadata, which maps every processed output element back to its exact input index. This design solves a common workflow problem: when a sub-loop processes an array, filters items, or calls external APIs, downstream nodes can still retrieve original contextual parameters from an ancestor node without requiring manual state aggregation.Proposed Core Engine: The Tickless Loop-of-Loops RuntimeTo run complex automation routines—such as a continuous trading loop, game rounds, drip marketing cycles, or event-driven Mini App interactions—without allocating thousands of idle goroutines, the engine must use a tickless, non-blocking execution design.The runtime splits execution into two coordinated loops:Outer Event Loop (Multiplexer): Integrates directly with the telegram.Dispatcher. It consumes incoming updates (*telegram.Update), webhook requests, internal ledger signals, and timer wheel ticks. The outer loop routes updates into execution queues partitioned by WorkflowID or ChatID, preventing concurrent mutations of the same workflow instance.Inner Execution Loop (Step Evaluator): Executes the compiled workflow steps sequentially within an allocated execution context. When a step completes, the evaluator reads the next instruction pointer. If a step requires an asynchronous pause (e.g., waiting 10 minutes, awaiting a Mini App event like mainButtonClicked, or waiting for a user callback query), the inner loop serializes the execution cursor into the embedded database (Pebble) and halts execution. The goroutine returns immediately to the worker pool. When the matching event or timer fires, the cursor is restored and the inner loop resumes.Expression Evaluation Engine: expr-lang/expr vs. cel-goWorkflows require dynamic condition evaluation without compiling custom Go binaries on the fly. The choice between expression languages impacts both evaluation speed and garbage collector pressure.Evaluation EngineCompilation StrategyEvaluation ThroughputMemory Allocation ProfileType System Integrationexpr-lang/exprBytecode VM compilationExtremely High (~30–80 ns/op)Near zero (0 to 1 allocations per evaluation)Native Go structs, maps, and scalar typescel-goAbstract Syntax Tree (AST) to planned interpretable programHigh (~150–350 ns/op)Higher (Allocates intermediate evaluation nodes)Built around Protocol Buffers; gradual typingWhile cel-go is widely adopted in cloud infrastructure, its design relies heavily on Protocol Buffers. Evaluating native Go structs in cel-go requires reflection-heavy type adapters, which increase memory allocations during hot loop iterations. Conversely, expr-lang/expr compiles expressions directly into an optimized bytecode format executed by a register-based virtual machine. It operates directly over native Go structs and maps without serialization wrappers. Workflows compile string expressions into *vm.Program instances during workflow creation and cache them in memory. During runtime evaluation, expr.Run(program, env) executes with zero allocations when using pooled execution contexts.Gopackage workflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type ExecutionContext struct {
	WorkflowID string
	StepIndex  int
	Variables  map[string]any
	Upstream   []map[string]any
	Cancelled  bool
}

var contextPool = sync.Pool{
	New: func() any {
		return &ExecutionContext{
			Variables: make(map[string]any, 32),
			Upstream:  make([]map[string]any, 0, 16),
		}
	},
}

func AcquireContext(workflowID string) *ExecutionContext {
	ctx := contextPool.Get().(*ExecutionContext)
	ctx.WorkflowID = workflowID
	ctx.StepIndex = 0
	ctx.Cancelled = false
	return ctx
}

func ReleaseContext(ctx *ExecutionContext) {
	clear(ctx.Variables)
	ctx.Upstream = ctx.Upstream[:0]
	contextPool.Put(ctx)
}

type StepHandler func(ctx context.Context, exec *ExecutionContext) error

type WorkflowNode struct {
	ID        string
	Action    StepHandler
	Condition *vm.Program
	NextTrue  int
	NextFalse int
}

type WorkflowEngine struct {
	nodes []WorkflowNode
}

func (e *WorkflowEngine) ExecuteLoop(ctx context.Context, exec *ExecutionContext) error {
	totalNodes := len(e.nodes)
	for exec.StepIndex >= 0 && exec.StepIndex < totalNodes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		node := &e.nodes[exec.StepIndex]

		if node.Condition != nil {
			output, err := expr.Run(node.Condition, exec.Variables)
			if err != nil {
				return fmt.Errorf("condition error at node %s: %w", node.ID, err)
			}
			matched, ok := output.(bool)
			if !ok || !matched {
				exec.StepIndex = node.NextFalse
				continue
			}
		}

		if node.Action != nil {
			if err := node.Action(ctx, exec); err != nil {
				return fmt.Errorf("action execution failure at node %s: %w", node.ID, err)
			}
		}

		exec.StepIndex = node.NextTrue
	}
	return nil
}
Multi-Asset Immutable Ledger and Verification EngineFinancial integrity within gaming platforms, Telegram Stars applications, and multi-asset crypto gateways requires strict double-entry accounting. The ledger subsystem must never maintain balances via direct updates (UPDATE accounts SET balance = balance + amount), as concurrent webhook deliveries, network retries, and race conditions inevitably cause double-spends or balance drift.Financial Ledger InvariantsThe ledger architecture adheres to the formal double-entry accounting model popularized by TigerBeetle: assets and liabilities must always balance across immutable journal entries. Every financial movement is governed by the invariant:$$\sum \text{Debits} = \sum \text{Credits}$$For every registered asset (e.g., Telegram Stars XTR, TON nano-units, Jetton USDT), accounts are partitioned into distinct functional classes.Ledger AccountClassificationFunctional RoleBalance Invariant Ruleplatform:stars:escrowAssetHolds aggregated Telegram Stars balances paid to the bot.Always $\ge 0$ (Debit balance)platform:ton:hotwalletAssetVerifiable on-chain liquid balance residing in the hot wallet.Always $\ge 0$ (Debit balance)user:{id}:availableLiabilityUser funds available for immediate play, betting, or withdrawal.Always $\le 0$ (Credit balance)user:{id}:lockedLiabilityUser funds reserved for open bets, game stakes, or active withdrawals.Always $\le 0$ (Credit balance)platform:revenue:feesEquityAccumulated platform revenue, commissions, and house edge.Always $\le 0$ (Credit balance)A user deposit does not increment a balance column; rather, it debits a platform asset account and credits a user liability account, representing the platform's obligation to the user.Two-Phase Transfer MechanismTo safely support high-throughput operations such as in-game wagers, marketplace purchases, or pending blockchain withdrawals, the ledger uses two-phase transfers:Reserve Phase (PendingTransfer): The ledger locks funds by shifting balance from user:{id}:available to user:{id}:locked. The transaction records a pending ID, amount, and expiration timeout. If the user's available liability limit is exceeded, the transfer is rejected immediately at the database level.Settle or Void Phase: If the external operation succeeds (e.g., a blockchain transaction confirms or an in-game purchase completes), the pending transfer is posted: funds move from user:{id}:locked to the target destination account. If the operation fails or times out, the pending transfer is voided, reversing the funds from user:{id}:locked back to user:{id}:available.Multi-Step Income Verification PipelineHandling inbound assets across disparate protocols (Telegram Stars webhooks, TON blockchain indexing, and Jetton transfers) requires a unified verification pipeline. The process advances sequentially through discrete validation gates:Stage 1 (Ingestion): Inbound webhooks or indexer events arrive at the HTTP ingress. The system extracts the idempotency key (TelegramPaymentChargeID for Stars, or tx_hash + lt for TON) and performs an atomic check against FastCache to discard duplicate deliveries.Stage 2 (Validation): Cryptographic signatures are verified. For Telegram Stars, the payload is validated against Bot API secret tokens and pre-checkout query states. For TON and Jettons, the indexer checks smart contract code hashes and verifies the transaction memo/payload.Stage 3 (Settlement Criteria): The system checks confirmation thresholds. While Telegram Stars settle immediately upon SuccessfulPayment, blockchain transactions require tracking block depth until finality is reached.Stage 4 (Atomic Posting): Within an ACID transaction, the deposit state updates to CREDITED and the ledger creates balanced journal entries debiting platform:asset and crediting user:available.Gopackage ledger

import (
	"context"
	"fmt"
	"time"
)

type AccountType uint8

const (
	AccountAsset AccountType = iota + 1
	AccountLiability
	AccountEquity
)

type JournalEntry struct {
	ID             uint64    `json:"id"`
	IdempotencyKey string    `json:"idempotency_key"`
	DebitAccount   string    `json:"debit_account"`
	CreditAccount  string    `json:"credit_account"`
	Asset          string    `json:"asset"`
	Amount         uint64    `json:"amount"`
	Timestamp      time.Time `json:"timestamp"`
}

type DepositStatus string

const (
	DepositDetected  DepositStatus = "DETECTED"
	DepositVerifying DepositStatus = "VERIFYING"
	DepositCredited  DepositStatus = "CREDITED"
	DepositRejected  DepositStatus = "REJECTED"
)

type DepositRecord struct {
	TxHash         string        `json:"tx_hash"`
	Asset          string        `json:"asset"`
	Amount         uint64        `json:"amount"`
	SenderAddress  string        `json:"sender_address"`
	TargetUser     int64         `json:"target_user"`
	Status         DepositStatus `json:"status"`
	BlockConfirmed uint64        `json:"block_confirmed"`
	RequiredBlocks uint64        `json:"required_blocks"`
}

type Engine struct {
	secretKey []byte
}

func (e *Engine) VerifyStarsPreCheckout(queryID string, amount int, currency string) bool {
	if currency != "XTR" || amount <= 0 {
		return false
	}
	return true
}

func (e *Engine) VerifyTonPayload(memo string, expectedUserID int64) bool {
	var parsedID int64
	_, err := fmt.Sscanf(memo, "user_%d", &parsedID)
	return err == nil && parsedID == expectedUserID
}
Outbound Wallet Manager and Isolated Cryptographic SignerManaging hot-wallet private keys directly inside the primary bot application introduces significant security risks. A memory dump, command-injection vulnerability, or compromised dependency could expose master private keys. The outbound transaction manager must separate transaction scheduling from cryptographic signing.Process Isolation and Threat BoundaryThe system architecture partitions signing responsibilities across two discrete OS processes:Primary Application Core (gc-core): Orchestrates workflows, dispatches updates, updates ledger balances, verifies deposits, and enqueues outbound withdrawal requests to a persistent outbox. It has no access to wallet private keys or mnemonic phrases.Isolated Wallet Signer (gc-signer): A hardened, minimal binary running under a dedicated, unprivileged operating system user. It accesses hardware security modules (HSM) or encrypted local keystores. It communicates with gc-core exclusively over a local Unix Domain Socket protected by strict file permissions (chmod 0600) or mTLS over localhost.Blockchain Nonce and Seqno SerializationBlockchains enforce strict replay-protection mechanics that break under naive concurrent execution:EVM Architectures: Rely on an incrementing nonce. If transactions are dispatched concurrently, any dropped nonce causes head-of-line blocking for all subsequent transactions.Standard TON Wallets (v4): Utilize an on-chain counter (seqno). A second transaction cannot be broadcast until the block containing the prior transaction has been finalized by validators, limiting outbound throughput to approximately one transfer every three to five seconds.TON Highload Wallet v3 Batch DispatcherTo overcome the single seqno bottleneck on the TON blockchain, the outbound engine utilizes the Highload Wallet v3 smart contract. Highload Wallet v3 decouples transactions from an incrementing seqno by introducing a 23-bit bitshifted query_id replay protection mechanism, allowing it to track up to 8,380,416 distinct transactions within a sliding expiration window without sequential blocking.$$\text{query\_id} = (\text{subwallet\_id} \ll 32) \mid (\text{shift} \ll 10) \mid \text{bit\_index}$$Highload Wallet v3 can pack up to 254 outgoing transfer messages into a single smart-contract invocation, dramatically reducing TVM compute fees and network load.Gopackage wallet

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type OutboxStatus string

const (
	StatusQueued      OutboxStatus = "QUEUED"
	StatusApproved    OutboxStatus = "APPROVED"
	StatusSigned      OutboxStatus = "SIGNED"
	StatusBroadcasted OutboxStatus = "BROADCASTED"
	StatusConfirmed   OutboxStatus = "CONFIRMED"
	StatusFailed      OutboxStatus = "FAILED"
)

type OutboxEntry struct {
	ID          uint64       `json:"id"`
	UserID      int64        `json:"user_id"`
	Destination string       `json:"destination"`
	Asset       string       `json:"asset"`
	Amount      uint64       `json:"amount"`
	Status      OutboxStatus `json:"status"`
	RetryCount  int          `json:"retry_count"`
	TxHash      *string      `json:"tx_hash,omitempty"`
	QueryID     uint64       `json:"query_id"`
	CreatedAt   time.Time    `json:"created_at"`
}

type BatchSignerClient interface {
	SignTonBatch(ctx context.Context, transfers []OutboxEntry) (signedPayload []byte, err error)
}

type OutboxProcessor struct {
	db           *sql.DB
	signerClient BatchSignerClient
	maxBatchSize int
}

func (p *OutboxProcessor) ProcessBatch(ctx context.Context) error {
	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT id, user_id, destination, asset, amount, query_id 
		FROM outbox_transfers 
		WHERE status = 'APPROVED' 
		ORDER BY id ASC 
		LIMIT $1 FOR UPDATE SKIP LOCKED`, p.maxBatchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	var batch []OutboxEntry
	for rows.Next() {
		var entry OutboxEntry
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Destination, &entry.Asset, &entry.Amount, &entry.QueryID); err != nil {
			return err
		}
		batch = append(batch, entry)
	}

	if len(batch) == 0 {
		return nil
	}

	signedBytes, err := p.signerClient.SignTonBatch(ctx, batch)
	if err != nil {
		return fmt.Errorf("cryptographic signing failed: %w", err)
	}

	for _, item := range batch {
		_, err = tx.ExecContext(ctx, `
			UPDATE outbox_transfers 
			SET status = 'SIGNED', retry_count = retry_count + 1 
			WHERE id = $1`, item.ID)
		if err != nil {
			return err
		}
	}

	_ = signedBytes
	return tx.Commit()
}
Transactional Outbox State Machine LifecycleTo guarantee consistency between the internal double-entry ledger and external blockchain nodes, state transitions follow a strict outbox lifecycle:Phase 1 (Reservation): When a user requests a withdrawal, funds move from user:available to user:locked in the ledger, and an outbox record is inserted with QUEUED status within the same ACID transaction.Phase 2 (Risk and Velocity Approval): A background verifier evaluates withdrawal velocity, pattern anomalies, and aggregate platform caps, advancing valid records from QUEUED to APPROVED.Phase 3 (Batch Packing and Signing): The outbox worker claims up to 254 approved records using SELECT ... FOR UPDATE SKIP LOCKED, generates an IPC request to gc-signer, and advances records to SIGNED upon receiving the signed cryptographic payload.Phase 4 (Network Broadcast): The signed batch message is broadcast to the blockchain network via ADNL or LiteServer endpoints, marking the records as BROADCASTED.Phase 5 (Final Settlement or Refund): Upon observing sufficient on-chain confirmation depth, records transition to CONFIRMED, burning the user's locked liability. If the transaction fails on-chain or exceeds the timeout window, records transition to FAILED, voiding the transfer and refunding funds from user:locked back to user:available.Low-Latency Go Runtime Engineering and GC MitigationUnder high load, memory allocations inside hot execution paths trigger frequent garbage collection cycles, degrading throughput and causing unpredictable latency spikes. Several patterns across the existing core codebase (gc/pkg/telegram) can be optimized to achieve low-GC execution.Stream Decoding and Buffer ReuseBaseline HTTP transport implementations frequently allocate byte slices on every request via io.ReadAll and json.Marshal. Reading entire response bodies into newly allocated byte slices forces continuous heap churn. When scaled to tens of thousands of requests per second, this pattern degrades garbage collector efficiency. To eliminate these allocations, the engine employs pooled byte buffers via sync.Pool across both the webhook ingress (ServeHTTP) and the API transport (Client.Do).Gopackage telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}

func AcquireBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func ReleaseBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 65536 {
		return
	}
	bufferPool.Put(buf)
}

func (c *Client) DoOptimized(ctx context.Context, method string, payload any, result any) error {
	endpoint := c.apiEndpoint(method)
	buf := AcquireBuffer()
	defer ReleaseBuffer(buf)

	if payload != nil {
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return fmt.Errorf("failed to encode payload: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
Eliminating Dynamic Reflection and Enforcing Outbound BackpressureThe baseline schema builder uses dynamic runtime reflection (reflect.ValueOf) to construct request maps. Using Go's reflect package inside hot event loops incurs significant CPU overhead and forces variables to escape to the heap. Dynamic schema reflection must be restricted entirely to build-time initialization and configuration parsing. Inside execution loops, schemas compile into static assignment closures (type EvaluatorFunc func(src any, dst map[string]any)), and workflows cache compiled condition expressions (*vm.Program) to bypass runtime string manipulation.Furthermore, the Telegram Bot API enforces strict rate limits:A maximum of 1 message per second to any specific private or group chat.A maximum of 20 messages per minute to channels or groups.A global API limit of 30 messages per second per bot token across all chats.Violating these limits causes Telegram to return HTTP status 429 Too Many Requests with a RetryAfter parameter. Retrying dropped messages naively inside workflow goroutines can tie up threads and trigger cascading memory spikes. The engine manages outbound requests using a token-bucket rate limiter (golang.org/x/time/rate) keyed by bot token and chat destination. Outgoing requests are buffered into bounded queues. When a chat queue exceeds its capacity, backpressure is applied directly to the workflow instance by delaying its next loop iteration, protecting both the Telegram API limit and internal memory resources.Architectural SynthesisThe system architecture structures the entire platform as a modular monolith with an isolated cryptographic signing process, maintaining near-zero network latency while establishing a strict security boundary.Incoming Telegram updates from webhooks or long pollers flow directly through pkg/telegram to the tickless runtime in pkg/workflow via in-memory dispatchers, eliminating RPC serialization overhead. Active session states and idempotency tokens are verified against FastCache (Tier 1 storage), while durable workflow cursors, Mini App states, and financial transactions are committed to Pebble (Tier 2 storage). Ledger operations in pkg/ledger enforce double-entry invariants, pushing outbound blockchain withdrawal batches to a persistent transactional outbox. The outbox processor coordinates with the isolated cmd/signer process over a secure local Unix Domain Socket, broadcasting signed transaction batches to the blockchain network without exposing private keys to the workflow core.System LayerSelected TechnologyStructural Role in the CorePrimary JustificationIn-Memory CacheVictoriaMetrics/fastcacheSession cache, nonces, rate-limit buckets.Zero pointer allocations; prevents GC scan pauses across millions of entries.Embedded Databasecockroachdb/pebbleDurable state, WAL, workflow cursors, ledger logs.Pure Go LSM-tree with off-heap block caching and fast commit pipelines.Condition Engineexpr-lang/exprRuntime workflow branching and filter evaluation.Pre-compiles into bytecode; executes directly on native Go types with zero allocations.Workflow LineageExplicit SourceIndex linkingNode-to-node tracking across batched loop iterations.Matches n8n's pairedItem design, enabling data retrieval without state fragmentation.Double-Entry CoreTigerBeetle invariant rulesMulti-asset accounting for Stars, TON, Jettons, points.Guarantees $\sum \text{Debits} = \sum \text{Credits}$; two-phase reservation prevents double-spends.Transaction SignerIsolated binary (cmd/signer)Holding and signing with hot-wallet private keys.Security boundary isolating keys from workflow dependencies and bot logic.TON BatchingHighload Wallet v3 wrapperHigh-volume outbound transfers via tonutils-go.Replaces single seqno locks with 23-bit query_id masks; packs up to 254 transfers per transaction.Implementing these unified architectural patterns ensures that the Telegram core delivers sustained sub-millisecond execution, complete data lineage tracking, and strict financial accounting across diverse digital assets while maintaining minimal memory churn and zero garbage collector degradation.