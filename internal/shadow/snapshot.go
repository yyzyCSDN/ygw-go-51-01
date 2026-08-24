package shadow

import (
	"sort"

	"deviceshadow/internal/model"
)

// Snapshot returns a deep copy of the document so the caller can read it
// safely even while other goroutines apply new updates.
func (s *Store) Snapshot(deviceID string) *model.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.documents[deviceID]
	if !ok || doc == nil {
		return nil
	}
	return doc.Clone()
}

// Get returns a deep copy of the device shadow, or nil when it does not exist.
func (s *Store) Get(deviceID string) *model.Document {
	return s.Snapshot(deviceID)
}

// List returns deep copies of every live shadow sorted by device id.
func (s *Store) List() []*model.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.documents))
	for deviceID := range s.documents {
		ids = append(ids, deviceID)
	}
	sort.Strings(ids)
	out := make([]*model.Document, 0, len(ids))
	for _, deviceID := range ids {
		out = append(out, s.documents[deviceID].Clone())
	}
	return out
}

// ListDesiredSince returns every committed desired write whose version exceeds
// the given watermark, in ascending version order. It is used by the reconnect
// sync to replay only newer writes and never to re-deliver state the device
// already received. Sorting guards the resync against history reordering or
// truncation: the device always applies the newest desired state last, so a
// stale desired state can never land after and clobber a newer push.
func (s *Store) ListDesiredSince(deviceID string, watermark int64) []model.DesiredOp {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	var ops []model.DesiredOp
	for _, event := range s.history[deviceID] {
		if event.Type != model.EventDesired || event.Version <= watermark {
			continue
		}
		if event.Snapshot == nil {
			continue
		}
		ops = append(ops, model.DesiredOp{
			DeviceID: deviceID,
			Props:    model.CloneMap(event.Snapshot.Desired),
			Version:  event.Version,
			Source:   "reconnect",
		})
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Version < ops[j].Version
	})
	return ops
}

// StateOf returns the state machine status of a device shadow.
func (s *Store) StateOf(deviceID string) model.State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.documents[deviceID]
	if !ok || doc == nil {
		return ""
	}
	return doc.State
}
