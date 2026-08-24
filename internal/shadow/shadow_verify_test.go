package shadow

import (
	"testing"

	"deviceshadow/internal/model"
)

func TestDeletedShadowNoResurrect(t *testing.T) {
	store := NewStore()
	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "20"},
		BaseVersion: 0,
		Version:     1,
	}, 100)
	store.Delete("d1", 200)
	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "21"},
		BaseVersion: 0,
		Version:     2,
	}, 300)

	if store.Get("d1") != nil {
		t.Fatalf("deleted shadow was resurrected by a late report")
	}
	if !store.IsTombstoned("d1") {
		t.Fatalf("tombstone was lost")
	}
}
