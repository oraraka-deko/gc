# Project Engineering Guidelines & Architectural Invariants

## 1. System Target & Client Architecture
- **Primary Client**: Flutter Web Client running within Telegram Mini App webviews.
- **Client/Server Responsibility Boundary**:
  - **Flutter Client (Heavy Compute / UI)**: Handles 100% of graphics, animations, game physics, math simulations, and UI slice rendering.
  - **Go Backend (Ultra-Lightweight & Fast)**: Strictly handles state transitions, double-entry ledger settlement, anti-cheat validation, and fast IO with minimal CPU/RAM footprint.
- **Domains & Infrastructure**:
  - Primary Domain: `esil.click`
  - Secondary Domain: `ochinchin.sbs`
  - Full DNS, reverse proxy, SSL, and webhook routing control.
- **Database & Persistence Tier**:
  - **Embedded Tier**: FastCache (zero-GC in-memory cache) + Pebble LSM-tree (sync WAL, durable journal).
  - **Cloud/Distributed Tier**: Upstash Redis (distributed rate limiting & pub/sub) + NeonDB PostgreSQL (relational storage).

## 2. Canonical Currencies & Live Rate Architecture
- **Production Asset Types**:
  1. **TON / Gram**: Stored in atomic nanoTON ($10^9$) as `uint64` / string (`ton_nano`).
  2. **Telegram Stars (`XTR`)**: Stored in integer atomic units (`stars`).
  3. **Tether (`USDT`)**: Stored in atomic microUSDT ($10^6$).
- **Live Rates Engine**:
  - Never use hardcoded, mocked, or stale rate values in production.
  - Utilize the multi-source fallback engine ([`examples/crypto_converter_bot/rates.go`](file:///w:/gc/examples/crypto_converter_bot/rates.go)) combining TonAPI.io $\to$ Binance $\to$ CoinGecko $\to$ OKX with a strict short-TTL in-memory cache.
- **Single Source of Truth**: All user balances bind to [`InventoryBalances`](file:///w:/gc/pkg/miniapp/account.go#L95) and transact through the [`pkg/ledger`](file:///w:/gc/pkg/ledger) double-entry core (`Reserve`, `Settle`, `Void`, `Deposit`).

## 3. Security, Authentication & Anti-Spoofing Protocols
- **Cryptographic `initData` Verification**:
  - The Flutter web client accesses Telegram Mini App APIs natively, but the Go backend **must always** cryptographically verify the Telegram `initData` signature (HMAC-SHA256 with bot token or Ed25519) and enforce maximum `auth_date` age.
- **Strict Identity Isolation (No Insecure User IDs in Requests)**:
  - Backend endpoints must **never** trust or accept `user_id` or sensitive user credentials passed in the HTTP request body, query parameters, or arbitrary headers.
  - User identity must always be extracted directly from the verified cryptographic session context (`initData.User.ID`).
  - *Testing Exception*: If a temporary test harness requires passing `user_id` in a payload, it must be marked with an explicit `// TODO: SECURITY` comment and logged in `docs/REPORT.md` for immediate removal.

## 4. Gifts & NFT Asset Lifecycle
Gifts and NFTs cannot be handled like fungible coins; they follow distinct multi-step protocols:
- **Gift Deposits**:
  - Users transfer gifts to a designated relay/business account managed by the bot token via Telegram Business / MTProto APIs.
  - The backend detects the gift receipt and automatically credits the corresponding [`CollectibleItem`](file:///w:/gc/pkg/miniapp/account.go#L87) into the user's inventory.
- **Simple (Non-Collectible) Gift Withdrawals**:
  - Automated in realtime: The bot utilizes its internal platform Telegram Stars reserve balance (funded from platform user deposits) to purchase and gift the item directly to the user via Bot API methods.
- **NFT / Rare Collectible Withdrawals**:
  - Escrow & Manual Admin Approval: The withdrawal request is placed into an escrow state (`pending_admin_delivery`), updating the user's UI status until the administrator reviews and processes the transfer. Admins can update the withdrawal status.

## 5. Full-Stack / Hybrid Client-Server Workflows
Certain features require coordinated orchestration across both Flutter and Go:
- **TON Connect**: Flutter client opens the native wallet modal; backend generates cryptographically signed proof challenges/nonces and verifies the signed payload.
- **Telegram Invoices**: Backend generates invoice links/payloads; Flutter client triggers Telegram native payment sheets; backend listens for `pre_checkout_query` and `successful_payment` updates.
- **Multi-Step Verification**: Client collects required context (e.g. biometrics, device fingerprints, or 2FA confirmations) and submits proof to the backend.

## 6. Autonomous Refactoring of Incoming Prototype/Game Logic
When backend logic for games, lootboxes, or mini-app modules is provided:
- **Recognize Mock/Test Artifacts**: Understand that prototype code often uses temporary, disconnected asset types, float balances, or standalone wallet mock classes created during isolated testing.
- **Proactive Canonicalization**: Automatically refactor these prototypes into the production core:
  - Replace custom mock wallets with [`pkg/ledger`](file:///w:/gc/pkg/ledger) double-entry transactions.
  - Map game stakes/wagers to two-phase `Reserve` (user available $\to$ locked).
  - Map wins/payouts to `Settle` (credited to user balance).
  - Map cancellations/refunds to `Void` (returned to user balance).
  - Bind game outcome updates to [`UserFinancialMetrics`](file:///w:/gc/pkg/miniapp/account.go#L110) (`HighestWinAmount`, `NetSpendVolume`, `TotalBonusClaimed`).
- **Autonomous Decision Making**: Apply these integrations directly without requiring the user to prompt for every type replacement or ledger bridge.

## 7. High-Concurrency & Load Testing Invariant (100+ Concurrent Users)
- Every feature, endpoint, and game engine must be designed, benchmarked, and tested to handle **100+ concurrent active users** simultaneously.
- Concurrent tests must simulate simultaneous gameplay, balance reservations, shop transactions, deposits, and withdrawals without race conditions, memory leaks, or database locking.

## 8. Incremental Implementation & Tracking
- When implementing extensive multi-step features, record clear `TODO:` markers and update [`docs/REPORT.md`](file:///w:/gc/docs/REPORT.md) detailing what is completed and what remains to be completed.
- Ensure that the repository builds cleanly (`go build ./...`) and passes tests (`go test ./...`) at every stage.

## 9. Proactive Collaborative Course-Correction
- As an engineering partner, if any user request contains architectural flaws, security risks, scalability bottlenecks, or logical contradictions, **do not blindly execute it**.
- Proactively flag the issue, explain the risk, and propose an optimal alternative solution before or while proceeding.
