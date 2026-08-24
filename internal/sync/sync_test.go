package sync

import (
	"testing"

	"deviceshadow/internal/device"
	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
)

func TestAllocatorSequentialVersions(t *testing.T) {
	allocator := NewAllocator()
	first := allocator.Next("d1")
	second := allocator.Next("d1")
	if first != 1 || second != 2 {
		t.Fatalf("allocator must hand out 1,2 in order, got %d,%d", first, second)
	}
	if allocator.Next("d2") != 1 {
		t.Fatalf("allocator must be per-device")
	}
}

func TestResumeDeliversOnlyNewerDesired(t *testing.T) {
	store := shadow.NewStore()
	registry := device.NewRegistry(2)
	syncsvc := NewService(store, registry)
	registry.Register("d1", 1)
	store.ApplyDesired(model.DesiredOp{DeviceID: "d1", Props: map[string]string{"mode": "fast"}, Version: 5, Source: "test"}, 1)
	syncsvc.Deliver("d1", model.DesiredOp{DeviceID: "d1", Props: map[string]string{"mode": "fast"}, Version: 5, Source: "test"})
	ops := syncsvc.Resume("d1")
	if len(ops) != 0 {
		t.Fatalf("nothing should be replayed after delivery, got %d ops", len(ops))
	}
}

func TestReplayPlanDeduplicates(t *testing.T) {
	plan := NewReplayPlan(
		[]model.DesiredOp{{DeviceID: "d1", Version: 5}},
		[]model.DesiredOp{{DeviceID: "d1", Version: 5}},
	)
	if plan.Count() != 1 {
		t.Fatalf("replay plan must deduplicate by version, got %d", plan.Count())
	}
}
