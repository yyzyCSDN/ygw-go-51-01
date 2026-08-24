package desired

import (
	"testing"

	"deviceshadow/internal/device"
	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

func TestApplyDesiredSingleDevice(t *testing.T) {
	store := shadow.NewStore()
	registry := device.NewRegistry(2)
	allocator := sync.NewAllocator()
	syncsvc := sync.NewService(store, registry)
	svc := NewService(store, allocator, syncsvc)
	registry.Register("d1", 1)

	doc, err := svc.ApplyDesired("d1", map[string]string{"mode": "fast"}, "test", 100)
	if err != nil {
		t.Fatalf("apply desired failed: %v", err)
	}
	if doc.Desired["mode"] != "fast" {
		t.Fatalf("desired value missing")
	}
	if doc.State != model.StateStale {
		t.Fatalf("desired write must leave shadow stale, got %s", doc.State)
	}
}

func TestBatchFullSuccess(t *testing.T) {
	store := shadow.NewStore()
	registry := device.NewRegistry(2)
	allocator := sync.NewAllocator()
	syncsvc := sync.NewService(store, registry)
	svc := NewBatchService(store, allocator, syncsvc)
	registry.Register("d1", 1)
	registry.Register("d2", 1)

	batch, err := svc.ApplyBatch([]model.BatchItem{
		{DeviceID: "d1", Props: map[string]string{"mode": "fast"}},
		{DeviceID: "d2", Props: map[string]string{"mode": "eco"}},
	}, 100)
	if err != nil {
		t.Fatalf("batch failed: %v", err)
	}
	if batch.State != model.BatchDone {
		t.Fatalf("successful batch must be done, got %s", batch.State)
	}
	if len(batch.Failed) != 0 {
		t.Fatalf("no failure expected: %v", batch.Failed)
	}
}
