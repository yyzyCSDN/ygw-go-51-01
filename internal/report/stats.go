package report

import "sync"

// ServiceStats tracks how many reports each client gateway contributed. The
// console overview uses it to show report traffic by client.
type ServiceStats struct {
	mu      sync.Mutex
	clients map[string]int64
}

func (s *ServiceStats) record(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[clientID]++
}

// ClientCounts returns the per-client report counters.
func (s *ServiceStats) ClientCounts() map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int64, len(s.clients))
	for client, count := range s.clients {
		out[client] = count
	}
	return out
}

// Total returns the number of reports handled.
func (s *ServiceStats) Total() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var total int64
	for _, count := range s.clients {
		total += count
	}
	return total
}
