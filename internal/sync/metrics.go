package sync

import "sync"

// Metrics tracks the delivery counters of the sync service. They feed the
// console overview so operators can see reconnect and offline activity.
type Metrics struct {
	mu         sync.Mutex
	delivered  int64
	replayed   int64
	cached     int64
	reconciled int64
}

func (m *Metrics) addDelivered(delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delivered += delta
}

func (m *Metrics) addReplayed(delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.replayed += delta
}

func (m *Metrics) addCached(delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cached += delta
}

func (m *Metrics) addReconciled(delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reconciled += delta
}

// Snapshot returns a point-in-time copy of the counters.
func (m *Metrics) Snapshot() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]int64{
		"delivered":  m.delivered,
		"replayed":   m.replayed,
		"cached":     m.cached,
		"reconciled": m.reconciled,
	}
}
