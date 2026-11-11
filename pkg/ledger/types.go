package ledger

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Supported standard asset symbols.
const (
	AssetStars  = "XTR"   // Telegram Stars (0 decimals)
	AssetTON    = "TON"   // TON (nanoTON, 9 decimals)
	AssetUSDT   = "USDT"  // Tether USD on TON (microUSDT, 6 decimals)
	AssetPoints = "POINT" // Internal reward points (0 decimals)
)

// Asset decimal constants.
const (
	DecimalsStars  = 0
	DecimalsTON    = 9
	DecimalsUSDT   = 6
	DecimalsPoints = 0
)

// Account types following standard double-entry accounting.
type AccountType string

const (
	AccountAsset     AccountType = "ASSET"     // Platform-held resources (e.g. Hot Wallets, Escrow, Bank)
	AccountLiability AccountType = "LIABILITY" // User deposits owed to users (User Main, User Locked)
	AccountEquity    AccountType = "EQUITY"    // Platform retained earnings, seed capital
	AccountRevenue   AccountType = "REVENUE"   // Collected commissions, platform fees
	AccountExpense   AccountType = "EXPENSE"   // On-chain network gas paid, infrastructure cost
)

// Direction of a posting.
type Direction string

const (
	Debit  Direction = "DEBIT"
	Credit Direction = "CREDIT"
)

// TransferStatus represents the lifecycle of a two-phase transfer.
type TransferStatus string

const (
	TransferPending TransferStatus = "PENDING"
	TransferSettled TransferStatus = "SETTLED"
	TransferVoided  TransferStatus = "VOIDED"
)

var (
	ErrUnbalancedEntry      = errors.New("ledger: total debits do not equal total credits for asset")
	ErrInsufficientFunds    = errors.New("ledger: insufficient available balance")
	ErrInvalidAmount        = errors.New("ledger: amount must be strictly greater than zero")
	ErrAccountNotFound      = errors.New("ledger: account not found")
	ErrDuplicateJournal     = errors.New("ledger: journal entry already exists")
	ErrTransferNotFound     = errors.New("ledger: pending transfer not found")
	ErrTransferNotPending   = errors.New("ledger: transfer is not in pending status")
	ErrAssetMismatch        = errors.New("ledger: asset symbol mismatch in account")
	ErrUnknownAccountType   = errors.New("ledger: unknown account type")
)

