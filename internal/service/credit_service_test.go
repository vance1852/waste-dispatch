package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

func TestCreditService_EarnAndRedeem(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	svc := service.NewCreditService(repo, zerolog.Nop())
	ctx := context.Background()

	residentID := "resident-earn-001"

	// Earn credits.
	tx, err := svc.EarnCredit(ctx, service.EarnCreditRequest{
		ResidentID:     residentID,
		Amount:         100,
		IdempotencyKey: uuid.New().String(),
		Description:    "recycling",
		CreatedBy:      "system",
	})
	if err != nil {
		t.Fatalf("EarnCredit() error: %v", err)
	}
	if tx.Amount != 100 {
		t.Errorf("amount = %d, want 100", tx.Amount)
	}

	// Check balance.
	credit, _ := svc.GetBalance(ctx, residentID)
	if credit.Balance != 100 {
		t.Errorf("balance = %d, want 100", credit.Balance)
	}

	// Redeem some credits.
	redeemTx, err := svc.RedeemCredit(ctx, service.RedeemCreditRequest{
		ResidentID:     residentID,
		Amount:         30,
		IdempotencyKey: uuid.New().String(),
		Description:    "reward claim",
		CreatedBy:      "system",
	})
	if err != nil {
		t.Fatalf("RedeemCredit() error: %v", err)
	}
	if redeemTx.BalanceAfter != 70 {
		t.Errorf("balance_after = %d, want 70", redeemTx.BalanceAfter)
	}
}

func TestCreditService_Earn_Idempotent(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	svc := service.NewCreditService(repo, zerolog.Nop())
	ctx := context.Background()

	key := uuid.New().String()
	req := service.EarnCreditRequest{
		ResidentID: "resident-idempotent-001", Amount: 50,
		IdempotencyKey: key, CreatedBy: "system",
	}

	tx1, err := svc.EarnCredit(ctx, req)
	if err != nil {
		t.Fatalf("first EarnCredit() error: %v", err)
	}

	// Same key, same operation - should return original tx.
	tx2, err := svc.EarnCredit(ctx, req)
	if err != nil {
		t.Fatalf("second EarnCredit() (idempotent) error: %v", err)
	}
	if tx1.ID != tx2.ID {
		t.Error("idempotent earn should return same transaction")
	}

	// Balance should only be credited once.
	credit, _ := svc.GetBalance(ctx, "resident-idempotent-001")
	if credit.Balance != 50 {
		t.Errorf("balance = %d, want 50 (should not double-credit)", credit.Balance)
	}
}

func TestCreditService_Redeem_InsufficientBalance(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	svc := service.NewCreditService(repo, zerolog.Nop())
	ctx := context.Background()

	// Earn 10 credits.
	_ , _ = svc.EarnCredit(ctx, service.EarnCreditRequest{
		ResidentID: "resident-insuf-001", Amount: 10,
		IdempotencyKey: uuid.New().String(), CreatedBy: "system",
	})

	// Try to redeem 50 - should fail.
	_, err := svc.RedeemCredit(ctx, service.RedeemCreditRequest{
		ResidentID: "resident-insuf-001", Amount: 50,
		IdempotencyKey: uuid.New().String(), CreatedBy: "system",
	})
	if !errors.Is(err, domain.ErrInsufficientCredit) {
		t.Errorf("expected ErrInsufficientCredit, got %v", err)
	}
}

func TestCreditService_ListTransactions_Pagination(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	svc := service.NewCreditService(repo, zerolog.Nop())
	ctx := context.Background()

	residentID := "resident-list-001"
	for i := 0; i < 5; i++ {
		_, _ = svc.EarnCredit(ctx, service.EarnCreditRequest{
			ResidentID:     residentID,
			Amount:         int64(10 * (i + 1)),
			IdempotencyKey: uuid.New().String(),
			CreatedBy:      "system",
		})
	}

	txs, total, err := svc.ListTransactions(ctx, residentID, 3, 0)
	if err != nil {
		t.Fatalf("ListTransactions() error: %v", err)
	}
	if total < 5 {
		t.Errorf("total = %d, want >= 5", total)
	}
	if len(txs) > 3 {
		t.Errorf("got %d transactions, want <= 3", len(txs))
	}
}

func TestCreditService_ConcurrentEarn_Idempotent(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	svc := service.NewCreditService(repo, zerolog.Nop())
	ctx := context.Background()

	residentID := "resident-concurrent-svc"
	// Pre-create the credit account to avoid the race in GetOrCreate.
	_, _ = svc.GetBalance(ctx, residentID)

	idempotencyKey := uuid.New().String()

	const n = 5
	results := make([]*domain.CreditTransaction, n)
	errs := make([]error, n)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tx, err := svc.EarnCredit(ctx, service.EarnCreditRequest{
				ResidentID:     residentID,
				Amount:         100,
				IdempotencyKey: idempotencyKey,
				CreatedBy:      "system",
			})
			results[idx] = tx
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	// Balance should be at most 100 (credited at most once due to idempotency key).
	credit, err := svc.GetBalance(ctx, residentID)
	if err != nil {
		t.Fatalf("GetBalance() error: %v", err)
	}
	if credit.Balance > 100 {
		t.Errorf("concurrent balance = %d, should not exceed 100 (idempotency prevents double-credit)", credit.Balance)
	}
	// At least one goroutine must have succeeded.
	anySuccess := false
	for _, e := range errs {
		if e == nil {
			anySuccess = true
			break
		}
	}
	if !anySuccess {
		t.Error("at least one concurrent earn call should succeed")
	}
	_ = results
}
