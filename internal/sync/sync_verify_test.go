package sync_test

import (
	"testing"

	"deviceshadow/internal/report"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

func TestReportVersionAdvances(t *testing.T) {
	store := shadow.NewStore()
	allocator := sync.NewAllocator()
	svc := report.NewService(store, allocator)

	_, _ = svc.HandleReport("d1", map[string]string{"temp": "20"}, 100, "verify", 100)
	_, _ = svc.HandleReport("d1", map[string]string{"humidity": "55"}, 100, "verify", 200)

	doc := store.Get("d1")
	if doc == nil {
		t.Fatalf("shadow missing after reports")
	}
	if doc.ReportedVersion != 2 {
		t.Fatalf("reported version did not advance: %d", doc.ReportedVersion)
	}
	if len(doc.Reported) != 2 {
		t.Fatalf("reports were lost: %v", doc.Reported)
	}
}
