package shadow

import (
	"testing"

	"deviceshadow/internal/model"
)

func TestApplyReportHappyPath(t *testing.T) {
	store := NewStore()
	doc := store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "24"},
		BaseVersion: 0,
		Version:     1,
	}, 100)
	if doc == nil {
		t.Fatalf("report was not applied")
	}
	if doc.State != model.StateSynced {
		t.Fatalf("reported shadow should be synced, got %s", doc.State)
	}
	if store.Get("d1").Reported["temp"] != "24" {
		t.Fatalf("reported value missing")
	}
}

func TestDeleteRemovesShadow(t *testing.T) {
	store := NewStore()
	store.ApplyReport(model.ReportOp{DeviceID: "d1", Props: map[string]string{"temp": "24"}, BaseVersion: 0, Version: 1}, 100)
	if err := store.Delete("d1", 200); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if store.Get("d1") != nil {
		t.Fatalf("shadow still exists after delete")
	}
	if !store.IsTombstoned("d1") {
		t.Fatalf("tombstone not recorded")
	}
}

func TestListSortedAndSnapshotIndependent(t *testing.T) {
	store := NewStore()
	store.ApplyReport(model.ReportOp{DeviceID: "b", Props: map[string]string{"p": "1"}, BaseVersion: 0, Version: 1}, 1)
	store.ApplyReport(model.ReportOp{DeviceID: "a", Props: map[string]string{"p": "2"}, BaseVersion: 0, Version: 1}, 1)
	list := store.List()
	if len(list) != 2 || list[0].DeviceID != "a" || list[1].DeviceID != "b" {
		t.Fatalf("list is not sorted: %v", list)
	}
	snap := store.Snapshot("a")
	snap.Reported["p"] = "mutated"
	if store.Get("a").Reported["p"] != "2" {
		t.Fatalf("snapshot is not an independent copy")
	}
}

func TestHistoryRecordsEvents(t *testing.T) {
	store := NewStore()
	store.ApplyReport(model.ReportOp{DeviceID: "d1", Props: map[string]string{"temp": "24"}, BaseVersion: 0, Version: 1}, 100)
	store.ApplyDesired(model.DesiredOp{DeviceID: "d1", Props: map[string]string{"mode": "fast"}, Version: 2}, 200)
	history := store.History("d1")
	if len(history) != 2 {
		t.Fatalf("expected two history events, got %d", len(history))
	}
	if history[1].Type != model.EventDesired {
		t.Fatalf("second event should be desired")
	}
}
