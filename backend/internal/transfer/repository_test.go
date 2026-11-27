package transfer

import (
	"atlas-financial-core/internal/account"
	"atlas-financial-core/internal/database"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_ = godotenv.Load("../../.env")
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		t.Skip("DATABASE_URL is not set; skipping integration test")
	}

	pool, err := database.DBConn(connStr)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func randomDigits(t *testing.T, length int) string {
	t.Helper()

	maxVal := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	n, err := rand.Int(rand.Reader, maxVal)
	if err != nil {
		t.Fatalf("failed to generate random digits: %v", err)
	}
	return fmt.Sprintf("%0*d", length, n.Int64())
}

func randomUUID(t *testing.T) string {
	t.Helper()

	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("failed to generate random uuid: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func createTestAccount(t *testing.T, pool *pgxpool.Pool, currency string, balance int64) account.Account {
	t.Helper()

	accNum := "99" + randomDigits(t, 8)
	query := `INSERT INTO accounts (account_number, owner_name, currency, balance) VALUES ($1, $2, $3, $4) RETURNING id, account_number, owner_name, currency, balance, created_at, updated_at`

	rows, err := pool.Query(context.Background(), query, accNum, "Test User", currency, balance)
	if err != nil {
		t.Fatalf("failed to insert test account: %v", err)
	}

	acc, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[account.Account])
	if err != nil {
		t.Fatalf("failed to scan test account: %v", err)
	}

	return acc
}

func registerCleanup(t *testing.T, pool *pgxpool.Pool, accountIDs ...string) {
	t.Helper()

	t.Cleanup(func() {
		ctx := context.Background()
		_, err := pool.Exec(ctx, `DELETE FROM transfers WHERE account_sender_id = ANY($1) OR account_receiver_id = ANY($1)`, accountIDs)
		if err != nil {
			t.Logf("cleanup transfers warning: %v", err)
		}
		_, err = pool.Exec(ctx, `DELETE FROM accounts WHERE id = ANY($1)`, accountIDs)
		if err != nil {
			t.Logf("cleanup accounts warning: %v", err)
		}
	})
}

func getAccountBalance(t *testing.T, pool *pgxpool.Pool, accountID string) int64 {
	t.Helper()

	var balance int64
	err := pool.QueryRow(context.Background(), `SELECT balance FROM accounts WHERE id = $1`, accountID).Scan(&balance)
	if err != nil {
		t.Fatalf("failed to query balance for account %s: %v", accountID, err)
	}
	return balance
}