// Account represents an account balance and its classification.
type Account struct {
	ID        string            `json:"id"`
	Type      AccountType       `json:"type"`
	Asset     string            `json:"asset"`
	Available uint64            `json:"available"`
	Locked    uint64            `json:"locked"`
	Version   uint64            `json:"version"`
	UpdatedAt int64             `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Posting represents an individual leg of a double-entry transaction.
type Posting struct {
	AccountID string    `json:"account_id"`
	Asset     string    `json:"asset"`
	Amount    uint64    `json:"amount"` // Strictly positive in atomic units
	Direction Direction `json:"direction"`
}

// JournalEntry represents an immutable set of balanced postings.
type JournalEntry struct {
	ID          string            `json:"id"`
	Reference   string            `json:"reference"` // External reference (tx hash, telegram charge_id, outbox_id)
	Description string            `json:"description"`
	Postings    []Posting         `json:"postings"`
	Timestamp   int64             `json:"timestamp"` // Unix nano
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PendingTransfer tracks a 2-phase reserve-settle transfer.
type PendingTransfer struct {
	TransferID  string            `json:"transfer_id"`
	FromAccount string            `json:"from_account"`
	ToAccount   string            `json:"to_account"`
	Asset       string            `json:"asset"`
	Amount      uint64            `json:"amount"`
	Status      TransferStatus    `json:"status"`
	CreatedAt   int64             `json:"created_at"`
	ExpiresAt   int64             `json:"expires_at"` // 0 if never
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Helper conversions between human floats and atomic integer units

// TONToNano converts human TON to nanoTON (10^9).
func TONToNano(ton float64) uint64 {
	return uint64(math.Round(ton * 1e9))
}

// NanoToTON converts nanoTON (10^9) to human TON.
func NanoToTON(nano uint64) float64 {
	return float64(nano) / 1e9
}

// USDTToMicro converts human USDT to microUSDT (10^6).
func USDTToMicro(usdt float64) uint64 {
	return uint64(math.Round(usdt * 1e6))
}

// MicroToUSDT converts microUSDT (10^6) to human USDT.
func MicroToUSDT(micro uint64) float64 {
	return float64(micro) / 1e6
}

// DecimalsForAsset returns standard decimals for known assets.
func DecimalsForAsset(asset string) int {
	switch strings.ToUpper(asset) {
	case AssetTON:
		return DecimalsTON
	case AssetUSDT:
		return DecimalsUSDT
	case AssetStars:
		return DecimalsStars
	case AssetPoints:
		return DecimalsPoints
	default:
		return 0
	}
}

// FormatAtomic formats an atomic integer amount into human string representation.
func FormatAtomic(amount uint64, asset string) string {
	dec := DecimalsForAsset(asset)
	if dec == 0 {
		return fmt.Sprintf("%d %s", amount, asset)
	}

	divisor := uint64(math.Pow10(dec))
	whole := amount / divisor
	frac := amount % divisor

	fracStr := fmt.Sprintf("%0*d", dec, frac)
	fracStr = strings.TrimRight(fracStr, "0")
	if fracStr == "" {
		return fmt.Sprintf("%d %s", whole, asset)
	}
	return fmt.Sprintf("%d.%s %s", whole, fracStr, asset)
}

// ParseToAtomic parses a human string into atomic uint64 units.
func ParseToAtomic(s string, asset string) (uint64, error) {
	s = strings.TrimSpace(s)
	dec := DecimalsForAsset(asset)
	if dec == 0 {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer amount %q: %w", s, err)
		}
		return v, nil
	}

	parts := strings.Split(s, ".")
	whole, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid whole number %q: %w", parts[0], err)
	}

	divisor := uint64(math.Pow10(dec))
	res := whole * divisor

	if len(parts) > 1 {
		fracStr := parts[1]
		if len(fracStr) > dec {
			fracStr = fracStr[:dec] // truncate
		} else {
			fracStr = fracStr + strings.Repeat("0", dec-len(fracStr))
		}
		frac, err := strconv.ParseUint(fracStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid fractional part %q: %w", fracStr, err)
		}
		res += frac
	}

	return res, nil
}

// Standard Account ID Generators
func UserMainAccount(userID int64, asset string) string {
	return fmt.Sprintf("user:%d:main:%s", userID, strings.ToUpper(asset))
}

func UserLockedAccount(userID int64, asset string) string {
	return fmt.Sprintf("user:%d:locked:%s", userID, strings.ToUpper(asset))
}

func SystemReserveAccount(asset string) string {
	return fmt.Sprintf("system:reserve:%s", strings.ToUpper(asset))
}

func SystemFeeAccount(asset string) string {
	return fmt.Sprintf("system:fee:%s", strings.ToUpper(asset))
}

func SystemExpenseAccount(asset string) string {
	return fmt.Sprintf("system:expense:%s", strings.ToUpper(asset))
}

// InferAccountType extracts normal classification from standard naming or defaults.
func InferAccountType(accountID string) AccountType {
	if strings.HasPrefix(accountID, "system:reserve:") {
		return AccountAsset
	}
	if strings.HasPrefix(accountID, "system:fee:") {
		return AccountRevenue
	}
	if strings.HasPrefix(accountID, "system:expense:") {
		return AccountExpense
	}
	if strings.HasPrefix(accountID, "system:equity:") {
		return AccountEquity
	}
	if strings.HasPrefix(accountID, "user:") {
		return AccountLiability
	}
	return AccountLiability
}

// ExtractAsset extracts the asset symbol from an account ID format (e.g. user:1:main:TON -> TON).
func ExtractAsset(accountID string) string {
	parts := strings.Split(accountID, ":")
	if len(parts) > 0 {
		return strings.ToUpper(parts[len(parts)-1])
	}
	return ""
}

// NewJournalEntry creates a new journal entry with current unix nano timestamp.
func NewJournalEntry(id, ref, desc string, postings []Posting) JournalEntry {
	return JournalEntry{
		ID:          id,
		Reference:   ref,
		Description: desc,
		Postings:    postings,
		Timestamp:   time.Now().UnixNano(),
		Metadata:    make(map[string]string),
	}
}
