package device

import (
	"testing"

	"deviceshadow/internal/model"
)

func TestRegisterAndConnectivity(t *testing.T) {
	registry := NewRegistry(4)
	dev := registry.Register("d1", 100)
	if dev == nil || dev.ID != "d1" {
		t.Fatalf("register failed")
	}
	if !dev.Online {
		t.Fatalf("new device should be online")
	}
	registry.SetOnline("d1", false)
	if dev.Online {
		t.Fatalf("device should be offline")
	}
	if registry.Count() != 1 {
		t.Fatalf("registry count mismatch")
	}
	if _, ok := registry.Get("d1"); !ok {
		t.Fatalf("device missing from registry")
	}
}

func TestAckWatermarkMovesForward(t *testing.T) {
	dev := NewDevice("d1", 1, 100)
	dev.Ack(4)
	dev.Ack(6)
	if dev.AckWatermark() != 6 {
		t.Fatalf("ack watermark must not regress")
	}
}

func TestOfflineCacheBuffersOps(t *testing.T) {
	cache := NewOfflineCache()
	op := desiredOp("d1", 7)
	cache.Add(op)
	if cache.Size() != 1 {
		t.Fatalf("cache size mismatch")
	}
	drained := cache.Drain()
	if len(drained) != 1 || drained[0].Version != 7 {
		t.Fatalf("drain returned wrong ops: %v", drained)
	}
	if cache.Size() != 0 {
		t.Fatalf("cache not cleared after drain")
	}
}

func desiredOp(deviceID string, version int64) model.DesiredOp {
	return model.DesiredOp{DeviceID: deviceID, Props: map[string]string{"mode": "fast"}, Version: version, Source: "test"}
}
