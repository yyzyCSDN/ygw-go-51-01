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

// TestBatchPartialFailureMustNotReportFullSuccess reproduces the regression
// where a partially delivered batch marked every shadow synced and the batch
// done, hiding the failure from operators. The failed shadow must stay stale,
// the batch must stay partial with the failed device recorded, and it must be
// queued for retry.
func TestBatchPartialFailureMustNotReportFullSuccess(t *testing.T) {
	store := shadow.NewStore()
	registry := device.NewRegistry(2)
	allocator := sync.NewAllocator()
	syncsvc := sync.NewService(store, registry)
	svc := NewBatchService(store, allocator, syncsvc)
	registry.Register("d1", 1)
	registry.Register("d2", 1)
	registry.SetOnline("d2", false) // d2 is offline, delivery of d2 fails

	batch, err := svc.ApplyBatch([]model.BatchItem{
		{DeviceID: "d1", Props: map[string]string{"mode": "fast"}},
		{DeviceID: "d2", Props: map[string]string{"mode": "eco"}},
	}, 100)
	if err != nil {
		t.Fatalf("batch failed: %v", err)
	}
	if batch.State != model.BatchPartial {
		t.Fatalf("partial failure must stay partial, got %s", batch.State)
	}
	if len(batch.Failed) != 1 || batch.Failed[0] != "d2" {
		t.Fatalf("failed device must be recorded as d2, got %v", batch.Failed)
	}
	if len(batch.Applied) != 1 || batch.Applied[0] != "d1" {
		t.Fatalf("delivered device must be recorded as d1, got %v", batch.Applied)
	}
	if store.StateOf("d2") != model.StateStale {
		t.Fatalf("failed device shadow must be stale, got %s", store.StateOf("d2"))
	}
	if store.StateOf("d1") != model.StateStale {
		t.Fatalf("delivered device shadow must be stale (awaiting ack), got %s", store.StateOf("d1"))
	}
	if len(svc.PendingBatches()) != 1 {
		t.Fatalf("partial batch must be queued for retry, got %v", svc.PendingBatches())
	}
}

// TestBatchPartialRetryRecovers verifies that a retried partial batch recovers
// once the failed device comes back online, the failure record is cleared and
// the batch closes.
func TestBatchPartialRetryRecovers(t *testing.T) {
	store := shadow.NewStore()
	registry := device.NewRegistry(2)
	allocator := sync.NewAllocator()
	syncsvc := sync.NewService(store, registry)
	svc := NewBatchService(store, allocator, syncsvc)
	registry.Register("d1", 1)
	registry.Register("d2", 1)
	registry.SetOnline("d2", false)

	batch, _ := svc.ApplyBatch([]model.BatchItem{
		{DeviceID: "d1", Props: map[string]string{"mode": "fast"}},
		{DeviceID: "d2", Props: map[string]string{"mode": "eco"}},
	}, 100)

	registry.SetOnline("d2", true)
	retried, done := svc.RetryOutstanding(batch.ID, 200)
	if !done {
		t.Fatalf("retry must close the batch once all items succeed")
	}
	if retried.State != model.BatchDone {
		t.Fatalf("recovered batch must be done, got %s", retried.State)
	}
	if len(retried.Failed) != 0 {
		t.Fatalf("recovered batch must have no failed items, got %v", retried.Failed)
	}
	if len(svc.PendingBatches()) != 0 {
		t.Fatalf("recovered batch must leave the retry queue, got %v", svc.PendingBatches())
	}
}
