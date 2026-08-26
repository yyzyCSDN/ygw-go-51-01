package sync_test

import (
	"testing"

	"deviceshadow/internal/device"
	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

func TestOfflineCacheOnlineSyncNoDup(t *testing.T) {
	store := shadow.NewStore()
	registry := device.NewRegistry(2)
	svc := sync.NewService(store, registry)
	registry.Register("d1", 1)

	store.ApplyDesired(model.DesiredOp{
		DeviceID: "d1",
		Props:    map[string]string{"mode": "fast"},
		Version:  5,
		Source:   "verify",
	}, 100)
	registry.SetOnline("d1", false)
	svc.Deliver("d1", model.DesiredOp{DeviceID: "d1", Props: map[string]string{"mode": "fast"}, Version: 5, Source: "verify"})
	registry.SetOnline("d1", true)

	delivered := svc.Reconcile("d1")
	count := 0
	for _, op := range delivered {
		if op.Version == 5 {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("same desired state delivered %d times after reconnect", count)
	}
}
