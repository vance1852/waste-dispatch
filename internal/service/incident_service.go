package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// IncidentService manages incident reporting and resolution.
type IncidentService struct {
	incidents repository.IncidentRepository
	log       zerolog.Logger
}

// NewIncidentService creates a new IncidentService.
func NewIncidentService(incidents repository.IncidentRepository, log zerolog.Logger) *IncidentService {
	return &IncidentService{incidents: incidents, log: log}
}

// ReportIncidentRequest carries data for reporting a new incident.
type ReportIncidentRequest struct {
	Type        domain.IncidentType
	Severity    domain.IncidentSeverity
	PointID     string
	VehicleID   string
	TaskID      string
	ReportedBy  string
	Description string
	OccurredAt  time.Time
}

// ReportIncident creates a new incident record.
func (s *IncidentService) ReportIncident(ctx context.Context, req ReportIncidentRequest) (*domain.Incident, error) {
	if req.OccurredAt.IsZero() {
		req.OccurredAt = time.Now().UTC()
	}

	incident := &domain.Incident{
		ID:          uuid.New().String(),
		Type:        req.Type,
		Severity:    req.Severity,
		Status:      domain.IncidentStatusOpen,
		PointID:     req.PointID,
		VehicleID:   req.VehicleID,
		TaskID:      req.TaskID,
		ReportedBy:  req.ReportedBy,
		Description: req.Description,
		OccurredAt:  req.OccurredAt,
	}

	if err := s.incidents.Create(ctx, incident); err != nil {
		return nil, err
	}

	s.log.Warn().
		Str("incident_id", incident.ID).
		Str("type", string(incident.Type)).
		Str("severity", string(incident.Severity)).
		Msg("incident reported")

	return incident, nil
}

// RegisterOverflowOnce records an overflow incident for a collection point only
// when that point has no incident of the same kind still being handled.
func (s *IncidentService) RegisterOverflowOnce(
	ctx context.Context,
	pointID string,
	description string,
) (*domain.Incident, bool, error) {
	active, err := s.incidents.CountActiveForPoint(ctx, pointID, domain.IncidentTypeOverflow)
	if err != nil {
		return nil, false, fmt.Errorf("check active overflow incidents: %w", err)
	}
	if active > 0 {
		s.log.Debug().
			Str("point_id", pointID).
			Int("active", active).
			Msg("overflow incident already tracked for collection point")
		return nil, false, nil
	}

	incident, err := s.ReportIncident(ctx, ReportIncidentRequest{
		Type:        domain.IncidentTypeOverflow,
		Severity:    domain.IncidentSeverityHigh,
		PointID:     pointID,
		ReportedBy:  "system",
		Description: description,
		OccurredAt:  time.Now().UTC(),
	})
	if err != nil {
		return nil, false, err
	}
	return incident, true, nil
}

// GetIncident retrieves an incident by ID.
func (s *IncidentService) GetIncident(ctx context.Context, id string) (*domain.Incident, error) {
	return s.incidents.GetByID(ctx, id)
}

// ListIncidents returns a paginated list of incidents.
func (s *IncidentService) ListIncidents(ctx context.Context, filter repository.IncidentFilter) ([]*domain.Incident, int, error) {
	return s.incidents.List(ctx, filter)
}

// AssignIncidentRequest carries assignment details.
type AssignIncidentRequest struct {
	IncidentID string
	AssignedTo string
}

// AssignIncident assigns an incident to a user and moves it to in_progress.
func (s *IncidentService) AssignIncident(ctx context.Context, req AssignIncidentRequest) (*domain.Incident, error) {
	incident, err := s.incidents.GetByID(ctx, req.IncidentID)
	if err != nil {
		return nil, err
	}

	incident.AssignedTo = req.AssignedTo
	incident.Status = domain.IncidentStatusInProgress

	if err := s.incidents.Update(ctx, incident); err != nil {
		return nil, err
	}

	s.log.Info().
		Str("incident_id", incident.ID).
		Str("assigned_to", req.AssignedTo).
		Msg("incident assigned")

	return incident, nil
}

// ResolveIncidentRequest carries resolution details.
type ResolveIncidentRequest struct {
	IncidentID  string
	Resolution  string
	ResolvedBy  string
}

// ResolveIncident marks an incident as resolved.
func (s *IncidentService) ResolveIncident(ctx context.Context, req ResolveIncidentRequest) (*domain.Incident, error) {
	incident, err := s.incidents.GetByID(ctx, req.IncidentID)
	if err != nil {
		return nil, err
	}

	if err := incident.Resolve(req.Resolution, req.ResolvedBy); err != nil {
		return nil, err
	}

	if err := s.incidents.Update(ctx, incident); err != nil {
		return nil, err
	}

	s.log.Info().
		Str("incident_id", incident.ID).
		Str("resolved_by", req.ResolvedBy).
		Msg("incident resolved")

	return incident, nil
}

// CloseIncident moves a resolved incident to closed.
func (s *IncidentService) CloseIncident(ctx context.Context, incidentID string) (*domain.Incident, error) {
	incident, err := s.incidents.GetByID(ctx, incidentID)
	if err != nil {
		return nil, err
	}

	if incident.Status != domain.IncidentStatusResolved {
		incident.Status = domain.IncidentStatusClosed
	} else {
		incident.Status = domain.IncidentStatusClosed
	}

	if err := s.incidents.Update(ctx, incident); err != nil {
		return nil, err
	}

	return incident, nil
}
