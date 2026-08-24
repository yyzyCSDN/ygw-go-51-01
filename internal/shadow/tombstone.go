package shadow

// IsTombstoned reports whether the device shadow was explicitly deleted.
func (s *Store) IsTombstoned(deviceID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.tombstones[deviceID]
	return ok
}

// GCTombstones drops tombstone records older than the given cutoff so the
// map cannot grow without bound. It never resurrects live documents.
func (s *Store) GCTombstones(cutoff int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for deviceID, deletedAt := range s.tombstones {
		if deletedAt < cutoff {
			delete(s.tombstones, deviceID)
			removed++
		}
	}
	return removed
}

// TombstoneCount returns how many tombstones are currently retained.
func (s *Store) TombstoneCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tombstones)
}
