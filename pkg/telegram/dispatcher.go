package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// UpdatesHandler defines the standard interface for processing incoming Telegram updates.
type UpdatesHandler interface {
	Handle(ctx context.Context, u *Update) error
}

// HandlerFunc is an adapter to allow the use of ordinary functions as UpdatesHandlers.
type HandlerFunc func(ctx context.Context, u *Update) error

func (f HandlerFunc) Handle(ctx context.Context, u *Update) error {
	return f(ctx, u)
}

// Middleware wraps a HandlerFunc with pre- or post-processing logic.
type Middleware func(next HandlerFunc) HandlerFunc

// CommandHandler handles command messages with parsed arguments.
type CommandHandler func(ctx context.Context, m *Message, args []string) error

// Dispatcher coordinates update routing, command parsing, and middleware chains.
type Dispatcher struct {
	mu               sync.RWMutex
	middlewares      []Middleware
	typeHandlers     map[UpdateType][]HandlerFunc
	commands         map[string]CommandHandler
	callbackPrefixes map[string]func(ctx context.Context, cq *CallbackQuery) error
	unhandledHandler HandlerFunc
}

// NewDispatcher initializes an empty Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		typeHandlers:     make(map[UpdateType][]HandlerFunc),
		commands:         make(map[string]CommandHandler),
		callbackPrefixes: make(map[string]func(ctx context.Context, cq *CallbackQuery) error),
	}
}

// Use adds one or more global middlewares to the execution pipeline.
func (d *Dispatcher) Use(mw ...Middleware) *Dispatcher {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.middlewares = append(d.middlewares, mw...)
	return d
}

// On registers a generic handler for a specific UpdateType.
func (d *Dispatcher) On(t UpdateType, h HandlerFunc) *Dispatcher {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.typeHandlers[t] = append(d.typeHandlers[t], h)
	return d
}

// OnUnhandled registers a fallback handler for updates with no matching handlers.
func (d *Dispatcher) OnUnhandled(h HandlerFunc) *Dispatcher {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.unhandledHandler = h
	return d
}

// OnMessage registers a handler for incoming messages.
func (d *Dispatcher) OnMessage(fn func(ctx context.Context, m *Message) error) *Dispatcher {
	return d.On(UpdateTypeMessage, func(ctx context.Context, u *Update) error {
		if u.Message != nil {
			return fn(ctx, u.Message)
		}
		return nil
	})
}

// OnCommand registers a handler for a specific text command (e.g. "start", "/help", "settings").
func (d *Dispatcher) OnCommand(command string, handler CommandHandler) *Dispatcher {
	d.mu.Lock()
	defer d.mu.Unlock()
	cmd := strings.ToLower(strings.TrimPrefix(command, "/"))
	d.commands[cmd] = handler
	return d
}

// OnEditedMessage registers a handler for edited messages.
func (d *Dispatcher) OnEditedMessage(fn func(ctx context.Context, m *Message) error) *Dispatcher {
	return d.On(UpdateTypeEditedMessage, func(ctx context.Context, u *Update) error {
		if u.EditedMessage != nil {
			return fn(ctx, u.EditedMessage)
		}
		return nil
	})
}

// OnChannelPost registers a handler for incoming channel posts.
func (d *Dispatcher) OnChannelPost(fn func(ctx context.Context, m *Message) error) *Dispatcher {
	return d.On(UpdateTypeChannelPost, func(ctx context.Context, u *Update) error {
		if u.ChannelPost != nil {
			return fn(ctx, u.ChannelPost)
		}
		return nil
	})
}

// OnCallbackQuery registers a generic handler for inline keyboard callback queries.
func (d *Dispatcher) OnCallbackQuery(fn func(ctx context.Context, cq *CallbackQuery) error) *Dispatcher {
	return d.On(UpdateTypeCallbackQuery, func(ctx context.Context, u *Update) error {
		if u.CallbackQuery != nil {
			return fn(ctx, u.CallbackQuery)
		}
		return nil
	})
}

// OnCallbackData registers a handler for callback queries whose data begins with prefix.
func (d *Dispatcher) OnCallbackData(prefix string, fn func(ctx context.Context, cq *CallbackQuery) error) *Dispatcher {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.callbackPrefixes[prefix] = fn
	return d
}

// OnInlineQuery registers a handler for inline queries.
func (d *Dispatcher) OnInlineQuery(fn func(ctx context.Context, iq *InlineQuery) error) *Dispatcher {
	return d.On(UpdateTypeInlineQuery, func(ctx context.Context, u *Update) error {
		if u.InlineQuery != nil {
			return fn(ctx, u.InlineQuery)
		}
		return nil
	})
}

