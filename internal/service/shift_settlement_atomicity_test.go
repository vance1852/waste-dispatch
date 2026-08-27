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

// TestShiftSettlementLeavesNothingHalfClosed checks that closing a collection
// shift is all-or-nothing. A shift often contains two runs to the same busy
// collection point; if the second run cannot be settled, the whole settlement
// must be refused and neither the first run nor that point's load may be left
// already updated.
func TestShiftSettlementLeavesNothingHalfClosed(t *testing.T) {
	db := openServiceTestDB(t)
	taskRepo := reposqlite.NewTaskRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	pointRepo := reposqlite.NewPointRepository(db)
	settlementRepo := reposqlite.NewShiftSettlementRepository(db)
	svc := service.NewTaskService(taskRepo, vehicleRepo, pointRepo, zerolog.Nop()).
		WithShiftSettlement(settlementRepo)
	ctx := context.Background()

	point := &domain.CollectionPoint{
		ID:            uuid.New().String(),
		Name:          "文昌里生活垃圾投放点",
		Address:       "文昌里 3 号",
		District:      "文昌",
		CapacityKg:    1200,
		CurrentLoadKg: 800,
		Status:        domain.PointStatusActive,
	}
	if err := pointRepo.Create(ctx, point); err != nil {
		t.Fatalf("Create point error: %v", err)
	}

	taskIDs := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		vehicle := &domain.Vehicle{
			ID:          uuid.New().String(),
			PlateNumber: "粤W" + uuid.New().String()[:6],
			Type:        domain.VehicleTypeCompactor,
			Capacity:    5000,
			Status:      domain.VehicleStatusIdle,
		}
		if err := vehicleRepo.Create(ctx, vehicle); err != nil {
			t.Fatalf("Create vehicle error: %v", err)
		}
		startedAt := time.Now().UTC().Add(-2 * time.Hour)
		task := &domain.CollectionTask{
			ID:          uuid.New().String(),
			PointID:     point.ID,
			VehicleID:   vehicle.ID,
			DriverID:    "driver-early",
			Status:      domain.TaskStatusInProgress,
			Priority:    domain.TaskPriorityNormal,
			ScheduledAt: startedAt,
			StartedAt:   &startedAt,
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Create task error: %v", err)
		}
		taskIDs = append(taskIDs, task.ID)
	}

	settleErr := svc.SettleShift(ctx, taskIDs, map[string]float64{
		taskIDs[0]: 300,
		taskIDs[1]: 250,
	})
	if settleErr == nil {
		t.Fatal("settling a shift whose second run cannot be applied must fail instead of reporting success")
	}

	firstTask, err := svc.GetTask(ctx, taskIDs[0])
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if firstTask.Status == domain.TaskStatusCompleted {
		t.Errorf(
			"first run of the shift was already closed (status=%s) although the shift settlement failed; "+
				"a refused settlement must not leave part of the shift closed",
			firstTask.Status,
		)
	}

	secondTask, err := svc.GetTask(ctx, taskIDs[1])
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if secondTask.Status == domain.TaskStatusCompleted {
		t.Errorf("second run was closed (status=%s) although the settlement failed", secondTask.Status)
	}

	storedPoint, err := pointRepo.GetByID(ctx, point.ID)
	if err != nil {
		t.Fatalf("GetByID point error: %v", err)
	}
	if storedPoint.CurrentLoadKg != 800 {
		t.Errorf(
			"collection point load became %.1f kg although the shift settlement failed, want it to stay 800",
			storedPoint.CurrentLoadKg,
		)
	}
}
