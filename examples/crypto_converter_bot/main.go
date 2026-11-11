package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gc/pkg/telegram"
)

func main() {
	tokenFlag := flag.String("token", "", "Telegram Bot Token (or set TELEGRAM_BOT_TOKEN)")
	flag.Parse()

	token := *tokenFlag
	if token == "" {
		token = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	if token == "" {
		log.Fatal("ERROR: No Telegram bot token provided. Set TELEGRAM_BOT_TOKEN environment variable or pass -token flag.")
	}

	client := telegram.NewClient(token, telegram.WithTimeout(15*time.Second))
	rateMgr := NewRateManager()
	dispatcher := telegram.NewDispatcher()

	// 1. Logging and Panic Recovery Middlewares
	dispatcher.Use(telegram.LoggingMiddleware(func(format string, args ...any) {
		log.Printf(format, args...)
	}))
	dispatcher.Use(telegram.RecoveryMiddleware(func(ctx context.Context, u *telegram.Update, r any) {
		log.Printf("[PANIC RECOVERED] on update %d: %v", u.UpdateID, r)
	}))

	// 2. /start and /help command handlers
	helpHandler := func(ctx context.Context, m *telegram.Message, args []string) error {
		helpText := `👋 <b>Welcome to Crypto & TON Live Converter Bot!</b>

Send any crypto or currency query in <b>direct chat, groups, or inline mode</b>:
• <code>1 ton</code>, <code>100 gram</code>, <code>500ton tron</code>
• <code>tron 500</code>, <code>usdt tron</code>, <code>ton1</code>, <code>gram</code>
• <code>tron usdt</code>, <code>ton to usdt</code>, <code>btc</code>, <code>eth</code>

⚡ <i>Powered by realtime data from TonAPI, Binance, CoinGecko & OKX!</i>`

		replyMarkup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{
					{Text: "💎 Check TON Price", CallbackData: telegram.StringPtr("query:1 ton")},
					{Text: "🔄 500 TON ➔ TRX", CallbackData: telegram.StringPtr("query:500 ton tron")},
				},
				{
					{Text: "🌐 Use in Inline Mode", SwitchInlineQuery: telegram.StringPtr("1 ton")},
				},
			},
		}

		_, err := client.SendMessage(ctx, &telegram.SendMessageParams{
			ChatID:      m.Chat.ID,
			Text:        helpText,
			ParseMode:   telegram.StringPtr("HTML"),
			ReplyMarkup: replyMarkup,
		})
		return err
	}

	dispatcher.OnCommand("start", helpHandler)
	dispatcher.OnCommand("help", helpHandler)

	// 3. Natural Language Message Handler (Direct Chats and Groups)
	dispatcher.OnMessage(func(ctx context.Context, m *telegram.Message) error {
		if m.Text == nil {
			return nil
		}
		text := strings.TrimSpace(*m.Text)
		if text == "" || strings.HasPrefix(text, "/") {
			return nil
		}

		query, ok := ParseConversionQuery(text)
		if !ok {
			// Not a crypto query, silently ignore to avoid noise in groups
			return nil
		}

		respText, markup, err := buildConversionResponse(ctx, rateMgr, query)
		if err != nil {
			log.Printf("Error processing query %q: %v", text, err)
			return nil
		}

		_, err = client.SendMessage(ctx, &telegram.SendMessageParams{
			ChatID:      m.Chat.ID,
			Text:        respText,
			ParseMode:   telegram.StringPtr("HTML"),
			ReplyMarkup: markup,
		})
		return err
	})

	// 4. Callback Query Handler (Refresh & Quick Queries)
	dispatcher.OnCallbackData("query:", func(ctx context.Context, cq *telegram.CallbackQuery) error {
		if cq.Data == nil {
			return nil
		}
		rawQuery := strings.TrimPrefix(*cq.Data, "query:")
		query, ok := ParseConversionQuery(rawQuery)
		if !ok {
			_, _ = client.AnswerCallbackQuery(ctx, &telegram.AnswerCallbackQueryParams{
				CallbackQueryID: cq.ID,
				Text:            telegram.StringPtr("Invalid query"),
			})
			return nil
		}

		respText, markup, err := buildConversionResponse(ctx, rateMgr, query)
		if err != nil {
			_, _ = client.AnswerCallbackQuery(ctx, &telegram.AnswerCallbackQueryParams{
				CallbackQueryID: cq.ID,
				Text:            telegram.StringPtr("Error refreshing rate"),
			})
			return nil
		}

		if cq.Message != nil {
			msgID := cq.Message.MessageID
			_, _ = client.EditMessageText(ctx, &telegram.EditMessageTextParams{
				ChatID:      cq.Message.Chat.ID,
				MessageID:   &msgID,
				Text:        respText,
				ParseMode:   telegram.StringPtr("HTML"),
				ReplyMarkup: markup,
			})
		}

		_, err = client.AnswerCallbackQuery(ctx, &telegram.AnswerCallbackQueryParams{
			CallbackQueryID: cq.ID,
			Text:            telegram.StringPtr("Rates updated!"),
		})
		return err
	})

	// 5. Inline Mode Handler (@BotName query in any chat)
	dispatcher.OnInlineQuery(func(ctx context.Context, iq *telegram.InlineQuery) error {
		queryText := strings.TrimSpace(iq.Query)
		if queryText == "" {
			queryText = "1 ton"
		}

		query, ok := ParseConversionQuery(queryText)
		if !ok {
			query, _ = ParseConversionQuery("1 ton")
		}

		respText, markup, err := buildConversionResponse(ctx, rateMgr, query)
		if err != nil {
			return nil
		}

		title := fmt.Sprintf("%.2f %s ➔ %s", query.Amount, query.FromCurrency, query.ToCurrency)
		description := "Tap to share realtime rate & conversion"

		results := []any{
			map[string]any{
				"type":        "article",
				"id":          fmt.Sprintf("rate_%d", time.Now().UnixNano()),
				"title":       title,
				"description": description,
				"input_message_content": map[string]any{
					"message_text": respText,
					"parse_mode":   "HTML",
				},
				"reply_markup": markup,
			},
		}

		_, err = client.AnswerInlineQuery(ctx, &telegram.AnswerInlineQueryParams{
			InlineQueryID: iq.ID,
			Results:       results,
			CacheTime:     telegram.IntPtr(10),
		})
		return err
	})

	// 6. Verify Bot Identity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	botUser, err := client.GetMe(ctx)
	cancel()
	if err != nil {
		log.Fatalf("Failed to authenticate bot: %v", err)
	}

	log.Printf("==================================================================")
	log.Printf(" Live Crypto Converter Bot Started!")
	log.Printf(" Bot Name:     %s (@%s)", botUser.FirstName, *botUser.Username)
	log.Printf(" Bot ID:       %d", botUser.ID)
	log.Printf(" Handling:     Groups, Direct Chat, Inline Mode")
	log.Printf(" Supported:    TON/Gram, TRX/Tron, USDT, BTC, ETH")
	log.Printf("==================================================================")

	// 7. Start Long Poller
	poller := telegram.NewLongPoller(client, dispatcher,
		telegram.WithPollTimeout(25),
		telegram.WithPollWorkers(2),
		telegram.WithDropPendingUpdates(true),
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	pollCtx, pollCancel := context.WithCancel(context.Background())
	go func() {
		<-sigChan
		log.Println("Shutting down bot...")
		pollCancel()
		poller.Stop()
	}()

	if err := poller.Start(pollCtx); err != nil {
		log.Fatalf("Poller exited with error: %v", err)
	}
	log.Println("Bot stopped gracefully.")
}

