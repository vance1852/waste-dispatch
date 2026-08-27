package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// PointRepository implements repository.PointRepository for SQLite.
type PointRepository struct {
	db    *sql.DB
	mu    sync.RWMutex
	cache map[string]*domain.CollectionPoint
}

// NewPointRepository creates a new SQLite-backed PointRepository.
func NewPointRepository(db *sql.DB) *PointRepository {
	return &PointRepository{db: db, cache: make(map[string]*domain.CollectionPoint)}
}

// cached returns a point previously loaded in this process, if any.
func (r *PointRepository) cached(id string) (*domain.CollectionPoint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	point, ok := r.cache[id]
	return point, ok
}

// remember stores a loaded point so repeated reads avoid another SQL round trip.
func (r *PointRepository) remember(point *domain.CollectionPoint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[point.ID] = point
}

// forget drops a cached point.
func (r *PointRepository) forget(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cache, id)
}

// Create inserts a new collection point record.
func (r *PointRepository) Create(ctx context.Context, p *domain.CollectionPoint) error {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.Version = 1

	cats, err := json.Marshal(p.WasteCategories)
	if err != nil {
		return fmt.Errorf("marshal categories: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO collection_points
		 (id, name, address, latitude, longitude, district, waste_categories, capacity_kg, current_load_kg, status, notes, created_at, updated_at, version)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Address, p.Latitude, p.Longitude, p.District,
		string(cats), p.CapacityKg, p.CurrentLoadKg, string(p.Status),
		p.Notes, p.CreatedAt, p.UpdatedAt, p.Version,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrPointAlreadyExists
		}
		return fmt.Errorf("create point: %w", err)
	}
	return nil
}

// GetByID retrieves a collection point by primary key.
func (r *PointRepository) GetByID(ctx context.Context, id string) (*domain.CollectionPoint, error) {
	if point, ok := r.cached(id); ok {
		return point, nil
	}
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, address, latitude, longitude, district, waste_categories, capacity_kg, current_load_kg, status, notes, created_at, updated_at, version
		 FROM collection_points WHERE id = ?`, id)
	point, err := scanPoint(row)
	if err != nil {
		return nil, err
	}
	r.remember(point)
	return point, nil
}

// UpdateWithVersion uses optimistic locking to update a point.
func (r *PointRepository) UpdateWithVersion(ctx context.Context, p *domain.CollectionPoint) error {
	oldVersion := p.Version
	p.UpdatedAt = time.Now().UTC()
	p.Version++

	cats, err := json.Marshal(p.WasteCategories)
	if err != nil {
		return fmt.Errorf("marshal categories: %w", err)
	}

	res, err := r.db.ExecContext(ctx,
		`UPDATE collection_points
		 SET name=?, address=?, latitude=?, longitude=?, district=?, waste_categories=?,
		     capacity_kg=?, current_load_kg=?, status=?, notes=?, updated_at=?, version=?
		 WHERE id=? AND version=?`,
		p.Name, p.Address, p.Latitude, p.Longitude, p.District, string(cats),
		p.CapacityKg, p.CurrentLoadKg, string(p.Status), p.Notes, p.UpdatedAt, p.Version,
		p.ID, oldVersion,
	)
	if err != nil {
		return fmt.Errorf("update point: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrPointVersionConflict
	}
	r.remember(p)
	return nil
}

// Delete removes a collection point.
func (r *PointRepository) Delete(ctx context.Context, id string) error {
	r.forget(id)
	res, err := r.db.ExecContext(ctx, `DELETE FROM collection_points WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete point: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrPointNotFound
	}
	return nil
}

// List returns paginated collection points with optional filters.
func (r *PointRepository) List(ctx context.Context, f repository.PointFilter) ([]*domain.CollectionPoint, int, error) {
	args := []interface{}{}
	where := "WHERE 1=1"

	if f.Status != "" {
		where += " AND status = ?"
		args = append(args, string(f.Status))
	}
	if f.District != "" {
		where += " AND district = ?"
		args = append(args, f.District)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM collection_points "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count points: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	query := fmt.Sprintf(
		`SELECT id, name, address, latitude, longitude, district, waste_categories, capacity_kg, current_load_kg, status, notes, created_at, updated_at, version
		 FROM collection_points %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list points: %w", err)
	}
	defer rows.Close()

	var points []*domain.CollectionPoint
	for rows.Next() {
		p, err := scanPointRow(rows)
		if err != nil {
			return nil, 0, err
		}
		points = append(points, p)
	}
	return points, total, rows.Err()
}

// ListOverThreshold returns points whose fill ratio exceeds threshold.
func (r *PointRepository) ListOverThreshold(ctx context.Context, threshold float64) ([]*domain.CollectionPoint, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, address, latitude, longitude, district, waste_categories, capacity_kg, current_load_kg, status, notes, created_at, updated_at, version
		 FROM collection_points
		 WHERE capacity_kg > 0 AND (current_load_kg / capacity_kg) >= ?
		 ORDER BY (current_load_kg / capacity_kg) DESC`, threshold)
	if err != nil {
		return nil, fmt.Errorf("list over threshold: %w", err)
	}
	defer rows.Close()

	var points []*domain.CollectionPoint
	for rows.Next() {
		p, err := scanPointRow(rows)
		if err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func scanPoint(row *sql.Row) (*domain.CollectionPoint, error) {
	p := &domain.CollectionPoint{}
	var catsJSON string
	err := row.Scan(
		&p.ID, &p.Name, &p.Address, &p.Latitude, &p.Longitude, &p.District,
		&catsJSON, &p.CapacityKg, &p.CurrentLoadKg, &p.Status,
		&p.Notes, &p.CreatedAt, &p.UpdatedAt, &p.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPointNotFound
		}
		return nil, fmt.Errorf("scan point: %w", err)
	}
	if catsJSON != "" {
		_ = json.Unmarshal([]byte(catsJSON), &p.WasteCategories)
	}
	return p, nil
}

func scanPointRow(rows *sql.Rows) (*domain.CollectionPoint, error) {
	p := &domain.CollectionPoint{}
	var catsJSON string
	err := rows.Scan(
		&p.ID, &p.Name, &p.Address, &p.Latitude, &p.Longitude, &p.District,
		&catsJSON, &p.CapacityKg, &p.CurrentLoadKg, &p.Status,
		&p.Notes, &p.CreatedAt, &p.UpdatedAt, &p.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("scan point row: %w", err)
	}
	if catsJSON != "" {
		_ = json.Unmarshal([]byte(catsJSON), &p.WasteCategories)
	}
	return p, nil
}
