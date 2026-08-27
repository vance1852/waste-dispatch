package audit

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// Auditor records audit events for write operations.
type Auditor struct {
	repo repository.AuditRepository
	log  zerolog.Logger
}

// NewAuditor creates a new Auditor.
func NewAuditor(repo repository.AuditRepository, log zerolog.Logger) *Auditor {
	return &Auditor{repo: repo, log: log}
}

// Log records an audit entry. It does not fail the caller on write errors.
func (a *Auditor) Log(ctx context.Context, entry domain.AuditLog) {
	entry.ID = uuid.New().String()
	if err := a.repo.Record(ctx, &entry); err != nil {
		a.log.Error().Err(err).Str("action", string(entry.Action)).
			Str("entity_type", entry.EntityType).
			Str("entity_id", entry.EntityID).
			Msg("failed to record audit log")
	}
}

// LogCreate records a create event.
func (a *Auditor) LogCreate(ctx context.Context, actorID string, actorRole domain.Role,
	entityType, entityID, newValue, requestID, ip string) {
	a.Log(ctx, domain.AuditLog{
		ActorID:    actorID,
		ActorRole:  actorRole,
		Action:     domain.AuditActionCreate,
		EntityType: entityType,
		EntityID:   entityID,
		NewValue:   newValue,
		RequestID:  requestID,
		IPAddress:  ip,
	})
}

// LogUpdate records an update event.
func (a *Auditor) LogUpdate(ctx context.Context, actorID string, actorRole domain.Role,
	entityType, entityID, oldValue, newValue, requestID, ip string) {
	a.Log(ctx, domain.AuditLog{
		ActorID:    actorID,
		ActorRole:  actorRole,
		Action:     domain.AuditActionUpdate,
		EntityType: entityType,
		EntityID:   entityID,
		OldValue:   oldValue,
		NewValue:   newValue,
		RequestID:  requestID,
		IPAddress:  ip,
	})
}

// LogDelete records a delete event.
func (a *Auditor) LogDelete(ctx context.Context, actorID string, actorRole domain.Role,
	entityType, entityID, requestID, ip string) {
	a.Log(ctx, domain.AuditLog{
		ActorID:    actorID,
		ActorRole:  actorRole,
		Action:     domain.AuditActionDelete,
		EntityType: entityType,
		EntityID:   entityID,
		RequestID:  requestID,
		IPAddress:  ip,
	})
}
