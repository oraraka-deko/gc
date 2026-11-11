package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// PollerOption allows customizing LongPoller settings.
type PollerOption func(*LongPoller)

// LongPoller manages a continuous getUpdates long polling loop.
type LongPoller struct {
	client             *Client
	handler            UpdatesHandler
	offset             int
	limit              int
	timeout            int
	allowedUpdates     []string
	pollInterval       time.Duration
	workers            int
	dropPendingUpdates bool
	errorHandler       func(err error)
	stopOnce           sync.Once
	cancelFunc         context.CancelFunc
}

// NewLongPoller creates a new LongPoller with sensible defaults.
func NewLongPoller(client *Client, handler UpdatesHandler, opts ...PollerOption) *LongPoller {
	p := &LongPoller{
		client:         client,
		handler:        handler,
		limit:          100,
		timeout:        30, // 30 seconds long-poll timeout
		pollInterval:   0,
		workers:        1,  // Sequential dispatching by default
		errorHandler:   func(err error) {},
	}

	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithPollTimeout sets the long-polling timeout in seconds (0-50).
func WithPollTimeout(seconds int) PollerOption {
	return func(p *LongPoller) {
		p.timeout = seconds
	}
}

// WithPollLimit sets maximum updates to fetch per request (1-100).
func WithPollLimit(limit int) PollerOption {
	return func(p *LongPoller) {
		p.limit = limit
	}
}

// WithPollOffset sets initial update offset.
func WithPollOffset(offset int) PollerOption {
	return func(p *LongPoller) {
		p.offset = offset
	}
}

// WithPollAllowedUpdates sets which update types to receive.
func WithPollAllowedUpdates(allowed ...string) PollerOption {
	return func(p *LongPoller) {
		p.allowedUpdates = allowed
	}
}

// WithPollInterval sets an optional sleep interval between long-poll requests.
func WithPollInterval(interval time.Duration) PollerOption {
	return func(p *LongPoller) {
		p.pollInterval = interval
	}
}

// WithPollWorkers configures concurrent worker goroutines for update dispatching.
func WithPollWorkers(workers int) PollerOption {
	return func(p *LongPoller) {
		if workers < 1 {
			workers = 1
		}
		p.workers = workers
	}
}

// WithDropPendingUpdates ensures old pending updates are dropped before polling starts.
func WithDropPendingUpdates(drop bool) PollerOption {
	return func(p *LongPoller) {
		p.dropPendingUpdates = drop
	}
}

// WithPollingErrorHandler sets an error handler for network or API errors.
func WithPollingErrorHandler(fn func(err error)) PollerOption {
	return func(p *LongPoller) {
		if fn != nil {
			p.errorHandler = fn
		}
	}
}

// Start begins the long polling loop and blocks until the context is canceled or a fatal error occurs.
func (p *LongPoller) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	p.cancelFunc = cancel
	defer cancel()

	// 1. Ensure webhook is removed before polling, since getUpdates and webhooks are mutually exclusive
	if _, err := p.client.DeleteWebhook(ctx, p.dropPendingUpdates); err != nil {
		p.errorHandler(fmt.Errorf("deleteWebhook warning before polling: %w", err))
	}

	// 2. Setup workers channel if concurrent processing is enabled
	var (
		jobsChan chan *Update
		wg       sync.WaitGroup
	)

	if p.workers > 1 {
		jobsChan = make(chan *Update, p.workers*2)
		for i := 0; i < p.workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for u := range jobsChan {
					if err := p.handler.Handle(ctx, u); err != nil {
						p.errorHandler(fmt.Errorf("handler error for update %d: %w", u.UpdateID, err))
					}
				}
			}()
		}
		defer func() {
			close(jobsChan)
			wg.Wait()
		}()
	}

	backoff := 500 * time.Millisecond
	maxBackoff := 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		params := &GetUpdatesParams{
			Offset:         &p.offset,
			Limit:          &p.limit,
			Timeout:        &p.timeout,
			AllowedUpdates: p.allowedUpdates,
		}

		updates, err := p.client.GetUpdates(ctx, params)
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}

			p.errorHandler(fmt.Errorf("getUpdates failed: %w", err))

			// Exponential backoff on errors
			select {
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			case <-ctx.Done():
				return nil
			}
			continue
		}

		// Reset backoff upon successful request
		backoff = 500 * time.Millisecond

		// Dispatch updates and advance offset
		for i := range updates {
			u := &updates[i]

			// Telegram offset rule: updateID + 1 to confirm receipt
			if u.UpdateID >= p.offset {
				p.offset = u.UpdateID + 1
			}

			if p.workers > 1 {
				select {
				case jobsChan <- u:
				case <-ctx.Done():
					return nil
				}
			} else {
				if err := p.handler.Handle(ctx, u); err != nil {
					p.errorHandler(fmt.Errorf("handler error for update %d: %w", u.UpdateID, err))
				}
			}
		}

		if p.pollInterval > 0 {
			select {
			case <-time.After(p.pollInterval):
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// Stop gracefully stops the polling loop.
func (p *LongPoller) Stop() {
	p.stopOnce.Do(func() {
		if p.cancelFunc != nil {
			p.cancelFunc()
		}
	})
}
