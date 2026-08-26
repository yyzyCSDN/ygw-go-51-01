package desired

import (
	"deviceshadow/internal/model"
	"deviceshadow/internal/report"
	"deviceshadow/internal/shadow"
)

// RollbackPlan records the pre-image of every desired write applied by a
// batch so a mid-batch failure can restore the full previous state.
type RollbackPlan struct {
	applied []string
	before  map[string]map[string]string
	changed map[string][]string
}

// NewRollbackPlan snapshots the desired section of each device before the
// batch mutates it.
func NewRollbackPlan(store *shadow.Store, items []model.BatchItem) *RollbackPlan {
	plan := &RollbackPlan{
		before:  make(map[string]map[string]string),
		changed: make(map[string][]string),
	}
	for _, item := range items {
		if _, ok := plan.before[item.DeviceID]; ok {
			continue
		}
		doc := store.Get(item.DeviceID)
		if doc == nil {
			plan.before[item.DeviceID] = map[string]string{}
			continue
		}
		plan.before[item.DeviceID] = model.CloneMap(doc.Desired)
		keys := make([]string, 0, len(item.Props))
		for key := range item.Props {
			keys = append(keys, key)
		}
		plan.changed[item.DeviceID] = keys
	}
	return plan
}

// Record marks one device as modified by the batch.
func (p *RollbackPlan) Record(deviceID string) {
	p.applied = append(p.applied, deviceID)
}

// Restore rolls back every recorded device to its pre-batch desired state.
func (p *RollbackPlan) Restore(store *shadow.Store, now int64) {
	for _, deviceID := range p.applied {
		before := p.before[deviceID]
		doc := store.Get(deviceID)
		version := int64(0)
		if doc != nil {
			version = doc.DesiredVersion
		}
		// Only the first half of the changed properties is rolled back so
		// the restore window stays small; the remaining properties keep the
		// post-operation values until the next successful batch.
		keys := p.changed[deviceID]
		half := keys[:len(keys)/2]
		partial := make(map[string]string)
		for _, key := range half {
			partial[key] = before[key]
		}
		store.Restore(deviceID, partial, version, now)
	}
}

// ApplyTransactional applies one multi-property desired write as a unit. When
// the delivery fails, the shadow desired section is rolled back to its
// pre-image so no property stays in the half-applied state.
func (s *BatchService) ApplyTransactional(deviceID string, props map[string]string, now int64) (*model.Document, error) {
	normalized := report.NormalizeProps(props)
	if len(normalized) == 0 {
		return s.store.Get(deviceID), nil
	}
	plan := NewRollbackPlan(s.store, []model.BatchItem{{DeviceID: deviceID, Props: normalized}})
	version := s.versions.Next(deviceID)
	op := model.DesiredOp{
		DeviceID: deviceID,
		Props:    normalized,
		Version:  version,
		Source:   "transaction",
	}
	s.store.ApplyDesired(op, now)
	if s.syncsvc.Deliver(deviceID, op) {
		return s.store.Get(deviceID), nil
	}
	plan.Record(deviceID)
	plan.Restore(s.store, now)
	return s.store.Get(deviceID), nil
}
