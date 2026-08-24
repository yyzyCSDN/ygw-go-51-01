package shadow

import (
	"testing"

	"deviceshadow/internal/model"
)

// TestReportedSurvivesDesiredPush is the integration-level regression test for
// the on-site state loss bug. It mirrors the reported scenario: a device
// reports on-site state, the platform then pushes desired state, and the
// reported section must survive so the shadow can still reconcile with the
// device. Desired and reported advance independently afterwards.
func TestReportedSurvivesDesiredPush(t *testing.T) {
	store := NewStore()

	// 1. device reports on-site state
	r := store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "24", "humidity": "55"},
		BaseVersion: 0,
		Version:     1,
	}, 100)
	if r.Reported["temp"] != "24" || r.Reported["humidity"] != "55" {
		t.Fatalf("report not stored: %v", r.Reported)
	}

	// 2. platform pushes desired state
	d := store.ApplyDesired(model.DesiredOp{
		DeviceID: "d1",
		Props:    map[string]string{"mode": "fast"},
		Version:  2,
		Source:   "console",
	}, 200)
	if d.Desired["mode"] != "fast" {
		t.Fatalf("desired not applied: %v", d.Desired)
	}
	if d.Reported["temp"] != "24" || d.Reported["humidity"] != "55" {
		t.Fatalf("reported on-site state lost after desired push: %v", d.Reported)
	}
	if d.ReportedVersion != 1 {
		t.Fatalf("reported version must not change on desired push: %d", d.ReportedVersion)
	}
	if d.DesiredVersion != 2 {
		t.Fatalf("desired version not advanced: %d", d.DesiredVersion)
	}
	if d.State != model.StateStale {
		t.Fatalf("shadow should be stale after desired push, got %s", d.State)
	}

	// 3. a later report advances reported independently; desired untouched
	d2 := store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "25"},
		BaseVersion: 2,
		Version:     3,
	}, 300)
	if d2.Reported["temp"] != "25" || d2.Reported["humidity"] != "55" {
		t.Fatalf("second report merge wrong: %v", d2.Reported)
	}
	if d2.Desired["mode"] != "fast" {
		t.Fatalf("desired lost after second report: %v", d2.Desired)
	}
}
