package domain

import (
	"errors"
	"time"
)

// VehicleStatus represents the operational state of a vehicle.
type VehicleStatus string

const (
	VehicleStatusIdle        VehicleStatus = "idle"
	VehicleStatusDispatched  VehicleStatus = "dispatched"
	VehicleStatusInService   VehicleStatus = "in_service"
	VehicleStatusMaintenance VehicleStatus = "maintenance"
	VehicleStatusRetired     VehicleStatus = "retired"
)

// IsValid checks if a VehicleStatus is one of the known values.
func (s VehicleStatus) IsValid() bool {
	switch s {
	case VehicleStatusIdle, VehicleStatusDispatched, VehicleStatusInService,
		VehicleStatusMaintenance, VehicleStatusRetired:
		return true
	}
	return false
}

// CanTransitionTo returns true if the vehicle can move to the next status.
func (s VehicleStatus) CanTransitionTo(next VehicleStatus) bool {
	allowed := map[VehicleStatus][]VehicleStatus{
		VehicleStatusIdle:        {VehicleStatusDispatched, VehicleStatusMaintenance, VehicleStatusRetired},
		VehicleStatusDispatched:  {VehicleStatusInService, VehicleStatusIdle},
		VehicleStatusInService:   {VehicleStatusIdle, VehicleStatusMaintenance},
		VehicleStatusMaintenance: {VehicleStatusIdle, VehicleStatusRetired},
		VehicleStatusRetired:     {},
	}
	for _, a := range allowed[s] {
		if a == next {
			return true
		}
	}
	return false
}

// VehicleType categorises a collection vehicle.
type VehicleType string

const (
	VehicleTypeCompactor   VehicleType = "compactor"
	VehicleTypeTruck       VehicleType = "truck"
	VehicleTypeElectric    VehicleType = "electric"
	VehicleTypeSpecialized VehicleType = "specialized"
)

// Vehicle is the core vehicle entity.
type Vehicle struct {
	ID             string        `json:"id"`
	PlateNumber    string        `json:"plate_number"`
	Type           VehicleType   `json:"type"`
	Capacity       float64       `json:"capacity_kg"`
	Status         VehicleStatus `json:"status"`
	DriverID       string        `json:"driver_id,omitempty"`
	LastServicedAt *time.Time    `json:"last_serviced_at,omitempty"`
	Notes          string        `json:"notes"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Version        int           `json:"version"`
}

// Assign sets the driver and transitions the vehicle to dispatched.
func (v *Vehicle) Assign(driverID string) error {
	if !v.Status.CanTransitionTo(VehicleStatusDispatched) {
		return ErrVehicleInvalidTransition
	}
	v.DriverID = driverID
	v.Status = VehicleStatusDispatched
	return nil
}

// Release removes the driver and transitions the vehicle back to idle.
func (v *Vehicle) Release() error {
	if !v.Status.CanTransitionTo(VehicleStatusIdle) {
		return ErrVehicleInvalidTransition
	}
	v.DriverID = ""
	v.Status = VehicleStatusIdle
	return nil
}

// Errors related to vehicles.
var (
	ErrVehicleNotFound          = errors.New("vehicle not found")
	ErrVehicleAlreadyExists     = errors.New("vehicle with this plate number already exists")
	ErrVehicleInvalidTransition = errors.New("invalid vehicle status transition")
	ErrVehicleNotAvailable      = errors.New("vehicle is not available for dispatch")
	ErrVehicleVersionConflict   = errors.New("vehicle was modified by another request, please retry")
)
