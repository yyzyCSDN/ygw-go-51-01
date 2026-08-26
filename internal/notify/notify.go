// Package notify implements the change notification hub. Subscribers receive
// consistent shadow snapshots; the hub coalesces bursts of changes per device
// so a subscriber never sees an intermediate half-updated state.
package notify

import (
	"sync"

	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
)

// Hub fans shadow change events out to subscribers.
type Hub struct {
	mu      sync.Mutex
	store   *shadow.Store
	hasher  *Hasher
	subs    map[string]map[string]chan *model.Document
	queued  map[string]*model.Document
	buckets map[string]uint64
}

// NewHub wires the hub to the shadow store.
func NewHub(store *shadow.Store) *Hub {
	return &Hub{
		store:   store,
		hasher:  NewHasher(4),
		subs:    make(map[string]map[string]chan *model.Document),
		queued:  make(map[string]*model.Document),
		buckets: make(map[string]uint64),
	}
}

// Subscribe registers a subscriber for one device and returns its id and
// snapshot channel. The channel receives deep copies only.
func (h *Hub) Subscribe(deviceID string, capacity int) (string, <-chan *model.Document) {
	h.mu.Lock()
	defer h.mu.Unlock()
	byID := h.subs[deviceID]
	if byID == nil {
		byID = make(map[string]chan *model.Document)
		h.subs[deviceID] = byID
	}
	ch := make(chan *model.Document, capacity)
	id := "sub-" + itoa(len(byID)+1)
	byID[id] = ch
	h.buckets[deviceID] = h.hasher.Bucket(TopicKey(deviceID))
	return id, ch
}

// Unsubscribe removes a subscriber.
func (h *Hub) Unsubscribe(deviceID, subID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	byID := h.subs[deviceID]
	if byID == nil {
		return
	}
	if ch, ok := byID[subID]; ok {
		delete(byID, subID)
		close(ch)
	}
	if len(byID) == 0 {
		delete(h.subs, deviceID)
	}
}

// Notify is the store commit hook. It replaces any older pending snapshot for
// the device and schedules one delivery so a burst of changes is coalesced.
func (h *Hub) Notify(event model.ChangeEvent) {
	if event.Type == model.EventDelete {
		h.mu.Lock()
		delete(h.queued, event.DeviceID)
		h.mu.Unlock()
		return
	}
	if event.Snapshot == nil {
		return
	}
	h.mu.Lock()
	h.queued[event.DeviceID] = event.Snapshot.Clone()
	h.mu.Unlock()
	go h.flush(event.DeviceID)
}

func (h *Hub) flush(deviceID string) {
	// Deliver while the device stays dirty. Re-reading the current snapshot
	// on every pass guarantees that a burst of changes coalesces into a final
	// consistent state instead of an intermediate half-update.
	for {
		h.mu.Lock()
		_, dirty := h.queued[deviceID]
		delete(h.queued, deviceID)
		byID := h.subs[deviceID]
		channels := make([]chan *model.Document, 0, len(byID))
		for _, ch := range byID {
			channels = append(channels, ch)
		}
		h.mu.Unlock()
		if !dirty {
			return
		}
		snap := h.store.Snapshot(deviceID)
		if snap == nil {
			continue
		}
		for _, ch := range channels {
			select {
			case ch <- snap.Clone():
			default:
			}
		}
	}
}

// Snapshot returns the current consistent snapshot for a device.
func (h *Hub) Snapshot(deviceID string) *model.Document {
	return h.store.Snapshot(deviceID)
}

// SubscriberCount returns how many subscribers listen to one device.
func (h *Hub) SubscriberCount(deviceID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs[deviceID])
}

// DevicesWithSubscribers returns the device ids that have at least one
// subscriber, used by the console overview.
func (h *Hub) DevicesWithSubscribers() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ids := make([]string, 0, len(h.subs))
	for id := range h.subs {
		ids = append(ids, id)
	}
	return ids
}

// ShardDistribution returns how many subscribed devices hash into each topic
// bucket, used by the console overview.
func (h *Hub) ShardDistribution() map[uint64]int {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make(map[uint64]int)
	for _, bucket := range h.buckets {
		out[bucket]++
	}
	return out
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	index := len(buf)
	for value > 0 {
		index--
		buf[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[index:])
}
