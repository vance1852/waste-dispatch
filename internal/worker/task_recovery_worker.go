package worker

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// TaskRecoveryWorker periodically checks for stale in-progress tasks and marks them failed.
type TaskRecoveryWorker struct {
	taskSvc  *service.TaskService
	interval time.Duration
	staleAge time.Duration
	log      zerolog.Logger
}

// NewTaskRecoveryWorker creates a new TaskRecoveryWorker.
func NewTaskRecoveryWorker(
	taskSvc *service.TaskService,
	interval time.Duration,
	staleAge time.Duration,
	log zerolog.Logger,
) *TaskRecoveryWorker {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	if staleAge <= 0 {
		staleAge = 4 * time.Hour
	}
	return &TaskRecoveryWorker{
		taskSvc:  taskSvc,
		interval: interval,
		staleAge: staleAge,
		log:      log,
	}
}

// Run starts the recovery worker loop and blocks until ctx is cancelled.
func (w *TaskRecoveryWorker) Run(ctx context.Context) {
	w.log.Info().
		Dur("interval", w.interval).
		Dur("stale_age", w.staleAge).
		Msg("task recovery worker started")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run once immediately on start.
	w.recover(ctx)

	for {
		select {
		case <-ctx.Done():
			w.log.Info().Msg("task recovery worker stopped")
			return
		case <-ticker.C:
			w.recover(ctx)
		}
	}
}

func (w *TaskRecoveryWorker) recover(ctx context.Context) {
	cutoff := time.Now().UTC().Add(-w.staleAge)
	count, err := w.taskSvc.RecoverStale(ctx, cutoff)
	if err != nil {
		w.log.Error().Err(err).Msg("task recovery failed")
		return
	}
	if count > 0 {
		w.log.Warn().Int("recovered", count).Msg("stale tasks recovered")
	}
}
