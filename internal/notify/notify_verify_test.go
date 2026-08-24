package notify

import (
	"testing"
	"time"

	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
)

func TestNotifySnapshotConsistent(t *testing.T) {
	store := shadow.NewStore()
	hub := NewHub(store)
	store.SetOnChange(hub.Notify)
	_, ch := hub.Subscribe("d1", 8)

	store.ApplyDesired(model.DesiredOp{
		DeviceID: "d1",
		Props:    map[string]string{"mode": "fast"},
		Version:  5,
		Source:   "verify",
	}, 100)
	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "27"},
		BaseVersion: 4,
		Version:     5,
	}, 200)

	final := store.Get("d1")
	var last *model.Document
	deadline := time.After(2 * time.Second)
drain:
	for {
		select {
		case snap := <-ch:
			last = snap
		case <-deadline:
			break drain
		}
	}
	if last == nil {
		t.Fatalf("subscriber received no notification")
	}
	if last.Desired["mode"] != final.Desired["mode"] ||
		last.Reported["temp"] != final.Reported["temp"] ||
		last.Version != final.Version {
		t.Fatalf(
			"notification carried a stale half snapshot: got desired=%v reported=%v version=%d, final desired=%v reported=%v version=%d",
			last.Desired, last.Reported, last.Version,
			final.Desired, final.Reported, final.Version,
		)
	}
}
