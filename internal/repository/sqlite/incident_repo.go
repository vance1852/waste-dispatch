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

// IncidentRepository implements repository.IncidentRepository for SQLite.
type IncidentRepository struct {
	db *sql.DB
}

// NewIncidentRepository creates a new SQLite-backed IncidentRepository.
func NewIncidentRepository(db *sql.DB) *IncidentRepository {
	return &IncidentRepository{db: db}
}

const incidentCols = `id, type, severity, status, point_id, vehicle_id, task_id,
	reported_by, assigned_to, description, resolution, occurred_at, resolved_at, created_at, updated_at, version`

// Create inserts a new incident record.
func (r *IncidentRepository) Create(ctx context.Context, i *domain.Incident) error {
	now := time.Now().UTC()
	i.CreatedAt = now
	i.UpdatedAt = now
	i.Version = 1

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO incidents
		 (id, type, severity, status, point_id, vehicle_id, task_id,
		  reported_by, assigned_to, description, resolution, occurred_at, resolved_at, created_at, updated_at, version)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		i.ID, string(i.Type), string(i.Severity), string(i.Status),
		nullString(i.PointID), nullString(i.VehicleID), nullString(i.TaskID),
		i.ReportedBy, nullString(i.AssignedTo),
		i.Description, i.Resolution, i.OccurredAt, i.ResolvedAt,
		i.CreatedAt, i.UpdatedAt, i.Version,
	)
	if err != nil {
		return fmt.Errorf("create incident: %w", err)
	}
	return nil
}

// GetByID retrieves an incident by primary key.
func (r *IncidentRepository) GetByID(ctx context.Context, id string) (*domain.Incident, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+incidentCols+` FROM incidents WHERE id = ?`, id)
	return scanIncident(row)
}

// Update updates all mutable incident fields.
func (r *IncidentRepository) Update(ctx context.Context, i *domain.Incident) error {
	i.UpdatedAt = time.Now().UTC()
	i.Version++

	_, err := r.db.ExecContext(ctx,
		`UPDATE incidents
		 SET type=?, severity=?, status=?, point_id=?, vehicle_id=?, task_id=?,
		     reported_by=?, assigned_to=?, description=?, resolution=?,
		     occurred_at=?, resolved_at=?, updated_at=?, version=?
		 WHERE id=?`,
		string(i.Type), string(i.Severity), string(i.Status),
		nullString(i.PointID), nullString(i.VehicleID), nullString(i.TaskID),
		i.ReportedBy, nullString(i.AssignedTo),
		i.Description, i.Resolution, i.OccurredAt, i.ResolvedAt,
		i.UpdatedAt, i.Version, i.ID,
	)
	if err != nil {
		return fmt.Errorf("update incident: %w", err)
	}
	return nil
}

// CountActiveForPoint reports how many incidents of a type still await closure.
func (r *IncidentRepository) CountActiveForPoint(
	ctx context.Context,
	pointID string,
	incidentType domain.IncidentType,
) (int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM incidents
		 WHERE point_id = ? AND type = ? AND status = ?`,
		pointID, string(incidentType), string(domain.IncidentStatusOpen),
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count active incidents: %w", err)
	}
	return total, nil
}

// List returns paginated incidents with optional filters.
func (r *IncidentRepository) List(ctx context.Context, f repository.IncidentFilter) ([]*domain.Incident, int, error) {
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
	if f.PointID != "" {
		where += " AND point_id = ?"
		args = append(args, f.PointID)
	}
	if f.VehicleID != "" {
		where += " AND vehicle_id = ?"
		args = append(args, f.VehicleID)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM incidents "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count incidents: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	query := fmt.Sprintf(`SELECT `+incidentCols+` FROM incidents %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list incidents: %w", err)
	}
	defer rows.Close()

	var incidents []*domain.Incident
	for rows.Next() {
		inc, err := scanIncidentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		incidents = append(incidents, inc)
	}
	return incidents, total, rows.Err()
}

func scanIncident(row *sql.Row) (*domain.Incident, error) {
	i := &domain.Incident{}
	var pointID, vehicleID, taskID, assignedTo sql.NullString
	var resolvedAt sql.NullTime
	err := row.Scan(
		&i.ID, &i.Type, &i.Severity, &i.Status,
		&pointID, &vehicleID, &taskID,
		&i.ReportedBy, &assignedTo,
		&i.Description, &i.Resolution, &i.OccurredAt, &resolvedAt,
		&i.CreatedAt, &i.UpdatedAt, &i.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrIncidentNotFound
		}
		return nil, fmt.Errorf("scan incident: %w", err)
	}
	if pointID.Valid {
		i.PointID = pointID.String
	}
	if vehicleID.Valid {
		i.VehicleID = vehicleID.String
	}
	if taskID.Valid {
		i.TaskID = taskID.String
	}
	if assignedTo.Valid {
		i.AssignedTo = assignedTo.String
	}
	if resolvedAt.Valid {
		i.ResolvedAt = &resolvedAt.Time
	}
	return i, nil
}

func scanIncidentRow(rows *sql.Rows) (*domain.Incident, error) {
	i := &domain.Incident{}
	var pointID, vehicleID, taskID, assignedTo sql.NullString
	var resolvedAt sql.NullTime
	err := rows.Scan(
		&i.ID, &i.Type, &i.Severity, &i.Status,
		&pointID, &vehicleID, &taskID,
		&i.ReportedBy, &assignedTo,
		&i.Description, &i.Resolution, &i.OccurredAt, &resolvedAt,
		&i.CreatedAt, &i.UpdatedAt, &i.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("scan incident row: %w", err)
	}
	if pointID.Valid {
		i.PointID = pointID.String
	}
	if vehicleID.Valid {
		i.VehicleID = vehicleID.String
	}
	if taskID.Valid {
		i.TaskID = taskID.String
	}
	if assignedTo.Valid {
		i.AssignedTo = assignedTo.String
	}
	if resolvedAt.Valid {
		i.ResolvedAt = &resolvedAt.Time
	}
	return i, nil
}
