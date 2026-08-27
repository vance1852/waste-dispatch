package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// TestStaleRecoverySweepStopsWhenCancelled checks that a cancelled recovery
// sweep does not keep rewriting collection tasks: once the caller cancels, the
// sweep must report the cancellation and leave the still in-progress tasks
// untouched so the next start-up can decide what to do with them.
func TestStaleRecoverySweepStopsWhenCancelled(t *testing.T) {
	db := openServiceTestDB(t)
	taskRepo := reposqlite.NewTaskRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	pointRepo := reposqlite.NewPointRepository(db)
	svc := service.NewTaskService(taskRepo, vehicleRepo, pointRepo, zerolog.Nop())
	ctx := context.Background()

	point := &domain.CollectionPoint{
		ID:         uuid.New().String(),
		Name:       "环城东路投放点",
		Address:    "环城东路 7 号",
		CapacityKg: 900,
		Status:     domain.PointStatusActive,
	}
	if err := pointRepo.Create(ctx, point); err != nil {
		t.Fatalf("Create point error: %v", err)
	}
	vehicle := &domain.Vehicle{
		ID:          uuid.New().String(),
		PlateNumber: "粤S90001",
		Type:        domain.VehicleTypeCompactor,
		Capacity:    5000,
		Status:      domain.VehicleStatusIdle,
	}
	if err := vehicleRepo.Create(ctx, vehicle); err != nil {
		t.Fatalf("Create vehicle error: %v", err)
	}

	startedAt := time.Now().UTC().Add(-9 * time.Hour)
	stale := &domain.CollectionTask{
		ID:          uuid.New().String(),
		PointID:     point.ID,
		VehicleID:   vehicle.ID,
		DriverID:    "driver-night",
		Status:      domain.TaskStatusInProgress,
		Priority:    domain.TaskPriorityNormal,
		ScheduledAt: startedAt,
		StartedAt:   &startedAt,
	}
	if err := taskRepo.Create(ctx, stale); err != nil {
		t.Fatalf("Create task error: %v", err)
	}

	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	recovered, err := svc.RecoverStale(cancelledCtx, time.Now().UTC().Add(-4*time.Hour))
	if err == nil {
		t.Error("RecoverStale on a cancelled context should report the cancellation instead of succeeding")
	}
	if recovered != 0 {
		t.Errorf("RecoverStale reported %d recovered tasks after cancellation, want 0", recovered)
	}

	stored, getErr := svc.GetTask(ctx, stale.ID)
	if getErr != nil {
		t.Fatalf("GetTask error: %v", getErr)
	}
	if stored.Status != domain.TaskStatusInProgress {
		t.Errorf(
			"collection task status = %s after a cancelled sweep, want it to stay in_progress; "+
				"the sweep kept writing after the caller cancelled",
			stored.Status,
		)
	}
	if stored.FailureReason != "" {
		t.Errorf(
			"collection task failure_reason = %q after a cancelled sweep, want it to stay empty",
			stored.FailureReason,
		)
	}
}
