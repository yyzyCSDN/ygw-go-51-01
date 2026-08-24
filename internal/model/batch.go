package model

// BatchState is the lifecycle of one desired delivery batch.
type BatchState string

const (
	// BatchPending means the batch has not been fully delivered yet.
	BatchPending BatchState = "pending"
	// BatchPartial means part of the batch failed and is scheduled for retry.
	BatchPartial BatchState = "partial"
	// BatchDone means every item either succeeded or was retried successfully.
	BatchDone BatchState = "done"
)

// BatchItem is one desired write inside a batch.
type BatchItem struct {
	DeviceID string            `json:"device_id"`
	Props    map[string]string `json:"props"`
	Priority int               `json:"priority"`
}

// Batch tracks the delivery progress of a group of desired writes.
type Batch struct {
	ID             string      `json:"id"`
	Items          []BatchItem `json:"items"`
	State          BatchState  `json:"state"`
	Applied        []string    `json:"applied"`
	Failed         []string    `json:"failed"`
	RetryRemaining int         `json:"retry_remaining"`
}

// NewBatch creates a pending batch from the given items.
func NewBatch(id string, items []BatchItem) *Batch {
	return &Batch{
		ID:             id,
		Items:          items,
		State:          BatchPending,
		RetryRemaining: len(items),
	}
}

// RecordApplied marks one device as successfully delivered. A device that
// previously failed and is now retried successfully is removed from the
// failed set so it is not re-delivered on the next retry pass.
func (b *Batch) RecordApplied(deviceID string) {
	b.Applied = append(b.Applied, deviceID)
	b.removeFailed(deviceID)
	if b.RetryRemaining > 0 {
		b.RetryRemaining--
	}
}

// MarkPartial records one failed device and moves the batch to the partial
// state so the retry loop keeps the outstanding item. A device already
// recorded as failed is not duplicated so the failure record stays accurate.
func (b *Batch) MarkPartial(deviceID string) {
	if !b.hasFailed(deviceID) {
		b.Failed = append(b.Failed, deviceID)
	}
	b.State = BatchPartial
}

// hasFailed reports whether a device is already in the failed set.
func (b *Batch) hasFailed(deviceID string) bool {
	for _, id := range b.Failed {
		if id == deviceID {
			return true
		}
	}
	return false
}

// removeFailed drops a device from the failed set, used when a retried
// delivery finally succeeds.
func (b *Batch) removeFailed(deviceID string) {
	for i, id := range b.Failed {
		if id == deviceID {
			b.Failed = append(b.Failed[:i], b.Failed[i+1:]...)
			return
		}
	}
}

// MarkDone closes the batch once every item has been delivered.
func (b *Batch) MarkDone() {
	b.State = BatchDone
	b.RetryRemaining = 0
}

// Outstanding returns the items that still need delivery.
func (b *Batch) Outstanding() []BatchItem {
	failed := make(map[string]struct{}, len(b.Failed))
	for _, deviceID := range b.Failed {
		failed[deviceID] = struct{}{}
	}
	var out []BatchItem
	for _, item := range b.Items {
		if _, ok := failed[item.DeviceID]; ok {
			out = append(out, item)
		}
	}
	return out
}
