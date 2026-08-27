package domain

import "testing"

func TestVehicleStatusTransitions(t *testing.T) {
	tests := []struct {
		from    VehicleStatus
		to      VehicleStatus
		allowed bool
	}{
		{VehicleStatusIdle, VehicleStatusDispatched, true},
		{VehicleStatusIdle, VehicleStatusMaintenance, true},
		{VehicleStatusIdle, VehicleStatusRetired, true},
		{VehicleStatusIdle, VehicleStatusInService, false},
		{VehicleStatusDispatched, VehicleStatusInService, true},
		{VehicleStatusDispatched, VehicleStatusIdle, true},
		{VehicleStatusDispatched, VehicleStatusMaintenance, false},
		{VehicleStatusInService, VehicleStatusIdle, true},
		{VehicleStatusInService, VehicleStatusMaintenance, true},
		{VehicleStatusMaintenance, VehicleStatusIdle, true},
		{VehicleStatusRetired, VehicleStatusIdle, false},
	}

	for _, tc := range tests {
		got := tc.from.CanTransitionTo(tc.to)
		if got != tc.allowed {
			t.Errorf("VehicleStatus(%s).CanTransitionTo(%s) = %v, want %v", tc.from, tc.to, got, tc.allowed)
		}
	}
}

func TestVehicle_Assign(t *testing.T) {
	v := &Vehicle{ID: "v1", Status: VehicleStatusIdle, Version: 1}
	if err := v.Assign("driver-1"); err != nil {
		t.Fatalf("Assign() unexpected error: %v", err)
	}
	if v.Status != VehicleStatusDispatched {
		t.Errorf("status = %s, want dispatched", v.Status)
	}
	if v.DriverID != "driver-1" {
		t.Errorf("DriverID = %q, want driver-1", v.DriverID)
	}
}

func TestVehicle_AssignFromMaintenance_Rejected(t *testing.T) {
	v := &Vehicle{Status: VehicleStatusMaintenance}
	if err := v.Assign("d1"); err == nil {
		t.Error("expected error assigning vehicle in maintenance")
	}
}

func TestVehicle_Release(t *testing.T) {
	v := &Vehicle{ID: "v1", Status: VehicleStatusDispatched, DriverID: "driver-1", Version: 1}
	if err := v.Release(); err != nil {
		t.Fatalf("Release() unexpected error: %v", err)
	}
	if v.Status != VehicleStatusIdle {
		t.Errorf("status = %s, want idle", v.Status)
	}
	if v.DriverID != "" {
		t.Errorf("DriverID should be empty after release, got %q", v.DriverID)
	}
}

func TestVehicle_ReleaseFromRetired_Rejected(t *testing.T) {
	v := &Vehicle{Status: VehicleStatusRetired}
	if err := v.Release(); err == nil {
		t.Error("expected error releasing a retired vehicle")
	}
}

func TestVehicleStatus_IsValid(t *testing.T) {
	valid := []VehicleStatus{VehicleStatusIdle, VehicleStatusDispatched, VehicleStatusInService, VehicleStatusMaintenance, VehicleStatusRetired}
	for _, s := range valid {
		if !s.IsValid() {
			t.Errorf("%s should be valid", s)
		}
	}
	if VehicleStatus("unknown").IsValid() {
		t.Error("unknown should not be a valid vehicle status")
	}
}
