package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// ShiftSettlementRepository closes a collection shift for SQLite.
type ShiftSettlementRepository struct {
	db *sql.DB
}

// NewShiftSettlementRepository creates a new SQLite-backed settlement repository.
func NewShiftSettlementRepository(db *sql.DB) *ShiftSettlementRepository {
	return &ShiftSettlementRepository{db: db}
}

// SettleShift persists every task completion and the matching point load.
func (r *ShiftSettlementRepository) SettleShift(ctx context.Context, entries []repository.ShiftSettlementEntry) error {
	for _, entry := range entries {
		if err := r.settleTask(ctx, entry.Task); err != nil {
			return fmt.Errorf("settle task %s: %w", entry.Task.ID, err)
		}
		if err := r.settlePoint(ctx, entry.Point); err != nil {
			return fmt.Errorf("settle point %s: %w", entry.Point.ID, err)
		}
	}
	return nil
}

func (r *ShiftSettlementRepository) settleTask(ctx context.Context, t *domain.CollectionTask) error {
	oldVersion := t.Version
	t.UpdatedAt = time.Now().UTC()
	t.Version++

	res, err := r.db.ExecContext(ctx,
		`UPDATE collection_tasks
		 SET status=?, completed_at=?, collected_weight_kg=?, updated_at=?, version=?
		 WHERE id=? AND version=?`,
		string(t.Status), t.CompletedAt, t.CollectedWeightKg, t.UpdatedAt, t.Version,
		t.ID, oldVersion,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrTaskVersionConflict
	}
	return nil
}

func (r *ShiftSettlementRepository) settlePoint(ctx context.Context, p *domain.CollectionPoint) error {
	cats, err := json.Marshal(p.WasteCategories)
	if err != nil {
		return err
	}
	oldVersion := p.Version
	p.UpdatedAt = time.Now().UTC()
	p.Version++

	res, err := r.db.ExecContext(ctx,
		`UPDATE collection_points
		 SET waste_categories=?, current_load_kg=?, status=?, updated_at=?, version=?
		 WHERE id=? AND version=?`,
		string(cats), p.CurrentLoadKg, string(p.Status), p.UpdatedAt, p.Version,
		p.ID, oldVersion,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrPointVersionConflict
	}
	return nil
}
