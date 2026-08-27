package domain

import (
	"errors"
	"time"
)

// CreditTxType classifies a credit transaction.
type CreditTxType string

const (
	CreditTxTypeEarn   CreditTxType = "earn"
	CreditTxTypeRedeem CreditTxType = "redeem"
	CreditTxTypeAdjust CreditTxType = "adjust"
	CreditTxTypeExpire CreditTxType = "expire"
)

// IsValid checks if a CreditTxType is known.
func (t CreditTxType) IsValid() bool {
	switch t {
	case CreditTxTypeEarn, CreditTxTypeRedeem, CreditTxTypeAdjust, CreditTxTypeExpire:
		return true
	}
	return false
}

// ResidentCredit tracks credit balance for a resident user.
type ResidentCredit struct {
	ID         string    `json:"id"`
	ResidentID string    `json:"resident_id"`
	Balance    int64     `json:"balance"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Version    int       `json:"version"`
}

// CreditTransaction records a single credit change with an idempotency key.
type CreditTransaction struct {
	ID             string       `json:"id"`
	ResidentID     string       `json:"resident_id"`
	Type           CreditTxType `json:"type"`
	Amount         int64        `json:"amount"`
	BalanceAfter   int64        `json:"balance_after"`
	IdempotencyKey string       `json:"idempotency_key"`
	RefID          string       `json:"ref_id,omitempty"`
	Description    string       `json:"description"`
	CreatedAt      time.Time    `json:"created_at"`
	CreatedBy      string       `json:"created_by"`
}

// Errors related to credits.
var (
	ErrCreditNotFound        = errors.New("resident credit record not found")
	ErrInsufficientCredit    = errors.New("insufficient credit balance")
	ErrDuplicateTransaction  = errors.New("duplicate transaction: idempotency key already used")
	ErrTransactionNotFound   = errors.New("credit transaction not found")
)
