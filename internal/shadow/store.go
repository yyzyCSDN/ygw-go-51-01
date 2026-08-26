// Package shadow implements the device shadow document store and its state
// machine. All document reads and writes are serialised by the store mutex so
// component interactions never observe a half-written shadow.
package shadow

import (
	"sync"

	"deviceshadow/internal/conflict"
	"deviceshadow/internal/model"
)

// Store keeps every device shadow in memory.
type Store struct {
	mu         sync.RWMutex
	historyMu  sync.Mutex
	documents  map[string]*model.Document
	tombstones map[string]int64
	history    map[string][]model.ChangeEvent
	policy     conflict.Policy
	onChange   func(model.ChangeEvent)
}

// NewStore creates an empty shadow store with the default conflict policy.
func NewStore() *Store {
	return &Store{
		documents:  make(map[string]*model.Document),
		tombstones: make(map[string]int64),
		history:    make(map[string][]model.ChangeEvent),
		policy:     conflict.DefaultPolicy(),
	}
}

// SetPolicy replaces the conflict policy used for report merging.
func (s *Store) SetPolicy(policy conflict.Policy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = policy
}

// SetOnChange registers the commit hook used by the notification hub.
func (s *Store) SetOnChange(hook func(model.ChangeEvent)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onChange = hook
}

// emit fires the commit hook outside the store lock.
func (s *Store) emit(event model.ChangeEvent) {
	s.mu.RLock()
	hook := s.onChange
	s.mu.RUnlock()
	if hook != nil {
		hook(event)
	}
}

// ensure returns the document for the device, creating it when it does not
// exist. Tombstoned devices are never resurrected by a late update.
func (s *Store) ensure(deviceID string, now int64) *model.Document {
	if _, tombstoned := s.tombstones[deviceID]; tombstoned {
		return nil
	}
	doc, ok := s.documents[deviceID]
	if !ok {
		doc = model.NewDocument(deviceID, now)
		s.documents[deviceID] = doc
	}
	return doc
}
