package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// CreditService manages resident credit balances with idempotent transactions.
type CreditService struct {
	credits repository.CreditRepository
	log     zerolog.Logger
}

// NewCreditService creates a new CreditService.
func NewCreditService(credits repository.CreditRepository, log zerolog.Logger) *CreditService {
	return &CreditService{credits: credits, log: log}
}

// EarnCreditRequest carries data for crediting a resident.
type EarnCreditRequest struct {
	ResidentID     string
	Amount         int64
	IdempotencyKey string
	RefID          string
	Description    string
	CreatedBy      string
}

// EarnCredit adds credits to a resident's balance (idempotent).
func (s *CreditService) EarnCredit(ctx context.Context, req EarnCreditRequest) (*domain.CreditTransaction, error) {
	// Check for duplicate idempotency key.
	existing, err := s.credits.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil {
		return existing, nil // Idempotent: return the existing transaction.
	}
	if err != domain.ErrTransactionNotFound {
		return nil, fmt.Errorf("check idempotency key: %w", err)
	}

	credit, err := s.credits.GetOrCreateByResidentID(ctx, req.ResidentID)
	if err != nil {
		return nil, fmt.Errorf("get credit account: %w", err)
	}

	credit.Balance += req.Amount
	if err := s.credits.UpdateBalance(ctx, credit); err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	tx := &domain.CreditTransaction{
		ID:             uuid.New().String(),
		ResidentID:     req.ResidentID,
		Type:           domain.CreditTxTypeEarn,
		Amount:         req.Amount,
		BalanceAfter:   credit.Balance,
		IdempotencyKey: req.IdempotencyKey,
		RefID:          req.RefID,
		Description:    req.Description,
		CreatedBy:      req.CreatedBy,
	}

	if err := s.credits.RecordTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("record transaction: %w", err)
	}

	s.log.Info().
		Str("resident_id", req.ResidentID).
		Int64("amount", req.Amount).
		Int64("balance_after", credit.Balance).
		Msg("credit earned")

	return tx, nil
}

// RedeemCreditRequest carries data for redeeming credits.
type RedeemCreditRequest struct {
	ResidentID     string
	Amount         int64
	IdempotencyKey string
	RefID          string
	Description    string
	CreatedBy      string
}

// RedeemCredit deducts credits from a resident's balance (idempotent).
func (s *CreditService) RedeemCredit(ctx context.Context, req RedeemCreditRequest) (*domain.CreditTransaction, error) {
	// Check for duplicate idempotency key.
	existing, err := s.credits.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if err != domain.ErrTransactionNotFound {
		return nil, fmt.Errorf("check idempotency key: %w", err)
	}

	if _, err := s.credits.GetOrCreateByResidentID(ctx, req.ResidentID); err != nil {
		return nil, fmt.Errorf("get credit account: %w", err)
	}

	// Deducting in a single statement keeps concurrent redemptions consistent.
	balanceAfter, err := s.credits.ApplyDelta(ctx, req.ResidentID, -req.Amount)
	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	tx := &domain.CreditTransaction{
		ID:             uuid.New().String(),
		ResidentID:     req.ResidentID,
		Type:           domain.CreditTxTypeRedeem,
		Amount:         req.Amount,
		BalanceAfter:   balanceAfter,
		IdempotencyKey: req.IdempotencyKey,
		RefID:          req.RefID,
		Description:    req.Description,
		CreatedBy:      req.CreatedBy,
	}

	if err := s.credits.RecordTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("record transaction: %w", err)
	}

	s.log.Info().
		Str("resident_id", req.ResidentID).
		Int64("amount", req.Amount).
		Int64("balance_after", balanceAfter).
		Msg("credit redeemed")

	return tx, nil
}

// GetBalance returns the current credit balance for a resident.
func (s *CreditService) GetBalance(ctx context.Context, residentID string) (*domain.ResidentCredit, error) {
	return s.credits.GetOrCreateByResidentID(ctx, residentID)
}

// ListTransactions returns paginated transaction history for a resident.
func (s *CreditService) ListTransactions(ctx context.Context, residentID string, limit, offset int) ([]*domain.CreditTransaction, int, error) {
	return s.credits.ListTransactions(ctx, residentID, limit, offset)
}
