package worker

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// IncidentWorker monitors collection points for overflow and auto-creates incidents.
type IncidentWorker struct {
	pointSvc    *service.PointService
	incidentSvc *service.IncidentService
	interval    time.Duration
	threshold   float64
	log         zerolog.Logger
}

// NewIncidentWorker creates a new IncidentWorker.
func NewIncidentWorker(
	pointSvc *service.PointService,
	incidentSvc *service.IncidentService,
	interval time.Duration,
	threshold float64,
	log zerolog.Logger,
) *IncidentWorker {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	if threshold <= 0 || threshold > 1.0 {
		threshold = 0.9
	}
	return &IncidentWorker{
		pointSvc:    pointSvc,
		incidentSvc: incidentSvc,
		interval:    interval,
		threshold:   threshold,
		log:         log,
	}
}

// Run starts the incident monitoring loop and blocks until ctx is cancelled.
func (w *IncidentWorker) Run(ctx context.Context) {
	w.log.Info().
		Dur("interval", w.interval).
		Float64("threshold", w.threshold).
		Msg("incident worker started")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info().Msg("incident worker stopped")
			return
		case <-ticker.C:
			w.checkOverflows(ctx)
		}
	}
}

func (w *IncidentWorker) checkOverflows(ctx context.Context) {
	points, err := w.pointSvc.ListOverCapacity(ctx, w.threshold)
	if err != nil {
		w.log.Error().Err(err).Msg("incident worker: failed to list over-capacity points")
		return
	}

	for _, p := range points {
		// Check if there's already an open overflow incident for this point.
		existing, _, err := w.incidentSvc.ListIncidents(ctx, repository.IncidentFilter{
			Status:  domain.IncidentStatusOpen,
			PointID: p.ID,
			Limit:   1,
		})
		if err != nil {
			w.log.Error().Err(err).Str("point_id", p.ID).Msg("incident worker: failed to check existing incidents")
			continue
		}
		// Skip if there's already an open overflow incident.
		for _, inc := range existing {
			if inc.Type == domain.IncidentTypeOverflow {
				w.log.Debug().Str("point_id", p.ID).Msg("incident worker: overflow incident already exists")
				goto nextPoint
			}
		}

		{
			_, err = w.incidentSvc.ReportIncident(ctx, service.ReportIncidentRequest{
				Type:        domain.IncidentTypeOverflow,
				Severity:    domain.IncidentSeverityHigh,
				PointID:     p.ID,
				ReportedBy:  "system",
				Description: "Auto-detected: collection point has exceeded capacity threshold",
				OccurredAt:  time.Now().UTC(),
			})
			if err != nil {
				w.log.Error().Err(err).Str("point_id", p.ID).Msg("incident worker: failed to create overflow incident")
				continue
			}
			w.log.Warn().
				Str("point_id", p.ID).
				Float64("fill_ratio", p.FillRatio()).
				Msg("incident worker: overflow incident created")
		}

	nextPoint:
	}
}
