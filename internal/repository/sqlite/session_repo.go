package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vance1852/waste-dispatch/internal/domain"
)

// SessionRepository implements repository.SessionRepository for SQLite.
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new SQLite-backed SessionRepository.
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

const sessionCols = `id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, last_seen_at, revoked_at`

// Create inserts a new session record.
func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	now := time.Now().UTC()
	s.CreatedAt = now
	s.LastSeenAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, last_seen_at, revoked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.TokenHash, s.UserAgent, s.IPAddress,
		s.ExpiresAt, s.CreatedAt, s.LastSeenAt, s.RevokedAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// GetByTokenHash retrieves a session by its hashed token.
func (r *SessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+sessionCols+` FROM sessions WHERE token_hash = ?`, tokenHash)
	return scanSession(row)
}

// GetByID retrieves a session by primary key.
func (r *SessionRepository) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+sessionCols+` FROM sessions WHERE id = ?`, id)
	return scanSession(row)
}

// UpdateLastSeen updates the last_seen_at timestamp for a session.
func (r *SessionRepository) UpdateLastSeen(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ? WHERE id = ?`,
		time.Now().UTC(), id,
	)
	return err
}

// Revoke marks a session as revoked.
func (r *SessionRepository) Revoke(ctx context.Context, id string) error {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

// RevokeAllForUser revokes all active sessions for a user.
func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`,
		now, userID,
	)
	return err
}

// PurgeIdle removes sessions that have not been seen for the given period.
func (r *SessionRepository) PurgeIdle(ctx context.Context, idleFor time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-idleFor)
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE last_seen_at < ?`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("purge idle sessions: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// DeleteExpired removes all expired or revoked sessions older than 30 days.
func (r *SessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)`,
		time.Now().UTC(), cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func scanSession(row *sql.Row) (*domain.Session, error) {
	s := &domain.Session{}
	var revokedAt sql.NullTime
	err := row.Scan(
		&s.ID, &s.UserID, &s.TokenHash, &s.UserAgent, &s.IPAddress,
		&s.ExpiresAt, &s.CreatedAt, &s.LastSeenAt, &revokedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("scan session: %w", err)
	}
	if revokedAt.Valid {
		s.RevokedAt = &revokedAt.Time
	}
	return s, nil
}
