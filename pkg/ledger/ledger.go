package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gc/pkg/storage"
)

// Ledger coordinates double-entry journals, accounts, and two-phase transfers.
type Ledger struct {
	store *storage.Storage
	mu    sync.RWMutex // Protects in-memory multi-account reservation pipelines
}

// New creates a new double-entry Ledger engine backed by storage.
func New(store *storage.Storage) *Ledger {
	return &Ledger{
		store: store,
	}
}

// GetAccount retrieves an account by ID, returning ErrAccountNotFound if absent.
func (l *Ledger) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	key := []byte(storage.NSAccount + accountID)
	data, err := l.store.Pebble.Get(key)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	var acc Account
	if err := json.Unmarshal(data, &acc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	}
	return &acc, nil
}

// getOrCreateAccount loads an existing account or initializes a default empty account.
func (l *Ledger) getOrCreateAccount(accountID string) (*Account, error) {
	acc, err := l.GetAccount(context.Background(), accountID)
	if err == nil {
		return acc, nil
	}
	if err != ErrAccountNotFound {
		return nil, err
	}

	// Initialize new account
	asset := ExtractAsset(accountID)
	accType := InferAccountType(accountID)

	return &Account{
		ID:        accountID,
		Type:      accType,
		Asset:     asset,
		Available: 0,
		Locked:    0,
		Version:   0,
		UpdatedAt: time.Now().UnixNano(),
		Metadata:  make(map[string]string),
	}, nil
}

// ValidatePostings checks that all postings are non-zero and sum(debits) == sum(credits) per asset.
func ValidatePostings(postings []Posting) error {
	if len(postings) < 2 {
		return errors.New("ledger: entry must contain at least 2 postings")
	}

	debits := make(map[string]uint64)
	credits := make(map[string]uint64)

	for _, p := range postings {
		if p.Amount == 0 {
			return ErrInvalidAmount
		}
		if p.AccountID == "" {
			return errors.New("ledger: posting AccountID cannot be empty")
		}
		asset := p.Asset
		if asset == "" {
			asset = ExtractAsset(p.AccountID)
		}
		if asset == "" {
			return errors.New("ledger: asset cannot be determined for posting")
		}

		if p.Direction == Debit {
			debits[asset] += p.Amount
		} else if p.Direction == Credit {
			credits[asset] += p.Amount
		} else {
			return errors.New("ledger: invalid posting direction")
		}
	}

	// Verify invariant for every asset
	for asset, totalDebit := range debits {
		totalCredit := credits[asset]
		if totalDebit != totalCredit {
			return fmt.Errorf("%w: asset %s debits=%d credits=%d", ErrUnbalancedEntry, asset, totalDebit, totalCredit)
		}
	}

	// Verify all credit assets have debits
	for asset, totalCredit := range credits {
		totalDebit := debits[asset]
		if totalDebit != totalCredit {
			return fmt.Errorf("%w: asset %s debits=%d credits=%d", ErrUnbalancedEntry, asset, totalDebit, totalCredit)
		}
	}

	return nil
}

