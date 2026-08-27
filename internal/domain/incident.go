package domain

import (
	"errors"
	"time"
)

// IncidentType categorises the kind of incident.
type IncidentType string

const (
	IncidentTypeOverflow     IncidentType = "overflow"
	IncidentTypeDamage       IncidentType = "damage"
	IncidentTypeIllegalDump  IncidentType = "illegal_dump"
	IncidentTypeVehicleFault IncidentType = "vehicle_fault"
	IncidentTypeOther        IncidentType = "other"
)

// IsValid checks if an IncidentType is known.
func (t IncidentType) IsValid() bool {
	switch t {
	case IncidentTypeOverflow, IncidentTypeDamage, IncidentTypeIllegalDump,
		IncidentTypeVehicleFault, IncidentTypeOther:
		return true
	}
	return false
}

// IncidentStatus represents the resolution state of an incident.
type IncidentStatus string

const (
	IncidentStatusOpen       IncidentStatus = "open"
	IncidentStatusInProgress IncidentStatus = "in_progress"
	IncidentStatusResolved   IncidentStatus = "resolved"
	IncidentStatusClosed     IncidentStatus = "closed"
)

// IsValid checks if an IncidentStatus is known.
func (s IncidentStatus) IsValid() bool {
	switch s {
	case IncidentStatusOpen, IncidentStatusInProgress, IncidentStatusResolved, IncidentStatusClosed:
		return true
	}
	return false
}

// IncidentSeverity indicates how serious an incident is.
type IncidentSeverity string

const (
	IncidentSeverityLow      IncidentSeverity = "low"
	IncidentSeverityMedium   IncidentSeverity = "medium"
	IncidentSeverityHigh     IncidentSeverity = "high"
	IncidentSeverityCritical IncidentSeverity = "critical"
)

// Incident represents an abnormal event in the waste dispatch system.
type Incident struct {
	ID           string           `json:"id"`
	Type         IncidentType     `json:"type"`
	Severity     IncidentSeverity `json:"severity"`
	Status       IncidentStatus   `json:"status"`
	PointID      string           `json:"point_id,omitempty"`
	VehicleID    string           `json:"vehicle_id,omitempty"`
	TaskID       string           `json:"task_id,omitempty"`
	ReportedBy   string           `json:"reported_by"`
	AssignedTo   string           `json:"assigned_to,omitempty"`
	Description  string           `json:"description"`
	Resolution   string           `json:"resolution,omitempty"`
	OccurredAt   time.Time        `json:"occurred_at"`
	ResolvedAt   *time.Time       `json:"resolved_at,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	Version      int              `json:"version"`
}

// Resolve transitions the incident to resolved and records resolution details.
func (i *Incident) Resolve(resolution string, resolvedBy string) error {
	if i.Status == IncidentStatusResolved || i.Status == IncidentStatusClosed {
		return ErrIncidentAlreadyResolved
	}
	now := time.Now().UTC()
	i.Resolution = resolution
	i.AssignedTo = resolvedBy
	i.ResolvedAt = &now
	i.Status = IncidentStatusResolved
	return nil
}

// Errors related to incidents.
var (
	ErrIncidentNotFound      = errors.New("incident not found")
	ErrIncidentAlreadyResolved = errors.New("incident is already resolved or closed")
)
