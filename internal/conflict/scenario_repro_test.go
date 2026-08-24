package conflict

import (
	"testing"

	"deviceshadow/internal/model"
)

// TestMergeDesiredPreservesReported is the regression test for the on-site
// state loss bug: a desired push must replace only the desired section. The
// reported section the device already pushed has to survive untouched, with
// its version unchanged, so the shadow can still reconcile with the device.
func TestMergeDesiredPreservesReported(t *testing.T) {
	current := model.NewDocument("d1", 1)
	current.Reported["temp"] = "24"
	current.Reported["humidity"] = "55"
	current.ReportedVersion = 1
	current.RefreshWatermark()

	merged := MergeDesired(current, model.DesiredOp{
		DeviceID: "d1",
		Props:    map[string]string{"mode": "fast"},
		Version:  2,
		Source:   "console",
	}, 2)

	if merged.Desired["mode"] != "fast" {
		t.Fatalf("desired not applied: %v", merged.Desired)
	}
	if merged.Reported["temp"] != "24" || merged.Reported["humidity"] != "55" {
		t.Fatalf("reported on-site state lost after desired push: %v", merged.Reported)
	}
	if merged.ReportedVersion != 1 {
		t.Fatalf("reported version must not advance on desired push: %d", merged.ReportedVersion)
	}
	if merged.DesiredVersion != 2 {
		t.Fatalf("desired version not advanced: %d", merged.DesiredVersion)
	}
	if merged.State != model.StateStale {
		t.Fatalf("shadow should be stale after desired push, got %s", merged.State)
	}
}
