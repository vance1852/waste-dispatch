package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vance1852/waste-dispatch/internal/domain"
)

// CreditRepository implements repository.CreditRepository for SQLite.
type CreditRepository struct {
	db *sql.DB
}

// NewCreditRepository creates a new SQLite-backed CreditRepository.
func NewCreditRepository(db *sql.DB) *CreditRepository {
	return &CreditRepository{db: db}
}

// GetOrCreateByResidentID returns existing credit account or creates one atomically.
func (r *CreditRepository) GetOrCreateByResidentID(ctx context.Context, residentID string) (*domain.ResidentCredit, error) {
	c := &domain.ResidentCredit{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, resident_id, balance, created_at, updated_at, version FROM resident_credits WHERE resident_id = ?`,
		residentID,
	).Scan(&c.ID, &c.ResidentID, &c.Balance, &c.CreatedAt, &c.UpdatedAt, &c.Version)
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get credit: %w", err)
	}

	// Create new credit account.
	now := time.Now().UTC()
	c = &domain.ResidentCredit{
		ID:         uuid.New().String(),
		ResidentID: residentID,
		Balance:    0,
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO resident_credits (id, resident_id, balance, created_at, updated_at, version) VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.ResidentID, c.Balance, c.CreatedAt, c.UpdatedAt, c.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("create credit: %w", err)
	}
	// Re-read in case another goroutine won the race.
	err = r.db.QueryRowContext(ctx,
		`SELECT id, resident_id, balance, created_at, updated_at, version FROM resident_credits WHERE resident_id = ?`,
		residentID,
	).Scan(&c.ID, &c.ResidentID, &c.Balance, &c.CreatedAt, &c.UpdatedAt, &c.Version)
	if err != nil {
		return nil, fmt.Errorf("re-read credit: %w", err)
	}
	return c, nil
}

// UpdateBalance persists an updated balance using optimistic locking.
func (r *CreditRepository) UpdateBalance(ctx context.Context, c *domain.ResidentCredit) error {
	oldVersion := c.Version
	c.UpdatedAt = time.Now().UTC()
	c.Version++

	res, err := r.db.ExecContext(ctx,
		`UPDATE resident_credits SET balance=?, updated_at=?, version=? WHERE id=? AND version=?`,
		c.Balance, c.UpdatedAt, c.Version, c.ID, oldVersion,
	)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("credit balance conflict: %w", domain.ErrCreditNotFound)
	}
	return nil
}

// RecordTransaction inserts a credit transaction record.
func (r *CreditRepository) RecordTransaction(ctx context.Context, tx *domain.CreditTransaction) error {
	now := time.Now().UTC()
	tx.CreatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO credit_transactions
		 (id, resident_id, type, amount, balance_after, idempotency_key, ref_id, description, created_at, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tx.ID, tx.ResidentID, string(tx.Type), tx.Amount, tx.BalanceAfter,
		tx.IdempotencyKey, nullString(tx.RefID), tx.Description, tx.CreatedAt, tx.CreatedBy,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrDuplicateTransaction
		}
		return fmt.Errorf("record transaction: %w", err)
	}
	return nil
}

// GetTransactionByIdempotencyKey retrieves an existing transaction by its idempotency key.
func (r *CreditRepository) GetTransactionByIdempotencyKey(ctx context.Context, key string) (*domain.CreditTransaction, error) {
	tx := &domain.CreditTransaction{}
	var refID sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, resident_id, type, amount, balance_after, idempotency_key, ref_id, description, created_at, created_by
		 FROM credit_transactions WHERE idempotency_key = ?`, key,
	).Scan(&tx.ID, &tx.ResidentID, &tx.Type, &tx.Amount, &tx.BalanceAfter,
		&tx.IdempotencyKey, &refID, &tx.Description, &tx.CreatedAt, &tx.CreatedBy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, fmt.Errorf("get transaction by key: %w", err)
	}
	if refID.Valid {
		tx.RefID = refID.String
	}
	return tx, nil
}

// ListTransactions returns paginated transactions for a resident.
func (r *CreditRepository) ListTransactions(ctx context.Context, residentID string, limit, offset int) ([]*domain.CreditTransaction, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM credit_transactions WHERE resident_id = ?`, residentID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, resident_id, type, amount, balance_after, idempotency_key, ref_id, description, created_at, created_by
		 FROM credit_transactions WHERE resident_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		residentID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var txs []*domain.CreditTransaction
	for rows.Next() {
		tx := &domain.CreditTransaction{}
		var refID sql.NullString
		if err := rows.Scan(
			&tx.ID, &tx.ResidentID, &tx.Type, &tx.Amount, &tx.BalanceAfter,
			&tx.IdempotencyKey, &refID, &tx.Description, &tx.CreatedAt, &tx.CreatedBy,
		); err != nil {
			return nil, 0, fmt.Errorf("scan transaction: %w", err)
		}
		if refID.Valid {
			tx.RefID = refID.String
		}
		txs = append(txs, tx)
	}
	return txs, total, rows.Err()
}
