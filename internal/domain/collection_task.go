package domain

import (
	"errors"
	"time"
)

// TaskStatus represents the lifecycle state of a collection task.
type TaskStatus string

const (
	TaskStatusScheduled   TaskStatus = "scheduled"
	TaskStatusInProgress  TaskStatus = "in_progress"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusFailed      TaskStatus = "failed"
	TaskStatusCancelled   TaskStatus = "cancelled"
)

// IsValid checks if a TaskStatus is one of the known values.
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusScheduled, TaskStatusInProgress, TaskStatusCompleted,
		TaskStatusFailed, TaskStatusCancelled:
		return true
	}
	return false
}

// IsTerminal returns true for states that cannot be left.
func (s TaskStatus) IsTerminal() bool {
	return s == TaskStatusCompleted || s == TaskStatusFailed || s == TaskStatusCancelled
}

// CanTransitionTo defines allowed state transitions.
func (s TaskStatus) CanTransitionTo(next TaskStatus) bool {
	allowed := map[TaskStatus][]TaskStatus{
		TaskStatusScheduled:  {TaskStatusInProgress, TaskStatusCancelled},
		TaskStatusInProgress: {TaskStatusCompleted, TaskStatusFailed},
		TaskStatusCompleted:  {},
		TaskStatusFailed:     {TaskStatusScheduled},
		TaskStatusCancelled:  {},
	}
	for _, a := range allowed[s] {
		if a == next {
			return true
		}
	}
	return false
}

// TaskPriority represents urgency of a collection task.
type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityNormal TaskPriority = "normal"
	TaskPriorityHigh   TaskPriority = "high"
	TaskPriorityUrgent TaskPriority = "urgent"
)

// CollectionTask represents a single waste collection job.
type CollectionTask struct {
	ID               string       `json:"id"`
	PointID          string       `json:"point_id"`
	VehicleID        string       `json:"vehicle_id"`
	DriverID         string       `json:"driver_id"`
	Status           TaskStatus   `json:"status"`
	Priority         TaskPriority `json:"priority"`
	ScheduledAt      time.Time    `json:"scheduled_at"`
	StartedAt        *time.Time   `json:"started_at,omitempty"`
	CompletedAt      *time.Time   `json:"completed_at,omitempty"`
	CollectedWeightKg float64     `json:"collected_weight_kg"`
	Notes            string       `json:"notes"`
	FailureReason    string       `json:"failure_reason,omitempty"`
	CreatedBy        string       `json:"created_by"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	Version          int          `json:"version"`
}

// Start transitions the task to in_progress and records the start time.
func (t *CollectionTask) Start() error {
	if !t.Status.CanTransitionTo(TaskStatusInProgress) {
		return ErrTaskInvalidTransition
	}
	now := time.Now().UTC()
	t.StartedAt = &now
	t.Status = TaskStatusInProgress
	return nil
}

// Complete transitions the task to completed and records weight and time.
func (t *CollectionTask) Complete(collectedKg float64) error {
	if !t.Status.CanTransitionTo(TaskStatusCompleted) {
		return ErrTaskInvalidTransition
	}
	now := time.Now().UTC()
	t.CompletedAt = &now
	t.CollectedWeightKg = collectedKg
	t.Status = TaskStatusCompleted
	return nil
}

// Fail transitions the task to failed with a reason.
func (t *CollectionTask) Fail(reason string) error {
	if !t.Status.CanTransitionTo(TaskStatusFailed) {
		return ErrTaskInvalidTransition
	}
	t.FailureReason = reason
	t.Status = TaskStatusFailed
	return nil
}

// Cancel transitions the task to cancelled.
func (t *CollectionTask) Cancel() error {
	if !t.Status.CanTransitionTo(TaskStatusCancelled) {
		return ErrTaskInvalidTransition
	}
	t.Status = TaskStatusCancelled
	return nil
}

// Errors related to collection tasks.
var (
	ErrTaskNotFound          = errors.New("collection task not found")
	ErrTaskInvalidTransition = errors.New("invalid task status transition")
	ErrTaskVersionConflict   = errors.New("collection task was modified by another request, please retry")
	ErrTaskAlreadyTerminal   = errors.New("collection task is already in a terminal state")
)
