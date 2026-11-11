package telegram_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gc/pkg/telegram"
)

func TestClient_GetMe_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot12345/getMe" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := telegram.ApiResponse[telegram.User]{
			Ok: true,
			Result: telegram.User{
				ID:        1234567,
				IsBot:     true,
				FirstName: "TestBot",
				Username:  telegramPtr("test_bot"),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := telegram.NewClient("12345", telegram.WithBaseURL(server.URL))
	user, err := client.GetMe(context.Background())
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}
	if user.ID != 1234567 || user.FirstName != "TestBot" {
		t.Errorf("unexpected user: %+v", user)
	}
}

func TestClient_SendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot12345/sendMessage" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var payload telegram.SendMessageParams
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if payload.Text != "Hello world" {
			t.Errorf("unexpected text: %s", payload.Text)
		}

		resp := telegram.ApiResponse[telegram.Message]{
			Ok: true,
			Result: telegram.Message{
				MessageID: 100,
				Date:      time.Now().Unix(),
				Text:      &payload.Text,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := telegram.NewClient("12345", telegram.WithBaseURL(server.URL))
	msg, err := client.SendMessage(context.Background(), &telegram.SendMessageParams{
		ChatID: 987654,
		Text:   "Hello world",
	})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msg.MessageID != 100 || *msg.Text != "Hello world" {
		t.Errorf("unexpected message: %+v", msg)
	}
}

func TestClient_MiniAppMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bot12345/answerWebAppQuery":
			resp := telegram.ApiResponse[telegram.SentWebAppMessage]{
				Ok: true,
				Result: telegram.SentWebAppMessage{
					InlineMessageID: telegramPtr("inline_msg_123"),
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case "/bot12345/savePreparedInlineMessage":
			resp := telegram.ApiResponse[telegram.PreparedInlineMessage]{
				Ok: true,
				Result: telegram.PreparedInlineMessage{
					ID:             "prep_msg_456",
					ExpirationDate: time.Now().Add(time.Hour).Unix(),
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case "/bot12345/savePreparedKeyboardButton":
			resp := telegram.ApiResponse[telegram.PreparedKeyboardButton]{
				Ok: true,
				Result: telegram.PreparedKeyboardButton{
					ID: "prep_btn_789",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := telegram.NewClient("12345", telegram.WithBaseURL(server.URL))

	// 1. AnswerWebAppQuery
	ans, err := client.AnswerWebAppQuery(context.Background(), &telegram.AnswerWebAppQueryParams{
		WebAppQueryID: "query_123",
		Result:        map[string]any{"type": "article", "id": "1", "title": "Test"},
	})
	if err != nil || *ans.InlineMessageID != "inline_msg_123" {
		t.Fatalf("AnswerWebAppQuery failed: %v", err)
	}

	// 2. SavePreparedInlineMessage
	prepMsg, err := client.SavePreparedInlineMessage(context.Background(), &telegram.SavePreparedInlineMessageParams{
		UserID: 123456,
		Result: map[string]any{"type": "article", "id": "1", "title": "Test"},
	})
	if err != nil || prepMsg.ID != "prep_msg_456" {
		t.Fatalf("SavePreparedInlineMessage failed: %v", err)
	}

	// 3. SavePreparedKeyboardButton
	prepBtn, err := client.SavePreparedKeyboardButton(context.Background(), &telegram.SavePreparedKeyboardButtonParams{
		UserID: 123456,
		Button: telegram.KeyboardButton{Text: "Launch Mini App"},
	})
	if err != nil || prepBtn.ID != "prep_btn_789" {
		t.Fatalf("SavePreparedKeyboardButton failed: %v", err)
	}
}

func TestClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := telegram.ApiResponse[telegram.User]{
			Ok:          false,
			ErrorCode:   401,
			Description: "Unauthorized",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := telegram.NewClient("invalid_token", telegram.WithBaseURL(server.URL))
	_, err := client.GetMe(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*telegram.APIError)
	if !ok {
		t.Fatalf("expected *telegram.APIError, got %T", err)
	}
	if apiErr.ErrorCode != 401 || apiErr.Description != "Unauthorized" {
		t.Errorf("unexpected API error: %+v", apiErr)
	}
}

func telegramPtr[T any](v T) *T {
	return &v
}
