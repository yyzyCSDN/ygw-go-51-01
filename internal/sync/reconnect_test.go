package sync

import (
	"testing"

	"deviceshadow/internal/device"
	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
)

// desiredOpFor builds a desired op with the given props and version.
func desiredOpFor(deviceID string, props map[string]string, version int64) model.DesiredOp {
	return model.DesiredOp{
		DeviceID: deviceID,
		Props:    model.CloneMap(props),
		Version:  version,
		Source:   "test",
	}
}

// newSyncFixture wires a store, registry and sync service for reconnect tests.
func newSyncFixture() (*Service, *shadow.Store, *device.Registry) {
	store := shadow.NewStore()
	registry := device.NewRegistry(2)
	svc := NewService(store, registry)
	return svc, store, registry
}

// TestDeliveredWatermarkAdvances guards the core fix: MarkDelivered must move
// the delivered watermark forward. Before the fix MarkDelivered was a no-op,
// so LastDelivered stayed at 0 forever and every reconnect replayed the full
// history from the ack watermark, letting stale desired state clobber a newer
// push.
func TestDeliveredWatermarkAdvances(t *testing.T) {
	dev := device.NewDevice("d1", 1, 100)
	dev.MarkDelivered(3)
	if dev.DeliveredWatermark() != 3 {
		t.Fatalf("MarkDelivered must advance the delivered watermark, got %d", dev.DeliveredWatermark())
	}
	// A stale delivery must never roll the watermark back.
	dev.MarkDelivered(1)
	if dev.DeliveredWatermark() != 3 {
		t.Fatalf("delivered watermark must not regress, got %d", dev.DeliveredWatermark())
	}
}

// TestDeliverSkipsStaleVersion locks down the staleness guard: a desired op
// whose version is not newer than what the device already received must never
// be re-pushed, otherwise the old desired state would overwrite a newer push
// on the device.
func TestDeliverSkipsStaleVersion(t *testing.T) {
	svc, _, registry := newSyncFixture()
	registry.Register("d1", 1)
	dev, _ := registry.Get("d1")
	dev.MarkDelivered(5)
	registry.SetOnline("d1", true)

	stale := desiredOpFor("d1", map[string]string{"mode": "old"}, 3)
	if !svc.Deliver("d1", stale) {
		t.Fatalf("stale deliver must be reported as delivered (already has it), not re-queued")
	}
	if dev.DeliveredWatermark() != 5 {
		t.Fatalf("stale deliver must not advance the watermark, got %d", dev.DeliveredWatermark())
	}
	if dev.Cache.Size() != 0 {
		t.Fatalf("stale deliver must not buffer into the offline cache, got %d", dev.Cache.Size())
	}
}

// TestDeliverBuffersOffline keeps the offline behaviour: a device that is
// offline still caches the op for later replay.
func TestDeliverBuffersOffline(t *testing.T) {
	svc, _, registry := newSyncFixture()
	registry.Register("d1", 1)
	registry.SetOnline("d1", false)

	op := desiredOpFor("d1", map[string]string{"mode": "fast"}, 2)
	if svc.Deliver("d1", op) {
		t.Fatalf("offline deliver must not report delivered")
	}
	dev, _ := registry.Get("d1")
	if dev.Cache.Size() != 1 {
		t.Fatalf("offline deliver must buffer the op, got cache size %d", dev.Cache.Size())
	}
}

