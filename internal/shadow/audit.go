package shadow

import "deviceshadow/internal/model"

const historyCap = 20

// record appends a change event to the per-device audit history.
func (s *Store) record(event model.ChangeEvent) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	history := s.history[event.DeviceID]
	history = append(history, event)
	if len(history) > historyCap {
		history = history[len(history)-historyCap:]
	}
	s.history[event.DeviceID] = history
}

// History returns the recent change events of a device in commit order.
func (s *Store) History(deviceID string) []model.ChangeEvent {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	history := s.history[deviceID]
	out := make([]model.ChangeEvent, len(history))
	for index, event := range history {
		out[index] = model.ChangeEvent{
			DeviceID:   event.DeviceID,
			Type:       event.Type,
			Version:    event.Version,
			Snapshot:   event.Snapshot.Clone(),
			OccurredAt: event.OccurredAt,
		}
	}
	return out
}
