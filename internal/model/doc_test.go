package model

import "testing"

func TestCloneIsIndependent(t *testing.T) {
	doc := NewDocument("d1", 100)
	doc.Desired["mode"] = "auto"
	doc.Reported["temp"] = "21"
	doc.DesiredVersion = 3
	doc.ReportedVersion = 2
	doc.RefreshWatermark()

	copyDoc := doc.Clone()
	copyDoc.Desired["mode"] = "manual"
	copyDoc.Reported["temp"] = "25"
	copyDoc.DesiredVersion = 9

	if doc.Desired["mode"] != "auto" {
		t.Fatalf("clone shares desired map")
	}
	if doc.Reported["temp"] != "21" {
		t.Fatalf("clone shares reported map")
	}
	if doc.Version != 3 {
		t.Fatalf("watermark should be 3, got %d", doc.Version)
	}
}

func TestMergeMapsOverlaysValues(t *testing.T) {
	merged := MergeMaps(map[string]string{"a": "1", "b": "2"}, map[string]string{"b": "9", "c": "3"})
	if merged["a"] != "1" || merged["b"] != "9" || merged["c"] != "3" {
		t.Fatalf("unexpected merge result: %v", merged)
	}
}

func TestBatchLifecycle(t *testing.T) {
	batch := NewBatch("b1", []BatchItem{
		{DeviceID: "d1", Props: map[string]string{"a": "1"}},
		{DeviceID: "d2", Props: map[string]string{"a": "2"}},
	})
	if batch.State != BatchPending {
		t.Fatalf("new batch must be pending")
	}
	batch.RecordApplied("d1")
	batch.MarkPartial("d2")
	if batch.State != BatchPartial {
		t.Fatalf("partial failure must set partial state")
	}
	outstanding := batch.Outstanding()
	if len(outstanding) != 1 || outstanding[0].DeviceID != "d2" {
		t.Fatalf("unexpected outstanding items: %v", outstanding)
	}
	batch.MarkDone()
	if batch.State != BatchDone || batch.RetryRemaining != 0 {
		t.Fatalf("batch did not close")
	}
}
