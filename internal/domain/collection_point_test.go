package domain

import "testing"

func TestCollectionPoint_FillRatio(t *testing.T) {
	p := &CollectionPoint{CapacityKg: 1000, CurrentLoadKg: 750}
	got := p.FillRatio()
	if got != 0.75 {
		t.Errorf("FillRatio() = %f, want 0.75", got)
	}
}

func TestCollectionPoint_FillRatio_ZeroCapacity(t *testing.T) {
	p := &CollectionPoint{CapacityKg: 0, CurrentLoadKg: 100}
	if p.FillRatio() != 0 {
		t.Error("FillRatio() should return 0 when capacity is 0")
	}
}

func TestCollectionPoint_IsOverThreshold(t *testing.T) {
	p := &CollectionPoint{CapacityKg: 1000, CurrentLoadKg: 850}
	if !p.IsOverThreshold(0.8) {
		t.Error("85% fill should be over 80% threshold")
	}
	if p.IsOverThreshold(0.9) {
		t.Error("85% fill should not be over 90% threshold")
	}
}

func TestCollectionPoint_UpdateStatus_Full(t *testing.T) {
	p := &CollectionPoint{CapacityKg: 1000, CurrentLoadKg: 1000, Status: PointStatusActive}
	p.UpdateStatus()
	if p.Status != PointStatusFull {
		t.Errorf("status = %s, want full", p.Status)
	}
}

func TestCollectionPoint_UpdateStatus_Active(t *testing.T) {
	p := &CollectionPoint{CapacityKg: 1000, CurrentLoadKg: 500, Status: PointStatusFull}
	p.UpdateStatus()
	if p.Status != PointStatusActive {
		t.Errorf("status = %s, want active", p.Status)
	}
}

func TestCollectionPoint_UpdateStatus_SuspendedPreserved(t *testing.T) {
	p := &CollectionPoint{CapacityKg: 1000, CurrentLoadKg: 1000, Status: PointStatusSuspended}
	p.UpdateStatus()
	if p.Status != PointStatusSuspended {
		t.Error("suspended status should not be overridden by UpdateStatus()")
	}
}

func TestCollectionPoint_UpdateStatus_ClosedPreserved(t *testing.T) {
	p := &CollectionPoint{CapacityKg: 1000, CurrentLoadKg: 1000, Status: PointStatusClosed}
	p.UpdateStatus()
	if p.Status != PointStatusClosed {
		t.Error("closed status should not be overridden by UpdateStatus()")
	}
}

func TestPointStatus_IsValid(t *testing.T) {
	for _, s := range []PointStatus{PointStatusActive, PointStatusFull, PointStatusSuspended, PointStatusClosed} {
		if !s.IsValid() {
			t.Errorf("%s should be valid", s)
		}
	}
	if PointStatus("invalid").IsValid() {
		t.Error("invalid should not be a valid point status")
	}
}
