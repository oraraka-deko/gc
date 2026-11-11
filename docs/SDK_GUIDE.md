# Telegram Bot & Mini App Auto SDK — Developer Guide

This SDK provides idiomatic Go implementations for Telegram Bot API operations, Telegram Mini App components, cryptographic validation, and an Auto Schema Builder.

---

## 1. Installation & Setup

Ensure Go 1.22+ or newer is installed. The SDK uses the standard Go library with zero external dependencies.

```bash
go test ./...
```

---

## 2. Telegram Bot API Client

### Initializing the Client

Never hardcode your bot token in source code. Load it from an environment variable:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"gc/pkg/telegram"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		panic("TELEGRAM_BOT_TOKEN environment variable is not set")
	}

	client := telegram.NewClient(token,
		telegram.WithTimeout(15*time.Second),
	)

	ctx := context.Background()
	user, err := client.GetMe(ctx)
	if err != nil {
		fmt.Printf("GetMe error: %v\n", err)
		return
	}

	fmt.Printf("Authenticated as @%s (ID: %d)\n", *user.Username, user.ID)
}
```

### Sending Messages with Keyboards & Mini App Launchers

```go
func sendWelcomeMessage(client *telegram.Client, chatID int64) error {
	ctx := context.Background()

	// Inline keyboard with a Mini App web_app button
	inlineMarkup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{
					Text: "Open Mini App",
					WebApp: &telegram.WebAppInfo{
						URL: "https://your-miniapp-domain.com",
					},
				},
			},
		},
	}

	params := &telegram.SendMessageParams{
		ChatID:      chatID,
		Text:        "Welcome! Click below to launch the Mini App:",
		ParseMode:   stringPtr("HTML"),
		ReplyMarkup: inlineMarkup,
	}

	msg, err := client.SendMessage(ctx, params)
	if err != nil {
		return err
	}
	fmt.Printf("Sent message ID: %d\n", msg.MessageID)
	return nil
}

