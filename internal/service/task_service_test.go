package service_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

func openServiceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// Use a temp file DB because golang-migrate opens a separate connection
	// that doesn't share the in-memory DB with our connection.
	tmpFile, err := os.CreateTemp("", "waste-dispatch-test-*.db")
	if err != nil {
		t.Fatalf("create temp db file: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	dsn := tmpFile.Name() + "?_foreign_keys=on&_busy_timeout=5000"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	migrationsPath := findMigrationsPath(t)
	driver, _ := sqlite3.WithInstance(db, &sqlite3.Config{})
	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "sqlite3", driver)
	if err != nil {
		t.Fatalf("create migrate: %v", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

func findMigrationsPath(t *testing.T) string {
	t.Helper()
	candidates := []string{"../../migrations", "../../../migrations", "migrations"}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	t.Fatal("migrations directory not found")
	return ""
}

func TestTaskService_CreateAndStartTask(t *testing.T) {
	db := openServiceTestDB(t)
	taskRepo := reposqlite.NewTaskRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	pointRepo := reposqlite.NewPointRepository(db)
	log := zerolog.Nop()
	svc := service.NewTaskService(taskRepo, vehicleRepo, pointRepo, log)
	ctx := context.Background()

	// Create prerequisite entities.
	point := &domain.CollectionPoint{
		ID: uuid.New().String(), Name: "Service-Test-Point", Address: "Test Addr",
		CapacityKg: 1000, Status: domain.PointStatusActive,
	}
	_ = pointRepo.Create(ctx, point)

	vehicle := &domain.Vehicle{
		ID: uuid.New().String(), PlateNumber: "粤X10001",
		Type: domain.VehicleTypeCompactor, Capacity: 5000, Status: domain.VehicleStatusIdle,
	}
	_ = vehicleRepo.Create(ctx, vehicle)

	// Create task.
	task, err := svc.CreateTask(ctx, service.CreateTaskRequest{
		PointID:     point.ID,
		VehicleID:   vehicle.ID,
		DriverID:    "driver-svc-1",
		Priority:    domain.TaskPriorityNormal,
		ScheduledAt: time.Now().UTC().Add(time.Hour),
		CreatedBy:   "op-1",
	})
	if err != nil {
		t.Fatalf("CreateTask() error: %v", err)
	}
	if task.Status != domain.TaskStatusScheduled {
		t.Errorf("status = %s, want scheduled", task.Status)
	}

	// Start it.
	started, err := svc.StartTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("StartTask() error: %v", err)
	}
	if started.Status != domain.TaskStatusInProgress {
		t.Errorf("status after start = %s, want in_progress", started.Status)
	}
	if started.StartedAt == nil {
		t.Error("StartedAt should be set after starting")
	}
}

func TestTaskService_CompleteTask_UpdatesPoint(t *testing.T) {
	db := openServiceTestDB(t)
	taskRepo := reposqlite.NewTaskRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	pointRepo := reposqlite.NewPointRepository(db)
	log := zerolog.Nop()
	svc := service.NewTaskService(taskRepo, vehicleRepo, pointRepo, log)
	ctx := context.Background()

	point := &domain.CollectionPoint{
		ID: uuid.New().String(), Name: "Full-Point", Address: "Addr2",
		CapacityKg: 1000, CurrentLoadKg: 800, Status: domain.PointStatusActive,
	}
	_ = pointRepo.Create(ctx, point)

	vehicle := &domain.Vehicle{
		ID: uuid.New().String(), PlateNumber: "粤X20001",
		Type: domain.VehicleTypeCompactor, Capacity: 5000, Status: domain.VehicleStatusIdle,
	}
	_ = vehicleRepo.Create(ctx, vehicle)

	task, _ := svc.CreateTask(ctx, service.CreateTaskRequest{
		PointID:     point.ID,
		VehicleID:   vehicle.ID,
		ScheduledAt: time.Now().UTC().Add(time.Hour),
	})
	_, _ = svc.StartTask(ctx, task.ID)

	completed, err := svc.CompleteTask(ctx, task.ID, 400.0)
	if err != nil {
		t.Fatalf("CompleteTask() error: %v", err)
	}
	if completed.Status != domain.TaskStatusCompleted {
		t.Errorf("status = %s, want completed", completed.Status)
	}
	if completed.CollectedWeightKg != 400.0 {
		t.Errorf("CollectedWeightKg = %f, want 400", completed.CollectedWeightKg)
	}

	// Point load should have decreased.
	updatedPoint, _ := pointRepo.GetByID(ctx, point.ID)
	if updatedPoint.CurrentLoadKg >= 800 {
		t.Errorf("point load = %f, should have decreased from 800", updatedPoint.CurrentLoadKg)
	}
}

func TestTaskService_FailTask(t *testing.T) {
	db := openServiceTestDB(t)
	taskRepo := reposqlite.NewTaskRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	pointRepo := reposqlite.NewPointRepository(db)
	log := zerolog.Nop()
	svc := service.NewTaskService(taskRepo, vehicleRepo, pointRepo, log)
	ctx := context.Background()

	point := &domain.CollectionPoint{
		ID: uuid.New().String(), Name: "Fail-Point", Address: "Addr3",
		CapacityKg: 1000, Status: domain.PointStatusActive,
	}
	_ = pointRepo.Create(ctx, point)
	vehicle := &domain.Vehicle{
		ID: uuid.New().String(), PlateNumber: "粤X30001",
		Type: domain.VehicleTypeCompactor, Capacity: 5000, Status: domain.VehicleStatusIdle,
	}
	_ = vehicleRepo.Create(ctx, vehicle)

	task, _ := svc.CreateTask(ctx, service.CreateTaskRequest{
		PointID: point.ID, VehicleID: vehicle.ID,
		ScheduledAt: time.Now().UTC().Add(time.Hour),
	})
	_, _ = svc.StartTask(ctx, task.ID)

	failed, err := svc.FailTask(ctx, task.ID, "engine failure")
	if err != nil {
		t.Fatalf("FailTask() error: %v", err)
	}
	if failed.Status != domain.TaskStatusFailed {
		t.Errorf("status = %s, want failed", failed.Status)
	}
	if failed.FailureReason != "engine failure" {
		t.Errorf("failure_reason = %q, want 'engine failure'", failed.FailureReason)
	}
}

func TestTaskService_CancelTask(t *testing.T) {
	db := openServiceTestDB(t)
	taskRepo := reposqlite.NewTaskRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	pointRepo := reposqlite.NewPointRepository(db)
	log := zerolog.Nop()
	svc := service.NewTaskService(taskRepo, vehicleRepo, pointRepo, log)
	ctx := context.Background()

	point := &domain.CollectionPoint{
		ID: uuid.New().String(), Name: "Cancel-Point", Address: "Addr4",
		CapacityKg: 1000, Status: domain.PointStatusActive,
	}
	_ = pointRepo.Create(ctx, point)
	vehicle := &domain.Vehicle{
		ID: uuid.New().String(), PlateNumber: "粤X40001",
		Type: domain.VehicleTypeCompactor, Capacity: 5000, Status: domain.VehicleStatusIdle,
	}
	_ = vehicleRepo.Create(ctx, vehicle)

	task, _ := svc.CreateTask(ctx, service.CreateTaskRequest{
		PointID: point.ID, VehicleID: vehicle.ID,
		ScheduledAt: time.Now().UTC().Add(time.Hour),
	})

	cancelled, err := svc.CancelTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("CancelTask() error: %v", err)
	}
	if cancelled.Status != domain.TaskStatusCancelled {
		t.Errorf("status = %s, want cancelled", cancelled.Status)
	}
}

func TestTaskService_RecoverStale(t *testing.T) {
	db := openServiceTestDB(t)
	taskRepo := reposqlite.NewTaskRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	pointRepo := reposqlite.NewPointRepository(db)
	log := zerolog.Nop()
	svc := service.NewTaskService(taskRepo, vehicleRepo, pointRepo, log)
	ctx := context.Background()

	point := &domain.CollectionPoint{
		ID: uuid.New().String(), Name: "Stale-Point", Address: "Addr5",
		CapacityKg: 1000, Status: domain.PointStatusActive,
	}
	_ = pointRepo.Create(ctx, point)
	vehicle := &domain.Vehicle{
		ID: uuid.New().String(), PlateNumber: "粤X50001",
		Type: domain.VehicleTypeCompactor, Capacity: 5000, Status: domain.VehicleStatusIdle,
	}
	_ = vehicleRepo.Create(ctx, vehicle)

	staleStart := time.Now().UTC().Add(-10 * time.Hour)
	staleTask := &domain.CollectionTask{
		ID:          uuid.New().String(),
		PointID:     point.ID,
		VehicleID:   vehicle.ID,
		Status:      domain.TaskStatusInProgress,
		Priority:    domain.TaskPriorityNormal,
		ScheduledAt: staleStart,
		StartedAt:   &staleStart,
	}
	_ = taskRepo.Create(ctx, staleTask)

	cutoff := time.Now().UTC().Add(-4 * time.Hour)
	count, err := svc.RecoverStale(ctx, cutoff)
	if err != nil {
		t.Fatalf("RecoverStale() error: %v", err)
	}
	if count < 1 {
		t.Errorf("RecoverStale() recovered %d tasks, want >= 1", count)
	}

	recovered, _ := svc.GetTask(ctx, staleTask.ID)
	if recovered.Status != domain.TaskStatusFailed {
		t.Errorf("recovered task status = %s, want failed", recovered.Status)
	}
}

func TestTaskService_CreateTask_InvalidPoint(t *testing.T) {
	db := openServiceTestDB(t)
	taskRepo := reposqlite.NewTaskRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	pointRepo := reposqlite.NewPointRepository(db)
	log := zerolog.Nop()
	svc := service.NewTaskService(taskRepo, vehicleRepo, pointRepo, log)
	ctx := context.Background()

	vehicle := &domain.Vehicle{
		ID: uuid.New().String(), PlateNumber: "粤X60001",
		Type: domain.VehicleTypeCompactor, Capacity: 5000, Status: domain.VehicleStatusIdle,
	}
	_ = vehicleRepo.Create(ctx, vehicle)

	_, err := svc.CreateTask(ctx, service.CreateTaskRequest{
		PointID:     "nonexistent-point",
		VehicleID:   vehicle.ID,
		ScheduledAt: time.Now().UTC().Add(time.Hour),
	})
	if err == nil {
		t.Error("expected error creating task with invalid point ID")
	}
}
