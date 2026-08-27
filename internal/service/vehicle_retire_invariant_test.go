package service_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// TestRetiringVehicleRespectsActiveDispatch checks that a collection vehicle
// which is still handed out to a driver cannot be retired straight away. The
// dispatch has to be given back first, otherwise the retired vehicle keeps a
// driver attached and that driver stays blocked from taking another vehicle.
func TestRetiringVehicleRespectsActiveDispatch(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	svc := service.NewVehicleService(repo, zerolog.Nop())
	ctx := context.Background()

	vehicle, err := svc.CreateVehicle(ctx, service.CreateVehicleRequest{
		PlateNumber: "粤Q70001",
		Type:        domain.VehicleTypeSpecialized,
		CapacityKg:  4200,
	})
	if err != nil {
		t.Fatalf("CreateVehicle error: %v", err)
	}

	if _, err := svc.AssignDriver(ctx, vehicle.ID, "driver-on-route"); err != nil {
		t.Fatalf("AssignDriver error: %v", err)
	}

	// The vehicle is out on a route, so retiring it now must be refused.
	if _, err := svc.RetireVehicle(ctx, vehicle.ID); err == nil {
		t.Error("retiring a vehicle that is still dispatched to a driver must be refused")
	}

	stillDispatched, err := svc.GetVehicle(ctx, vehicle.ID)
	if err != nil {
		t.Fatalf("GetVehicle error: %v", err)
	}
	if stillDispatched.Status == domain.VehicleStatusRetired {
		t.Errorf(
			"vehicle was retired while driver %q was still holding it; a retired vehicle must not keep an active dispatch",
			stillDispatched.DriverID,
		)
	}
	if stillDispatched.Status == domain.VehicleStatusRetired && stillDispatched.DriverID != "" {
		t.Errorf("retired vehicle still carries driver_id %q", stillDispatched.DriverID)
	}

	// After the dispatch is handed back, retiring must succeed and leave no driver.
	if _, err := svc.ReleaseVehicle(ctx, vehicle.ID); err != nil {
		t.Fatalf("ReleaseVehicle error: %v", err)
	}
	retired, err := svc.RetireVehicle(ctx, vehicle.ID)
	if err != nil {
		t.Fatalf("RetireVehicle after release error: %v", err)
	}
	if retired.Status != domain.VehicleStatusRetired {
		t.Errorf("vehicle status = %s after retiring an idle vehicle, want retired", retired.Status)
	}
	if retired.DriverID != "" {
		t.Errorf("retired vehicle driver_id = %q, want empty", retired.DriverID)
	}
}
