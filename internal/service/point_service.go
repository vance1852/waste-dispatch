package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// PointService handles collection point management.
type PointService struct {
	points repository.PointRepository
	log    zerolog.Logger
}

// NewPointService creates a new PointService.
func NewPointService(points repository.PointRepository, log zerolog.Logger) *PointService {
	return &PointService{points: points, log: log}
}

// CreatePointRequest holds data for creating a collection point.
type CreatePointRequest struct {
	Name            string
	Address         string
	Latitude        float64
	Longitude       float64
	District        string
	WasteCategories []domain.WasteCategory
	CapacityKg      float64
	Notes           string
}

// CreatePoint registers a new collection point.
func (s *PointService) CreatePoint(ctx context.Context, req CreatePointRequest) (*domain.CollectionPoint, error) {
	p := &domain.CollectionPoint{
		ID:              uuid.New().String(),
		Name:            req.Name,
		Address:         req.Address,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		District:        req.District,
		WasteCategories: req.WasteCategories,
		CapacityKg:      req.CapacityKg,
		Notes:           req.Notes,
		Status:          domain.PointStatusActive,
	}

	if err := s.points.Create(ctx, p); err != nil {
		return nil, err
	}

	s.log.Info().Str("point_id", p.ID).Str("name", p.Name).Msg("collection point created")
	return p, nil
}

// GetPoint retrieves a collection point by ID.
func (s *PointService) GetPoint(ctx context.Context, id string) (*domain.CollectionPoint, error) {
	return s.points.GetByID(ctx, id)
}

// ListPoints returns a paginated list of collection points.
func (s *PointService) ListPoints(ctx context.Context, filter repository.PointFilter) ([]*domain.CollectionPoint, int, error) {
	return s.points.List(ctx, filter)
}

// UpdateLoadRequest carries an updated load reading for a collection point.
type UpdateLoadRequest struct {
	PointID       string
	CurrentLoadKg float64
}

// UpdateLoad updates the current load at a collection point and recalculates status.
func (s *PointService) UpdateLoad(ctx context.Context, req UpdateLoadRequest) (*domain.CollectionPoint, error) {
	p, err := s.points.GetByID(ctx, req.PointID)
	if err != nil {
		return nil, err
	}

	p.CurrentLoadKg = req.CurrentLoadKg
	p.UpdateStatus()

	if err := s.points.UpdateWithVersion(ctx, p); err != nil {
		return nil, err
	}

	if p.Status == domain.PointStatusFull {
		s.log.Warn().
			Str("point_id", p.ID).
			Float64("fill_ratio", p.FillRatio()).
			Msg("collection point is full")
	}

	return p, nil
}

// UpdateStatus changes the operational status of a collection point.
func (s *PointService) UpdateStatus(ctx context.Context, pointID string, status domain.PointStatus) (*domain.CollectionPoint, error) {
	p, err := s.points.GetByID(ctx, pointID)
	if err != nil {
		return nil, err
	}

	if !status.IsValid() {
		return nil, domain.ErrPointNotFound
	}
	p.Status = status

	if err := s.points.UpdateWithVersion(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

// DeletePoint removes a collection point.
func (s *PointService) DeletePoint(ctx context.Context, id string) error {
	return s.points.Delete(ctx, id)
}

// ListOverCapacity returns collection points whose fill ratio exceeds threshold (0.0-1.0).
func (s *PointService) ListOverCapacity(ctx context.Context, threshold float64) ([]*domain.CollectionPoint, error) {
	if threshold <= 0 {
		threshold = 0.8
	}
	return s.points.ListOverThreshold(ctx, threshold)
}
