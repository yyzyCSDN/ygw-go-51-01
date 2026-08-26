package desired

import (
	"testing"

	"deviceshadow/internal/device"
	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

func TestBatchRollbackComplete(t *testing.T) {
	store := shadow.NewStore()
	registry := device.NewRegistry(2)
	allocator := sync.NewAllocator()
	syncsvc := sync.NewService(store, registry)
	svc := NewBatchService(store, allocator, syncsvc)
	registry.Register("d1", 1)
	store.ApplyDesired(model.DesiredOp{
		DeviceID: "d1",
		Props:    map[string]string{"a": "1", "b": "1", "c": "1"},
		Version:  1,
		Source:   "seed",
	}, 100)

	registry.SetOnline("d1", false)
	_, err := svc.ApplyTransactional("d1", map[string]string{"a": "2", "b": "2", "c": "3"}, 200)
	if err != nil {
		t.Fatalf("transactional batch failed: %v", err)
	}
	doc := store.Get("d1")
	if doc == nil {
		t.Fatalf("shadow missing after rollback")
	}
	if doc.Desired["a"] != "1" || doc.Desired["b"] != "1" || doc.Desired["c"] != "1" {
		t.Fatalf("rollback left post-operation values behind: %v", doc.Desired)
	}
}
