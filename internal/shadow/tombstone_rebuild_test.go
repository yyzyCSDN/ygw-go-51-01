package shadow

import (
	"testing"

	"deviceshadow/internal/model"
)

// TestLateReportDoesNotRebuildDeletedShadow guards the deletion flow: a report
// that was buffered in the gateway and arrives after Delete must not recreate
// the shadow. The tombstone stays for its retention window and intercepts the
// late write, so the device does not reappear in listings or notifications.
func TestLateReportDoesNotRebuildDeletedShadow(t *testing.T) {
	store := NewStore()
	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "24"},
		BaseVersion: 0,
		Version:     1,
	}, 100)
	if err := store.Delete("d1", 200); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Late report from the gateway buffer arrives after the delete.
	doc := store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "99"},
		BaseVersion: 1,
		Version:     2,
	}, 300)
	if doc != nil {
		t.Fatalf("late report rebuilt the deleted shadow")
	}
	if store.Get("d1") != nil {
		t.Fatalf("deleted shadow was resurrected by a late report")
	}
	if len(store.List()) != 0 {
		t.Fatalf("deleted device reappeared in the device list")
	}
	if !store.IsTombstoned("d1") {
		t.Fatalf("tombstone must remain within the retention window")
	}
}

// TestLateDesiredDoesNotRebuildDeletedShadow makes the same guarantee for the
// desired write path, which also routes through ensure.
func TestLateDesiredDoesNotRebuildDeletedShadow(t *testing.T) {
	store := NewStore()
	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "24"},
		BaseVersion: 0,
		Version:     1,
	}, 100)
	store.Delete("d1", 200)

	doc := store.ApplyDesired(model.DesiredOp{
		DeviceID: "d1",
		Props:    map[string]string{"mode": "fast"},
		Version:  2,
		Source:   "console",
	}, 300)
	if doc != nil {
		t.Fatalf("late desired write rebuilt the deleted shadow")
	}
	if store.Get("d1") != nil {
		t.Fatalf("deleted shadow was resurrected by a late desired write")
	}
}

// TestFreshReportRebuildsShadowAfterRetentionExpiry confirms that once the
// tombstone is collected past the retention cutoff, a fresh update may create a
// new shadow again: the block only lasts for the retention window.
func TestFreshReportRebuildsShadowAfterRetentionExpiry(t *testing.T) {
	store := NewStore()
	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "24"},
		BaseVersion: 0,
		Version:     1,
	}, 100)
	store.Delete("d1", 200)

	store.GCTombstones(250) // tombstone recorded at 200 < cutoff, so it is dropped

	doc := store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "5"},
		BaseVersion: 0,
		Version:     2,
	}, 300)
	if doc == nil {
		t.Fatalf("fresh report after retention expiry should rebuild the shadow")
	}
	if store.IsTombstoned("d1") {
		t.Fatalf("tombstone should have been collected past the retention cutoff")
	}
}
