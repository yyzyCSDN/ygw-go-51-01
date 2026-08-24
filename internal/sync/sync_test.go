package sync

import (
	"testing"

	"deviceshadow/internal/model"
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

func TestReplayPlanDeduplicates(t *testing.T) {
	plan := NewReplayPlan(
		[]model.DesiredOp{{DeviceID: "d1", Version: 5}},
		[]model.DesiredOp{{DeviceID: "d1", Version: 5}},
	)
	if plan.Count() != 1 {
		t.Fatalf("replay plan must deduplicate by version, got %d", plan.Count())
	}
}
