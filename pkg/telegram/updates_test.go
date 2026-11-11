package telegram_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gc/pkg/telegram"
)

func TestUpdate_Unmarshal_And_Types(t *testing.T) {
	testCases := []struct {
		name         string
		jsonPayload  string
		expectedType telegram.UpdateType
		expectedText string
		hasSender    bool
		hasChat      bool
	}{
		{
			name: "Regular Message",
			jsonPayload: `{
				"update_id": 10001,
				"message": {
					"message_id": 1,
					"date": 1700000000,
					"chat": {"id": 12345, "type": "private"},
					"from": {"id": 67890, "is_bot": false, "first_name": "Alice"},
					"text": "/start hello"
				}
			}`,
			expectedType: telegram.UpdateTypeMessage,
			expectedText: "/start hello",
			hasSender:    true,
			hasChat:      true,
		},
		{
			name: "Callback Query",
			jsonPayload: `{
				"update_id": 10002,
				"callback_query": {
					"id": "cb_123",
					"from": {"id": 67890, "is_bot": false, "first_name": "Alice"},
					"chat_instance": "inst_1",
					"data": "action:confirm"
				}
			}`,
			expectedType: telegram.UpdateTypeCallbackQuery,
			expectedText: "action:confirm",
			hasSender:    true,
			hasChat:      false,
		},
		{
			name: "Purchased Paid Media",
			jsonPayload: `{
				"update_id": 10003,
				"purchased_paid_media": {
					"from": {"id": 67890, "is_bot": false, "first_name": "Alice"},
					"paid_media_payload": "premium_pack_1"
				}
			}`,
			expectedType: telegram.UpdateTypePurchasedPaidMedia,
			hasSender:    true,
			hasChat:      false,
		},
		{
			name: "Guest Message",
			jsonPayload: `{
				"update_id": 10004,
				"guest_message": {
					"message_id": 2,
					"date": 1700000001,
					"chat": {"id": 12345, "type": "private"},
					"from": {"id": 67890, "is_bot": false, "first_name": "Alice"},
					"guest_query_id": "gq_456",
					"text": "Hello guest"
				}
			}`,
			expectedType: telegram.UpdateTypeGuestMessage,
			expectedText: "",
			hasSender:    true,
			hasChat:      true,
		},
		{
			name: "Chat Join Request",
			jsonPayload: `{
				"update_id": 10005,
				"chat_join_request": {
					"chat": {"id": -1001234567, "type": "supergroup", "title": "Dev Group"},
					"from": {"id": 67890, "is_bot": false, "first_name": "Alice"},
					"user_chat_id": 67890,
					"date": 1700000002
				}
			}`,
			expectedType: telegram.UpdateTypeChatJoinRequest,
			hasSender:    true,
			hasChat:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := telegram.UnmarshalUpdate([]byte(tc.jsonPayload))
			if err != nil {
				t.Fatalf("failed to unmarshal update: %v", err)
			}

			if u.Type() != tc.expectedType {
				t.Errorf("expected type %s, got %s", tc.expectedType, u.Type())
			}

			if tc.expectedText != "" && u.Text() != tc.expectedText {
				t.Errorf("expected text %q, got %q", tc.expectedText, u.Text())
			}

			if tc.hasSender && u.Sender() == nil {
				t.Errorf("expected sender, got nil")
			}

			if tc.hasChat && u.Chat() == nil {
				t.Errorf("expected chat, got nil")
			}

			// Test re-marshaling
			marshaled, err := telegram.MarshalUpdate(u)
			if err != nil || len(marshaled) == 0 {
				t.Fatalf("failed to re-marshal update: %v", err)
			}
		})
	}
}

func TestDispatcher_Routing_And_Middlewares(t *testing.T) {
	d := telegram.NewDispatcher()

	var order []string
	var commandReceived string
	var commandArgs []string
	var callbackReceived string

	// Middlewares
	d.Use(func(next telegram.HandlerFunc) telegram.HandlerFunc {
		return func(ctx context.Context, u *telegram.Update) error {
			order = append(order, "mw1_start")
			err := next(ctx, u)
			order = append(order, "mw1_end")
			return err
		}
	})

	d.Use(func(next telegram.HandlerFunc) telegram.HandlerFunc {
		return func(ctx context.Context, u *telegram.Update) error {
			order = append(order, "mw2_start")
			err := next(ctx, u)
			order = append(order, "mw2_end")
			return err
		}
	})

	// Register Command
	d.OnCommand("start", func(ctx context.Context, m *telegram.Message, args []string) error {
		order = append(order, "command_start")
		commandReceived = "start"
		commandArgs = args
		return nil
	})

	// Register Callback query prefix
	d.OnCallbackData("btn:", func(ctx context.Context, cq *telegram.CallbackQuery) error {
		order = append(order, "callback_btn")
		callbackReceived = *cq.Data
		return nil
	})

	ctx := context.Background()

	// 1. Dispatch command update
	cmdText := "/start arg1 arg2"
	updateCmd := &telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 10,
			Text:      &cmdText,
		},
	}

	if err := d.Handle(ctx, updateCmd); err != nil {
		t.Fatalf("dispatch command failed: %v", err)
	}

	if commandReceived != "start" || len(commandArgs) != 2 || commandArgs[0] != "arg1" || commandArgs[1] != "arg2" {
		t.Errorf("unexpected command parsed: %s, args: %v", commandReceived, commandArgs)
	}

	// Verify middleware order
	expectedOrder := []string{"mw1_start", "mw2_start", "command_start", "mw2_end", "mw1_end"}
	if fmt.Sprintf("%v", order) != fmt.Sprintf("%v", expectedOrder) {
		t.Errorf("unexpected middleware execution order: %v, expected: %v", order, expectedOrder)
	}

	// 2. Dispatch callback query update
	cbData := "btn:submit"
	updateCb := &telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb_1",
			Data: &cbData,
		},
	}

	if err := d.Handle(ctx, updateCb); err != nil {
		t.Fatalf("dispatch callback failed: %v", err)
	}

	if callbackReceived != "btn:submit" {
		t.Errorf("unexpected callback data: %s", callbackReceived)
	}
}

