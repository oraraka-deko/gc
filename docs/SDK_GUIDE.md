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

## 6. Running the Verification Suite

Run the test suite against Telegram Bot API servers using your token via flag or environment variable:

```bash
# Using CLI flag:
go run cmd/testbot/main.go -token <YOUR_BOT_TOKEN>

# Or using Environment Variable:
# On Linux/macOS:
export TELEGRAM_BOT_TOKEN="<YOUR_BOT_TOKEN>"
go run cmd/testbot/main.go

# On Windows (PowerShell):
$env:TELEGRAM_BOT_TOKEN="<YOUR_BOT_TOKEN>"
go run cmd/testbot/main.go
```
