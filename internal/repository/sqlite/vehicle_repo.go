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

// VehicleRepository implements repository.VehicleRepository for SQLite.
type VehicleRepository struct {
	db *sql.DB
}

// NewVehicleRepository creates a new SQLite-backed VehicleRepository.
func NewVehicleRepository(db *sql.DB) *VehicleRepository {
	return &VehicleRepository{db: db}
}

const vehicleColumns = `id, plate_number, type, capacity_kg, status, driver_id, last_serviced_at, notes, created_at, updated_at, version`

// Create inserts a new vehicle record.
func (r *VehicleRepository) Create(ctx context.Context, v *domain.Vehicle) error {
	now := time.Now().UTC()
	v.CreatedAt = now
	v.UpdatedAt = now
	v.Version = 1

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO vehicles (id, plate_number, type, capacity_kg, status, driver_id, last_serviced_at, notes, created_at, updated_at, version)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.PlateNumber, string(v.Type), v.Capacity, string(v.Status),
		nullString(v.DriverID), v.LastServicedAt, v.Notes, v.CreatedAt, v.UpdatedAt, v.Version,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrVehicleAlreadyExists
		}
		return fmt.Errorf("create vehicle: %w", err)
	}
	return nil
}

// GetByID retrieves a vehicle by primary key.
func (r *VehicleRepository) GetByID(ctx context.Context, id string) (*domain.Vehicle, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+vehicleColumns+` FROM vehicles WHERE id = ?`, id)
	return scanVehicle(row)
}

// GetByPlate retrieves a vehicle by plate number.
func (r *VehicleRepository) GetByPlate(ctx context.Context, plate string) (*domain.Vehicle, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+vehicleColumns+` FROM vehicles WHERE plate_number = ?`, plate)
	return scanVehicle(row)
}

// UpdateWithVersion uses optimistic locking to update a vehicle.
func (r *VehicleRepository) UpdateWithVersion(ctx context.Context, v *domain.Vehicle) error {
	oldVersion := v.Version
	v.UpdatedAt = time.Now().UTC()
	v.Version++

	res, err := r.db.ExecContext(ctx,
		`UPDATE vehicles SET plate_number=?, type=?, capacity_kg=?, status=?, driver_id=?,
		 last_serviced_at=?, notes=?, updated_at=?, version=?
		 WHERE id=? AND version=?`,
		v.PlateNumber, string(v.Type), v.Capacity, string(v.Status),
		nullString(v.DriverID), v.LastServicedAt, v.Notes, v.UpdatedAt, v.Version,
		v.ID, oldVersion,
	)
	if err != nil {
		return fmt.Errorf("update vehicle: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrVehicleVersionConflict
	}
	return nil
}

// Delete removes a vehicle by ID.
func (r *VehicleRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM vehicles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete vehicle: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrVehicleNotFound
	}
	return nil
}

// List returns paginated vehicles with optional filters.
func (r *VehicleRepository) List(ctx context.Context, f repository.VehicleFilter) ([]*domain.Vehicle, int, error) {
	args := []interface{}{}
	where := "WHERE 1=1"

	if f.Status != "" {
		where += " AND status = ?"
		args = append(args, string(f.Status))
	}
	if f.Type != "" {
		where += " AND type = ?"
		args = append(args, string(f.Type))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM vehicles "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count vehicles: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	query := fmt.Sprintf(`SELECT `+vehicleColumns+` FROM vehicles %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list vehicles: %w", err)
	}
	defer rows.Close()

	var vehicles []*domain.Vehicle
	for rows.Next() {
		v, err := scanVehicleRow(rows)
		if err != nil {
			return nil, 0, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, total, rows.Err()
}

// ListAvailable returns all idle vehicles.
func (r *VehicleRepository) ListAvailable(ctx context.Context) ([]*domain.Vehicle, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+vehicleColumns+` FROM vehicles WHERE status = ? ORDER BY created_at`,
		string(domain.VehicleStatusIdle),
	)
	if err != nil {
		return nil, fmt.Errorf("list available vehicles: %w", err)
	}
	defer rows.Close()

	var vehicles []*domain.Vehicle
	for rows.Next() {
		v, err := scanVehicleRow(rows)
		if err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, rows.Err()
}

func scanVehicle(row *sql.Row) (*domain.Vehicle, error) {
	v := &domain.Vehicle{}
	var driverID sql.NullString
	var lastServicedAt sql.NullTime
	err := row.Scan(
		&v.ID, &v.PlateNumber, &v.Type, &v.Capacity, &v.Status,
		&driverID, &lastServicedAt, &v.Notes, &v.CreatedAt, &v.UpdatedAt, &v.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrVehicleNotFound
		}
		return nil, fmt.Errorf("scan vehicle: %w", err)
	}
	if driverID.Valid {
		v.DriverID = driverID.String
	}
	if lastServicedAt.Valid {
		v.LastServicedAt = &lastServicedAt.Time
	}
	return v, nil
}

func scanVehicleRow(rows *sql.Rows) (*domain.Vehicle, error) {
	v := &domain.Vehicle{}
	var driverID sql.NullString
	var lastServicedAt sql.NullTime
	err := rows.Scan(
		&v.ID, &v.PlateNumber, &v.Type, &v.Capacity, &v.Status,
		&driverID, &lastServicedAt, &v.Notes, &v.CreatedAt, &v.UpdatedAt, &v.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("scan vehicle row: %w", err)
	}
	if driverID.Valid {
		v.DriverID = driverID.String
	}
	if lastServicedAt.Valid {
		v.LastServicedAt = &lastServicedAt.Time
	}
	return v, nil
}
