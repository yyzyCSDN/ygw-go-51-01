package desired

import (
	"testing"

	"deviceshadow/internal/device"
	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

func TestBatchDesiredPartialFailure(t *testing.T) {
	store := shadow.NewStore()
	registry := device.NewRegistry(2)
	allocator := sync.NewAllocator()
	syncsvc := sync.NewService(store, registry)
	svc := NewBatchService(store, allocator, syncsvc)
	registry.Register("d1", 1)
	registry.Register("d2", 1)
	registry.SetOnline("d2", false)

	batch, err := svc.ApplyBatch([]model.BatchItem{
		{DeviceID: "d1", Props: map[string]string{"mode": "fast"}},
		{DeviceID: "d2", Props: map[string]string{"mode": "eco"}},
	}, 100)
	if err != nil {
		t.Fatalf("batch failed: %v", err)
	}
	if batch.State != model.BatchPartial {
		t.Fatalf("partially failed batch must stay partial, got %s", batch.State)
	}
	if len(batch.Failed) != 1 || batch.Failed[0] != "d2" {
		t.Fatalf("failed items not recorded: %v", batch.Failed)
	}
	if len(batch.Outstanding()) != 1 {
		t.Fatalf("retry set is missing the failed item")
	}
	if doc := store.Get("d2"); doc == nil || doc.State != model.StateStale {
		t.Fatalf("failed device shadow must stay stale, got %v", doc)
	}
	if len(svc.PendingBatches()) != 1 {
		t.Fatalf("batch was not scheduled for retry")
	}
}
