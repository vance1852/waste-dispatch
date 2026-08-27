package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// TestRejectedPointLoadDoesNotChangeReportedLoad checks that a rejected load
// reading never becomes visible: when a collection point load update is refused
// because another operator already changed the point, later reads of that point
// must still report the load that is actually stored in the database.
func TestRejectedPointLoadDoesNotChangeReportedLoad(t *testing.T) {
	db := openServiceTestDB(t)
	pointRepo := reposqlite.NewPointRepository(db)
	svc := service.NewPointService(pointRepo, zerolog.Nop())
	ctx := context.Background()

	point := &domain.CollectionPoint{
		ID:            uuid.New().String(),
		Name:          "临江路生活垃圾投放点",
		Address:       "临江路 18 号",
		District:      "临江",
		CapacityKg:    1000,
		CurrentLoadKg: 400,
		Status:        domain.PointStatusActive,
	}
	if err := pointRepo.Create(ctx, point); err != nil {
		t.Fatalf("Create point error: %v", err)
	}

	// Warm up the read path the same way a dashboard request would.
	if _, err := svc.GetPoint(ctx, point.ID); err != nil {
		t.Fatalf("initial GetPoint error: %v", err)
	}

	// Another operator commits a change out of band, so the next update from this
	// service instance is stale and must be refused.
	if _, err := db.ExecContext(ctx,
		`UPDATE collection_points SET current_load_kg = ?, version = version + 1 WHERE id = ?`,
		450.0, point.ID,
	); err != nil {
		t.Fatalf("out-of-band update error: %v", err)
	}

	_, updateErr := svc.UpdateLoad(ctx, service.UpdateLoadRequest{
		PointID:       point.ID,
		CurrentLoadKg: 960,
	})
	if updateErr == nil {
		t.Fatal("UpdateLoad should have been rejected because the point was changed concurrently")
	}

	reread, err := svc.GetPoint(ctx, point.ID)
	if err != nil {
		t.Fatalf("GetPoint after rejected update error: %v", err)
	}
	if reread.CurrentLoadKg == 960 {
		t.Errorf(
			"rejected load reading 960 kg is being reported back (current_load_kg=%.1f); "+
				"a refused update must not become visible to later reads",
			reread.CurrentLoadKg,
		)
	}
	if reread.CurrentLoadKg != 450 {
		t.Errorf(
			"reported current_load_kg = %.1f, want 450 (the value actually stored in the database)",
			reread.CurrentLoadKg,
		)
	}
	if reread.Status == domain.PointStatusFull {
		t.Errorf("point status = %s, want it to stay active because the 960 kg reading was refused", reread.Status)
	}
}
