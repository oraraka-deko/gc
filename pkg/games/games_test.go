package games

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"gc/pkg/ledger"
	"gc/pkg/storage"
)

func setupTestGameSuite(t testing.TB) (*GameServer, *ledger.Ledger, func()) {
	tmpDir, err := os.MkdirTemp("", "game_test_*")
	if err != nil {
		t.Fatal(err)
	}

	store, err := storage.New(storage.Config{
		PebblePath:     filepath.Join(tmpDir, "pebble"),
		FastCacheBytes: 32 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	led := ledger.New(store)
	srv := NewGameServer(led, store)

	cleanup := func() {
		srv.Close()
		_ = store.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return srv, led, cleanup
}

func TestBallEscape_FullLifecycle(t *testing.T) {
	srv, led, cleanup := setupTestGameSuite(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(101)

	// 1. Deposit 500 Stars
	_, err := led.Deposit(ctx, userID, ledger.AssetStars, 500, "dep:be:1", "Deposit")
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// 2. Start round with 100 Stars
	res, err := srv.BallEscape.StartRound(ctx, userID, ledger.AssetStars, 100, "medium")
	if err != nil {
		t.Fatalf("start round failed: %v", err)
	}

	if res.RoundID <= 0 || res.ServerSeed == "" || res.Sig == "" {
		t.Fatalf("invalid start round result: %+v", res)
	}
	if res.UpdatedBalance != 400 {
		t.Fatalf("expected 400 Stars available after 100 Stars reserve, got %d", res.UpdatedBalance)
	}

	// 3. Test tampered signature claim -> must fail
	_, err = srv.BallEscape.ClaimRound(ctx, userID, res.RoundID, res.ServerSeed, "tampered_sig", 2.0, true, 100)
	if err == nil {
		t.Fatalf("expected tampered sig to fail claim")
	}

	// 4. Test valid win claim (2.5x multiplier)
	claimRes, err := srv.BallEscape.ClaimRound(ctx, userID, res.RoundID, res.ServerSeed, res.Sig, 2.5, true, 100)
	if err != nil {
		t.Fatalf("valid claim failed: %v", err)
	}

	// Expected payout = 100 * 2.5 = 250 Stars
	if claimRes.PayoutAtomic != 250 {
		t.Fatalf("expected 250 Stars payout, got %d", claimRes.PayoutAtomic)
	}

	// Balance should now be: 400 + 250 = 650 Stars
	userAcc, _ := led.GetAccount(ctx, ledger.UserMainAccount(userID, ledger.AssetStars))
	if userAcc.Available != 650 {
		t.Fatalf("expected 650 Stars balance, got %d", userAcc.Available)
	}

	// 5. Replay claim attempt -> must fail (round consumed)
	_, err = srv.BallEscape.ClaimRound(ctx, userID, res.RoundID, res.ServerSeed, res.Sig, 2.5, true, 100)
	if err == nil {
		t.Fatalf("expected replay claim to fail")
	}
}

func TestTowerClimb_FullLifecycle(t *testing.T) {
	srv, led, cleanup := setupTestGameSuite(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(202)

	// Deposit 10 TON (10 * 10^9 nanoTON)
	depositNano := ledger.TONToNano(10.0)
	_, err := led.Deposit(ctx, userID, ledger.AssetTON, depositNano, "dep:tw:1", "Deposit")
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// 1. Start round on "easy" (1 trap per row) with 2 TON
	betNano := ledger.TONToNano(2.0)
	startRes, err := srv.Tower.StartRound(ctx, userID, ledger.AssetTON, betNano, "easy")
	if err != nil {
		t.Fatalf("start tower failed: %v", err)
	}

	if startRes.SeedHash == "" || startRes.RoundID <= 0 {
		t.Fatalf("invalid start result: %+v", startRes)
	}
	if startRes.UpdatedBalance != ledger.TONToNano(8.0) {
		t.Fatalf("expected 8 TON available, got %d", startRes.UpdatedBalance)
	}

	// Retrieve session to inspect bomb row
	srv.Tower.mu.RLock()
	session := srv.Tower.sessions[userID]
	bombColRow0 := session.BombCols[0][0]
	srv.Tower.mu.RUnlock()

	// 2. Pick a safe column for step 0
	safeCol := (bombColRow0 + 1) % 5
	stepRes, err := srv.Tower.PickTile(ctx, userID, startRes.RoundID, 0, safeCol)
	if err != nil {
		t.Fatalf("pick tile failed: %v", err)
	}
	if stepRes.Status != "playing" || stepRes.Step != 1 {
		t.Fatalf("expected step 1 playing, got status=%s, step=%d", stepRes.Status, stepRes.Step)
	}

	// 3. Cashout at step 1
	cashRes, err := srv.Tower.Cashout(ctx, userID, startRes.RoundID)
	if err != nil {
		t.Fatalf("cashout failed: %v", err)
	}
	if cashRes.Status != "cashedOut" || cashRes.PayoutAtomic == 0 {
		t.Fatalf("invalid cashout: %+v", cashRes)
	}

	// Verify balance is credited
	userAcc, _ := led.GetAccount(ctx, ledger.UserMainAccount(userID, ledger.AssetTON))
	if userAcc.Available <= ledger.TONToNano(8.0) {
		t.Fatalf("balance not increased after cashout: %d", userAcc.Available)
	}
}

func TestTerritory_PotAndWinner(t *testing.T) {
	srv, led, cleanup := setupTestGameSuite(t)
	defer cleanup()

	ctx := context.Background()
	user1 := int64(301)
	user2 := int64(302)

	// Deposit 1000 Stars to each
	_, _ = led.Deposit(ctx, user1, ledger.AssetStars, 1000, "dep:ter:1", "Dep")
	_, _ = led.Deposit(ctx, user2, ledger.AssetStars, 1000, "dep:ter:2", "Dep")

	// Place bets
	err := srv.Territory.PlaceBet(ctx, user1, "alice", "Alice", "", ledger.AssetStars, 150)
	if err != nil {
		t.Fatalf("place bet user1 failed: %v", err)
	}
	err = srv.Territory.PlaceBet(ctx, user2, "bob", "Bob", "", ledger.AssetStars, 250)
	if err != nil {
		t.Fatalf("place bet user2 failed: %v", err)
	}

	st := srv.Territory.GetCurrentState()
	if st.TotalPotStars < 400 {
		t.Fatalf("expected total pot >= 400 Stars, got %d", st.TotalPotStars)
	}
	if len(st.Bets) < 2 {
		t.Fatalf("expected >= 2 bets, got %d", len(st.Bets))
	}
}

func TestConcurrency_100Users_AllGamesSimultaneous(t *testing.T) {
	srv, led, cleanup := setupTestGameSuite(t)
	defer cleanup()

	ctx := context.Background()
	const userCount = 105 // 105 concurrent users (> 100 invariant)
	var wg sync.WaitGroup
	errCh := make(chan error, userCount*5)

	// 1. Initial funding for all 105 users
	for i := 1; i <= userCount; i++ {
		uid := int64(i)
		_, err := led.Deposit(ctx, uid, ledger.AssetStars, 1000, fmt.Sprintf("dep:init:%d", uid), "Init Stars")
		if err != nil {
			t.Fatalf("deposit failed for user %d: %v", uid, err)
		}
	}

	// 2. Launch 105 concurrent players distributed across all games
	for i := 1; i <= userCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			uid := int64(idx)

			switch idx % 3 {
			case 0:
				// Ball Escape User: Play round and claim
				res, err := srv.BallEscape.StartRound(ctx, uid, ledger.AssetStars, 50, "easy")
				if err != nil {
					errCh <- fmt.Errorf("user %d BE start failed: %w", uid, err)
					return
				}
				_, err = srv.BallEscape.ClaimRound(ctx, uid, res.RoundID, res.ServerSeed, res.Sig, 1.5, true, 50)
				if err != nil {
					errCh <- fmt.Errorf("user %d BE claim failed: %w", uid, err)
					return
				}

			case 1:
				// Tower Climb User: Start, pick safe tile, cash out
				st, err := srv.Tower.StartRound(ctx, uid, ledger.AssetStars, 30, "easy")
				if err != nil {
					errCh <- fmt.Errorf("user %d Tower start failed: %w", uid, err)
					return
				}
				stepRes, err := srv.Tower.PickTile(ctx, uid, st.RoundID, 0, 0)
				if err != nil {
					errCh <- fmt.Errorf("user %d Tower pick failed: %w", uid, err)
					return
				}
				if stepRes.Status == "playing" {
					_, err = srv.Tower.Cashout(ctx, uid, st.RoundID)
					if err != nil {
						errCh <- fmt.Errorf("user %d Tower cashout failed: %w", uid, err)
						return
					}
				}

			case 2:
				// Territory Pot User: Place bet
				err := srv.Territory.PlaceBet(ctx, uid, fmt.Sprintf("player_%d", uid), "Player", "", ledger.AssetStars, 20)
				if err != nil {
					errCh <- fmt.Errorf("user %d Territory bet failed: %w", uid, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent test error: %v", err)
	}
}

func Benchmark_100ConcurrentUsers_GamesLoad(b *testing.B) {
	srv, led, cleanup := setupTestGameSuite(b)
	defer cleanup()

	ctx := context.Background()
	const workers = 100

	// Fund accounts
	for i := 1; i <= workers; i++ {
		uid := int64(10000 + i)
		_, _ = led.Deposit(ctx, uid, ledger.AssetStars, 1_000_000, fmt.Sprintf("bench:dep:%d", uid), "Bench Init")
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		workerID := int64(10001 + (CryptoRandInt(workers)))
		for pb.Next() {
			res, err := srv.BallEscape.StartRound(ctx, workerID, ledger.AssetStars, 10, "easy")
			if err == nil {
				_, _ = srv.BallEscape.ClaimRound(ctx, workerID, res.RoundID, res.ServerSeed, res.Sig, 1.2, true, 10)
			}
		}
	})
}
