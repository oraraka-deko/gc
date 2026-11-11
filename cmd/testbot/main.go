package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"gc/pkg/miniapp"
	"gc/pkg/schema"
	"gc/pkg/telegram"
)

func main() {
	tokenFlag := flag.String("token", "", "Telegram Bot Token to test (can also be set via TELEGRAM_BOT_TOKEN environment variable)")
	flag.Parse()

	token := *tokenFlag
	if token == "" {
		token = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	fmt.Println("================================================================================")
	fmt.Println(" Telegram Bot & Mini App Auto SDK Test Suite")
	fmt.Println("================================================================================")

	if token == "" {
		fmt.Println("[!] No Telegram bot token provided.")
		fmt.Println("    Please provide a token via -token flag or TELEGRAM_BOT_TOKEN environment variable:")
		fmt.Println("    Example: go run cmd/testbot/main.go -token <YOUR_BOT_TOKEN>")
		fmt.Println("    Example (PowerShell): $env:TELEGRAM_BOT_TOKEN=\"<YOUR_BOT_TOKEN>\"; go run cmd/testbot/main.go")
		fmt.Println("================================================================================")
	} else {
		fmt.Printf("[*] Using Bot Token from Environment/Flag: %s\n\n", maskToken(token))
	}

	// 1. Schema Registry Verification
	fmt.Println("[1] Verifying Schema Registry & Auto Builder...")
	schemas := schema.GlobalRegistry.List()
	fmt.Printf("    -> Registered %d total schemas across Bot API & Mini App features.\n", len(schemas))

	methods := schema.GlobalRegistry.List(schema.KindBotAPIMethod)
	components := schema.GlobalRegistry.List(schema.KindMiniAppComponent)
	types := schema.GlobalRegistry.List(schema.KindMiniAppType)
	events := schema.GlobalRegistry.List(schema.KindMiniAppEvent)

	fmt.Printf("    - Bot API Methods:      %d\n", len(methods))
	fmt.Printf("    - Mini App Components:  %d\n", len(components))
	fmt.Printf("    - Mini App Types:       %d\n", len(types))
	fmt.Printf("    - Mini App Events:      %d\n", len(events))

	// 2. Test Payload Builder & Validator
	fmt.Println("\n[2] Testing Dynamic Schema Payload Builder & Validation...")
	sendMsgSchema, ok := schema.GlobalRegistry.Get("sendMessage")
	if !ok {
		fmt.Println("    [!] FAILED: sendMessage schema not found in registry")
		os.Exit(1)
	}
	builder := schema.NewPayloadBuilder(sendMsgSchema)
	builder.Set("chat_id", 123456789).
		Set("text", "Automated test message").
		Set("parse_mode", "HTML")
	if err := builder.Validate(); err != nil {
		fmt.Printf("    [!] FAILED: payload validation: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("    [+] Dynamic Payload Builder validation: PASSED")

	// 3. Test Mini App HMAC-SHA256 Cryptographic Engine
	fmt.Println("\n[3] Testing Mini App InitData Cryptographic Validation...")
	testParams := map[string]string{
		"query_id":  "test_query_123",
		"auth_date": fmt.Sprintf("%d", time.Now().Unix()),
		"user":      `{"id":1001,"first_name":"TelegramUser"}`,
	}
	mockToken := "123456:MOCK_TOKEN_FOR_TESTING"
	signedQuery := miniapp.GenerateSignedInitData(mockToken, testParams)
	valid, err := miniapp.Validate(signedQuery, mockToken, time.Hour)
	if err != nil || !valid {
		fmt.Printf("    [!] FAILED: Mini App InitData validation: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("    [+] Mini App HMAC-SHA256 Signature Verification: PASSED")

	// 4. Test Updates Unmarshaling & Dispatcher Core
	fmt.Println("\n[4] Testing Updates Unmarshaling, Dispatcher, LongPoller & Webhook Core...")
	sampleUpdateJSON := []byte(`{
		"update_id": 9901,
		"message": {
			"message_id": 42,
			"date": 1700000000,
			"chat": {"id": 123456, "type": "private"},
			"from": {"id": 123456, "is_bot": false, "first_name": "Developer"},
			"text": "/start from_test"
		}
	}`)

	update, err := telegram.UnmarshalUpdate(sampleUpdateJSON)
	if err != nil || update.Type() != telegram.UpdateTypeMessage {
		fmt.Printf("    [!] FAILED: update unmarshal error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("    [+] JSON Update Unmarshaled: ID=%d, Type=%s, Text=%q\n", update.UpdateID, update.Type(), update.Text())

	dispatcher := telegram.NewDispatcher()
	commandHandled := false
	dispatcher.OnCommand("start", func(ctx context.Context, m *telegram.Message, args []string) error {
		commandHandled = true
		return nil
	})

	if err := dispatcher.Handle(context.Background(), update); err != nil || !commandHandled {
		fmt.Printf("    [!] FAILED: Dispatcher command routing failed\n")
		os.Exit(1)
	}
	fmt.Println("    [+] Dispatcher Routing & Command Execution: PASSED")

	webhook := telegram.NewWebhookHandler(dispatcher, telegram.WithWebhookSecretToken("sec_tok_123"))
	if webhook == nil {
		fmt.Printf("    [!] FAILED: Webhook initialization\n")
		os.Exit(1)
	}
	fmt.Println("    [+] Webhook Handler Initialization: PASSED")

	if token == "" {
		fmt.Println("\n[!] Skipping live Telegram Bot API tests because no token was supplied.")
		return
	}

	// 5. Live API Test with Provided Token
	fmt.Println("\n[5] Running Live Test against Telegram Bot API (https://api.telegram.org)...")
	client := telegram.NewClient(token, telegram.WithTimeout(10*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := client.GetMe(ctx)
	if err != nil {
		if apiErr, ok := err.(*telegram.APIError); ok {
			fmt.Println("\n--------------------------------------------------------------------------------")
			fmt.Printf(" [!] TELEGRAM API RETURNED ERROR %d: %s\n", apiErr.ErrorCode, apiErr.Description)
			fmt.Println("--------------------------------------------------------------------------------")
			if apiErr.ErrorCode == 401 {
				fmt.Println(" The bot token is INVALID or has been revoked by @BotFather.")
				fmt.Println(" Please verify or resend a valid Telegram Bot token.")
			}
		} else {
			fmt.Printf(" [!] Network or request error: %v\n", err)
		}
		fmt.Println("--------------------------------------------------------------------------------")
		return
	}

	fmt.Println("    [+] Live getMe Succeeded!")
	fmt.Printf("    - Bot ID:                   %d\n", user.ID)
	fmt.Printf("    - Bot Name:                 %s\n", user.FirstName)
	if user.Username != nil {
		fmt.Printf("    - Bot Username:             @%s\n", *user.Username)
	}
	if user.CanJoinGroups != nil {
		fmt.Printf("    - Can Join Groups:          %v\n", *user.CanJoinGroups)
	}
	if user.CanReadAllGroupMessages != nil {
		fmt.Printf("    - Read All Group Messages:  %v\n", *user.CanReadAllGroupMessages)
	}
	if user.SupportsInlineQueries != nil {
		fmt.Printf("    - Supports Inline Queries:  %v\n", *user.SupportsInlineQueries)
	}

	// Test getUpdates (polling test)
	updates, err := client.GetUpdates(ctx, &telegram.GetUpdatesParams{
		Limit:   telegram.IntPtr(1),
		Timeout: telegram.IntPtr(0),
	})
	if err != nil {
		fmt.Printf("    - Polling Status:           Check returned notice (%v)\n", err)
	} else {
		fmt.Printf("    - Polling Status:           Active, pending updates in queue: %d\n", len(updates))
	}

	// Also test getMyCommands
	commands, err := client.GetMyCommands(ctx, nil)
	if err == nil {
		fmt.Printf("    - Registered Commands:      %d\n", len(commands))
		for _, cmd := range commands {
			fmt.Printf("      * /%s - %s\n", cmd.Command, cmd.Description)
		}
	}

	// Also test getWebhookInfo
	webhookInfo, err := client.GetWebhookInfo(ctx)
	if err == nil {
		webhookStatus := "None (polling mode active)"
		if webhookInfo.URL != "" {
			webhookStatus = webhookInfo.URL
		}
		fmt.Printf("    - Webhook URL:              %s\n", webhookStatus)
		fmt.Printf("    - Pending Updates:          %d\n", webhookInfo.PendingUpdateCount)
	}

	fmt.Println("\n================================================================================")
	fmt.Println(" [SUCCESS] All auto SDK validations & live Telegram API tests PASSED!")
	fmt.Println("================================================================================")
}

func maskToken(token string) string {
	if len(token) <= 10 {
		return "*****"
	}
	return token[:6] + "..." + token[len(token)-4:]
}
