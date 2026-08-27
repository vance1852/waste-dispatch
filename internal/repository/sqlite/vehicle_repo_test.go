package sqlite_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
)

func TestVehicleRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	ctx := context.Background()

	v := &domain.Vehicle{
		ID:          uuid.New().String(),
		PlateNumber: "粤A12345",
		Type:        domain.VehicleTypeCompactor,
		Capacity:    8000,
		Status:      domain.VehicleStatusIdle,
	}

	if err := repo.Create(ctx, v); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, v.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.PlateNumber != v.PlateNumber {
		t.Errorf("plate = %q, want %q", got.PlateNumber, v.PlateNumber)
	}
	if got.Status != domain.VehicleStatusIdle {
		t.Errorf("status = %s, want idle", got.Status)
	}
	if got.Version != 1 {
		t.Errorf("version = %d, want 1", got.Version)
	}
}

func TestVehicleRepository_UpdateWithVersion_OptimisticLock(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	ctx := context.Background()

	v := &domain.Vehicle{
		ID:          uuid.New().String(),
		PlateNumber: "粤B99999",
		Type:        domain.VehicleTypeTruck,
		Capacity:    5000,
		Status:      domain.VehicleStatusIdle,
	}
	_ = repo.Create(ctx, v)

	// First update should succeed.
	v.Status = domain.VehicleStatusDispatched
	v.DriverID = "driver-1"
	if err := repo.UpdateWithVersion(ctx, v); err != nil {
		t.Fatalf("first UpdateWithVersion() error: %v", err)
	}
	if v.Version != 2 {
		t.Errorf("version after first update = %d, want 2", v.Version)
	}

	// Simulate concurrent update by reverting version.
	v.Version = 1
	v.Status = domain.VehicleStatusIdle
	err := repo.UpdateWithVersion(ctx, v)
	if err != domain.ErrVehicleVersionConflict {
		t.Errorf("expected ErrVehicleVersionConflict on stale version, got %v", err)
	}
}

func TestVehicleRepository_DuplicatePlate_Rejected(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	ctx := context.Background()

	v1 := &domain.Vehicle{ID: uuid.New().String(), PlateNumber: "粤C11111", Type: domain.VehicleTypeElectric, Capacity: 2000, Status: domain.VehicleStatusIdle}
	_ = repo.Create(ctx, v1)

	v2 := &domain.Vehicle{ID: uuid.New().String(), PlateNumber: "粤C11111", Type: domain.VehicleTypeElectric, Capacity: 2000, Status: domain.VehicleStatusIdle}
	err := repo.Create(ctx, v2)
	if err != domain.ErrVehicleAlreadyExists {
		t.Errorf("expected ErrVehicleAlreadyExists, got %v", err)
	}
}

func TestVehicleRepository_ListAvailable(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	ctx := context.Background()

	idle1 := &domain.Vehicle{ID: uuid.New().String(), PlateNumber: "粤D10001", Type: domain.VehicleTypeCompactor, Capacity: 6000, Status: domain.VehicleStatusIdle}
	idle2 := &domain.Vehicle{ID: uuid.New().String(), PlateNumber: "粤D10002", Type: domain.VehicleTypeCompactor, Capacity: 6000, Status: domain.VehicleStatusIdle}
	dispatched := &domain.Vehicle{ID: uuid.New().String(), PlateNumber: "粤D10003", Type: domain.VehicleTypeCompactor, Capacity: 6000, Status: domain.VehicleStatusDispatched}

	for _, v := range []*domain.Vehicle{idle1, idle2, dispatched} {
		if err := repo.Create(ctx, v); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	available, err := repo.ListAvailable(ctx)
	if err != nil {
		t.Fatalf("ListAvailable() error: %v", err)
	}

	idleCount := 0
	for _, v := range available {
		if v.Status == domain.VehicleStatusIdle {
			idleCount++
		}
	}
	if idleCount < 2 {
		t.Errorf("expected >= 2 idle vehicles, got %d", idleCount)
	}
	for _, v := range available {
		if v.Status != domain.VehicleStatusIdle {
			t.Errorf("ListAvailable returned non-idle vehicle status=%s", v.Status)
		}
	}
}

func TestVehicleRepository_Delete(t *testing.T) {
	db := openTestDB(t)
	repo := reposqlite.NewVehicleRepository(db)
	ctx := context.Background()

	v := &domain.Vehicle{ID: uuid.New().String(), PlateNumber: "粤E55555", Type: domain.VehicleTypeTruck, Capacity: 3000, Status: domain.VehicleStatusIdle}
	_ = repo.Create(ctx, v)

	if err := repo.Delete(ctx, v.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	if _, err := repo.GetByID(ctx, v.ID); err != domain.ErrVehicleNotFound {
		t.Error("expected ErrVehicleNotFound after deletion")
	}
}
