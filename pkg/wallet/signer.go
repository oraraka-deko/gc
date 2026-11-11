package wallet

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// BatchSigner defines the interface for signing outbound transaction batches.
type BatchSigner interface {
	SignBatch(ctx context.Context, transfers []OutboxTx) (*SignedBatch, error)
	PublicKey() string
	Close() error
}

// MockSigner is an in-process Ed25519 signer for development and automated testing.
type MockSigner struct {
	pubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey
	mu      sync.Mutex
}

// NewMockSigner creates an in-process mock signer with an ephemeral Ed25519 keypair.
func NewMockSigner() (*MockSigner, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 key: %w", err)
	}
	return &MockSigner{
		pubKey:  pub,
		privKey: priv,
	}, nil
}

// PublicKey returns the hex-encoded Ed25519 public key.
func (s *MockSigner) PublicKey() string {
	return hex.EncodeToString(s.pubKey)
}

// SignBatch packs transfers into a simulated Highload Wallet v3 cell and signs with Ed25519.
func (s *MockSigner) SignBatch(ctx context.Context, transfers []OutboxTx) (*SignedBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(transfers) == 0 {
		return nil, ErrEmptyBatch
	}

	txIDs := make([]string, len(transfers))
	hasher := sha256.New()

	for i, tx := range transfers {
		txIDs[i] = tx.ID
		hasher.Write([]byte(fmt.Sprintf("%s:%s:%d:%s;", tx.ID, tx.Asset, tx.Amount, tx.DestinationAddress)))
	}

	// Digest of the batch
	batchHash := hasher.Sum(nil)

	// Ed25519 signature over batch hash
	sig := ed25519.Sign(s.privKey, batchHash)

	// Construct simulated Highload Wallet v3 BoC (Header + Signature + Hash)
	bocPayload := append(sig, batchHash...)
	bocBase64 := base64.StdEncoding.EncodeToString(bocPayload)

	batchID := fmt.Sprintf("batch_%d_%s", time.Now().UnixNano(), hex.EncodeToString(batchHash[:8]))

	return &SignedBatch{
		BatchID:   batchID,
		TxIDs:     txIDs,
		BOCBase64: bocBase64,
		CreatedAt: time.Now().UnixNano(),
	}, nil
}

// Close is a no-op for MockSigner.
func (s *MockSigner) Close() error {
	return nil
}

// IPCSignerClient connects to an isolated external signer daemon over local TCP / socket.
type IPCSignerClient struct {
	addr   string
	conn   net.Conn
	reader *bufio.Reader
	pubKey string
	mu     sync.Mutex
}

// SignerRequest represents a JSON-RPC-style request to the signer daemon.
type SignerRequest struct {
	Action    string     `json:"action"` // "get_public_key" or "sign_batch"
	Transfers []OutboxTx `json:"transfers,omitempty"`
}

// SignerResponse represents a response from the signer daemon.
type SignerResponse struct {
	Success   bool         `json:"success"`
	Error     string       `json:"error,omitempty"`
	PublicKey string       `json:"public_key,omitempty"`
	Batch     *SignedBatch `json:"batch,omitempty"`
}

// NewIPCSignerClient creates a new client connected to the signer daemon.
func NewIPCSignerClient(addr string) (*IPCSignerClient, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to signer daemon at %s: %w", addr, err)
	}

	client := &IPCSignerClient{
		addr:   addr,
		conn:   conn,
		reader: bufio.NewReader(conn),
	}

	// Fetch public key
	pubKey, err := client.fetchPublicKey()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to fetch public key from signer daemon: %w", err)
	}
	client.pubKey = pubKey

	return client, nil
}

func (c *IPCSignerClient) fetchPublicKey() (string, error) {
	req := SignerRequest{Action: "get_public_key"}
	reqBytes, _ := json.Marshal(req)
	reqBytes = append(reqBytes, '\n')

	if _, err := c.conn.Write(reqBytes); err != nil {
		return "", err
	}

	respLine, err := c.reader.ReadBytes('\n')
	if err != nil {
		return "", err
	}

	var resp SignerResponse
	if err := json.Unmarshal(respLine, &resp); err != nil {
		return "", err
	}

	if !resp.Success {
		return "", errors.New(resp.Error)
	}

	return resp.PublicKey, nil
}

// PublicKey returns the remote signer's public key.
func (c *IPCSignerClient) PublicKey() string {
	return c.pubKey
}

// SignBatch sends batch to signer daemon.
func (c *IPCSignerClient) SignBatch(ctx context.Context, transfers []OutboxTx) (*SignedBatch, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(transfers) == 0 {
		return nil, ErrEmptyBatch
	}

	req := SignerRequest{
		Action:    "sign_batch",
		Transfers: transfers,
	}
	reqBytes, _ := json.Marshal(req)
	reqBytes = append(reqBytes, '\n')

	if _, err := c.conn.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("failed to send request to signer: %w", err)
	}

	respLine, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response from signer: %w", err)
	}

	var resp SignerResponse
	if err := json.Unmarshal(respLine, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response from signer: %w", err)
	}

	if !resp.Success {
		return nil, errors.New(resp.Error)
	}

	return resp.Batch, nil
}

// Close disconnects from the signer daemon.
func (c *IPCSignerClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
