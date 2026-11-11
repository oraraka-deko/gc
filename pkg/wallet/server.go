package wallet

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// SignerServer is an isolated daemon server that signs batches over TCP/IPC.
type SignerServer struct {
	listener net.Listener
	pubKey   ed25519.PublicKey
	privKey  ed25519.PrivateKey
	quit     chan struct{}
	wg       sync.WaitGroup
}

// NewSignerServer creates and binds a new SignerServer.
func NewSignerServer(addr string, priv ed25519.PrivateKey) (*SignerServer, error) {
	if priv == nil {
		_, p, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		priv = p
	}

	pub := priv.Public().(ed25519.PublicKey)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind signer server to %s: %w", addr, err)
	}

	s := &SignerServer{
		listener: ln,
		pubKey:   pub,
		privKey:  priv,
		quit:     make(chan struct{}),
	}

	s.wg.Add(1)
	go s.serve()

	return s, nil
}

// Addr returns the listener network address.
func (s *SignerServer) Addr() string {
	return s.listener.Addr().String()
}

// PublicKey returns hex-encoded public key.
func (s *SignerServer) PublicKey() string {
	return hex.EncodeToString(s.pubKey)
}

func (s *SignerServer) serve() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				continue
			}
		}

		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			defer c.Close()
			s.handleConn(c)
		}(conn)
	}
}

func (s *SignerServer) handleConn(conn net.Conn) {
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}

		var req SignerRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendResp(conn, SignerResponse{Success: false, Error: "invalid json request"})
			continue
		}

		switch req.Action {
		case "get_public_key":
			s.sendResp(conn, SignerResponse{
				Success:   true,
				PublicKey: hex.EncodeToString(s.pubKey),
			})

		case "sign_batch":
			if len(req.Transfers) == 0 {
				s.sendResp(conn, SignerResponse{Success: false, Error: "empty transfers"})
				continue
			}

			txIDs := make([]string, len(req.Transfers))
			hasher := sha256.New()
			for i, tx := range req.Transfers {
				txIDs[i] = tx.ID
				hasher.Write([]byte(fmt.Sprintf("%s:%s:%d:%s;", tx.ID, tx.Asset, tx.Amount, tx.DestinationAddress)))
			}

			batchHash := hasher.Sum(nil)
			sig := ed25519.Sign(s.privKey, batchHash)

			bocPayload := append(sig, batchHash...)
			bocBase64 := base64.StdEncoding.EncodeToString(bocPayload)

			batchID := fmt.Sprintf("daemon_batch_%d_%s", time.Now().UnixNano(), hex.EncodeToString(batchHash[:8]))

			s.sendResp(conn, SignerResponse{
				Success: true,
				Batch: &SignedBatch{
					BatchID:   batchID,
					TxIDs:     txIDs,
					BOCBase64: bocBase64,
					CreatedAt: time.Now().UnixNano(),
				},
			})

		default:
			s.sendResp(conn, SignerResponse{Success: false, Error: "unknown action"})
		}
	}
}

func (s *SignerServer) sendResp(conn net.Conn, resp SignerResponse) {
	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	_, _ = conn.Write(data)
}

// Stop shuts down the server.
func (s *SignerServer) Stop() error {
	close(s.quit)
	err := s.listener.Close()
	s.wg.Wait()
	return err
}