func stringPtr(s string) *string { return &s }
```

### Answering Mini App Queries from the Bot

When a Mini App triggers `sendData` or an inline query interaction, answer it with `AnswerWebAppQuery`:

```go
func answerApp(client *telegram.Client, queryID string) error {
	ctx := context.Background()

	result := map[string]any{
		"type":  "article",
		"id":    "1",
		"title": "Mini App Action Completed",
		"input_message_content": map[string]any{
			"message_text": "Item successfully selected in Mini App!",
		},
	}

	res, err := client.AnswerWebAppQuery(ctx, &telegram.AnswerWebAppQueryParams{
		WebAppQueryID: queryID,
		Result:        result,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Answered WebApp Query, inline message ID: %v\n", res.InlineMessageID)
	return nil
}
```

---

## 3. Mini App Cryptographic Validation

Telegram passes a signed query string (`initData`) to your Mini App. The backend must validate the HMAC-SHA256 signature before trusting any user or chat data.

### Validating `initData` with the Bot Token

```go
package main

import (
	"fmt"
	"os"
	"time"

	"gc/pkg/miniapp"
)

func handleMiniAppAuth(rawQueryString string) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	// Validate HMAC-SHA256 signature and ensure auth_date is within 1 hour
	valid, err := miniapp.Validate(rawQueryString, botToken, 1*time.Hour)
	if err != nil || !valid {
		fmt.Printf("Invalid or expired initData: %v\n", err)
		return
	}

	// Parse fields into strongly-typed struct
	data, err := miniapp.ParseInitData(rawQueryString)
	if err != nil {
		fmt.Printf("Failed to parse initData: %v\n", err)
		return
	}

	fmt.Printf("Authenticated User: %s (ID: %d)\n", data.User.FirstName, data.User.ID)
	if data.Chat != nil {
		fmt.Printf("Launched from Chat: %s (ID: %d)\n", data.Chat.Title, data.Chat.ID)
	}
}
```

### Validating Third-Party `initData` with Ed25519

If validating data on behalf of a third party without access to the bot token:

```go
func handleThirdPartyAuth(rawQueryString string, botID string) {
	// isTestEnv = false for production Telegram environment
	valid, err := miniapp.ValidateThirdParty(rawQueryString, botID, false, 24*time.Hour)
	if err != nil || !valid {
		fmt.Printf("Ed25519 signature validation failed: %v\n", err)
		return
	}
	fmt.Println("Third-party Telegram signature verified successfully!")
}
```

---

## 4. Mini App Component Builders

Construct and manage the state of Mini App UI and hardware components in Go:

### Bottom Buttons (Main & Secondary)

```go
package main

import (
	"fmt"

	"gc/pkg/miniapp"
)

func configureBottomButtons() {
	// Main Button with custom styling
	mainBtn := miniapp.NewMainButton("Pay 50 Stars").
		SetColors("#2481cc", "#ffffff").
		SetShine(true)

	// Secondary Button placed to the left
	secondaryBtn := miniapp.NewSecondaryButton("Cancel").
		SetPosition("left")

	fmt.Printf("Main Button: %+v\n", mainBtn)
	fmt.Printf("Secondary Button: %+v\n", secondaryBtn)
}
```

### Haptic Feedback & Storage Operations

```go
func configureFeedbackAndStorage() {
	haptic := &miniapp.HapticFeedback{}
	impactPayload := haptic.ImpactOccurred("medium")
	successPayload := haptic.NotificationOccurred("success")

	cloud := &miniapp.CloudStorage{}
	setItemPayload := cloud.SetItem("theme_mode", "dark")

	secure := &miniapp.SecureStorage{}
	restoreItemPayload := secure.RestoreItem("auth_session_token")

	fmt.Printf("Haptic payload: %v\n", impactPayload)
	fmt.Printf("Success payload: %v\n", successPayload)
	fmt.Printf("Cloud Storage: %v\n", setItemPayload)
	fmt.Printf("Secure Storage: %v\n", restoreItemPayload)
}
```

---

## 5. Schema Builder & Dynamic Validation

The `pkg/schema` package enables runtime validation, dynamic payload construction, and JSON Schema generation.

### Dynamic Payload Validation

```go
package main

import (
	"fmt"

	"gc/pkg/schema"
)

func buildAndValidateMessage() {
	// 1. Fetch schema from registry
	sendMsgSchema, ok := schema.GlobalRegistry.Get("sendMessage")
	if !ok {
		panic("schema not found")
	}

	// 2. Build payload with validation
	builder := schema.NewPayloadBuilder(sendMsgSchema)
	builder.
		Set("chat_id", 123456789).
		Set("text", "Hello from dynamic builder!").
		Set("parse_mode", "HTML")

	payload, err := builder.Build()
	if err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}

	fmt.Println("Payload is valid and ready to dispatch:")
	fmt.Printf("%+v\n", payload)
}
```

### Exporting Standard JSON Schema (draft-07)

```go
func exportJSONSchemas() {
	schemaObj, _ := schema.GlobalRegistry.Get("sendMessage")
	jsonSchemaString, err := schemaObj.ToJSONSchemaString()
	if err != nil {
		panic(err)
	}

	fmt.Println(jsonSchemaString)
}
```

---

## 6. Updates Handling: JSON Serialization, Dispatcher, Long Polling & Webhooks

Telegram provides two mutually exclusive mechanisms to receive incoming updates: **Long Polling** (`getUpdates`) and **Webhooks**. The SDK provides a unified `Dispatcher` that works seamlessly with both methods.

### 1. JSON Serialization & Type Detection

Every incoming update contains at most one of the optional update fields. The SDK dynamically inspects the update:

```go
package main

import (
	"fmt"
	"gc/pkg/telegram"
)

func processRawUpdate(jsonData []byte) {
	update, err := telegram.UnmarshalUpdate(jsonData)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Update ID: %d, Type: %s\n", update.UpdateID, update.Type())
	fmt.Printf("Sender: %v, Chat: %v\n", update.Sender(), update.Chat())
}
```

### 2. Setting Up the Dispatcher & Commands

The `Dispatcher` provides typed routing, command argument parsing, and middleware chaining:

```go
package main

import (
	"context"
	"fmt"
	"gc/pkg/telegram"
)

