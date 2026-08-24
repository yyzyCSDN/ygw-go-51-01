// Package desired implements desired state writes: single device writes,
// batch delivery with partial failure tracking, and rollback.
package desired

import (
	"deviceshadow/internal/model"
	"deviceshadow/internal/report"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

// Service applies desired state writes and pushes them to devices.
type Service struct {
	store    *shadow.Store
	versions *sync.VersionAllocator
	syncsvc  *sync.Service
}

// NewService wires the desired service to the store, allocator and sync.
func NewService(store *shadow.Store, versions *sync.VersionAllocator, syncsvc *sync.Service) *Service {
	return &Service{store: store, versions: versions, syncsvc: syncsvc}
}

// ApplyDesired commits one desired write, allocates the next version and
// pushes the op to the device. Offline devices receive the op through the
// sync cache so the change is delivered after reconnect.
func (s *Service) ApplyDesired(deviceID string, props map[string]string, source string, now int64) (*model.Document, error) {
	if err := report.ValidateDeviceID(deviceID); err != nil {
		return nil, err
	}
	normalized := report.NormalizeProps(props)
	if len(normalized) == 0 {
		return s.store.Get(deviceID), nil
	}
	version := s.versions.Next(deviceID)
	op := model.DesiredOp{
		DeviceID: deviceID,
		Props:    normalized,
		Version:  version,
		Source:   source,
	}
	s.store.ApplyDesired(op, now)
	s.syncsvc.Deliver(deviceID, op)
	return s.store.Get(deviceID), nil
}
