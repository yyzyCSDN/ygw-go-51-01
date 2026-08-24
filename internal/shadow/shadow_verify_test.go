package shadow

import (
	"testing"

	"deviceshadow/internal/model"
)

func TestDesiredReportVersionConflict(t *testing.T) {
	store := NewStore()
	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "27"},
		BaseVersion: 0,
		Version:     1,
	}, 100)
	store.ApplyDesired(model.DesiredOp{
		DeviceID: "d1",
		Props:    map[string]string{"mode": "fast"},
		Version:  5,
		Source:   "verify",
	}, 200)

	doc := store.Get("d1")
	if doc == nil {
		t.Fatalf("shadow missing after desired write")
	}
	if doc.Desired["mode"] != "fast" {
		t.Fatalf("desired state lost during merge: %v", doc.Desired)
	}
	if doc.Reported["temp"] != "27" {
		t.Fatalf("desired write cleared the reported on-site state: %v", doc.Reported)
	}
}
