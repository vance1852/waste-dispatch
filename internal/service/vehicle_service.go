package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// VehicleService handles vehicle management operations.
type VehicleService struct {
	vehicles repository.VehicleRepository
	log      zerolog.Logger
}

// NewVehicleService creates a new VehicleService.
func NewVehicleService(vehicles repository.VehicleRepository, log zerolog.Logger) *VehicleService {
	return &VehicleService{vehicles: vehicles, log: log}
}

// CreateVehicleRequest holds data for creating a vehicle.
type CreateVehicleRequest struct {
	PlateNumber string
	Type        domain.VehicleType
	CapacityKg  float64
	Notes       string
}

// CreateVehicle registers a new vehicle in the system.
func (s *VehicleService) CreateVehicle(ctx context.Context, req CreateVehicleRequest) (*domain.Vehicle, error) {
	v := &domain.Vehicle{
		ID:          uuid.New().String(),
		PlateNumber: req.PlateNumber,
		Type:        req.Type,
		Capacity:    req.CapacityKg,
		Status:      domain.VehicleStatusIdle,
		Notes:       req.Notes,
	}

	if err := s.vehicles.Create(ctx, v); err != nil {
		return nil, err
	}

	s.log.Info().Str("vehicle_id", v.ID).Str("plate", v.PlateNumber).Msg("vehicle created")
	return v, nil
}

// GetVehicle retrieves a vehicle by its ID.
func (s *VehicleService) GetVehicle(ctx context.Context, id string) (*domain.Vehicle, error) {
	return s.vehicles.GetByID(ctx, id)
}

// ListVehicles returns a paginated list of vehicles with optional filters.
func (s *VehicleService) ListVehicles(ctx context.Context, filter repository.VehicleFilter) ([]*domain.Vehicle, int, error) {
	return s.vehicles.List(ctx, filter)
}

// AssignDriver assigns a driver to a vehicle using optimistic locking.
func (s *VehicleService) AssignDriver(ctx context.Context, vehicleID, driverID string) (*domain.Vehicle, error) {
	v, err := s.vehicles.GetByID(ctx, vehicleID)
	if err != nil {
		return nil, err
	}

	if err := v.Assign(driverID); err != nil {
		return nil, err
	}

	if err := s.vehicles.UpdateWithVersion(ctx, v); err != nil {
		return nil, fmt.Errorf("assign driver: %w", err)
	}

	s.log.Info().Str("vehicle_id", vehicleID).Str("driver_id", driverID).Msg("driver assigned")
	return v, nil
}

// ReleaseVehicle removes the driver and sets vehicle back to idle.
func (s *VehicleService) ReleaseVehicle(ctx context.Context, vehicleID string) (*domain.Vehicle, error) {
	v, err := s.vehicles.GetByID(ctx, vehicleID)
	if err != nil {
		return nil, err
	}

	if err := v.Release(); err != nil {
		return nil, err
	}

	if err := s.vehicles.UpdateWithVersion(ctx, v); err != nil {
		return nil, fmt.Errorf("release vehicle: %w", err)
	}

	s.log.Info().Str("vehicle_id", vehicleID).Msg("vehicle released")
	return v, nil
}

// UpdateStatus transitions a vehicle to a new status.
func (s *VehicleService) UpdateStatus(ctx context.Context, vehicleID string, newStatus domain.VehicleStatus) (*domain.Vehicle, error) {
	v, err := s.vehicles.GetByID(ctx, vehicleID)
	if err != nil {
		return nil, err
	}

	if !v.Status.CanTransitionTo(newStatus) {
		return nil, domain.ErrVehicleInvalidTransition
	}
	v.Status = newStatus

	if err := s.vehicles.UpdateWithVersion(ctx, v); err != nil {
		return nil, err
	}

	s.log.Info().Str("vehicle_id", vehicleID).Str("status", string(newStatus)).Msg("vehicle status updated")
	return v, nil
}

// DeleteVehicle removes a vehicle from the system.
func (s *VehicleService) DeleteVehicle(ctx context.Context, id string) error {
	return s.vehicles.Delete(ctx, id)
}

// ListAvailableVehicles returns all idle vehicles.
func (s *VehicleService) ListAvailableVehicles(ctx context.Context) ([]*domain.Vehicle, error) {
	return s.vehicles.ListAvailable(ctx)
}
