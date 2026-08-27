package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// TestRedeemingMoreThanEarnedIsRefused checks that a resident cannot exchange
// more classification credits than they actually hold: the redemption must be
// refused, the balance must stay untouched, and no redemption record may appear
// in the resident's credit history.
func TestRedeemingMoreThanEarnedIsRefused(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewCreditRepository(db)
	svc := service.NewCreditService(repo, zerolog.Nop())
	ctx := context.Background()

	residentID := "resident-overdraft-" + uuid.New().String()[:8]

	if _, err := svc.EarnCredit(ctx, service.EarnCreditRequest{
		ResidentID:     residentID,
		Amount:         40,
		IdempotencyKey: uuid.New().String(),
		Description:    "厨余垃圾正确投放奖励",
		CreatedBy:      "operator-day",
	}); err != nil {
		t.Fatalf("EarnCredit error: %v", err)
	}

	_, redeemErr := svc.RedeemCredit(ctx, service.RedeemCreditRequest{
		ResidentID:     residentID,
		Amount:         120,
		IdempotencyKey: uuid.New().String(),
		Description:    "兑换环保礼品",
		CreatedBy:      "operator-day",
	})
	if redeemErr == nil {
		t.Error("redeeming more credits than the resident holds must be refused")
	} else if !isInsufficientCredit(redeemErr) {
		t.Errorf("refusal reason %v is not recognisable as an insufficient balance", redeemErr)
	}

	credit, err := svc.GetBalance(ctx, residentID)
	if err != nil {
		t.Fatalf("GetBalance error: %v", err)
	}
	if credit.Balance < 0 {
		t.Errorf("resident balance went negative (%d) after an over-sized redemption", credit.Balance)
	}
	if credit.Balance != 40 {
		t.Errorf("resident balance = %d after a refused redemption, want it to stay 40", credit.Balance)
	}

	history, total, err := svc.ListTransactions(ctx, residentID, 20, 0)
	if err != nil {
		t.Fatalf("ListTransactions error: %v", err)
	}
	if total != 1 {
		t.Errorf("credit history holds %d records, want only the single earn record", total)
	}
	for _, entry := range history {
		if entry.Type == domain.CreditTxTypeRedeem {
			t.Errorf("a redemption record %s was written although the redemption was refused", entry.ID)
		}
	}
}

func isInsufficientCredit(err error) bool {
	return errors.Is(err, domain.ErrInsufficientCredit)
}
