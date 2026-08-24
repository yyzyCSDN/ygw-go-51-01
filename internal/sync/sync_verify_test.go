package sync

import (
	"testing"

	"deviceshadow/internal/device"
	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
)

func TestReconnectSyncNoStaleVersion(t *testing.T) {
	store := shadow.NewStore()
	registry := device.NewRegistry(2)
	svc := NewService(store, registry)
	dev := registry.Register("d1", 1)

	for _, version := range []int64{4, 5, 6} {
		op := model.DesiredOp{DeviceID: "d1", Props: map[string]string{"mode": "v"}, Version: version, Source: "verify"}
		store.ApplyDesired(op, 100)
		svc.Deliver("d1", op)
	}
	dev.Ack(3)

	ops := svc.Resume("d1")
	for _, op := range ops {
		if op.Version < 6 {
			t.Fatalf("stale desired state replayed on reconnect: v%d", op.Version)
		}
	}
}
