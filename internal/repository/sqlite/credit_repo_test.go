package sqlite_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
)

func TestCreditRepository_GetOrCreate_NewResident(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	ctx := context.Background()

	credit, err := repo.GetOrCreateByResidentID(ctx, "resident-001")
	if err != nil {
		t.Fatalf("GetOrCreateByResidentID() error: %v", err)
	}
	if credit.Balance != 0 {
		t.Errorf("initial balance = %d, want 0", credit.Balance)
	}
	if credit.ResidentID != "resident-001" {
		t.Errorf("resident_id = %q, want resident-001", credit.ResidentID)
	}
}

func TestCreditRepository_GetOrCreate_Idempotent(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	ctx := context.Background()

	c1, err := repo.GetOrCreateByResidentID(ctx, "resident-002")
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	c2, err := repo.GetOrCreateByResidentID(ctx, "resident-002")
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if c1.ID != c2.ID {
		t.Error("GetOrCreate should return same record on second call")
	}
}

func TestCreditRepository_UpdateBalance_OptimisticLock(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	ctx := context.Background()

	credit, _ := repo.GetOrCreateByResidentID(ctx, "resident-003")
	credit.Balance = 100

	if err := repo.UpdateBalance(ctx, credit); err != nil {
		t.Fatalf("first UpdateBalance() error: %v", err)
	}
	if credit.Version != 2 {
		t.Errorf("version = %d, want 2", credit.Version)
	}

	// Stale update.
	credit.Version = 1
	credit.Balance = 999
	if err := repo.UpdateBalance(ctx, credit); err == nil {
		t.Error("expected error on stale UpdateBalance, got nil")
	}
}

func TestCreditRepository_RecordTransaction_Idempotency(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	ctx := context.Background()

	credit, _ := repo.GetOrCreateByResidentID(ctx, "resident-004")
	credit.Balance = 200
	_ = repo.UpdateBalance(ctx, credit)

	tx := &domain.CreditTransaction{
		ID:             uuid.New().String(),
		ResidentID:     "resident-004",
		Type:           domain.CreditTxTypeEarn,
		Amount:         50,
		BalanceAfter:   250,
		IdempotencyKey: "idempotency-key-001",
		Description:    "recycling reward",
		CreatedBy:      "system",
	}

	// First insert should succeed.
	if err := repo.RecordTransaction(ctx, tx); err != nil {
		t.Fatalf("first RecordTransaction() error: %v", err)
	}

	// Duplicate idempotency key should return unique constraint error.
	tx2 := *tx
	tx2.ID = uuid.New().String()
	if err := repo.RecordTransaction(ctx, &tx2); err != domain.ErrDuplicateTransaction {
		t.Errorf("expected ErrDuplicateTransaction, got %v", err)
	}
}

func TestCreditRepository_GetTransactionByIdempotencyKey(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	ctx := context.Background()

	_, _ = repo.GetOrCreateByResidentID(ctx, "resident-005")
	tx := &domain.CreditTransaction{
		ID:             uuid.New().String(),
		ResidentID:     "resident-005",
		Type:           domain.CreditTxTypeEarn,
		Amount:         30,
		BalanceAfter:   30,
		IdempotencyKey: "key-005",
		CreatedBy:      "op",
	}
	_ = repo.RecordTransaction(ctx, tx)

	got, err := repo.GetTransactionByIdempotencyKey(ctx, "key-005")
	if err != nil {
		t.Fatalf("GetTransactionByIdempotencyKey() error: %v", err)
	}
	if got.Amount != 30 {
		t.Errorf("amount = %d, want 30", got.Amount)
	}
}

func TestCreditRepository_GetTransactionByIdempotencyKey_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	ctx := context.Background()

	_, err := repo.GetTransactionByIdempotencyKey(ctx, "does-not-exist")
	if err != domain.ErrTransactionNotFound {
		t.Errorf("expected ErrTransactionNotFound, got %v", err)
	}
}

func TestCreditRepository_ConcurrentEarn_NoDuplicates(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	ctx := context.Background()

	_, _ = repo.GetOrCreateByResidentID(ctx, "resident-concurrent")

	const goroutines = 5
	results := make([]error, goroutines)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tx := &domain.CreditTransaction{
				ID:             uuid.New().String(),
				ResidentID:     "resident-concurrent",
				Type:           domain.CreditTxTypeEarn,
				Amount:         10,
				BalanceAfter:   int64(10 * (idx + 1)),
				IdempotencyKey: "concurrent-key", // same key for all
				CreatedBy:      "system",
			}
			results[idx] = repo.RecordTransaction(ctx, tx)
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, err := range results {
		if err == nil {
			successCount++
		}
	}
	if successCount != 1 {
		t.Errorf("exactly 1 goroutine should succeed with same idempotency key, got %d", successCount)
	}
}
