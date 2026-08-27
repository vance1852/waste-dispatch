package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/repository"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// TestOverflowRegistrationSkipsPointsAlreadyBeingHandled checks that a collection
// point which already has an overflow report under way does not collect a second
// report. Once a supervisor picks the report up, the point is still overflowing,
// so the periodic sweep must recognise the in-progress report and stay silent.
func TestOverflowRegistrationSkipsPointsAlreadyBeingHandled(t *testing.T) {
	db := openServiceTestDB(t)
	pointRepo := reposqlite.NewPointRepository(db)
	incidentRepo := reposqlite.NewIncidentRepository(db)
	svc := service.NewIncidentService(incidentRepo, zerolog.Nop())
	ctx := context.Background()

	point := &domain.CollectionPoint{
		ID:            uuid.New().String(),
		Name:          "同乐街投放点",
		Address:       "同乐街 21 号",
		District:      "同乐",
		CapacityKg:    900,
		CurrentLoadKg: 880,
		Status:        domain.PointStatusActive,
	}
	if err := pointRepo.Create(ctx, point); err != nil {
		t.Fatalf("Create point error: %v", err)
	}

	first, created, err := svc.RegisterOverflowOnce(ctx, point.ID, "巡检发现投放点接近溢满")
	if err != nil {
		t.Fatalf("first RegisterOverflowOnce error: %v", err)
	}
	if !created || first == nil {
		t.Fatal("the first sweep over an overflowing point must register one report")
	}

	// A supervisor picks the report up; the point is still overflowing.
	if _, err := svc.AssignIncident(ctx, service.AssignIncidentRequest{
		IncidentID: first.ID,
		AssignedTo: "supervisor-day",
	}); err != nil {
		t.Fatalf("AssignIncident error: %v", err)
	}

	second, createdAgain, err := svc.RegisterOverflowOnce(ctx, point.ID, "巡检发现投放点接近溢满")
	if err != nil {
		t.Fatalf("second RegisterOverflowOnce error: %v", err)
	}
	if createdAgain {
		t.Errorf(
			"a second overflow report was registered for the same point while report %s is still being handled; "+
				"the sweep must recognise reports that are already picked up",
			first.ID,
		)
	}
	if second != nil {
		t.Errorf("no new report should be produced, got %s", second.ID)
	}

	reports, total, err := incidentRepo.List(ctx, repository.IncidentFilter{
		PointID: point.ID,
		Type:    domain.IncidentTypeOverflow,
		Limit:   20,
	})
	if err != nil {
		t.Fatalf("List incidents error: %v", err)
	}
	if total != 1 {
		t.Errorf("collection point accumulated %d overflow reports, want exactly 1; reports=%d", total, len(reports))
	}
}
