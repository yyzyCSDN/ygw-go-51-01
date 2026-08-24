// Package device manages the device registry: registration, connectivity and
// the per-device delivery watermark used by the reconnect sync.
package device

import (
	"sort"
	"sync"

	"github.com/cespare/xxhash/v2"
)

// Registry keeps every known device and its connectivity state.
type Registry struct {
	mu      sync.RWMutex
	devices map[string]*Device
	shards  int
}

// NewRegistry creates a registry that groups devices into hash shards for the
// console overview.
func NewRegistry(shards int) *Registry {
	if shards < 1 {
		shards = 8
	}
	return &Registry{
		devices: make(map[string]*Device),
		shards:  shards,
	}
}

// Register adds a device and returns its handle.
func (r *Registry) Register(deviceID string, now int64) *Device {
	r.mu.Lock()
	defer r.mu.Unlock()
	if dev, ok := r.devices[deviceID]; ok {
		return dev
	}
	dev := NewDevice(deviceID, uint64(xxhash.Sum64String(deviceID)%uint64(r.shards)), now)
	r.devices[deviceID] = dev
	return dev
}

// Get returns the device handle and whether it is registered.
func (r *Registry) Get(deviceID string) (*Device, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dev, ok := r.devices[deviceID]
	return dev, ok
}

// Unregister removes a device from the registry.
func (r *Registry) Unregister(deviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.devices, deviceID)
}

// SetOnline flips the connectivity flag of a device.
func (r *Registry) SetOnline(deviceID string, online bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if dev, ok := r.devices[deviceID]; ok {
		dev.Online = online
	}
}

// List returns every registered device sorted by id.
func (r *Registry) List() []*Device {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.devices))
	for id := range r.devices {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]*Device, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.devices[id])
	}
	return out
}

// Count returns the number of registered devices.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.devices)
}

// BySlot returns the devices that hash to a given shard.
func (r *Registry) BySlot(slot uint64) []*Device {
	var out []*Device
	for _, dev := range r.List() {
		if dev.Slot == slot {
			out = append(out, dev)
		}
	}
	return out
}

// Shards returns the configured shard count of the registry.
func (r *Registry) Shards() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.shards
}