// RecordJournalEntry validates and atomically records a double-entry journal entry.
func (l *Ledger) RecordJournalEntry(ctx context.Context, entry JournalEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.ID == "" {
		return errors.New("ledger: journal entry ID cannot be empty")
	}

	// 1. Validate Double-Entry Invariant
	if err := ValidatePostings(entry.Postings); err != nil {
		return err
	}

	// 2. Check for duplicate entry or reference
	jnlKey := []byte(storage.NSJournal + entry.ID)
	if _, err := l.store.Pebble.Get(jnlKey); err == nil {
		return ErrDuplicateJournal
	}

	if entry.Reference != "" {
		refKey := []byte("led:ref:" + entry.Reference)
		if _, err := l.store.Pebble.Get(refKey); err == nil {
			return fmt.Errorf("ledger: reference %s already recorded", entry.Reference)
		}
	}

	if entry.Timestamp == 0 {
		entry.Timestamp = time.Now().UnixNano()
	}

	// 3. Load all involved accounts and calculate balance adjustments
	type accountDelta struct {
		account *Account
		netDiff int64 // positive increases balance, negative decreases
	}

	accountMap := make(map[string]*accountDelta)

	for _, p := range entry.Postings {
		ad, ok := accountMap[p.AccountID]
		if !ok {
			acc, err := l.getOrCreateAccount(p.AccountID)
			if err != nil {
				return err
			}
			ad = &accountDelta{account: acc, netDiff: 0}
			accountMap[p.AccountID] = ad
		}

		// Calculate balance impact according to standard accounting rules:
		// ASSET & EXPENSE: Debit increases (+), Credit decreases (-)
		// LIABILITY, EQUITY & REVENUE: Credit increases (+), Debit decreases (-)
		var sign int64
		switch ad.account.Type {
		case AccountAsset, AccountExpense:
			if p.Direction == Debit {
				sign = 1
			} else {
				sign = -1
			}
		case AccountLiability, AccountEquity, AccountRevenue:
			if p.Direction == Credit {
				sign = 1
			} else {
				sign = -1
			}
		default:
			return fmt.Errorf("%w: %s", ErrUnknownAccountType, ad.account.Type)
		}

		ad.netDiff += sign * int64(p.Amount)
	}

	// 4. Verify no account experiences negative balance overdraft
	for accID, ad := range accountMap {
		newAvail := int64(ad.account.Available) + ad.netDiff
		if newAvail < 0 {
			return fmt.Errorf("%w on account %s: available=%d requested delta=%d",
				ErrInsufficientFunds, accID, ad.account.Available, ad.netDiff)
		}
	}

	// 5. Apply changes into atomic Pebble Batch
	batch := l.store.Pebble.NewBatch()
	defer batch.Close()

	now := time.Now().UnixNano()

	// Update accounts
	for _, ad := range accountMap {
		ad.account.Available = uint64(int64(ad.account.Available) + ad.netDiff)
		ad.account.Version++
		ad.account.UpdatedAt = now

		data, err := json.Marshal(ad.account)
		if err != nil {
			return err
		}
		if err := batch.Set([]byte(storage.NSAccount+ad.account.ID), data); err != nil {
			return err
		}

		// Write account chronological index: led:ach:<accountID>:<timestampNanoPad>:<entryID>
		achKey := []byte(fmt.Sprintf("%s%s:%020d:%s", storage.NSAccountHistory, ad.account.ID, entry.Timestamp, entry.ID))
		if err := batch.Set(achKey, []byte(entry.ID)); err != nil {
			return err
		}
	}

	// Write Journal Entry
	entryData, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if err := batch.Set(jnlKey, entryData); err != nil {
		return err
	}

	// Write reference index
	if entry.Reference != "" {
		refKey := []byte("led:ref:" + entry.Reference)
		if err := batch.Set(refKey, []byte(entry.ID)); err != nil {
			return err
		}
	}

	// 6. Commit batch synchronously for durable WAL safety
	if err := batch.Commit(true); err != nil {
		return fmt.Errorf("failed to commit journal batch: %w", err)
	}

	// 7. Store idempotency in FastCache for ultra-fast replay rejection
	if entry.Reference != "" {
		l.store.Fast.SetIdempotency("ref:"+entry.Reference, []byte(entry.ID), 24*time.Hour)
	}

	return nil
}

