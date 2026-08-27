package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
)

func TestTaskRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	// Insert prerequisite point and vehicle via their respective repos.
	pointRepo := reposqlite.NewPointRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	taskRepo := reposqlite.NewTaskRepository(db)
	ctx := context.Background()

	point := &domain.CollectionPoint{
		ID:         uuid.New().String(),
		Name:       "Test Point",
		Address:    "123 Main St",
		CapacityKg: 1000,
		Status:     domain.PointStatusActive,
	}
	if err := pointRepo.Create(ctx, point); err != nil {
		t.Fatalf("create point: %v", err)
	}

	vehicle := &domain.Vehicle{
		ID:          uuid.New().String(),
		PlateNumber: "粤F10001",
		Type:        domain.VehicleTypeCompactor,
		Capacity:    5000,
		Status:      domain.VehicleStatusIdle,
	}
	if err := vehicleRepo.Create(ctx, vehicle); err != nil {
		t.Fatalf("create vehicle: %v", err)
	}

	task := &domain.CollectionTask{
		ID:          uuid.New().String(),
		PointID:     point.ID,
		VehicleID:   vehicle.ID,
		DriverID:    "driver-1",
		Status:      domain.TaskStatusScheduled,
		Priority:    domain.TaskPriorityNormal,
		ScheduledAt: time.Now().UTC().Add(time.Hour),
		CreatedBy:   "operator-1",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := taskRepo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Status != domain.TaskStatusScheduled {
		t.Errorf("status = %s, want scheduled", got.Status)
	}
	if got.Version != 1 {
		t.Errorf("version = %d, want 1", got.Version)
	}
}

func TestTaskRepository_UpdateWithVersion_OptimisticLock(t *testing.T) {
	db := openTestDB(t)
	pointRepo := reposqlite.NewPointRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	taskRepo := reposqlite.NewTaskRepository(db)
	ctx := context.Background()

	point := &domain.CollectionPoint{ID: uuid.New().String(), Name: "P1", Address: "A1", CapacityKg: 500, Status: domain.PointStatusActive}
	_ = pointRepo.Create(ctx, point)
	vehicle := &domain.Vehicle{ID: uuid.New().String(), PlateNumber: "粤G20001", Type: domain.VehicleTypeCompactor, Capacity: 5000, Status: domain.VehicleStatusIdle}
	_ = vehicleRepo.Create(ctx, vehicle)

	task := &domain.CollectionTask{
		ID:          uuid.New().String(),
		PointID:     point.ID,
		VehicleID:   vehicle.ID,
		Status:      domain.TaskStatusScheduled,
		Priority:    domain.TaskPriorityNormal,
		ScheduledAt: time.Now().UTC().Add(time.Hour),
	}
	_ = taskRepo.Create(ctx, task)

	// Transition to in_progress.
	now := time.Now().UTC()
	task.StartedAt = &now
	task.Status = domain.TaskStatusInProgress
	if err := taskRepo.UpdateWithVersion(ctx, task); err != nil {
		t.Fatalf("first update error: %v", err)
	}
	if task.Version != 2 {
		t.Errorf("version = %d, want 2", task.Version)
	}

	// Stale update should fail.
	task.Version = 1
	err := taskRepo.UpdateWithVersion(ctx, task)
	if err != domain.ErrTaskVersionConflict {
		t.Errorf("expected ErrTaskVersionConflict, got %v", err)
	}
}

func TestTaskRepository_ListStaleInProgress(t *testing.T) {
	db := openTestDB(t)
	pointRepo := reposqlite.NewPointRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	taskRepo := reposqlite.NewTaskRepository(db)
	ctx := context.Background()

	point := &domain.CollectionPoint{ID: uuid.New().String(), Name: "P2", Address: "A2", CapacityKg: 500, Status: domain.PointStatusActive}
	_ = pointRepo.Create(ctx, point)
	vehicle := &domain.Vehicle{ID: uuid.New().String(), PlateNumber: "粤H30001", Type: domain.VehicleTypeCompactor, Capacity: 5000, Status: domain.VehicleStatusIdle}
	_ = vehicleRepo.Create(ctx, vehicle)

	// Create an in_progress task with a very old start time.
	staleStart := time.Now().UTC().Add(-5 * time.Hour)
	task := &domain.CollectionTask{
		ID:          uuid.New().String(),
		PointID:     point.ID,
		VehicleID:   vehicle.ID,
		Status:      domain.TaskStatusInProgress,
		Priority:    domain.TaskPriorityNormal,
		ScheduledAt: staleStart,
		StartedAt:   &staleStart,
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Use a cutoff 4 hours ago; the task started 5 hours ago so it should appear.
	cutoff := time.Now().UTC().Add(-4 * time.Hour)
	stale, err := taskRepo.ListStaleInProgress(ctx, cutoff)
	if err != nil {
		t.Fatalf("ListStaleInProgress() error: %v", err)
	}
	found := false
	for _, s := range stale {
		if s.ID == task.ID {
			found = true
		}
	}
	if !found {
		t.Error("stale task not returned by ListStaleInProgress")
	}
}

func TestTaskRepository_List_FilterByStatus(t *testing.T) {
	db := openTestDB(t)
	pointRepo := reposqlite.NewPointRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	taskRepo := reposqlite.NewTaskRepository(db)
	ctx := context.Background()

	point := &domain.CollectionPoint{ID: uuid.New().String(), Name: "P3", Address: "A3", CapacityKg: 500, Status: domain.PointStatusActive}
	_ = pointRepo.Create(ctx, point)
	vehicle := &domain.Vehicle{ID: uuid.New().String(), PlateNumber: "粤I40001", Type: domain.VehicleTypeCompactor, Capacity: 5000, Status: domain.VehicleStatusIdle}
	_ = vehicleRepo.Create(ctx, vehicle)

	for i := 0; i < 3; i++ {
		v2 := &domain.Vehicle{ID: uuid.New().String(), PlateNumber: uuid.New().String()[:9], Type: domain.VehicleTypeCompactor, Capacity: 5000, Status: domain.VehicleStatusIdle}
		_ = vehicleRepo.Create(ctx, v2)
		task := &domain.CollectionTask{
			ID:          uuid.New().String(),
			PointID:     point.ID,
			VehicleID:   v2.ID,
			Status:      domain.TaskStatusScheduled,
			Priority:    domain.TaskPriorityNormal,
			ScheduledAt: time.Now().UTC().Add(time.Duration(i) * time.Hour),
		}
		_ = taskRepo.Create(ctx, task)
	}

	tasks, total, err := taskRepo.List(ctx, repository.TaskFilter{
		Status: domain.TaskStatusScheduled,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total < 3 {
		t.Errorf("total = %d, want >= 3", total)
	}
	for _, task := range tasks {
		if task.Status != domain.TaskStatusScheduled {
			t.Errorf("got task with status %s, want scheduled", task.Status)
		}
	}
}
