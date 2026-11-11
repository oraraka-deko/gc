package telegram

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// WebhookConfig contains settings for registering a webhook with Telegram.
type WebhookConfig struct {
	URL                string   `json:"url"`
	SecretToken        string   `json:"secret_token,omitempty"`
	MaxConnections     *int     `json:"max_connections,omitempty"`
	AllowedUpdates     []string `json:"allowed_updates,omitempty"`
	DropPendingUpdates bool     `json:"drop_pending_updates,omitempty"`
	IPAddress          *string  `json:"ip_address,omitempty"`
}

// WebhookOption configures the WebhookHandler.
type WebhookOption func(*WebhookHandler)

// WebhookHandler is an http.Handler that receives incoming Telegram webhook POST requests.
type WebhookHandler struct {
	handler      UpdatesHandler
	secretToken  string
	maxBodySize  int64
	workers      int
	jobsChan     chan *Update
	wg           sync.WaitGroup
	errorHandler func(w http.ResponseWriter, r *http.Request, err error)
}

// NewWebhookHandler initializes an HTTP handler for Telegram updates.
func NewWebhookHandler(handler UpdatesHandler, opts ...WebhookOption) *WebhookHandler {
	wh := &WebhookHandler{
		handler:      handler,
		maxBodySize:  10 * 1024 * 1024, // 10 MB maximum payload limit
		errorHandler: defaultWebhookErrorHandler,
	}

	for _, opt := range opts {
		opt(wh)
	}

	if wh.workers > 1 {
		wh.jobsChan = make(chan *Update, wh.workers*4)
		for i := 0; i < wh.workers; i++ {
			wh.wg.Add(1)
			go func() {
				defer wh.wg.Done()
				for u := range wh.jobsChan {
					_ = wh.handler.Handle(context.Background(), u)
				}
			}()
		}
	}

	return wh
}

// WithWebhookSecretToken sets the secret token expected in the X-Telegram-Bot-Api-Secret-Token header.
func WithWebhookSecretToken(secret string) WebhookOption {
	return func(wh *WebhookHandler) {
		wh.secretToken = secret
	}
}

// WithWebhookMaxBodySize limits the maximum request body size accepted from Telegram.
func WithWebhookMaxBodySize(maxBytes int64) WebhookOption {
	return func(wh *WebhookHandler) {
		wh.maxBodySize = maxBytes
	}
}

// WithWebhookWorkers enables asynchronous processing using a worker pool.
func WithWebhookWorkers(workers int) WebhookOption {
	return func(wh *WebhookHandler) {
		if workers < 1 {
			workers = 1
		}
		wh.workers = workers
	}
}

// WithWebhookErrorHandler sets a custom HTTP error handler.
func WithWebhookErrorHandler(fn func(w http.ResponseWriter, r *http.Request, err error)) WebhookOption {
	return func(wh *WebhookHandler) {
		if fn != nil {
			wh.errorHandler = fn
		}
	}
}

// Close gracefully flushes any queued updates when workers are enabled.
func (wh *WebhookHandler) Close() {
	if wh.jobsChan != nil {
		close(wh.jobsChan)
		wh.wg.Wait()
	}
}

// ServeHTTP implements http.Handler.
func (wh *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Telegram only sends updates via POST
	if r.Method != http.MethodPost {
		wh.errorHandler(w, r, fmt.Errorf("method not allowed: %s", r.Method))
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 2. Validate Secret Token if configured
	if wh.secretToken != "" {
		receivedToken := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if subtle.ConstantTimeCompare([]byte(receivedToken), []byte(wh.secretToken)) != 1 {
			wh.errorHandler(w, r, fmt.Errorf("unauthorized: invalid secret token"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	// 3. Read request body safely
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, wh.maxBodySize))
	_ = r.Body.Close()
	if err != nil {
		wh.errorHandler(w, r, fmt.Errorf("failed to read body: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Unmarshal update
	update, err := UnmarshalUpdate(bodyBytes)
	if err != nil {
		wh.errorHandler(w, r, fmt.Errorf("failed to unmarshal update: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 5. Acknowledge receipt to Telegram with 200 OK immediately
	w.WriteHeader(http.StatusOK)

	// 6. Dispatch update
	if wh.workers > 1 && wh.jobsChan != nil {
		wh.jobsChan <- update
	} else {
		if err := wh.handler.Handle(r.Context(), update); err != nil {
			wh.errorHandler(w, r, fmt.Errorf("handler error for update %d: %w", update.UpdateID, err))
		}
	}
}

func defaultWebhookErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	// Telegram expects standard HTTP status codes
}

// SetupWebhook registers the webhook with Telegram via the Bot API.
func SetupWebhook(ctx context.Context, client *Client, cfg WebhookConfig) (bool, error) {
	params := &SetWebhookParams{
		URL:                cfg.URL,
		IPAddress:          cfg.IPAddress,
		MaxConnections:     cfg.MaxConnections,
		AllowedUpdates:     cfg.AllowedUpdates,
		DropPendingUpdates: &cfg.DropPendingUpdates,
	}
	if cfg.SecretToken != "" {
		params.SecretToken = &cfg.SecretToken
	}

	return client.SetWebhook(ctx, params)
}

// RemoveWebhook deletes the webhook integration from Telegram.
func RemoveWebhook(ctx context.Context, client *Client, dropPendingUpdates bool) (bool, error) {
	return client.DeleteWebhook(ctx, dropPendingUpdates)
}
