package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vance1852/waste-dispatch/internal/domain"
)

// AuditRepository implements repository.AuditRepository for SQLite.
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new SQLite-backed AuditRepository.
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Record inserts an audit log entry.
func (r *AuditRepository) Record(ctx context.Context, log *domain.AuditLog) error {
	log.CreatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_logs
		 (id, actor_id, actor_role, action, entity_type, entity_id, old_value, new_value, request_id, ip_address, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.ActorID, string(log.ActorRole), string(log.Action),
		log.EntityType, log.EntityID,
		nullString(log.OldValue), nullString(log.NewValue),
		nullString(log.RequestID), nullString(log.IPAddress),
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("record audit: %w", err)
	}
	return nil
}

// List returns paginated audit logs for a given entity.
func (r *AuditRepository) List(ctx context.Context, entityType, entityID string, limit, offset int) ([]*domain.AuditLog, int, error) {
	args := []interface{}{entityType, entityID}
	where := "WHERE entity_type = ? AND entity_id = ?"

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_logs "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, actor_id, actor_role, action, entity_type, entity_id, old_value, new_value, request_id, ip_address, created_at
		 FROM audit_logs WHERE entity_type = ? AND entity_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		entityType, entityID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		l := &domain.AuditLog{}
		var oldVal, newVal, reqID, ipAddr sql.NullString
		if err := rows.Scan(
			&l.ID, &l.ActorID, &l.ActorRole, &l.Action,
			&l.EntityType, &l.EntityID,
			&oldVal, &newVal, &reqID, &ipAddr, &l.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		if oldVal.Valid {
			l.OldValue = oldVal.String
		}
		if newVal.Valid {
			l.NewValue = newVal.String
		}
		if reqID.Valid {
			l.RequestID = reqID.String
		}
		if ipAddr.Valid {
			l.IPAddress = ipAddr.String
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}
