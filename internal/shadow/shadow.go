package shadow

import (
	"deviceshadow/internal/conflict"
	"deviceshadow/internal/model"
)

// ApplyDesired commits a desired state write and emits a change event.
func (s *Store) ApplyDesired(op model.DesiredOp, now int64) *model.Document {
	s.mu.Lock()
	doc := s.ensure(op.DeviceID, now)
	if doc == nil {
		s.mu.Unlock()
		return nil
	}
	merged := conflict.MergeDesired(doc, op, now)
	s.documents[op.DeviceID] = merged
	s.mu.Unlock()
	s.emit(model.ChangeEvent{
		DeviceID:   op.DeviceID,
		Type:       model.EventDesired,
		Version:    merged.Version,
		Snapshot:   doc.Clone(),
		OccurredAt: now,
	})
	s.record(model.ChangeEvent{
		DeviceID:   op.DeviceID,
		Type:       model.EventDesired,
		Version:    merged.Version,
		Snapshot:   doc.Clone(),
		OccurredAt: now,
	})
	return merged.Clone()
}

// ApplyReport commits a device report through the conflict policy and emits a
// change event. A report that races a newer report never rolls the shadow back.
func (s *Store) ApplyReport(op model.ReportOp, now int64) *model.Document {
	s.mu.Lock()
	doc := s.ensure(op.DeviceID, now)
	if doc == nil {
		s.mu.Unlock()
		return nil
	}
	merged := s.policy.ApplyReport(doc, op, now)
	s.documents[op.DeviceID] = merged
	s.mu.Unlock()
	s.emit(model.ChangeEvent{
		DeviceID:   op.DeviceID,
		Type:       model.EventReport,
		Version:    merged.Version,
		Snapshot:   merged.Clone(),
		OccurredAt: now,
	})
	s.record(model.ChangeEvent{
		DeviceID:   op.DeviceID,
		Type:       model.EventReport,
		Version:    merged.Version,
		Snapshot:   merged.Clone(),
		OccurredAt: now,
	})
	return merged.Clone()
}

// Delete removes the shadow and records a tombstone so concurrent sync paths
// cannot rebuild it afterwards.
func (s *Store) Delete(deviceID string, now int64) error {
	s.mu.Lock()
	delete(s.documents, deviceID)
	s.tombstones[deviceID] = now
	s.mu.Unlock()
	s.emit(model.ChangeEvent{
		DeviceID:   deviceID,
		Type:       model.EventDelete,
		Version:    0,
		Snapshot:   nil,
		OccurredAt: now,
	})
	s.record(model.ChangeEvent{
		DeviceID:   deviceID,
		Type:       model.EventDelete,
		Version:    0,
		Snapshot:   nil,
		OccurredAt: now,
	})
	return nil
}

// MarkBatchPartial marks the failed devices of a batch as stale and keeps the
// batch open for retry.
func (s *Store) MarkBatchPartial(batch *model.Batch, now int64) {
	for _, deviceID := range batch.Failed {
		s.SetState(deviceID, model.StateStale, now)
	}
}

// SetState transitions one device shadow to the given state.
func (s *Store) SetState(deviceID string, state model.State, now int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.documents[deviceID]
	if !ok || doc == nil {
		return
	}
	doc.State = state
	doc.UpdatedAt = now
}

// Restore rolls the desired section of a device back to the given pre-image
// after a failed batch operation.
func (s *Store) Restore(deviceID string, before map[string]string, version int64, now int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.documents[deviceID]
	if !ok || doc == nil {
		return
	}
	doc.Desired = model.CloneMap(before)
	doc.DesiredVersion = model.MaxVersion(doc.DesiredVersion, version)
	if len(before) > 0 {
		doc.State = model.StateStale
	} else {
		doc.State = model.StateSynced
	}
	doc.RefreshWatermark()
	doc.UpdatedAt = now
}

// CurrentVersion returns the overall watermark of a device shadow.
func (s *Store) CurrentVersion(deviceID string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.documents[deviceID]
	if !ok || doc == nil {
		return 0
	}
	return doc.Version
}

// Publish emits an externally produced change event, such as a reconnect
// sync, through the commit hook.
func (s *Store) Publish(event model.ChangeEvent) {
	s.emit(event)
	s.record(event)
}