func TestDispatcher_RecoveryMiddleware(t *testing.T) {
	d := telegram.NewDispatcher()
	var recoveredPanic any

	d.Use(telegram.RecoveryMiddleware(func(ctx context.Context, u *telegram.Update, panicVal any) {
		recoveredPanic = panicVal
	}))

	d.OnMessage(func(ctx context.Context, m *telegram.Message) error {
		panic("simulated fatal crash")
	})

	text := "boom"
	update := &telegram.Update{
		UpdateID: 1,
		Message:  &telegram.Message{Text: &text},
	}

	err := d.Handle(context.Background(), update)
	if err == nil {
		t.Fatal("expected error from panic, got nil")
	}
	if recoveredPanic != "simulated fatal crash" {
		t.Errorf("unexpected panic caught: %v", recoveredPanic)
	}
}

func TestLongPoller_PollingLoop(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bot12345/deleteWebhook":
			resp := telegram.ApiResponse[bool]{Ok: true, Result: true}
			_ = json.NewEncoder(w).Encode(resp)
		case "/bot12345/getUpdates":
			count := atomic.AddInt32(&requestCount, 1)
			var updates []telegram.Update
			if count == 1 {
				msgText := "Update 1"
				updates = []telegram.Update{
					{
						UpdateID: 500,
						Message:  &telegram.Message{MessageID: 1, Text: &msgText},
					},
					{
						UpdateID: 501,
						Message:  &telegram.Message{MessageID: 2, Text: &msgText},
					},
				}
			}
			resp := telegram.ApiResponse[[]telegram.Update]{Ok: true, Result: updates}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := telegram.NewClient("12345", telegram.WithBaseURL(server.URL))
	d := telegram.NewDispatcher()

	var processedUpdates []int
	var mu sync.Mutex

	d.OnMessage(func(ctx context.Context, m *telegram.Message) error {
		mu.Lock()
		processedUpdates = append(processedUpdates, m.MessageID)
		mu.Unlock()
		return nil
	})

	poller := telegram.NewLongPoller(client, d,
		telegram.WithPollTimeout(1),
		telegram.WithPollInterval(10*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := poller.Start(ctx)
	if err != nil {
		t.Fatalf("poller failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(processedUpdates) != 2 || processedUpdates[0] != 1 || processedUpdates[1] != 2 {
		t.Errorf("unexpected processed updates: %v", processedUpdates)
	}
}

func TestWebhookHandler_ServeHTTP(t *testing.T) {
	d := telegram.NewDispatcher()
	var receivedUpdateID int

	d.OnMessage(func(ctx context.Context, m *telegram.Message) error {
		receivedUpdateID = m.MessageID
		return nil
	})

	secret := "super-secret-token"
	webhook := telegram.NewWebhookHandler(d, telegram.WithWebhookSecretToken(secret))

	// 1. Method Not Allowed (GET)
	reqGet := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	recGet := httptest.NewRecorder()
	webhook.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", recGet.Code)
	}

	// 2. Unauthorized (Missing or Invalid Secret Token)
	body := []byte(`{"update_id": 999, "message": {"message_id": 77}}`)
	reqUnauthorized := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	reqUnauthorized.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-token")
	recUnauthorized := httptest.NewRecorder()
	webhook.ServeHTTP(recUnauthorized, reqUnauthorized)
	if recUnauthorized.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", recUnauthorized.Code)
	}

	// 3. Success (Valid POST with Secret Token)
	reqValid := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	reqValid.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	recValid := httptest.NewRecorder()
	webhook.ServeHTTP(recValid, reqValid)
	if recValid.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", recValid.Code)
	}
	if receivedUpdateID != 77 {
		t.Errorf("expected update to be dispatched with message ID 77, got %d", receivedUpdateID)
	}
}
