package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// TestEarnCredit_SameKeyDoubleCredits verifies the idempotency guarantee of EarnCredit:
// when the same idempotency_key is submitted twice, the resident's balance should only
// increase once, and the second call should return the original transaction without error.
//
// If the idempotency check is missing, calling EarnCredit twice with the same key will
// update the balance twice (charging the resident double), and the second call will fail
// with a duplicate-key error — the caller receives an error AND the balance has already
// been incremented.
func TestEarnCredit_SameKeyDoubleCredits(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	svc := service.NewCreditService(repo, zerolog.Nop())
	ctx := context.Background()

	residentID := "resident-idempotency-check-" + uuid.New().String()[:8]
	idempotencyKey := "earn-key-" + uuid.New().String()

	req := service.EarnCreditRequest{
		ResidentID:     residentID,
		Amount:         200,
		IdempotencyKey: idempotencyKey,
		Description:    "垃圾分类奖励",
		CreatedBy:      "system",
	}

	// First call: must succeed.
	tx1, err := svc.EarnCredit(ctx, req)
	if err != nil {
		t.Fatalf("first EarnCredit() unexpected error: %v", err)
	}
	if tx1 == nil {
		t.Fatal("first EarnCredit returned nil transaction")
	}

	// Second call with the SAME idempotency key: must NOT return an error,
	// and must NOT credit the resident a second time.
	tx2, err := svc.EarnCredit(ctx, req)
	if err != nil {
		t.Errorf(
			"second EarnCredit() with same idempotency_key returned error %q; "+
				"idempotency guarantee requires returning the original transaction without error",
			err,
		)
	}

	if err == nil && tx2 != nil && tx1.ID != tx2.ID {
		t.Errorf(
			"second EarnCredit() returned a different transaction (first=%s, second=%s); "+
				"idempotent calls must return the same transaction",
			tx1.ID, tx2.ID,
		)
	}

	// Balance must be exactly 200 (credited once), not 400 (credited twice).
	credit, balErr := svc.GetBalance(ctx, residentID)
	if balErr != nil {
		t.Fatalf("GetBalance() error: %v", balErr)
	}
	if credit.Balance != 200 {
		t.Errorf(
			"resident balance = %d, want 200; same idempotency_key caused double-credit "+
				"(balance was incremented %d times instead of once)",
			credit.Balance, credit.Balance/200,
		)
	}
}
