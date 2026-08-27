package domain

import "time"

// AuditAction describes what kind of change was made.
type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
	AuditActionLogin  AuditAction = "login"
	AuditActionLogout AuditAction = "logout"
)

// AuditLog records a change event for compliance and troubleshooting.
type AuditLog struct {
	ID         string      `json:"id"`
	ActorID    string      `json:"actor_id"`
	ActorRole  Role        `json:"actor_role"`
	Action     AuditAction `json:"action"`
	EntityType string      `json:"entity_type"`
	EntityID   string      `json:"entity_id"`
	OldValue   string      `json:"old_value,omitempty"`
	NewValue   string      `json:"new_value,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
	IPAddress  string      `json:"ip_address,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}
