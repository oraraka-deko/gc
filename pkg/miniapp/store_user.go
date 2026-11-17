package miniapp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gc/pkg/storage"
)

// StoreUserManager provides persistence for MiniAppUser accounts using DurableStore.
type StoreUserManager struct {
	store storage.DurableStore
}

// NewStoreUserManager wraps any DurableStore (e.g. WriteBehindStore).
func NewStoreUserManager(store storage.DurableStore) *StoreUserManager {
	return &StoreUserManager{store: store}
}

// UserKey returns canonical key for a user profile.
func UserKey(userID int64) []byte {
	return []byte(fmt.Sprintf("%s%d", storage.NSUser, userID))
}

// UserGiftKey returns canonical key for a user's collectible gift item.
func UserGiftKey(userID int64, assetID string) []byte {
	return []byte(fmt.Sprintf("%s%d:%s", storage.NSUserGifts, userID, assetID))
}

// GetUser loads user profile from RAM/DurableStore.
func (m *StoreUserManager) GetUser(ctx context.Context, userID int64) (*MiniAppUser, error) {
	data, err := m.store.Get(UserKey(userID))
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	var user MiniAppUser
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}
	return &user, nil
}

// SaveUser persists user profile into RAM and schedules background write-behind sync.
func (m *StoreUserManager) SaveUser(ctx context.Context, user *MiniAppUser) error {
	if user == nil {
		return nil
	}
	user.UpdatedAt = time.Now().UnixMilli()

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return m.store.Set(UserKey(user.UserID), data, false)
}

// SaveGift persists a collectible gift for a user.
func (m *StoreUserManager) SaveGift(ctx context.Context, userID int64, gift CollectibleItem) error {
	data, err := json.Marshal(gift)
	if err != nil {
		return err
	}
	return m.store.Set(UserGiftKey(userID, gift.AssetID), data, false)
}

// GetUserGifts loads all collectible gifts for a user via fast in-memory PrefixScan.
func (m *StoreUserManager) GetUserGifts(ctx context.Context, userID int64) ([]CollectibleItem, error) {
	prefix := []byte(fmt.Sprintf("%s%d:", storage.NSUserGifts, userID))
	var gifts []CollectibleItem

	err := m.store.PrefixScan(prefix, func(key, val []byte) bool {
		var item CollectibleItem
		if err := json.Unmarshal(val, &item); err == nil {
			gifts = append(gifts, item)
		}
		return true
	})

	return gifts, err
}

// EnsureUser loads or creates a default user account from authenticated WebAppUser.
func (m *StoreUserManager) EnsureUser(ctx context.Context, webUser *WebAppUser) (*MiniAppUser, error) {
	if webUser == nil {
		return nil, fmt.Errorf("webUser cannot be nil")
	}

	user, err := m.GetUser(ctx, webUser.ID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		user = NewUserAccount(webUser)
		if err := m.SaveUser(ctx, user); err != nil {
			return nil, err
		}
	} else if webUser.FirstName != "" && (user.InitUser == nil || user.InitUser.FirstName != webUser.FirstName) {
		user.InitUser = webUser
		_ = m.SaveUser(ctx, user)
	}

	return user, nil
}

// UserTaskKey returns canonical key for a user's task progress.
func UserTaskKey(userID int64, taskID string) []byte {
	return []byte(fmt.Sprintf("usr:tsk:%d:%s", userID, taskID))
}

// SaveTaskProgress persists a user's task progress into RAM and WriteBehindStore.
func (m *StoreUserManager) SaveTaskProgress(ctx context.Context, p UserTaskProgress) error {
	p.LastProgressUpdate = time.Now().UnixMilli()
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return m.store.Set(UserTaskKey(p.UserID, p.TaskID), data, false)
}

// GetUserTasks returns all recorded task progress for a user via fast prefix scan.
func (m *StoreUserManager) GetUserTasks(ctx context.Context, userID int64) ([]UserTaskProgress, error) {
	prefix := []byte(fmt.Sprintf("usr:tsk:%d:", userID))
	var tasks []UserTaskProgress

	err := m.store.PrefixScan(prefix, func(key, val []byte) bool {
		var p UserTaskProgress
		if err := json.Unmarshal(val, &p); err == nil {
			tasks = append(tasks, p)
		}
		return true
	})

	return tasks, err
}