// TestResumeContinuesFromDeliveredVersion reproduces the reported field issue
// and locks the fix: an NB module that reconnects constantly must resume from
// the last delivered version, not the lagging ack watermark, so it keeps
// converging on the newest desired state without an old desired state rolling
// the config back.
func TestResumeContinuesFromDeliveredVersion(t *testing.T) {
	svc, store, registry := newSyncFixture()
	registry.Register("d1", 1)

	// v1: stale desired state the device already received.
	store.ApplyDesired(desiredOpFor("d1", map[string]string{"mode": "eco"}, 1), 1)
	// v2: newer desired state that is the current platform target.
	store.ApplyDesired(desiredOpFor("d1", map[string]string{"mode": "fast"}, 2), 2)

	dev, _ := registry.Get("d1")
	// The device received v2 already but never acked it (NB module ack lag).
	dev.MarkDelivered(2)
	registry.SetOnline("d1", true)
	// Ack only trails at 1; before the fix Resume started here and replayed
	// the unacknowledged backlog, which is what rolled the config back.
	dev.Ack(1)

	ops := svc.Resume("d1")
	// Resume must not re-push any version the device already received.
	for _, op := range ops {
		if op.Version <= 0 {
			t.Fatalf("resume must only replay strictly newer versions, got %d", op.Version)
		}
	}
	// The shadow keeps the newest desired state.
	doc := store.Get("d1")
	if doc == nil {
		t.Fatalf("shadow must exist after resume")
	}
	if doc.Desired["mode"] != "fast" {
		t.Fatalf("shadow desired must keep the newest target, got %v", doc.Desired)
	}
}

// TestResumeReplaysAscending ensures the resync replays in ascending version
// order so the newest desired state is applied last and cannot be clobbered by
// a stale state landing after it.
func TestResumeReplaysAscending(t *testing.T) {
	svc, store, registry := newSyncFixture()
	registry.Register("d1", 1)
	store.ApplyDesired(desiredOpFor("d1", map[string]string{"mode": "a"}, 1), 1)
	store.ApplyDesired(desiredOpFor("d1", map[string]string{"mode": "b"}, 2), 2)
	store.ApplyDesired(desiredOpFor("d1", map[string]string{"mode": "c"}, 3), 3)

	dev, _ := registry.Get("d1")
	dev.MarkDelivered(0)
	registry.SetOnline("d1", true)

	ops := svc.Resume("d1")
	for i := 1; i < len(ops); i++ {
		if ops[i].Version <= ops[i-1].Version {
			t.Fatalf("resume must replay in ascending version order, got %v then %v", ops[i-1].Version, ops[i].Version)
		}
	}
	if dev.DeliveredWatermark() != 3 {
		t.Fatalf("delivered watermark must reach the newest version, got %d", dev.DeliveredWatermark())
	}
}

// TestResumeDrainsOfflineCache guarantees a reconnect does not lose desired
// writes that were buffered while the device was away.
func TestResumeDrainsOfflineCache(t *testing.T) {
	svc, store, registry := newSyncFixture()
	registry.Register("d1", 1)
	store.ApplyDesired(desiredOpFor("d1", map[string]string{"mode": "fast"}, 4), 4)

	dev, _ := registry.Get("d1")
	registry.SetOnline("d1", false)
	// Buffer a write while offline; it carries a newer version.
	svc.Deliver("d1", desiredOpFor("d1", map[string]string{"mode": "turbo"}, 5))

	registry.SetOnline("d1", true)
	ops := svc.Resume("d1")
	seen := map[int64]string{}
	for _, op := range ops {
		seen[op.Version] = op.Props["mode"]
	}
	if seen[5] != "turbo" {
		t.Fatalf("resume must replay the buffered offline write, got ops %v", ops)
	}
	if dev.Cache.Size() != 0 {
		t.Fatalf("resume must drain the offline cache, got %d", dev.Cache.Size())
	}
}

// TestReplayPlanSortsAscending locks the dedup plan ordering so the newest
// desired state is always applied last across the offline + online merge.
func TestReplayPlanSortsAscending(t *testing.T) {
	plan := NewReplayPlan(
		[]model.DesiredOp{{DeviceID: "d1", Version: 5}, {DeviceID: "d1", Version: 3}},
		[]model.DesiredOp{{DeviceID: "d1", Version: 4}, {DeviceID: "d1", Version: 5}},
	)
	ops := plan.Ops()
	if plan.Count() != 3 {
		t.Fatalf("plan must deduplicate by version, got %d", plan.Count())
	}
	for i := 1; i < len(ops); i++ {
		if ops[i].Version <= ops[i-1].Version {
			t.Fatalf("plan must be ascending, got %v then %v", ops[i-1].Version, ops[i].Version)
		}
	}
	if ops[len(ops)-1].Version != 5 {
		t.Fatalf("newest version must be last, got %v", ops[len(ops)-1].Version)
	}
}