func getSymbolIcon(sym string) string {
	switch sym {
	case "TON":
		return "💎"
	case "TRX":
		return "🔴"
	case "USDT", "USD":
		return "💵"
	case "BTC":
		return "₿"
	case "ETH":
		return "⟠"
	default:
		return "🪙"
	}
}

func getSymbolFullName(sym string) string {
	switch sym {
	case "TON":
		return "TON (The Open Network / Gram)"
	case "TRX":
		return "TRON (TRX)"
	case "USDT":
		return "Tether (USDT)"
	case "USD":
		return "US Dollar (USD)"
	case "BTC":
		return "Bitcoin (BTC)"
	case "ETH":
		return "Ethereum (ETH)"
	default:
		return sym
	}
}

func buildConversionResponse(ctx context.Context, rm *RateManager, q *ConversionQuery) (string, *telegram.InlineKeyboardMarkup, error) {
	fromRate, err := rm.GetRate(ctx, q.FromCurrency)
	if err != nil {
		return "", nil, fmt.Errorf("unable to fetch rate for %s: %w", q.FromCurrency, err)
	}

	fromIcon := getSymbolIcon(q.FromCurrency)
	updatedTime := time.Now().Format("15:04:05")

	var sb strings.Builder

	// Case 1: Single coin rate in USD (e.g. "ton", "1 ton", "gram", "usdt")
	if q.IsSingleCoin || q.ToCurrency == "USD" || q.ToCurrency == "USDT" {
		totalUSD := q.Amount * fromRate.PriceUSD
		sb.WriteString(fmt.Sprintf("%s <b>%s</b>\n", fromIcon, getSymbolFullName(q.FromCurrency)))
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")

		if q.Amount == 1.0 {
			sb.WriteString(fmt.Sprintf("💵 <b>Price:</b> <code>$%.4f USD</code>\n", fromRate.PriceUSD))
		} else {
			sb.WriteString(fmt.Sprintf("💵 <b>%s %s</b> = <b>$%.2f USD</b>\n", formatFloat(q.Amount), q.FromCurrency, totalUSD))
			sb.WriteString(fmt.Sprintf("📊 <b>Rate:</b> 1 %s ≈ $%.4f USD\n", q.FromCurrency, fromRate.PriceUSD))
		}

		trendEmoji := "📈"
		if strings.HasPrefix(q.FromCurrency, "-") || strings.Contains(fromRate.Change24h, "-") || strings.Contains(fromRate.Change24h, "−") {
			trendEmoji = "📉"
		}
		sb.WriteString(fmt.Sprintf("%s <b>24h Change:</b> <code>%s</code>\n", trendEmoji, fromRate.Change24h))
		sb.WriteString(fmt.Sprintf("⚡ <b>Source:</b> <i>%s</i>\n", fromRate.Source))
		sb.WriteString(fmt.Sprintf("🕒 <b>Updated:</b> %s UTC", updatedTime))
	} else {
		// Case 2: Conversion between two currencies (e.g. 500 TON to TRX, TRX to TON)
		toRate, err := rm.GetRate(ctx, q.ToCurrency)
		if err != nil {
			return "", nil, fmt.Errorf("unable to fetch rate for %s: %w", q.ToCurrency, err)
		}

		toIcon := getSymbolIcon(q.ToCurrency)
		unitExchangeRate := fromRate.PriceUSD / toRate.PriceUSD
		convertedTotal := q.Amount * unitExchangeRate
		totalUSD := q.Amount * fromRate.PriceUSD

		sb.WriteString(fmt.Sprintf("🔄 <b>%s %s ➔ %s</b>\n", formatFloat(q.Amount), q.FromCurrency, q.ToCurrency))
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString(fmt.Sprintf("%s <b>%s %s</b> = %s <b>%s %s</b>\n",
			fromIcon, formatFloat(q.Amount), q.FromCurrency,
			toIcon, formatFloat(convertedTotal), q.ToCurrency))
		sb.WriteString(fmt.Sprintf("💵 <b>Value in USD:</b> ~$%.2f USD\n", totalUSD))
		sb.WriteString(fmt.Sprintf("📊 <b>Rate:</b> 1 %s ≈ %s %s\n", q.FromCurrency, formatFloat(unitExchangeRate), q.ToCurrency))
		sb.WriteString(fmt.Sprintf("📈 <b>24h Trend:</b> %s <code>%s</code> | %s <code>%s</code>\n",
			q.FromCurrency, fromRate.Change24h, q.ToCurrency, toRate.Change24h))
		sb.WriteString(fmt.Sprintf("⚡ <b>Source:</b> <i>%s</i>\n", fromRate.Source))
		sb.WriteString(fmt.Sprintf("🕒 <b>Updated:</b> %s UTC", updatedTime))
	}

	// Create Refresh & Share button
	refreshQuery := fmt.Sprintf("%s %s", formatFloat(q.Amount), q.FromCurrency)
	if !q.IsSingleCoin && q.ToCurrency != "USD" {
		refreshQuery += " " + q.ToCurrency
	}

	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "🔄 Refresh Rate", CallbackData: telegram.StringPtr("query:" + refreshQuery)},
				{Text: "🌐 Share / Inline", SwitchInlineQuery: telegram.StringPtr(refreshQuery)},
			},
		},
	}

	return sb.String(), markup, nil
}

func formatFloat(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%.0f", v)
	}
	if v >= 100 {
		return fmt.Sprintf("%.2f", v)
	}
	if v >= 1 {
		return fmt.Sprintf("%.4f", v)
	}
	return fmt.Sprintf("%.6f", v)
}
