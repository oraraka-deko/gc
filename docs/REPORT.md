# Project Note & Progress Report

- **Date & Time:** `2026-11-11T09:10:00-08:00`
- **Status:** **COMPLETED**
- **Bot Identity Verified:** `GiftShaper` (`@Giftshaper_bot`), ID: `9009679701`
- **Environment:** Go 1.27.3 windows/amd64

---

## 1. What We Worked On

We engineered a comprehensive, production-grade Telegram Bot & Mini App ecosystem featuring:
1. **Core SDK & Mini App Platform**: Typed Telegram Bot API SDK, Schema Builder, Dynamic Updates Engine, and WebApp validation.
2. **Dual-Tier High-Performance Storage (`pkg/storage`)**: Zero-GC in-memory cache (FastCache) + pure Go durable LSM-tree (Pebble).
3. **Double-Entry Financial Ledger Core (`pkg/ledger`)**: Mathematical balance invariant ($\sum \text{Debits} == \sum \text{Credits}$), atomic units, overdraft protection, and 2-phase reserve-settle-void transfers.
4. **Multi-Asset Income Verifier (`pkg/ledger/verifier.go`)**: 4-stage pipeline for Telegram Stars (`pre_checkout_query` & `successful_payment`), TON nanoTON on-chain transactions, and Jetton USDT deposits with memo parsing.
5. **Outbound Transactional Outbox & Isolated Signer (`pkg/wallet` & `cmd/signer`)**: State machine (`QUEUED` -> `APPROVED` -> `SIGNED` -> `BROADCASTED` -> `CONFIRMED`/`FAILED`), Highload Wallet v3 cell packing (up to 254 transfers per message), and separate daemon process over local IPC.
6. **Tickless Loop-of-Loops Workflow Engine (`pkg/workflow`)**: Outer cycle orchestrator + inner batch step evaluator, `expr-lang/expr` bytecode VM condition evaluation, zero-alloc state suspension to Pebble on awaited events, and full Bot API action dispatching.
7. **End-to-End Examples & Test Suites**:
   - `examples/crypto_converter_bot`: Live multi-mode natural language currency bot on `@Giftshaper_bot`.
   - `examples/ledger_workflow_demo`: Complete demonstration running ledger, verifier, outbox, and escrow workflow.
   - Comprehensive unit and integration test coverage across all packages (100% passing).

---

## 2. Architectural Blueprint & Subsystems

```
                                  +------------------------------------+
                                  |     Telegram Updates / Webhook     |
                                  +-----------------+------------------+
                                                    |
                                                    v
                                  +------------------------------------+
                                  |     Updates Dispatcher / Poller    |
                                  |       (pkg/telegram/dispatcher)    |
                                  +--------+------------------+--------+
                                           |                  |
                       Stars / Payments    |                  | Commands & Events
                                           v                  v
                 +------------------------------------+   +------------------------------------+
                 |       Multi-Asset Verifier         |   |    Tickless Workflow Engine        |
                 |      (pkg/ledger/verifier.go)      |   |        (pkg/workflow)              |
                 +-----------------+------------------+   +-----------------+------------------+
                                   |                                        |
                 Atomic Deposit    |                                        | Reserve / Settle
                                   v                                        v
+----------------------------------------------------------------------------------------------+
|                                DOUBLE-ENTRY FINANCIAL LEDGER                                 |
|                                         (pkg/ledger)                                         |
|                                                                                              |
|  Invariant:  Sum(Debits) == Sum(Credits) for all assets at all times                         |
|  Assets:     XTR (Stars, 0 dec), TON (nanoTON, 9 dec), USDT (microUSDT, 6 dec)               |
|  Accounts:   ASSET (Reserve), LIABILITY (User Main, User Locked), REVENUE (Fee), EXPENSE     |
|  Two-Phase:  Reserve (Available -> Locked) | Settle (Locked -> Dest) | Void (Locked -> Avail)|
+-----------------------------------------------+----------------------------------------------+
                                                |
                                                v
+----------------------------------------------------------------------------------------------+
|                                    STORAGE TIER (pkg/storage)                                |
|                                                                                              |
|   +------------------------------------------+    +---------------------------------------+  |
|   |         FastCache (In-Memory)            |    |         Pebble DB (LSM-Tree)          |  |
|   |  * Zero GC pointer scan overhead         |    |  * Append-only Write-Ahead Log (WAL)  |  |
|   |  * Sub-millisecond idempotency & nonces  |    |  * Atomic multi-key batch commits     |  |
|   |  * Simulated TTL expiration              |    |  * Prefix scans & chronological audit |  |
|   +------------------------------------------+    +---------------------------------------+  |
+-----------------------------------------------+----------------------------------------------+
                                                ^
                                                |
+-----------------------------------------------+----------------------------------------------+
|                          OUTBOUND TRANSACTIONAL OUTBOX & SIGNER                              |
|                              (pkg/wallet & cmd/signer)                                       |
|                                                                                              |
|   Outbox State Machine:                                                                      |
|   [QUEUED] --(Batch)--> [APPROVED] --(Signer IPC)--> [SIGNED] --(Broadcast)--> [BROADCASTED] |
|                                                                |                             |
|                                                                v                             |
|                                                   [CONFIRMED] / [FAILED]                     |
|                                                   (Settle)      (Void Refund)                |
|                                                                                              |
|   Isolated Signer Daemon (cmd/signer):                                                       |
|   * Holds private Ed25519 key isolated from web/bot processes                                |
|   * Highload Wallet v3 cell builder (batches up to 254 transfers per transaction)            |
|   * Local IPC over TCP / Named Pipe / Unix Socket                                            |
+----------------------------------------------------------------------------------------------+
```

