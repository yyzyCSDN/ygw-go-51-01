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

// RecordApplied marks one device as successfully delivered.
func (b *Batch) RecordApplied(deviceID string) {
	b.Applied = append(b.Applied, deviceID)
	if b.RetryRemaining > 0 {
		b.RetryRemaining--
	}
}

// MarkPartial records one failed device and moves the batch to the partial
// state so the retry loop keeps the outstanding items.
func (b *Batch) MarkPartial(deviceID string) {
	b.Failed = append(b.Failed, deviceID)
	b.State = BatchPartial
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