func createDispatcher() *telegram.Dispatcher {
	d := telegram.NewDispatcher()

	// Global logging middleware
	d.Use(telegram.LoggingMiddleware(func(format string, args ...any) {
		fmt.Printf(format+"\n", args...)
	}))

	// Panic recovery middleware
	d.Use(telegram.RecoveryMiddleware(func(ctx context.Context, u *telegram.Update, r any) {
		fmt.Printf("Recovered from panic on update %d: %v\n", u.UpdateID, r)
	}))

	// Handle /start command with arguments (e.g. /start referral_123)
	d.OnCommand("start", func(ctx context.Context, m *telegram.Message, args []string) error {
		fmt.Printf("Received /start from user %d with args: %v\n", m.From.ID, args)
		return nil
	})

	// Handle incoming messages
	d.OnMessage(func(ctx context.Context, m *telegram.Message) error {
		if m.Text != nil {
			fmt.Printf("Message: %s\n", *m.Text)
		}
		return nil
	})

	// Handle callback query buttons
	d.OnCallbackData("action:", func(ctx context.Context, cq *telegram.CallbackQuery) error {
		fmt.Printf("Callback triggered: %s\n", *cq.Data)
		return nil
	})

	// Handle chat join requests
	d.OnChatJoinRequest(func(ctx context.Context, cjr *telegram.ChatJoinRequest) error {
		fmt.Printf("User %d requested to join chat %s\n", cjr.From.ID, cjr.Chat.Title)
		return nil
	})

	return d
}
```

### 3. Long Polling Method

Long polling continuously requests updates from Telegram. `LongPoller` automatically removes any old webhook, advances update offsets (`offset = updateID + 1`), supports worker pools, and handles exponential backoff:

```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gc/pkg/telegram"
)

