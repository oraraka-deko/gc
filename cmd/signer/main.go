package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gc/pkg/wallet"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9099", "Bind address for isolated signer daemon")
	keyHex := flag.String("key", "", "Hex-encoded Ed25519 private key (optional, random if empty)")
	flag.Parse()

	var privKey ed25519.PrivateKey
	if *keyHex != "" {
		k, err := hex.DecodeString(*keyHex)
		if err != nil || len(k) != ed25519.PrivateKeySize {
			log.Fatalf("invalid private key hex: %v", err)
		}
		privKey = ed25519.PrivateKey(k)
	}

	server, err := wallet.NewSignerServer(*addr, privKey)
	if err != nil {
		log.Fatalf("failed to start signer server: %v", err)
	}
	defer server.Stop()

	log.Printf("[Signer Daemon] Listening on %s", server.Addr())
	log.Printf("[Signer Daemon] Public Key: %s", server.PublicKey())

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("[Signer Daemon] Shutting down...")
}
