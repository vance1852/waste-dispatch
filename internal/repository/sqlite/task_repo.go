package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// TaskRepository implements repository.TaskRepository for SQLite.
type TaskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a new SQLite-backed TaskRepository.
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

const taskCols = `id, point_id, vehicle_id, driver_id, status, priority, scheduled_at,
	started_at, completed_at, collected_weight_kg, notes, failure_reason, created_by, created_at, updated_at, version`

// Create inserts a new collection task.
func (r *TaskRepository) Create(ctx context.Context, t *domain.CollectionTask) error {
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	t.Version = 1

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO collection_tasks
		 (id, point_id, vehicle_id, driver_id, status, priority, scheduled_at,
		  started_at, completed_at, collected_weight_kg, notes, failure_reason, created_by, created_at, updated_at, version)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.PointID, t.VehicleID, t.DriverID, string(t.Status), string(t.Priority),
		t.ScheduledAt, t.StartedAt, t.CompletedAt, t.CollectedWeightKg,
		t.Notes, t.FailureReason, t.CreatedBy, t.CreatedAt, t.UpdatedAt, t.Version,
	)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

// GetByID retrieves a task by primary key.
func (r *TaskRepository) GetByID(ctx context.Context, id string) (*domain.CollectionTask, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+taskCols+` FROM collection_tasks WHERE id = ?`, id)
	return scanTask(row)
}

// UpdateWithVersion uses optimistic locking to update a task.
func (r *TaskRepository) UpdateWithVersion(ctx context.Context, t *domain.CollectionTask) error {
	oldVersion := t.Version
	t.UpdatedAt = time.Now().UTC()
	t.Version++

	res, err := r.db.ExecContext(ctx,
		`UPDATE collection_tasks
		 SET point_id=?, vehicle_id=?, driver_id=?, status=?, priority=?, scheduled_at=?,
		     started_at=?, completed_at=?, collected_weight_kg=?, notes=?, failure_reason=?,
		     created_by=?, updated_at=?, version=?
		 WHERE id=? AND version=?`,
		t.PointID, t.VehicleID, t.DriverID, string(t.Status), string(t.Priority), t.ScheduledAt,
		t.StartedAt, t.CompletedAt, t.CollectedWeightKg, t.Notes, t.FailureReason,
		t.CreatedBy, t.UpdatedAt, t.Version, t.ID, oldVersion,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrTaskVersionConflict
	}
	return nil
}

// Delete removes a task by ID.
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM collection_tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

// List returns paginated tasks with optional filters.
func (r *TaskRepository) List(ctx context.Context, f repository.TaskFilter) ([]*domain.CollectionTask, int, error) {
	args := []interface{}{}
	where := "WHERE 1=1"

	if f.Status != "" {
		where += " AND status = ?"
		args = append(args, string(f.Status))
	}
	if f.VehicleID != "" {
		where += " AND vehicle_id = ?"
		args = append(args, f.VehicleID)
	}
	if f.DriverID != "" {
		where += " AND driver_id = ?"
		args = append(args, f.DriverID)
	}
	if f.PointID != "" {
		where += " AND point_id = ?"
		args = append(args, f.PointID)
	}
	if f.From != nil {
		where += " AND scheduled_at >= ?"
		args = append(args, *f.From)
	}
	if f.To != nil {
		where += " AND scheduled_at <= ?"
		args = append(args, *f.To)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM collection_tasks "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	query := fmt.Sprintf(`SELECT `+taskCols+` FROM collection_tasks %s ORDER BY scheduled_at DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.CollectionTask
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}
	return tasks, total, rows.Err()
}

// ListStaleInProgress returns in_progress tasks started before olderThan.
func (r *TaskRepository) ListStaleInProgress(ctx context.Context, olderThan time.Time) ([]*domain.CollectionTask, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+taskCols+` FROM collection_tasks
		 WHERE status = ? AND started_at IS NOT NULL AND started_at < ?
		 ORDER BY started_at`,
		string(domain.TaskStatusInProgress), olderThan,
	)
	if err != nil {
		return nil, fmt.Errorf("list stale in-progress: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.CollectionTask
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func scanTask(row *sql.Row) (*domain.CollectionTask, error) {
	t := &domain.CollectionTask{}
	var startedAt, completedAt sql.NullTime
	err := row.Scan(
		&t.ID, &t.PointID, &t.VehicleID, &t.DriverID, &t.Status, &t.Priority,
		&t.ScheduledAt, &startedAt, &completedAt,
		&t.CollectedWeightKg, &t.Notes, &t.FailureReason,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, fmt.Errorf("scan task: %w", err)
	}
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return t, nil
}

func scanTaskRow(rows *sql.Rows) (*domain.CollectionTask, error) {
	t := &domain.CollectionTask{}
	var startedAt, completedAt sql.NullTime
	err := rows.Scan(
		&t.ID, &t.PointID, &t.VehicleID, &t.DriverID, &t.Status, &t.Priority,
		&t.ScheduledAt, &startedAt, &completedAt,
		&t.CollectedWeightKg, &t.Notes, &t.FailureReason,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("scan task row: %w", err)
	}
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return t, nil
}