func runLongPolling() {
	client := telegram.NewClient(os.Getenv("TELEGRAM_BOT_TOKEN"))
	dispatcher := createDispatcher()

	poller := telegram.NewLongPoller(client, dispatcher,
		telegram.WithPollTimeout(30),                    // 30 second long poll
		telegram.WithPollWorkers(4),                     // 4 concurrent workers
		telegram.WithDropPendingUpdates(true),           // Drop old backlog on start
		telegram.WithPollInterval(50*time.Millisecond),  // Jitter
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Blocks until context is canceled (SIGINT/SIGTERM)
	if err := poller.Start(ctx); err != nil {
		panic(err)
	}
}
```

### 4. Webhook Method

For production environments behind HTTPS, the `WebhookHandler` implements `http.Handler` and validates the `X-Telegram-Bot-Api-Secret-Token` header:

```go
package main

import (
	"context"
	"net/http"
	"os"

	"gc/pkg/telegram"
)

func runWebhook() {
	client := telegram.NewClient(os.Getenv("TELEGRAM_BOT_TOKEN"))
	dispatcher := createDispatcher()

	secretToken := "your-secret-token-12345"
	webhookURL := "https://your-domain.com/telegram-webhook"

	// 1. Register webhook with Telegram
	ctx := context.Background()
	_, err := telegram.SetupWebhook(ctx, client, telegram.WebhookConfig{
		URL:                webhookURL,
		SecretToken:        secretToken,
		DropPendingUpdates: false,
	})
	if err != nil {
		panic(err)
	}

	// 2. Start HTTP server with WebhookHandler
	webhookHandler := telegram.NewWebhookHandler(dispatcher,
		telegram.WithWebhookSecretToken(secretToken),
		telegram.WithWebhookWorkers(8), // Async worker pool
	)

	http.Handle("/telegram-webhook", webhookHandler)
	http.ListenAndServe(":8080", nil)
}
```

---

## 7. Running the Verification Suite

Run the test suite against Telegram Bot API servers using your token via flag or environment variable:

```bash
# Using CLI flag:
go run cmd/testbot/main.go -token <YOUR_BOT_TOKEN>

# Or using Environment Variable:
# On Linux/macOS:
export TELEGRAM_BOT_TOKEN="<YOUR_BOT_TOKEN>"
go run cmd/testbot/main.go

---

## 8. Example Bot: Live Crypto & TON Converter (`examples/crypto_converter_bot`)

A complete, production-grade example bot that handles realtime cryptocurrency rates, conversions, and 24h market trends across **Direct Chat**, **Groups**, and **Inline Mode**.

### Supported Queries

The bot features a flexible natural-language tokenizer that parses queries regardless of spacing, capitalization, or order:
- **Single Tokens / Prices**: `1 ton`, `1 TOn`, `1Ton`, `1grAm` (Gram = TON), `gram1`, `ton1`, `gram   1`, `gram`, `ton`, `usdt`
- **Conversions**: `500ton tron`, `tron 500`, `tron ton`, `usdt tron`, `tron usdt`, `500 ton to tron`, `btc to usdt`

### Architecture & Components

1. **`rates.go` (Multi-Source Realtime Engine)**:
   - Queries **TonAPI.io** first for TON, TRX, and USDT rates with 24-hour differential strings.
   - Falls back gracefully to **Binance**, **CoinGecko**, and **OKX** on timeouts or connection issues.
   - In-memory 20-second cache prevents rate limiting while keeping data accurate.

2. **`parser.go` (Flexible Tokenizer)**:
   - Normalizes text, filters filler words (`to`, `into`, `convert`), and handles glued numbers (`500ton` -> `500` + `TON`).
   - Maps aliases: `gram` -> `TON`, `tron` -> `TRX`, `tether` / `$` -> `USDT`.

3. **`main.go` (Dispatcher & Multi-Mode Routing)**:
   - **Direct Chats & Groups**: Responds directly with formatted HTML prices and inline keyboards.
   - **Inline Mode**: Responds to `@Giftshaper_bot <query>` with instantaneous `InlineQueryResultArticle` cards that users can send to any chat.
   - **Interactive Callbacks**: Tapping `[🔄 Refresh Rate]` instantly refreshes the message with the latest live market price.

### Running the Example

```bash
# Using flag:
go run ./examples/crypto_converter_bot -token <YOUR_BOT_TOKEN>

# Or using environment variable:
$env:TELEGRAM_BOT_TOKEN="<YOUR_BOT_TOKEN>"
go run ./examples/crypto_converter_bot
```

---

## 9. Storage Tier: FastCache & Pebble (`pkg/storage`)

The storage tier provides a unified dual-tier caching and durable persistence architecture:

```go
package main

import (
	"time"
	"gc/pkg/storage"
)

func main() {
	store, err := storage.New(storage.Config{
		PebblePath:     "./data/pebble",
		FastCacheBytes: 64 * 1024 * 1024, // 64 MB zero-GC in-memory cache
	})
	if err != nil {
		panic(err)
	}
	defer store.Close()

	// 1. In-Memory FastCache with TTL and Idempotency
	store.Fast.Set("key:1", []byte("value"), 5*time.Minute)
	
	// Atomic check-and-set idempotency key (returns true if first, false if duplicate)
	if store.Fast.SetIdempotency("req:9988", []byte("1"), 24*time.Hour) {
		// Process request
	}

	// Nonce replay check
	if !store.Fast.CheckNonce("nonce_abc123", 10*time.Minute) {
		// Replay detected!
	}

	// 2. Durable Pebble LSM-Tree with Atomic Batches
	batch := store.Pebble.NewBatch()
	_ = batch.Set([]byte(storage.NSAccount+"user:1:main:TON"), []byte("..."))
	_ = batch.Commit(true) // Synchronous disk WAL flush

	// Range prefix iteration
	_ = store.Pebble.PrefixScan([]byte(storage.NSAccount), func(k, v []byte) bool {
		// iterate accounts
		return true // continue
	})
}
```

---

## 10. Double-Entry Financial Ledger (`pkg/ledger`)

The ledger provides a strict double-entry accounting engine where $\sum \text{Debits} == \sum \text{Credits}$ for all operations. All amounts are stored in atomic `uint64` units to prevent any floating-point precision loss.

### Assets & Units
- **Telegram Stars (`XTR`)**: 0 decimals (integer Stars).
- **TON (`TON`)**: 9 decimals ($10^9$ nanoTON).
- **USDT (`USDT`)**: 6 decimals ($10^6$ microUSDT).

### Deposit, Withdraw, and Transfer

```go
package main

import (
	"context"
	"fmt"
	"gc/pkg/ledger"
	"gc/pkg/storage"
)

func main() {
	store, _ := storage.New(storage.Config{PebblePath: "./data/pebble"})
	defer store.Close()
	led := ledger.New(store)
	ctx := context.Background()

	userID := int64(12345)

	// 1. Deposit 5.5 TON (5,500,000,000 nanoTON)
	depositAmount := ledger.TONToNano(5.5)
	_, _ = led.Deposit(ctx, userID, ledger.AssetTON, depositAmount, "dep_tx_001", "User deposit")

	// 2. Query Balance
	acc, _ := led.GetAccount(ctx, ledger.UserMainAccount(userID, ledger.AssetTON))
	fmt.Printf("User Available: %s\n", ledger.FormatAtomic(acc.Available, ledger.AssetTON)) // 5.5 TON

	// 3. Instant Internal Transfer (User 12345 -> User 67890)
	_, _ = led.InternalTransfer(ctx, 12345, 67890, ledger.AssetTON, ledger.TONToNano(1.0), "xfr_002", "Gift")

	// 4. Two-Phase Transfer (Reserve -> Settle / Void)
	transferID := "escrow_tx_999"
	pt, _ := led.Reserve(ctx, transferID, 12345, ledger.UserMainAccount(67890, ledger.AssetTON), ledger.AssetTON, ledger.TONToNano(2.0), nil)
	// Funds are now moved from user:12345:main:TON to user:12345:locked:TON

	// Option A: Settle when condition met
	_ = led.Settle(ctx, transferID)

	// Option B: Void to refund locked funds back to available
	// _ = led.Void(ctx, transferID, "user cancelled order")
}
```

---

## 11. Multi-Asset Income Verifier & Transactional Outbox

### Multi-Asset Verifier (`pkg/ledger/verifier.go`)
Validates incoming payments from Telegram Stars, on-chain TON coin, and Jetton USDT:

```go
pipeline := ledger.NewIncomePipeline(led, store)

// Telegram Stars pre-checkout validation
ok, errMsg, err := pipeline.ValidatePreCheckout(ctx, preCheckoutQuery)

// Telegram Stars successful payment credit
entry, err := pipeline.ProcessStarsPayment(ctx, successfulPayment, user)

// TON on-chain deposit with memo routing
entry, err := pipeline.ProcessTONDeposit(ctx, txHash, fromAddr, toAddr, amountNano, "user_12345")

// Jetton USDT deposit
entry, err := pipeline.ProcessJettonDeposit(ctx, txHash, "USDT", fromAddr, amountMicro, "12345")
```

### Outbound Transactional Outbox & Signer (`pkg/wallet`)

Guarantees zero double-spends and safe state transitions for withdrawals:

```go
signer, _ := wallet.NewMockSigner()
outbox := wallet.NewOutboxManager(store, led, signer, wallet.OutboxConfig{
	MaxBatchSize: 254, // Highload Wallet v3 limit
})

// 1. Enqueue (automatically reserves user funds in ledger)
tx, _ := outbox.Enqueue(ctx, userID, ledger.AssetTON, ledger.TONToNano(3.0), "EQDestAddress...", "memo")

// 2. Batch Sign (bundles queued transfers into a Highload v3 message)
batch, _ := outbox.ProcessNextBatch(ctx)

// 3. Broadcast to TON network
_ = outbox.MarkBroadcasted(ctx, tx.ID, "0xchain_hash...")

// 4. Confirm on-chain finalize (settles ledger transfer)
_ = outbox.Confirm(ctx, tx.ID, "0xchain_hash...")

// If broadcast fails permanently:
// _ = outbox.Fail(ctx, tx.ID, "network error") // Automatically refunds user funds
```

---

## 12. Tickless Loop-of-Loops Workflow Engine (`pkg/workflow`)

An advanced workflow engine executing outer cycle loops and inner batch steps with tickless state suspension:

```go
wfEngine := workflow.NewEngine(store, led, botClient)

// Register Workflow Schema
wf := &workflow.WorkflowDef{
	ID:   "p2p_trade",
	Name: "P2P Trade Workflow",
	Steps: []workflow.StepDef{
		{
			ID:   "reserve_step",
			Type: workflow.StepActionLedgerReserve,
			Params: map[string]any{
				"from_user":  int64(1001),
				"to_account": ledger.UserMainAccount(2002, ledger.AssetTON),
				"asset":      ledger.AssetTON,
				"amount":     ledger.TONToNano(5.0),
			},
		},
		{
			ID:   "wait_for_confirmation",
			Type: workflow.StepActionWaitEvent,
			Params: map[string]any{
				"event": "buyer_confirmed",
			},
		},
		{
			ID:   "settle_step",
			Type: workflow.StepActionLedgerSettle,
		},
	},
}
_ = wfEngine.RegisterWorkflow(wf)

// 1. Start Instance: Runs until wait_for_confirmation, serializes state to Pebble, and halts goroutine
inst, _ := wfEngine.StartInstance(ctx, "p2p_trade", nil)
fmt.Println(inst.State) // "SUSPENDED"

// 2. Resume when external event occurs
resumed, _ := wfEngine.ResumeInstance(ctx, inst.InstanceID, "buyer_confirmed", "confirmed_by_buyer")
fmt.Println(resumed.State) // "COMPLETED"
```

---

## 13. Full End-to-End Integration Demo (`examples/ledger_workflow_demo`)

Run the complete integrated demonstration combining the storage tier, double-entry ledger, outbox manager, and tickless workflow engine:

```bash
go run ./examples/ledger_workflow_demo
```