// OnChosenInlineResult registers a handler for chosen inline results.
func (d *Dispatcher) OnChosenInlineResult(fn func(ctx context.Context, cir *ChosenInlineResult) error) *Dispatcher {
	return d.On(UpdateTypeChosenInlineResult, func(ctx context.Context, u *Update) error {
		if u.ChosenInlineResult != nil {
			return fn(ctx, u.ChosenInlineResult)
		}
		return nil
	})
}

// OnChatJoinRequest registers a handler for chat join requests.
func (d *Dispatcher) OnChatJoinRequest(fn func(ctx context.Context, cjr *ChatJoinRequest) error) *Dispatcher {
	return d.On(UpdateTypeChatJoinRequest, func(ctx context.Context, u *Update) error {
		if u.ChatJoinRequest != nil {
			return fn(ctx, u.ChatJoinRequest)
		}
		return nil
	})
}

// OnChatMember registers a handler for chat member status changes.
func (d *Dispatcher) OnChatMember(fn func(ctx context.Context, cm *ChatMemberUpdated) error) *Dispatcher {
	return d.On(UpdateTypeChatMember, func(ctx context.Context, u *Update) error {
		if u.ChatMember != nil {
			return fn(ctx, u.ChatMember)
		}
		return nil
	})
}

// OnMyChatMember registers a handler for bot's own member status changes.
func (d *Dispatcher) OnMyChatMember(fn func(ctx context.Context, cm *ChatMemberUpdated) error) *Dispatcher {
	return d.On(UpdateTypeMyChatMember, func(ctx context.Context, u *Update) error {
		if u.MyChatMember != nil {
			return fn(ctx, u.MyChatMember)
		}
		return nil
	})
}

// OnPoll registers a handler for poll state changes.
func (d *Dispatcher) OnPoll(fn func(ctx context.Context, p *Poll) error) *Dispatcher {
	return d.On(UpdateTypePoll, func(ctx context.Context, u *Update) error {
		if u.Poll != nil {
			return fn(ctx, u.Poll)
		}
		return nil
	})
}

// OnPollAnswer registers a handler for non-anonymous poll answers.
func (d *Dispatcher) OnPollAnswer(fn func(ctx context.Context, pa *PollAnswer) error) *Dispatcher {
	return d.On(UpdateTypePollAnswer, func(ctx context.Context, u *Update) error {
		if u.PollAnswer != nil {
			return fn(ctx, u.PollAnswer)
		}
		return nil
	})
}

// OnMessageReaction registers a handler for message reaction changes.
func (d *Dispatcher) OnMessageReaction(fn func(ctx context.Context, mr *MessageReactionUpdated) error) *Dispatcher {
	return d.On(UpdateTypeMessageReaction, func(ctx context.Context, u *Update) error {
		if u.MessageReaction != nil {
			return fn(ctx, u.MessageReaction)
		}
		return nil
	})
}

// OnBusinessConnection registers a handler for Telegram Business connections.
func (d *Dispatcher) OnBusinessConnection(fn func(ctx context.Context, bc *BusinessConnection) error) *Dispatcher {
	return d.On(UpdateTypeBusinessConnection, func(ctx context.Context, u *Update) error {
		if u.BusinessConnection != nil {
			return fn(ctx, u.BusinessConnection)
		}
		return nil
	})
}

// OnBusinessMessage registers a handler for messages from a connected business account.
func (d *Dispatcher) OnBusinessMessage(fn func(ctx context.Context, bm *Message) error) *Dispatcher {
	return d.On(UpdateTypeBusinessMessage, func(ctx context.Context, u *Update) error {
		if u.BusinessMessage != nil {
			return fn(ctx, u.BusinessMessage)
		}
		return nil
	})
}

// OnPurchasedPaidMedia registers a handler for paid media purchases.
func (d *Dispatcher) OnPurchasedPaidMedia(fn func(ctx context.Context, ppm *PaidMediaPurchased) error) *Dispatcher {
	return d.On(UpdateTypePurchasedPaidMedia, func(ctx context.Context, u *Update) error {
		if u.PurchasedPaidMedia != nil {
			return fn(ctx, u.PurchasedPaidMedia)
		}
		return nil
	})
}

// OnGuestMessage registers a handler for guest messages.
func (d *Dispatcher) OnGuestMessage(fn func(ctx context.Context, gm *Message) error) *Dispatcher {
	return d.On(UpdateTypeGuestMessage, func(ctx context.Context, u *Update) error {
		if u.GuestMessage != nil {
			return fn(ctx, u.GuestMessage)
		}
		return nil
	})
}

