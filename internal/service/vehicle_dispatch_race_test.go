package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// TestConcurrentDriverAssignmentKeepsSingleOwner checks that a collection vehicle
// can only be handed to one driver: when two dispatchers assign the same idle
// vehicle at the same moment, exactly one call may succeed, the other must be
// rejected, and the persisted vehicle must show the winning driver.
func TestConcurrentDriverAssignmentKeepsSingleOwner(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	svc := service.NewVehicleService(repo, zerolog.Nop())
	ctx := context.Background()

	vehicle, err := svc.CreateVehicle(ctx, service.CreateVehicleRequest{
		PlateNumber: "粤R80001",
		Type:        domain.VehicleTypeCompactor,
		CapacityKg:  6000,
	})
	if err != nil {
		t.Fatalf("CreateVehicle error: %v", err)
	}

	drivers := []string{"driver-morning-shift", "driver-evening-shift"}
	start := make(chan struct{})
	results := make([]error, len(drivers))
	assigned := make([]*domain.Vehicle, len(drivers))

	var wg sync.WaitGroup
	for i, driverID := range drivers {
		wg.Add(1)
		go func(idx int, driver string) {
			defer wg.Done()
			<-start
			v, assignErr := svc.AssignDriver(ctx, vehicle.ID, driver)
			results[idx] = assignErr
			assigned[idx] = v
		}(i, driverID)
	}
	close(start)
	wg.Wait()

	success := 0
	var winner string
	for idx, assignErr := range results {
		if assignErr == nil {
			success++
			if assigned[idx] != nil {
				winner = assigned[idx].DriverID
			}
		}
	}

	if success != 1 {
		t.Errorf(
			"concurrent assignment of one idle vehicle produced %d successful hand-overs, want exactly 1; results=%v",
			success, results,
		)
	}

	stored, err := svc.GetVehicle(ctx, vehicle.ID)
	if err != nil {
		t.Fatalf("GetVehicle error: %v", err)
	}
	if stored.Status != domain.VehicleStatusDispatched {
		t.Errorf("stored vehicle status = %s, want dispatched", stored.Status)
	}
	if success == 1 && stored.DriverID != winner {
		t.Errorf(
			"stored driver_id = %q but the only successful hand-over reported %q; a losing request overwrote the owner",
			stored.DriverID, winner,
		)
	}
	if stored.DriverID != drivers[0] && stored.DriverID != drivers[1] {
		t.Errorf("stored driver_id = %q is not one of the dispatched drivers", stored.DriverID)
	}
}
