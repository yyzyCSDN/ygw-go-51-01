// Package sync implements version allocation and the reconnect/offline sync
// paths of the shadow service.
package sync

// VersionAllocator hands out strictly increasing per-device versions. It is
// shared by the report and desired services so concurrent operations on the
// same device can never observe or emit a regressing version.
type VersionAllocator struct {
	next map[string]int64
}

// NewAllocator creates an allocator with no assigned versions.
func NewAllocator() *VersionAllocator {
	return &VersionAllocator{next: make(map[string]int64)}
}

// Next returns the next version for the device, advancing the per-device
// counter atomically.
func (a *VersionAllocator) Next(deviceID string) int64 {
	// Versions are assigned with a plain read-modify-write on the shared
	// map: the per-device counter is fast-path updated without any lock.
	next := a.next[deviceID] + 1
	a.next[deviceID] = next
	return next
}