// Reserve locks user funds in a two-phase transfer by moving them to user:locked account.
func (l *Ledger) Reserve(ctx context.Context, transferID string, fromUserID int64, toAccount string, asset string, amount uint64, metadata map[string]string) (*PendingTransfer, error) {
	if amount == 0 {
		return nil, ErrInvalidAmount
	}
	if transferID == "" {
		return nil, errors.New("transferID cannot be empty")
	}

	penKey := []byte(storage.NSPending + transferID)
	if _, err := l.store.Pebble.Get(penKey); err == nil {
		return nil, fmt.Errorf("transfer %s already exists", transferID)
	}

	fromMain := UserMainAccount(fromUserID, asset)
	fromLocked := UserLockedAccount(fromUserID, asset)

	// In double-entry:
	// Debit user:main (reduces user available liability balance)
	// Credit user:locked (increases user locked liability balance)
	journalID := fmt.Sprintf("jnl:res:%s", transferID)
	entry := NewJournalEntry(journalID, "res:"+transferID, fmt.Sprintf("Reserve %s for transfer %s", FormatAtomic(amount, asset), transferID), []Posting{
		{AccountID: fromMain, Asset: asset, Amount: amount, Direction: Debit},
		{AccountID: fromLocked, Asset: asset, Amount: amount, Direction: Credit},
	})
	entry.Metadata = metadata

	if err := l.RecordJournalEntry(ctx, entry); err != nil {
		return nil, fmt.Errorf("reserve journal failed: %w", err)
	}

	pt := &PendingTransfer{
		TransferID:  transferID,
		FromAccount: fromMain,
		ToAccount:   toAccount,
		Asset:       asset,
		Amount:      amount,
		Status:      TransferPending,
		CreatedAt:   time.Now().UnixNano(),
		Metadata:    metadata,
	}

	ptData, err := json.Marshal(pt)
	if err != nil {
		return nil, err
	}
	if err := l.store.Pebble.Set(penKey, ptData, true); err != nil {
		return nil, err
	}

	return pt, nil
}

// Settle releases locked funds to the destination account.
func (l *Ledger) Settle(ctx context.Context, transferID string) error {
	penKey := []byte(storage.NSPending + transferID)
	ptData, err := l.store.Pebble.Get(penKey)
	if err != nil {
		if err == storage.ErrNotFound {
			return ErrTransferNotFound
		}
		return err
	}

	var pt PendingTransfer
	if err := json.Unmarshal(ptData, &pt); err != nil {
		return err
	}

	if pt.Status != TransferPending {
		return fmt.Errorf("%w: status is %s", ErrTransferNotPending, pt.Status)
	}

	fromLocked := strings.Replace(pt.FromAccount, ":main:", ":locked:", 1)

	// Settle Journal:
	// Debit user:locked (clears locked liability balance)
	// Credit destination account (increases destination liability or platform account)
	journalID := fmt.Sprintf("jnl:set:%s", transferID)
	entry := NewJournalEntry(journalID, "set:"+transferID, fmt.Sprintf("Settle transfer %s", transferID), []Posting{
		{AccountID: fromLocked, Asset: pt.Asset, Amount: pt.Amount, Direction: Debit},
		{AccountID: pt.ToAccount, Asset: pt.Asset, Amount: pt.Amount, Direction: Credit},
	})

	if err := l.RecordJournalEntry(ctx, entry); err != nil {
		return fmt.Errorf("settle journal failed: %w", err)
	}

	pt.Status = TransferSettled
	updatedData, err := json.Marshal(pt)
	if err != nil {
		return err
	}

	return l.store.Pebble.Set(penKey, updatedData, true)
}

// Void cancels a pending transfer and returns locked funds back to available balance.
func (l *Ledger) Void(ctx context.Context, transferID string, reason string) error {
	penKey := []byte(storage.NSPending + transferID)
	ptData, err := l.store.Pebble.Get(penKey)
	if err != nil {
		if err == storage.ErrNotFound {
			return ErrTransferNotFound
		}
		return err
	}

	var pt PendingTransfer
	if err := json.Unmarshal(ptData, &pt); err != nil {
		return err
	}

	if pt.Status != TransferPending {
		return fmt.Errorf("%w: status is %s", ErrTransferNotPending, pt.Status)
	}

	fromLocked := strings.Replace(pt.FromAccount, ":main:", ":locked:", 1)

	// Void Journal:
	// Debit user:locked (reduces locked balance)
	// Credit user:main (restores available balance)
	journalID := fmt.Sprintf("jnl:void:%s", transferID)
	entry := NewJournalEntry(journalID, "void:"+transferID, fmt.Sprintf("Void transfer %s: %s", transferID, reason), []Posting{
		{AccountID: fromLocked, Asset: pt.Asset, Amount: pt.Amount, Direction: Debit},
		{AccountID: pt.FromAccount, Asset: pt.Asset, Amount: pt.Amount, Direction: Credit},
	})

	if err := l.RecordJournalEntry(ctx, entry); err != nil {
		return fmt.Errorf("void journal failed: %w", err)
	}

	pt.Status = TransferVoided
	if pt.Metadata == nil {
		pt.Metadata = make(map[string]string)
	}
	pt.Metadata["void_reason"] = reason

	updatedData, err := json.Marshal(pt)
	if err != nil {
		return err
	}

	return l.store.Pebble.Set(penKey, updatedData, true)
}

