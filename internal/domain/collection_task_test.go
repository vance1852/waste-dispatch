package domain

import (
	"testing"
	"time"
)

func TestTaskStatusTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    TaskStatus
		to      TaskStatus
		allowed bool
	}{
		{"scheduled -> in_progress", TaskStatusScheduled, TaskStatusInProgress, true},
		{"scheduled -> cancelled", TaskStatusScheduled, TaskStatusCancelled, true},
		{"scheduled -> completed", TaskStatusScheduled, TaskStatusCompleted, false},
		{"in_progress -> completed", TaskStatusInProgress, TaskStatusCompleted, true},
		{"in_progress -> failed", TaskStatusInProgress, TaskStatusFailed, true},
		{"in_progress -> scheduled", TaskStatusInProgress, TaskStatusScheduled, false},
		{"completed -> any", TaskStatusCompleted, TaskStatusScheduled, false},
		{"cancelled -> any", TaskStatusCancelled, TaskStatusInProgress, false},
		{"failed -> scheduled", TaskStatusFailed, TaskStatusScheduled, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.from.CanTransitionTo(tc.to)
			if got != tc.allowed {
				t.Errorf("CanTransitionTo(%s->%s) = %v, want %v", tc.from, tc.to, got, tc.allowed)
			}
		})
	}
}

func TestCollectionTask_Start(t *testing.T) {
	task := &CollectionTask{
		ID:          "t1",
		Status:      TaskStatusScheduled,
		ScheduledAt: time.Now().UTC(),
		Version:     1,
	}

	if err := task.Start(); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if task.Status != TaskStatusInProgress {
		t.Errorf("status = %s, want %s", task.Status, TaskStatusInProgress)
	}
	if task.StartedAt == nil {
		t.Error("StartedAt should be set after Start()")
	}
}

func TestCollectionTask_StartFromInvalidState(t *testing.T) {
	task := &CollectionTask{Status: TaskStatusCompleted}
	if err := task.Start(); err == nil {
		t.Error("expected error when starting a completed task")
	}
}

func TestCollectionTask_Complete(t *testing.T) {
	now := time.Now().UTC()
	task := &CollectionTask{
		ID:          "t1",
		Status:      TaskStatusInProgress,
		StartedAt:   &now,
		ScheduledAt: now,
		Version:     1,
	}

	const collected = 120.5
	if err := task.Complete(collected); err != nil {
		t.Fatalf("Complete() unexpected error: %v", err)
	}
	if task.Status != TaskStatusCompleted {
		t.Errorf("status = %s, want %s", task.Status, TaskStatusCompleted)
	}
	if task.CollectedWeightKg != collected {
		t.Errorf("CollectedWeightKg = %f, want %f", task.CollectedWeightKg, collected)
	}
	if task.CompletedAt == nil {
		t.Error("CompletedAt should be set after Complete()")
	}
}

func TestCollectionTask_CompleteFromScheduled_Rejected(t *testing.T) {
	task := &CollectionTask{Status: TaskStatusScheduled}
	if err := task.Complete(0); err == nil {
		t.Error("expected ErrTaskInvalidTransition when completing a scheduled task")
	}
}

func TestCollectionTask_Fail(t *testing.T) {
	now := time.Now().UTC()
	task := &CollectionTask{
		Status:    TaskStatusInProgress,
		StartedAt: &now,
		Version:   1,
	}
	const reason = "vehicle breakdown"
	if err := task.Fail(reason); err != nil {
		t.Fatalf("Fail() unexpected error: %v", err)
	}
	if task.Status != TaskStatusFailed {
		t.Errorf("status = %s, want %s", task.Status, TaskStatusFailed)
	}
	if task.FailureReason != reason {
		t.Errorf("FailureReason = %q, want %q", task.FailureReason, reason)
	}
}

func TestCollectionTask_Cancel(t *testing.T) {
	task := &CollectionTask{Status: TaskStatusScheduled, Version: 1}
	if err := task.Cancel(); err != nil {
		t.Fatalf("Cancel() unexpected error: %v", err)
	}
	if task.Status != TaskStatusCancelled {
		t.Errorf("status = %s, want %s", task.Status, TaskStatusCancelled)
	}
}

func TestCollectionTask_CancelFromInProgress_Rejected(t *testing.T) {
	now := time.Now().UTC()
	task := &CollectionTask{Status: TaskStatusInProgress, StartedAt: &now}
	if err := task.Cancel(); err == nil {
		t.Error("expected error when cancelling an in-progress task")
	}
}

func TestCollectionTask_FailedCanRetryToScheduled(t *testing.T) {
	task := &CollectionTask{Status: TaskStatusFailed}
	if !task.Status.CanTransitionTo(TaskStatusScheduled) {
		t.Error("a failed task should be retryable back to scheduled")
	}
}

func TestTaskStatus_IsTerminal(t *testing.T) {
	terminal := []TaskStatus{TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminal := []TaskStatus{TaskStatusScheduled, TaskStatusInProgress}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("%s should not be terminal", s)
		}
	}
}