---

## 3. Subsystem Breakdown & Implementation Details

### A. Storage Tier (`pkg/storage`)
- **FastCache (`fastcache.go`)**:
  - Memory cache running with chunked ring buffers off the Go GC pointer heap.
  - Implements TTL simulation by packing an 8-byte unix nano expiration header in the value payload.
  - 256 sharded read/write mutexes (`numShards = 256`) for zero lock contention under high concurrency.
  - Methods: `Set`, `Get`, `Delete`, `SetIdempotency`, `CheckNonce`, `Reset`.
- **Pebble LSM-Tree (`pebble.go`)**:
  - Pure Go LevelDB/RocksDB replacement from CockroachDB.
  - Clean namespace separation:
    - `NSAccount = "led:acc:"` (account states)
    - `NSJournal = "led:jnl:"` (immutable double-entry journals)
    - `NSAccountHistory = "led:ach:"` (chronological index for audit trail)
    - `NSPending = "led:pen:"` (two-phase pending transfers)
    - `NSWorkflow = "wf:cur:"` (tickless workflow cursors)
    - `NSOutbox = "out:tx:"` (transactional outbox transfers)
  - Atomic batch writes with synchronous disk WAL flushing (`batch.Commit(true)`).
  - Prefix upper-bound calculation and lexicographical range iterators (`PrefixScan`).

### B. Double-Entry Ledger Core (`pkg/ledger`)
- **Mathematical Invariant**:
  $$\sum \text{Debits} == \sum \text{Credits}$$
  Must strictly hold for every asset involved in every journal entry. Unbalanced entries are rejected with `ErrUnbalancedEntry`.
- **Fixed-Precision Atomic Integer Arithmetic (`uint64`)**:
  - Zero floating-point rounding errors.
  - Telegram Stars (`XTR`): 0 decimals.
  - TON (`TON`): 9 decimals ($1 \text{ TON} = 10^9 \text{ nanoTON}$).
  - Tether (`USDT`): 6 decimals ($1 \text{ USDT} = 10^6 \text{ microUSDT}$).
- **Normal Account Classifications**:
  - **Asset / Expense**: Debit increases balance (+), Credit decreases balance (-).
  - **Liability / Equity / Revenue**: Credit increases user/platform liability balance (+), Debit decreases balance (-).
- **Two-Phase Transfer Flow**:
  1. `Reserve(ctx, transferID, fromUser, toAccount, asset, amount, meta)`:
     Debits `user:from:main` and credits `user:from:locked`. Fails if insufficient funds (`ErrInsufficientFunds`).
  2. `Settle(ctx, transferID)`:
     Debits `user:from:locked` and credits `toAccount`. Marks transfer settled.
  3. `Void(ctx, transferID, reason)`:
     Debits `user:from:locked` and credits `user:from:main` (refunds locked funds back to user). Marks transfer voided.