// Deposit records an external inflow (Stars, on-chain TON/USDT) into a user's account.
func (l *Ledger) Deposit(ctx context.Context, userID int64, asset string, amount uint64, ref string, desc string) (*JournalEntry, error) {
	entryID := fmt.Sprintf("jnl:dep:%s", ref)
	reserveAcc := SystemReserveAccount(asset)
	userAcc := UserMainAccount(userID, asset)

	// Deposit:
	// Debit system:reserve (Asset normal debit balance: platform cash increases)
	// Credit user:main (Liability normal credit balance: user available balance increases)
	entry := NewJournalEntry(entryID, ref, desc, []Posting{
		{AccountID: reserveAcc, Asset: asset, Amount: amount, Direction: Debit},
		{AccountID: userAcc, Asset: asset, Amount: amount, Direction: Credit},
	})

	if err := l.RecordJournalEntry(ctx, entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// Withdraw records a user withdrawal out of the system.
func (l *Ledger) Withdraw(ctx context.Context, userID int64, asset string, amount uint64, ref string, desc string) (*JournalEntry, error) {
	entryID := fmt.Sprintf("jnl:wdr:%s", ref)
	reserveAcc := SystemReserveAccount(asset)
	userAcc := UserMainAccount(userID, asset)

	// Withdraw:
	// Debit user:main (Liability debit: user available balance decreases)
	// Credit system:reserve (Asset credit: platform cash decreases)
	entry := NewJournalEntry(entryID, ref, desc, []Posting{
		{AccountID: userAcc, Asset: asset, Amount: amount, Direction: Debit},
		{AccountID: reserveAcc, Asset: asset, Amount: amount, Direction: Credit},
	})

	if err := l.RecordJournalEntry(ctx, entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// InternalTransfer performs an immediate atomic transfer between two users.
func (l *Ledger) InternalTransfer(ctx context.Context, fromUserID, toUserID int64, asset string, amount uint64, ref string, desc string) (*JournalEntry, error) {
	entryID := fmt.Sprintf("jnl:xfr:%s", ref)
	fromAcc := UserMainAccount(fromUserID, asset)
	toAcc := UserMainAccount(toUserID, asset)

	// Internal Transfer:
	// Debit fromUser:main (reduces sender balance)
	// Credit toUser:main (increases receiver balance)
	entry := NewJournalEntry(entryID, ref, desc, []Posting{
		{AccountID: fromAcc, Asset: asset, Amount: amount, Direction: Debit},
		{AccountID: toAcc, Asset: asset, Amount: amount, Direction: Credit},
	})

	if err := l.RecordJournalEntry(ctx, entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// GetAccountHistory retrieves historical journal entries for an account chronologically.
func (l *Ledger) GetAccountHistory(ctx context.Context, accountID string, limit int) ([]JournalEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	prefix := []byte(fmt.Sprintf("%s%s:", storage.NSAccountHistory, accountID))
	var entries []JournalEntry

	err := l.store.Pebble.PrefixScan(prefix, func(k, v []byte) bool {
		entryID := string(v)
		jnlKey := []byte(storage.NSJournal + entryID)
		jnlData, err := l.store.Pebble.Get(jnlKey)
		if err == nil {
			var je JournalEntry
			if err := json.Unmarshal(jnlData, &je); err == nil {
				entries = append(entries, je)
			}
		}
		return len(entries) < limit
	})

	return entries, err
}
