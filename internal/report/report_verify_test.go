package report

import (
	"testing"

	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

func TestOutOfOrderReportNoStaleOverwrite(t *testing.T) {
	store := shadow.NewStore()
	allocator := sync.NewAllocator()
	svc := NewService(store, allocator)

	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "30"},
		BaseVersion: 0,
		Version:     5,
	}, 100)
	doc, err := svc.HandleReport("d1", map[string]string{"temp": "20"}, 3, "verify", 200)
	if err != nil {
		t.Fatalf("report handling failed: %v", err)
	}
	if doc.Reported["temp"] != "30" {
		t.Fatalf("stale report overwrote newer on-site state: %v", doc.Reported)
	}
	if doc.ReportedVersion != 5 {
		t.Fatalf("reported version regressed: %d", doc.ReportedVersion)
	}
}
