package wallet

import (
	"errors"
)

// OutboxState describes the lifecycle of an outbound transaction.
type OutboxState string

const (
	StateQueued      OutboxState = "QUEUED"      // Funds reserved in ledger, awaiting batch signer
	StateApproved    OutboxState = "APPROVED"    // Selected for current batch
	StateSigned      OutboxState = "SIGNED"      // Packed into Highload v3 message and cryptographically signed
	StateBroadcasted OutboxState = "BROADCASTED" // Submitted to TON lite-servers / API
	StateConfirmed   OutboxState = "CONFIRMED"   // Finalized on-chain; ledger settled
	StateFailed      OutboxState = "FAILED"      // Terminal failure; ledger voided & funds restored
)

var (
	ErrTxNotFound    = errors.New("outbox: transaction not found")
	ErrInvalidState  = errors.New("outbox: invalid transaction state transition")
	ErrEmptyBatch    = errors.New("outbox: no queued transactions to batch")
	ErrSignerClosed  = errors.New("signer: signer connection closed")
)

// OutboxTx represents an individual outbound transfer managed by the transactional outbox.
type OutboxTx struct {
	ID                 string      `json:"id"`
	UserID             int64       `json:"user_id"`
	TransferID         string      `json:"transfer_id"` // Corresponds to ledger.PendingTransfer
	Asset              string      `json:"asset"`
	Amount             uint64      `json:"amount"` // Atomic units
	DestinationAddress string      `json:"destination_address"`
	Memo               string      `json:"memo,omitempty"`
	State              OutboxState `json:"state"`
	SignedBOC          string      `json:"signed_boc,omitempty"` // Base64 encoded BoC
	TxHash             string      `json:"tx_hash,omitempty"`    // On-chain transaction hash
	RetryCount         int         `json:"retry_count"`
	LastError          string      `json:"last_error,omitempty"`
	CreatedAt          int64       `json:"created_at"`
	UpdatedAt          int64       `json:"updated_at"`
}

// SignedBatch represents a batch of outbound transfers bundled into a Highload Wallet v3 cell.
type SignedBatch struct {
	BatchID   string   `json:"batch_id"`
	TxIDs     []string `json:"tx_ids"`
	BOCBase64 string   `json:"boc_base64"`
	CreatedAt int64    `json:"created_at"`
}
