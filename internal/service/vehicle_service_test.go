package service_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

func TestVehicleService_CreateAndGet(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	svc := service.NewVehicleService(repo, zerolog.Nop())
	ctx := context.Background()

	v, err := svc.CreateVehicle(ctx, service.CreateVehicleRequest{
		PlateNumber: "粤Z10001",
		Type:        domain.VehicleTypeCompactor,
		CapacityKg:  8000,
	})
	if err != nil {
		t.Fatalf("CreateVehicle() error: %v", err)
	}

	got, err := svc.GetVehicle(ctx, v.ID)
	if err != nil {
		t.Fatalf("GetVehicle() error: %v", err)
	}
	if got.Status != domain.VehicleStatusIdle {
		t.Errorf("new vehicle status = %s, want idle", got.Status)
	}
}

func TestVehicleService_AssignAndRelease(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	svc := service.NewVehicleService(repo, zerolog.Nop())
	ctx := context.Background()

	v, _ := svc.CreateVehicle(ctx, service.CreateVehicleRequest{
		PlateNumber: "粤Z20001",
		Type:        domain.VehicleTypeElectric,
		CapacityKg:  3000,
	})

	// Assign driver.
	assigned, err := svc.AssignDriver(ctx, v.ID, "driver-abc")
	if err != nil {
		t.Fatalf("AssignDriver() error: %v", err)
	}
	if assigned.Status != domain.VehicleStatusDispatched {
		t.Errorf("status = %s, want dispatched", assigned.Status)
	}
	if assigned.DriverID != "driver-abc" {
		t.Errorf("DriverID = %q, want driver-abc", assigned.DriverID)
	}

	// Release.
	released, err := svc.ReleaseVehicle(ctx, v.ID)
	if err != nil {
		t.Fatalf("ReleaseVehicle() error: %v", err)
	}
	if released.Status != domain.VehicleStatusIdle {
		t.Errorf("status after release = %s, want idle", released.Status)
	}
	if released.DriverID != "" {
		t.Errorf("DriverID after release = %q, want empty", released.DriverID)
	}
}

func TestVehicleService_AssignDriver_RejectsNonIdleVehicle(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	svc := service.NewVehicleService(repo, zerolog.Nop())
	ctx := context.Background()

	v, _ := svc.CreateVehicle(ctx, service.CreateVehicleRequest{
		PlateNumber: "粤Z30001",
		Type:        domain.VehicleTypeCompactor,
		CapacityKg:  5000,
	})

	if _, err := svc.AssignDriver(ctx, v.ID, "driver-A"); err != nil {
		t.Fatalf("first AssignDriver error: %v", err)
	}

	// A vehicle that is no longer idle must not accept another driver.
	if _, err := svc.AssignDriver(ctx, v.ID, "driver-B"); err == nil {
		t.Error("expected error when assigning a driver to a dispatched vehicle")
	}

	reloaded, err := svc.GetVehicle(ctx, v.ID)
	if err != nil {
		t.Fatalf("GetVehicle error: %v", err)
	}
	if reloaded.DriverID != "driver-A" {
		t.Errorf("driver_id = %q, want driver-A to be preserved", reloaded.DriverID)
	}
}

func TestVehicleService_ListVehicles(t *testing.T) {
	db := openServiceTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	svc := service.NewVehicleService(repo, zerolog.Nop())
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _ = svc.CreateVehicle(ctx, service.CreateVehicleRequest{
			PlateNumber: "粤Z" + string(rune('A'+i)) + "0001",
			Type:        domain.VehicleTypeTruck,
			CapacityKg:  4000,
		})
	}

	vehicles, total, err := svc.ListVehicles(ctx, repository.VehicleFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListVehicles() error: %v", err)
	}
	if total < 3 {
		t.Errorf("total = %d, want >= 3", total)
	}
	_ = vehicles
}

func TestVehicleService_DeleteVehicle(t *testing.T) {	db := openServiceTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	svc := service.NewVehicleService(repo, zerolog.Nop())
	ctx := context.Background()

	v, _ := svc.CreateVehicle(ctx, service.CreateVehicleRequest{
		PlateNumber: "粤Z40001",
		Type:        domain.VehicleTypeTruck,
		CapacityKg:  2000,
	})

	if err := svc.DeleteVehicle(ctx, v.ID); err != nil {
		t.Fatalf("DeleteVehicle() error: %v", err)
	}

	_, err := svc.GetVehicle(ctx, v.ID)
	if err != domain.ErrVehicleNotFound {
		t.Errorf("expected ErrVehicleNotFound after deletion, got %v", err)
	}
}
