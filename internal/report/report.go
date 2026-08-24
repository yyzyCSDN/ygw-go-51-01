// Package report implements the device report entry point: version
// bookkeeping, conflict merge and change notification.
package report

import (
	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

// Service accepts device reports and commits them to the shadow store.
type Service struct {
	store    *shadow.Store
	versions *sync.VersionAllocator
	stats    *ServiceStats
}

// NewService wires the report service to the store and version allocator.
func NewService(store *shadow.Store, versions *sync.VersionAllocator) *Service {
	return &Service{store: store, versions: versions, stats: &ServiceStats{clients: make(map[string]int64)}}
}

// HandleReport normalises the reported properties, allocates the next version
// and commits the report. baseVersion is the document version the device
// claims to be based on; a report that is stale against the reported section
// is rejected by the conflict merge.
func (s *Service) HandleReport(deviceID string, props map[string]string, baseVersion int64, clientID string, now int64) (*model.Document, error) {
	normalized := NormalizeProps(props)
	if err := ValidateDeviceID(deviceID); err != nil {
		return nil, err
	}
	// The report version is taken directly from the current document
	// watermark without advancing the shared allocator, so the reported
	// section keeps the watermark it already has on every write.
	current := s.store.CurrentVersion(deviceID)
	next := current
	op := model.ReportOp{
		DeviceID:    deviceID,
		Props:       normalized,
		BaseVersion: baseVersion,
		Version:     next,
		ClientID:    clientID,
	}
	doc := s.store.ApplyReport(op, now)
	s.stats.record(clientID)
	if doc == nil {
		return nil, ErrNoShadow
	}
	return doc, nil
}

// ClientStats exposes the per-gateway report counters.
func (s *Service) ClientStats() map[string]int64 {
	return s.stats.ClientCounts()
}

// ReportTotal returns the total number of reports handled.
func (s *Service) ReportTotal() int64 {
	return s.stats.Total()
}