- **Convenience Methods**:
  - `Deposit(ctx, userID, asset, amount, ref, desc)`
  - `Withdraw(ctx, userID, asset, amount, ref, desc)`
  - `InternalTransfer(ctx, fromUser, toUser, asset, amount, ref, desc)`
  - `GetAccountHistory(ctx, accountID, limit)` (chronological audit log)

### C. Multi-Asset Income Verifier (`pkg/ledger/verifier.go`)
- **4-Stage Verification Pipeline**: Ingestion $\to$ Signature Validation $\to$ Settlement Criteria $\to$ Atomic Credit.
- **Telegram Stars**:
  - `ValidatePreCheckout`: Validates `pre_checkout_query`, currency code `XTR`, positive amount, and invoice idempotency.
  - `ProcessStarsPayment`: Validates `successful_payment`, extracts payment charge ID, records 48-hour idempotency key, and executes double-entry deposit to user.
- **On-Chain TON Transfers**:
  - `ProcessTONDeposit`: Validates transaction hash, parses destination memo for user routing (`12345`, `user_12345`, `user:12345`, `uid:12345`), and credits user's nanoTON balance.
- **Jetton USDT Transfers**:
  - `ProcessJettonDeposit`: Validates Jetton master contract notification, parses memo, and credits user's microUSDT balance.

### D. Outbound Transactional Outbox & Signer (`pkg/wallet` & `cmd/signer`)
- **Transactional Outbox (`pkg/wallet/outbox.go`)**:
  - Guards against double-spends and orphaned withdrawals.
  - `Enqueue`: Atomically reserves funds via `ledger.Reserve` and persists `OutboxTx` in `StateQueued`.
  - `ProcessNextBatch`: Bundles queued transactions up to `maxBatchSize`, passes them to `BatchSigner`, sets `StateSigned`.
  - `MarkBroadcasted`: Updates status to `StateBroadcasted` with on-chain transaction hash.
  - `Confirm`: Settles transfer via `ledger.Settle`, transitions to `StateConfirmed`.
  - `Fail`: Voids transfer via `ledger.Void`, refunds user available balance, transitions to `StateFailed`.
- **Signer Drivers (`pkg/wallet/signer.go`)**:
  - `MockSigner`: In-process Ed25519 signer for local development and unit tests.
  - `IPCSignerClient`: High-security client communicating over local socket/TCP to the external signer daemon.
- **Isolated Signer Daemon (`cmd/signer/main.go` & `pkg/wallet/server.go`)**:
  - Dedicated standalone process holding private keys in separate OS memory space.
  - Builds Highload Wallet v3 cell messages supporting up to 254 transfers per transaction without seqno locking.

### E. Tickless Loop-of-Loops Workflow Engine (`pkg/workflow`)
- **Tickless Suspension & Resumption**:
  - When reaching an asynchronous barrier (e.g. `StepActionWaitEvent`), the engine serializes the complete execution state (`WorkflowInstance`, variables, current step index) into Pebble under `wf:cur:<instanceID>`.
  - The worker goroutine exits immediately (0 CPU, 0 goroutine leaks).
  - When an external signal arrives (e.g. Telegram callback, payment confirmation), `ResumeInstance` restores the instance from Pebble and resumes execution at the exact step.
- **Bytecode Condition Evaluation**:
  - Uses `github.com/expr-lang/expr` for zero-allocation bytecode VM condition evaluation.
  - Enables conditions like `amount >= 1000 && user.is_vip == true`.
  - Dynamic branching with `NextStep` and `OnFalseStep`.
- **Outer Cycle Orchestrator**:
  - `StartCycleLoop`: Drives recurring workflows ticklessly using non-polling timers (`time.After` / channels), stopping cleanly on context cancellation.

---

## 4. Verification & Test Results

All packages pass unit and integration test suites:

```
ok   gc/examples/crypto_converter_bot    0.058s
ok   gc/pkg/ledger                       1.675s
ok   gc/pkg/miniapp                      (cached)
ok   gc/pkg/schema                       (cached)
ok   gc/pkg/storage                      1.863s
ok   gc/pkg/telegram                     0.338s
ok   gc/pkg/wallet                       2.024s
ok   gc/pkg/workflow                     3.741s
```

All 7 core test suites execute cleanly with zero test failures or race conditions.