func countTransfersByKey(t *testing.T, pool *pgxpool.Pool, key string) int {
	t.Helper()

	var count int
	err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM transfers WHERE idempotency_key = $1`, key).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count transfers for key %s: %v", key, err)
	}
	return count
}

func TestRepository_ExecuteTransfer_Success(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	const senderInitial int64 = 500_000
	const receiverInitial int64 = 200_000
	const transferAmount int64 = 150_000
	initialTotal := senderInitial + receiverInitial

	sender := createTestAccount(t, pool, "IDR", senderInitial)
	receiver := createTestAccount(t, pool, "IDR", receiverInitial)
	registerCleanup(t, pool, sender.ID, receiver.ID)

	idemKey := "test-success-" + randomUUID(t)
	input := Transfer{
		AccountSenderID:   sender.ID,
		AccountReceiverID: receiver.ID,
		Amount:            transferAmount,
		Purpose:           "integration test transfer",
		IdempotencyKey:    idemKey,
	}

	res, err := repo.ExecuteTransfer(context.Background(), input)
	if err != nil {
		t.Fatalf("ExecuteTransfer() unexpected error = %v", err)
	}

	if res.ID == "" {
		t.Error("expected non-empty transfer ID")
	}

	if res.Amount != transferAmount {
		t.Errorf("transfer amount = %d, want %d", res.Amount, transferAmount)
	}

	senderAfter := getAccountBalance(t, pool, sender.ID)
	receiverAfter := getAccountBalance(t, pool, receiver.ID)

	if senderAfter != senderInitial-transferAmount {
		t.Errorf("sender balance = %d, want %d", senderAfter, senderInitial-transferAmount)
	}

	if receiverAfter != receiverInitial+transferAmount {
		t.Errorf("receiver balance = %d, want %d", receiverAfter, receiverInitial+transferAmount)
	}

	if senderAfter+receiverAfter != initialTotal {
		t.Errorf("total balance = %d, want %d (conservation violated)", senderAfter+receiverAfter, initialTotal)
	}

	if count := countTransfersByKey(t, pool, idemKey); count != 1 {
		t.Errorf("transfer row count = %d, want 1", count)
	}
}

func TestRepository_ExecuteTransfer_RetrySameKey(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	svc := NewService(repo)

	const senderInitial int64 = 500_000
	const receiverInitial int64 = 100_000
	const transferAmount int64 = 100_000

	sender := createTestAccount(t, pool, "IDR", senderInitial)
	receiver := createTestAccount(t, pool, "IDR", receiverInitial)
	registerCleanup(t, pool, sender.ID, receiver.ID)

	idemKey := "test-retry-" + randomUUID(t)
	input := Transfer{
		AccountSenderID:   sender.ID,
		AccountReceiverID: receiver.ID,
		Amount:            transferAmount,
		Purpose:           "retry test",
		IdempotencyKey:    idemKey,
	}

	first, err := svc.ExecuteTransfer(context.Background(), input)
	if err != nil {
		t.Fatalf("first ExecuteTransfer() error = %v", err)
	}

	second, err := svc.ExecuteTransfer(context.Background(), input)
	if err != nil {
		t.Fatalf("retry ExecuteTransfer() error = %v", err)
	}

	if second.ID != first.ID {
		t.Errorf("retry returned ID %q, want %q", second.ID, first.ID)
	}

	_, err = repo.ExecuteTransfer(context.Background(), input)
	if !errors.Is(err, ErrDuplicateIdempotencyKey) {
		t.Errorf("direct repo duplicate call error = %v, want %v", err, ErrDuplicateIdempotencyKey)
	}

	senderAfter := getAccountBalance(t, pool, sender.ID)
	receiverAfter := getAccountBalance(t, pool, receiver.ID)

	if senderAfter != senderInitial-transferAmount {
		t.Errorf("sender balance = %d, want %d (debited more than once)", senderAfter, senderInitial-transferAmount)
	}

	if receiverAfter != receiverInitial+transferAmount {
		t.Errorf("receiver balance = %d, want %d (credited more than once)", receiverAfter, receiverInitial+transferAmount)
	}

	if count := countTransfersByKey(t, pool, idemKey); count != 1 {
		t.Errorf("transfer row count = %d, want 1", count)
	}
}

func TestRepository_ExecuteTransfer_Concurrent(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	svc := NewService(repo)

	t.Run("concurrent same idempotency key", func(t *testing.T) {
		sender := createTestAccount(t, pool, "IDR", 500_000)
		receiver := createTestAccount(t, pool, "IDR", 100_000)
		registerCleanup(t, pool, sender.ID, receiver.ID)

		const workers = 5
		const transferAmount int64 = 100_000
		idemKey := "test-concurrent-idem-" + randomUUID(t)

		var wg sync.WaitGroup
		results := make([]Transfer, workers)
		errs := make([]error, workers)

		for i := range workers {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx], errs[idx] = svc.ExecuteTransfer(context.Background(), Transfer{
					AccountSenderID:   sender.ID,
					AccountReceiverID: receiver.ID,
					Amount:            transferAmount,
					Purpose:           "concurrent retry",
					IdempotencyKey:    idemKey,
				})
			}(i)
		}
		wg.Wait()

		firstID := ""
		for i := range workers {
			if errs[i] != nil {
				t.Fatalf("worker %d failed: %v", i, errs[i])
			}
			if firstID == "" {
				firstID = results[i].ID
			} else if results[i].ID != firstID {
				t.Errorf("worker %d got transfer ID %q, want %q", i, results[i].ID, firstID)
			}
		}

		if got := getAccountBalance(t, pool, sender.ID); got != 400_000 {
			t.Errorf("sender balance = %d, want 400000", got)
		}
		if got := getAccountBalance(t, pool, receiver.ID); got != 200_000 {
			t.Errorf("receiver balance = %d, want 200000", got)
		}
		if count := countTransfersByKey(t, pool, idemKey); count != 1 {
			t.Errorf("transfer row count = %d, want 1", count)
		}
	})

	t.Run("concurrent cross transfer deadlock free", func(t *testing.T) {
		accA := createTestAccount(t, pool, "IDR", 500_000)
		accB := createTestAccount(t, pool, "IDR", 500_000)
		registerCleanup(t, pool, accA.ID, accB.ID)

		const pairs = 3
		const aToB int64 = 50_000
		const bToA int64 = 30_000

		var wg sync.WaitGroup
		errCh := make(chan error, pairs*2)

		for i := range pairs {
			wg.Add(2)
			go func(idx int) {
				defer wg.Done()
				_, err := repo.ExecuteTransfer(context.Background(), Transfer{
					AccountSenderID:   accA.ID,
					AccountReceiverID: accB.ID,
					Amount:            aToB,
					Purpose:           fmt.Sprintf("A to B %d", idx),
					IdempotencyKey:    "test-cross-ab-" + randomUUID(t),
				})
				if err != nil {
					errCh <- err
				}
			}(i)

			go func(idx int) {
				defer wg.Done()
				_, err := repo.ExecuteTransfer(context.Background(), Transfer{
					AccountSenderID:   accB.ID,
					AccountReceiverID: accA.ID,
					Amount:            bToA,
					Purpose:           fmt.Sprintf("B to A %d", idx),
					IdempotencyKey:    "test-cross-ba-" + randomUUID(t),
				})
				if err != nil {
					errCh <- err
				}
			}(i)
		}
		wg.Wait()
		close(errCh)

		for err := range errCh {
			t.Fatalf("cross transfer failed: %v", err)
		}

		wantA := 500_000 - pairs*aToB + pairs*bToA
		wantB := 500_000 + pairs*aToB - pairs*bToA
		gotA := getAccountBalance(t, pool, accA.ID)
		gotB := getAccountBalance(t, pool, accB.ID)

		if gotA != wantA {
			t.Errorf("account A balance = %d, want %d", gotA, wantA)
		}
		if gotB != wantB {
			t.Errorf("account B balance = %d, want %d", gotB, wantB)
		}
		if gotA+gotB != 1_000_000 {
			t.Errorf("total balance = %d, want 1000000", gotA+gotB)
		}
	})

	t.Run("concurrent overdraft protection", func(t *testing.T) {
		sender := createTestAccount(t, pool, "IDR", 100_000)
		receiver := createTestAccount(t, pool, "IDR", 0)
		registerCleanup(t, pool, sender.ID, receiver.ID)

		const workers = 5
		var wg sync.WaitGroup
		errs := make([]error, workers)

		for i := range workers {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				_, errs[idx] = repo.ExecuteTransfer(context.Background(), Transfer{
					AccountSenderID:   sender.ID,
					AccountReceiverID: receiver.ID,
					Amount:            100_000,
					Purpose:           "overdraft race",
					IdempotencyKey:    "test-overdraft-" + randomUUID(t),
				})
			}(i)
		}
		wg.Wait()

		successCount := 0
		insufficientCount := 0
		for _, err := range errs {
			switch {
			case err == nil:
				successCount++
			case errors.Is(err, ErrInsufficientBalance):
				insufficientCount++
			default:
				t.Fatalf("unexpected error during overdraft test: %v", err)
			}
		}

		if successCount != 1 {
			t.Errorf("success count = %d, want 1", successCount)
		}
		if insufficientCount != workers-1 {
			t.Errorf("insufficient balance count = %d, want %d", insufficientCount, workers-1)
		}
		if got := getAccountBalance(t, pool, sender.ID); got != 0 {
			t.Errorf("sender balance = %d, want 0", got)
		}
		if got := getAccountBalance(t, pool, receiver.ID); got != 100_000 {
			t.Errorf("receiver balance = %d, want 100000", got)
		}
	})
}

func TestRepository_ExecuteTransfer_Rollback(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	t.Run("insufficient balance rolls back", func(t *testing.T) {
		sender := createTestAccount(t, pool, "IDR", 50_000)
		receiver := createTestAccount(t, pool, "IDR", 100_000)
		registerCleanup(t, pool, sender.ID, receiver.ID)

		idemKey := "test-rollback-balance-" + randomUUID(t)
		_, err := repo.ExecuteTransfer(context.Background(), Transfer{
			AccountSenderID:   sender.ID,
			AccountReceiverID: receiver.ID,
			Amount:            100_000,
			Purpose:           "rollback test",
			IdempotencyKey:    idemKey,
		})
		if !errors.Is(err, ErrInsufficientBalance) {
			t.Fatalf("expected ErrInsufficientBalance, got %v", err)
		}

		if got := getAccountBalance(t, pool, sender.ID); got != 50_000 {
			t.Errorf("sender balance changed to %d, want 50000", got)
		}
		if got := getAccountBalance(t, pool, receiver.ID); got != 100_000 {
			t.Errorf("receiver balance changed to %d, want 100000", got)
		}
		if count := countTransfersByKey(t, pool, idemKey); count != 0 {
			t.Errorf("transfer row count = %d, want 0", count)
		}
	})

	t.Run("currency mismatch rolls back", func(t *testing.T) {
		sender := createTestAccount(t, pool, "IDR", 500_000)
		receiver := createTestAccount(t, pool, "USD", 500_000)
		registerCleanup(t, pool, sender.ID, receiver.ID)

		idemKey := "test-rollback-currency-" + randomUUID(t)
		_, err := repo.ExecuteTransfer(context.Background(), Transfer{
			AccountSenderID:   sender.ID,
			AccountReceiverID: receiver.ID,
			Amount:            100_000,
			Purpose:           "currency mismatch test",
			IdempotencyKey:    idemKey,
		})
		if !errors.Is(err, ErrCurrenciesNotMatch) {
			t.Fatalf("expected ErrCurrenciesNotMatch, got %v", err)
		}

		if got := getAccountBalance(t, pool, sender.ID); got != 500_000 {
			t.Errorf("sender balance changed to %d, want 500000", got)
		}
		if got := getAccountBalance(t, pool, receiver.ID); got != 500_000 {
			t.Errorf("receiver balance changed to %d, want 500000", got)
		}
		if count := countTransfersByKey(t, pool, idemKey); count != 0 {
			t.Errorf("transfer row count = %d, want 0", count)
		}
	})

	t.Run("receiver not found rolls back", func(t *testing.T) {
		sender := createTestAccount(t, pool, "IDR", 500_000)
		registerCleanup(t, pool, sender.ID)

		idemKey := "test-rollback-notfound-" + randomUUID(t)
		_, err := repo.ExecuteTransfer(context.Background(), Transfer{
			AccountSenderID:   sender.ID,
			AccountReceiverID: randomUUID(t),
			Amount:            100_000,
			Purpose:           "missing receiver test",
			IdempotencyKey:    idemKey,
		})
		if !errors.Is(err, ErrAccountNotFound) {
			t.Fatalf("expected ErrAccountNotFound, got %v", err)
		}

		if got := getAccountBalance(t, pool, sender.ID); got != 500_000 {
			t.Errorf("sender balance changed to %d, want 500000", got)
		}
		if count := countTransfersByKey(t, pool, idemKey); count != 0 {
			t.Errorf("transfer row count = %d, want 0", count)
		}
	})
}