// OnPreCheckoutQuery registers a handler for pre-checkout queries.
func (d *Dispatcher) OnPreCheckoutQuery(fn func(ctx context.Context, pcq *PreCheckoutQuery) error) *Dispatcher {
	return d.On(UpdateTypePreCheckoutQuery, func(ctx context.Context, u *Update) error {
		if u.PreCheckoutQuery != nil {
			return fn(ctx, u.PreCheckoutQuery)
		}
		return nil
	})
}

// OnShippingQuery registers a handler for shipping queries.
func (d *Dispatcher) OnShippingQuery(fn func(ctx context.Context, sq *ShippingQuery) error) *Dispatcher {
	return d.On(UpdateTypeShippingQuery, func(ctx context.Context, u *Update) error {
		if u.ShippingQuery != nil {
			return fn(ctx, u.ShippingQuery)
		}
		return nil
	})
}

// Handle executes the middleware chain and dispatches the update to matched handlers.
func (d *Dispatcher) Handle(ctx context.Context, u *Update) error {
	if u == nil {
		return nil
	}

	coreHandler := func(c context.Context, upd *Update) error {
		return d.dispatch(c, upd)
	}

	d.mu.RLock()
	mws := make([]Middleware, len(d.middlewares))
	copy(mws, d.middlewares)
	d.mu.RUnlock()

	// Chain middlewares in order (outermost first)
	finalHandler := coreHandler
	for i := len(mws) - 1; i >= 0; i-- {
		finalHandler = mws[i](finalHandler)
	}

	return finalHandler(ctx, u)
}

func (d *Dispatcher) dispatch(ctx context.Context, u *Update) error {
	uType := u.Type()
	handled := false

	// 1. Check for command matching if it is a message
	if uType == UpdateTypeMessage && u.Message != nil && u.Message.Text != nil {
		text := strings.TrimSpace(*u.Message.Text)
		if strings.HasPrefix(text, "/") {
			cmd, args := parseCommand(text)
			d.mu.RLock()
			cmdHandler, ok := d.commands[cmd]
			d.mu.RUnlock()
			if ok && cmdHandler != nil {
				handled = true
				if err := cmdHandler(ctx, u.Message, args); err != nil {
					return err
				}
			}
		}
	}

	// 2. Check for callback query prefix handlers
	if uType == UpdateTypeCallbackQuery && u.CallbackQuery != nil && u.CallbackQuery.Data != nil {
		data := *u.CallbackQuery.Data
		d.mu.RLock()
		for prefix, cbHandler := range d.callbackPrefixes {
			if strings.HasPrefix(data, prefix) {
				handled = true
				d.mu.RUnlock()
				if err := cbHandler(ctx, u.CallbackQuery); err != nil {
					return err
				}
				d.mu.RLock()
				break
			}
		}
		d.mu.RUnlock()
	}

	// 3. Dispatch to type handlers
	d.mu.RLock()
	handlers, hasTypeHandlers := d.typeHandlers[uType]
	unhandled := d.unhandledHandler
	d.mu.RUnlock()

	if hasTypeHandlers && len(handlers) > 0 {
		handled = true
		for _, h := range handlers {
			if err := h(ctx, u); err != nil {
				return err
			}
		}
	}

	// 4. Fallback if not handled
	if !handled && unhandled != nil {
		return unhandled(ctx, u)
	}

	return nil
}

// parseCommand splits text into command name (without / and @bot) and arguments.
func parseCommand(text string) (string, []string) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", nil
	}

	cmdToken := strings.TrimPrefix(fields[0], "/")
	// Remove bot username suffix if present: /command@BotName
	if idx := strings.Index(cmdToken, "@"); idx != -1 {
		cmdToken = cmdToken[:idx]
	}

	cmd := strings.ToLower(cmdToken)
	var args []string
	if len(fields) > 1 {
		args = fields[1:]
	}
	return cmd, args
}

// Built-in Middlewares

// RecoveryMiddleware catches any panics within handlers and passes them to recoveryHandler.
func RecoveryMiddleware(recoveryHandler func(ctx context.Context, u *Update, panicVal any)) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, u *Update) (err error) {
			defer func() {
				if r := recover(); r != nil {
					if recoveryHandler != nil {
						recoveryHandler(ctx, u, r)
					}
					err = fmt.Errorf("panic in update handler: %v", r)
				}
			}()
			return next(ctx, u)
		}
	}
}

// LoggingMiddleware logs every received update and any errors.
func LoggingMiddleware(logger func(format string, args ...any)) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, u *Update) error {
			if logger != nil {
				logger("[Update %d] type=%s text=%q sender=%v", u.UpdateID, u.Type(), u.Text(), u.Sender())
			}
			err := next(ctx, u)
			if err != nil && logger != nil {
				logger("[Update %d] handler error: %v", u.UpdateID, err)
			}
			return err
		}
	}
}
