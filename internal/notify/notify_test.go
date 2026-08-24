package notify

import (
	"testing"
	"time"

	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
)

func TestSubscribeReceivesSnapshot(t *testing.T) {
	store := shadow.NewStore()
	hub := NewHub(store)
	store.SetOnChange(hub.Notify)
	subID, ch := hub.Subscribe("d1", 4)
	defer hub.Unsubscribe("d1", subID)

	store.ApplyReport(model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"temp": "25"},
		BaseVersion: 0,
		Version:     1,
	}, 100)

	select {
	case snap := <-ch:
		if snap.Reported["temp"] != "25" {
			t.Fatalf("subscriber got wrong snapshot: %v", snap.Reported)
		}
	case <-time.After(time.Second):
		t.Fatalf("subscriber did not receive snapshot")
	}
	if hub.SubscriberCount("d1") != 1 {
		t.Fatalf("subscriber count mismatch")
	}
}

func TestTopicHasherBuckets(t *testing.T) {
	hasher := NewHasher(4)
	first := hasher.Bucket(TopicKey("d1"))
	if first >= hasher.Shards() {
		t.Fatalf("bucket out of range")
	}
	if hasher.Bucket(TopicKey("d1")) != first {
		t.Fatalf("topic hash must be stable")
	}
}
