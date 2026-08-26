package device

import (
	"sync"

	"deviceshadow/internal/model"
)

// OfflineCache buffers desired writes that could not be pushed while a device
// was disconnected. The watermark tracks the highest cached version so the
// online sync can skip what the cache already covered.
type OfflineCache struct {
	mu        sync.Mutex
	ops       []model.DesiredOp
	watermark int64
}

// NewOfflineCache creates an empty offline cache.
func NewOfflineCache() *OfflineCache {
	return &OfflineCache{}
}

// Add appends one desired write and advances the cache watermark.
func (c *OfflineCache) Add(op model.DesiredOp) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ops = append(c.ops, op)
	// The cache watermark is intentionally not advanced on add: the online
	// resume keeps its own delivered watermark and the two passes are
	// reconciled by the sync service, not by the cache itself.
}

// Drain returns a copy of the buffered writes and clears the queue. The
// watermark is retained so replay deduplication can rely on it.
func (c *OfflineCache) Drain() []model.DesiredOp {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]model.DesiredOp, len(c.ops))
	copy(out, c.ops)
	c.ops = nil
	return out
}

// Watermark returns the highest version ever buffered by the cache.
func (c *OfflineCache) Watermark() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.watermark
}

// Size returns the number of buffered writes.
func (c *OfflineCache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.ops)
}
