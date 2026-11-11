package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gc/pkg/ledger"
	"gc/pkg/storage"
	"gc/pkg/telegram"
	"gc/pkg/wallet"
	"gc/pkg/workflow"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	tokenFlag := flag.String("token", "", "Telegram Bot Token")
	dataDir := flag.String("data", "./data", "Data directory for Pebble and storage")
	flag.Parse()

	if *tokenFlag != "" {
		token = *tokenFlag
	}

	log.Println("=== Initializing High-Throughput Ledger & Workflow Engine ===")

	// 1. Initialize Storage Tier (FastCache + Pebble)
	store, err := storage.New(storage.Config{
		PebblePath:     filepath.Join(*dataDir, "pebble"),
		FastCacheBytes: 64 * 1024 * 1024, // 64 MB
	})
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}
	defer store.Close()
	log.Println("[Storage] FastCache & Pebble LSM-tree initialized.")

	// 2. Initialize Double-Entry Ledger Engine
	led := ledger.New(store)
	log.Println("[Ledger] Double-entry core online. Invariant: Sum(Debits) == Sum(Credits).")

	// 3. Initialize Multi-Asset Income Verifier
	incomeVerifier := ledger.NewIncomePipeline(led, store)
	log.Println("[Verifier] Multi-asset income pipeline ready for Stars, TON, and Jetton USDT.")

	// 4. Initialize Outbound Wallet Manager with In-Process or Daemon Signer
	signer, err := wallet.NewMockSigner()
	if err != nil {
		log.Fatalf("failed to init signer: %v", err)
	}
	defer signer.Close()

	outbox := wallet.NewOutboxManager(store, led, signer, wallet.OutboxConfig{
		MaxBatchSize: 254, // Highload Wallet v3 limit
		BatchTimeout: 500 * time.Millisecond,
	})
	log.Printf("[Outbox] Transactional outbox initialized for %d transfers (Signer Public Key: %s).",
		254, signer.PublicKey())
	_ = outbox // Ready for outbound processing jobs

	// 5. Initialize Workflow Engine
	var botClient *telegram.Client
	if token != "" {
		botClient = telegram.NewClient(token)
	}

	wfEngine := workflow.NewEngine(store, led, botClient)

	// Register an automated Escrow & Settlement Workflow
	escrowWorkflow := &workflow.WorkflowDef{
		ID:   "escrow_payment",
		Name: "P2P Escrow & Settlement Workflow",
		Steps: []workflow.StepDef{
			{
				ID:   "reserve_escrow",
				Type: workflow.StepActionLedgerReserve,
				Params: map[string]any{
					"from_user":  int64(1001),
					"to_account": ledger.UserMainAccount(2002, ledger.AssetTON),
					"asset":      ledger.AssetTON,
					"amount":     ledger.TONToNano(2.5),
				},
			},
			{
				ID:   "wait_for_buyer_confirmation",
				Type: workflow.StepActionWaitEvent,
				Params: map[string]any{
					"event": "buyer_confirmed",
				},
			},
			{
				ID:   "settle_escrow",
				Type: workflow.StepActionLedgerSettle,
			},
		},
		CycleConfig: workflow.CycleConfig{
			Interval: 10 * time.Second,
		},
	}

	if err := wfEngine.RegisterWorkflow(escrowWorkflow); err != nil {
		log.Fatalf("failed to register workflow: %v", err)
	}
	log.Println("[Workflow] Escrow & Settlement tickless workflow registered.")

	// 6. Demonstrate Multi-Asset Ingestion & Workflow Execution
	ctx := context.Background()

	// Initial User Deposit: 10 TON
	log.Println("[Demo] Crediting User 1001 with 10 TON initial deposit...")
	_, err = led.Deposit(ctx, 1001, ledger.AssetTON, ledger.TONToNano(10.0), "dep:demo:1001", "Demo Deposit")
	if err != nil {
		log.Fatalf("deposit failed: %v", err)
	}

	acc, _ := led.GetAccount(ctx, ledger.UserMainAccount(1001, ledger.AssetTON))
	log.Printf("[Demo] User 1001 Balance: %s", ledger.FormatAtomic(acc.Available, ledger.AssetTON))

	// Start Workflow Instance (suspends at wait_for_buyer_confirmation)
	log.Println("[Demo] Starting Escrow Workflow Instance...")
	inst, err := wfEngine.StartInstance(ctx, "escrow_payment", nil)
	if err != nil {
		log.Fatalf("start workflow failed: %v", err)
	}
	log.Printf("[Demo] Instance %s status: %s (Awaiting: %s). Goroutine halted, state saved in Pebble.",
		inst.InstanceID, inst.State, inst.AwaitingEvent)

	// Check reserved balance
	accUser, _ := led.GetAccount(ctx, ledger.UserMainAccount(1001, ledger.AssetTON))
	accLocked, _ := led.GetAccount(ctx, ledger.UserLockedAccount(1001, ledger.AssetTON))
	log.Printf("[Demo] User 1001 Available: %s, Locked: %s",
		ledger.FormatAtomic(accUser.Available, ledger.AssetTON),
		ledger.FormatAtomic(accLocked.Available, ledger.AssetTON))

	// Simulate event arrival: Buyer confirms delivery!
	log.Println("[Demo] Emulating Buyer Confirmation Event...")
	resumed, err := wfEngine.ResumeInstance(ctx, inst.InstanceID, "buyer_confirmed", "goods_received_ok")
	if err != nil {
		log.Fatalf("resume failed: %v", err)
	}
	log.Printf("[Demo] Workflow resumed to completion! Final status: %s", resumed.State)

	// Check final balances
	accSender, _ := led.GetAccount(ctx, ledger.UserMainAccount(1001, ledger.AssetTON))
	accReceiver, _ := led.GetAccount(ctx, ledger.UserMainAccount(2002, ledger.AssetTON))
	log.Printf("[Demo] Final Balance User 1001 (Sender): %s", ledger.FormatAtomic(accSender.Available, ledger.AssetTON))
	log.Printf("[Demo] Final Balance User 2002 (Receiver): %s", ledger.FormatAtomic(accReceiver.Available, ledger.AssetTON))

	// 7. If bot token is provided, wire up Dispatcher for Telegram Bot API
	if token != "" && botClient != nil {
		log.Println("[Bot] Starting Telegram bot dispatcher...")
		dispatcher := telegram.NewDispatcher()

		// Command: /balance
		dispatcher.OnCommand("balance", func(ctx context.Context, m *telegram.Message, args []string) error {
			if m.From == nil {
				return nil
			}

			tonAcc, _ := led.GetAccount(ctx, ledger.UserMainAccount(m.From.ID, ledger.AssetTON))
			usdtAcc, _ := led.GetAccount(ctx, ledger.UserMainAccount(m.From.ID, ledger.AssetUSDT))
			starsAcc, _ := led.GetAccount(ctx, ledger.UserMainAccount(m.From.ID, ledger.AssetStars))

			var tonBal, usdtBal, starsBal uint64
			if tonAcc != nil {
				tonBal = tonAcc.Available
			}
			if usdtAcc != nil {
				usdtBal = usdtAcc.Available
			}
			if starsAcc != nil {
				starsBal = starsAcc.Available
			}

			msg := fmt.Sprintf(
				"<b>Your Financial Ledger Balance:</b>\n\n"+
					"💎 TON: <code>%s</code>\n"+
					"💵 USDT: <code>%s</code>\n"+
					"⭐ Stars: <code>%d XTR</code>\n\n"+
					"<i>Backed by atomic double-entry accounting.</i>",
				ledger.FormatAtomic(tonBal, ledger.AssetTON),
				ledger.FormatAtomic(usdtBal, ledger.AssetUSDT),
				starsBal,
			)

			_, err := botClient.SendMessage(ctx, &telegram.SendMessageParams{
				ChatID:    m.Chat.ID,
				Text:      msg,
				ParseMode: telegram.StringPtr("HTML"),
			})
			return err
		})

		// Handle Telegram Stars PreCheckoutQuery
		dispatcher.OnPreCheckoutQuery(func(ctx context.Context, q *telegram.PreCheckoutQuery) error {
			ok, errMsg, err := incomeVerifier.ValidatePreCheckout(ctx, q)
			if err != nil || !ok {
				_, _ = botClient.AnswerPreCheckoutQuery(ctx, &telegram.AnswerPreCheckoutQueryParams{
					PreCheckoutQueryID: q.ID,
					OK:                 false,
					ErrorMessage:       telegram.StringPtr(errMsg),
				})
				return nil
			}
			_, err = botClient.AnswerPreCheckoutQuery(ctx, &telegram.AnswerPreCheckoutQueryParams{
				PreCheckoutQueryID: q.ID,
				OK:                 true,
			})
			return err
		})

		poller := telegram.NewLongPoller(botClient, dispatcher,
			telegram.WithPollTimeout(20),
			telegram.WithPollWorkers(8),
		)

		botCtx, cancelBot := context.WithCancel(context.Background())
		defer cancelBot()

		go func() {
			if err := poller.Start(botCtx); err != nil {
				log.Printf("[Bot Poller] Stopped: %v", err)
			}
		}()
		log.Println("[Bot] Telegram poller online.")
	}

	log.Println("=== System is fully operational. Press Ctrl+C to terminate. ===")

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down cleanly...")
}
