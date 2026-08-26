package shadow

import (
	"testing"

	"deviceshadow/internal/model"
)

func TestReadNoPartialShadowState(t *testing.T) {
	store := NewStore()
	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"p0": "0"},
		BaseVersion: 100,
		Version:     1,
	}, 100)

	observed := store.Get("d1")
	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"p1": "1"},
		BaseVersion: 100,
		Version:     2,
	}, 200)

	if _, ok := observed.Reported["p1"]; ok {
		t.Fatalf("delivered snapshot was polluted by a later report: %v", observed.Reported)
	}
	if observed.Reported["p0"] != "0" {
		t.Fatalf("delivered snapshot lost its original content: %v", observed.Reported)
	}
	doc := store.Get("d1")
	if doc == nil || len(doc.Reported) != 2 {
		t.Fatalf("store state incomplete: %v", doc)
	}
}
