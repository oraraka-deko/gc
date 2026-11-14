package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gc/pkg/games"
	"gc/pkg/ledger"
	"gc/pkg/storage"
)

func main() {
	port := flag.Int("port", 8080, "Port to run the HTTP/WS backend server on")
	dataDir := flag.String("data", "./data", "Directory for Pebble database and local storage")
	botTokenFlag := flag.String("token", "", "Telegram Bot Token for initData verification")
	flag.Parse()

	botToken := *botTokenFlag
	if botToken == "" {
		botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	log.Printf("================================================================================")
	log.Printf(" Starting MiniApp Unified Backend Server on :%d", *port)
	log.Printf(" Storage directory: %s", *dataDir)
	if botToken != "" {
		log.Printf(" Bot Token configured: %s...%s", botToken[:4], botToken[len(botToken)-4:])
	} else {
		log.Printf(" [NOTICE] No bot token provided. Operating in Sandbox/Dev mode.")
	}
	log.Printf("================================================================================")

	// 1. Initialize Storage Tier (Pebble + FastCache)
	store, err := storage.New(storage.Config{
		PebblePath:     filepath.Join(*dataDir, "pebble"),
		FastCacheBytes: 32 * 1024 * 1024, // 32MB in-memory cache
	})
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()
	log.Printf("[+] Storage tier initialized (Pebble LSM-tree + FastCache)")

	// 2. Initialize Double-Entry Ledger Core
	led := ledger.New(store)
	log.Printf("[+] Double-entry ledger core online")

	// 3. Initialize Unified Game Server (Territory, BallEscape, Tower, Hub, Sync)
	gameServer := games.NewGameServer(led, store, botToken)
	defer gameServer.Close()
	log.Printf("[+] Game engines and ClientHub online")

	// 4. Setup HTTP Router & Register Endpoints
	mux := http.NewServeMux()
	gameServer.RegisterRoutes(mux)

	// Health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		games.EnableCORS(w)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","time":%d}`, time.Now().Unix())
	})

	serverAddr := fmt.Sprintf("0.0.0.0:%d", *port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// 5. Start Server in background goroutine
	go func() {
		log.Printf("[*] HTTP & WebSocket Server listening on http://localhost:%d", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server ListenAndServe error: %v", err)
		}
	}()

	// 6. Graceful Shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\n[*] Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("[+] Server stopped cleanly.")
}
