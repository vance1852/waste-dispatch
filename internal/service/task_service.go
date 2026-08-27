package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
)

// TaskService manages collection task lifecycle.
type TaskService struct {
	tasks    repository.TaskRepository
	vehicles repository.VehicleRepository
	points   repository.PointRepository
	log      zerolog.Logger
}

// NewTaskService creates a new TaskService.
func NewTaskService(
	tasks repository.TaskRepository,
	vehicles repository.VehicleRepository,
	points repository.PointRepository,
	log zerolog.Logger,
) *TaskService {
	return &TaskService{tasks: tasks, vehicles: vehicles, points: points, log: log}
}

// CreateTaskRequest holds data for scheduling a new collection task.
type CreateTaskRequest struct {
	PointID     string
	VehicleID   string
	DriverID    string
	Priority    domain.TaskPriority
	ScheduledAt time.Time
	Notes       string
	CreatedBy   string
}

// CreateTask schedules a new collection task, validating point and vehicle existence.
func (s *TaskService) CreateTask(ctx context.Context, req CreateTaskRequest) (*domain.CollectionTask, error) {
	// Validate point exists.
	if _, err := s.points.GetByID(ctx, req.PointID); err != nil {
		return nil, fmt.Errorf("invalid point_id: %w", err)
	}

	// Validate vehicle exists.
	v, err := s.vehicles.GetByID(ctx, req.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("invalid vehicle_id: %w", err)
	}

	if v.Status != domain.VehicleStatusIdle && v.Status != domain.VehicleStatusDispatched {
		return nil, domain.ErrVehicleNotAvailable
	}

	priority := req.Priority
	if priority == "" {
		priority = domain.TaskPriorityNormal
	}

	task := &domain.CollectionTask{
		ID:          uuid.New().String(),
		PointID:     req.PointID,
		VehicleID:   req.VehicleID,
		DriverID:    req.DriverID,
		Status:      domain.TaskStatusScheduled,
		Priority:    priority,
		ScheduledAt: req.ScheduledAt,
		Notes:       req.Notes,
		CreatedBy:   req.CreatedBy,
	}

	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, err
	}

	s.log.Info().
		Str("task_id", task.ID).
		Str("point_id", task.PointID).
		Str("vehicle_id", task.VehicleID).
		Msg("collection task scheduled")

	return task, nil
}

// GetTask retrieves a task by ID.
func (s *TaskService) GetTask(ctx context.Context, id string) (*domain.CollectionTask, error) {
	return s.tasks.GetByID(ctx, id)
}

// ListTasks returns a paginated, filtered list of tasks.
func (s *TaskService) ListTasks(ctx context.Context, filter repository.TaskFilter) ([]*domain.CollectionTask, int, error) {
	return s.tasks.List(ctx, filter)
}

// StartTask transitions a task to in_progress.
func (s *TaskService) StartTask(ctx context.Context, taskID string) (*domain.CollectionTask, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if err := task.Start(); err != nil {
		return nil, err
	}

	if err := s.tasks.UpdateWithVersion(ctx, task); err != nil {
		return nil, err
	}

	s.log.Info().Str("task_id", taskID).Msg("collection task started")
	return task, nil
}

// CompleteTask transitions a task to completed and updates the collection point load.
func (s *TaskService) CompleteTask(ctx context.Context, taskID string, collectedKg float64) (*domain.CollectionTask, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if err := task.Complete(collectedKg); err != nil {
		return nil, err
	}

	if err := s.tasks.UpdateWithVersion(ctx, task); err != nil {
		return nil, err
	}

	// Update point load after successful collection.
	point, err := s.points.GetByID(ctx, task.PointID)
	if err == nil {
		newLoad := point.CurrentLoadKg - collectedKg
		if newLoad < 0 {
			newLoad = 0
		}
		point.CurrentLoadKg = newLoad
		point.UpdateStatus()
		if updateErr := s.points.UpdateWithVersion(ctx, point); updateErr != nil {
			s.log.Warn().Err(updateErr).Str("point_id", task.PointID).Msg("failed to update point load after task completion")
		}
	}

	s.log.Info().
		Str("task_id", taskID).
		Float64("collected_kg", collectedKg).
		Msg("collection task completed")

	return task, nil
}

// FailTask transitions a task to failed with a reason.
func (s *TaskService) FailTask(ctx context.Context, taskID, reason string) (*domain.CollectionTask, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if err := task.Fail(reason); err != nil {
		return nil, err
	}

	if err := s.tasks.UpdateWithVersion(ctx, task); err != nil {
		return nil, err
	}

	s.log.Warn().Str("task_id", taskID).Str("reason", reason).Msg("collection task failed")
	return task, nil
}

// CancelTask cancels a scheduled task.
func (s *TaskService) CancelTask(ctx context.Context, taskID string) (*domain.CollectionTask, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if err := task.Cancel(); err != nil {
		return nil, err
	}

	if err := s.tasks.UpdateWithVersion(ctx, task); err != nil {
		return nil, err
	}

	s.log.Info().Str("task_id", taskID).Msg("collection task cancelled")
	return task, nil
}

// RecoverStale reschedules or fails tasks that have been in_progress too long.
func (s *TaskService) RecoverStale(ctx context.Context, staleCutoff time.Time) (int, error) {
	tasks, err := s.tasks.ListStaleInProgress(ctx, staleCutoff)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, t := range tasks {
		if err := t.Fail("auto-recovery: task exceeded maximum in-progress duration"); err != nil {
			s.log.Error().Err(err).Str("task_id", t.ID).Msg("failed to mark stale task as failed")
			continue
		}
		if err := s.tasks.UpdateWithVersion(ctx, t); err != nil {
			s.log.Error().Err(err).Str("task_id", t.ID).Msg("failed to save recovered task")
			continue
		}
		count++
	}

	return count, nil
}
