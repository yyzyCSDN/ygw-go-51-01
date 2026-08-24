package sync

import (
	"deviceshadow/internal/device"
	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
)

// Service coordinates reconnect resync and offline replay for the shadow
// store. Every delivery advances the device watermark so a reconnect never
// replays state the device already received.
type Service struct {
	store   *shadow.Store
	devices *device.Registry
	metrics *Metrics
}

// NewService wires the sync service to the shadow store and device registry.
func NewService(store *shadow.Store, devices *device.Registry) *Service {
	return &Service{store: store, devices: devices, metrics: &Metrics{}}
}

// Deliver pushes one desired op to a device. Offline devices receive the op
// through their cache instead of an immediate delivery.
func (s *Service) Deliver(deviceID string, op model.DesiredOp) bool {
	dev, ok := s.devices.Get(deviceID)
	if !ok {
		return false
	}
	if !dev.Online {
		dev.CacheDesired(op)
		s.metrics.addCached(1)
		return false
	}
	dev.MarkDelivered(op.Version)
	s.metrics.addDelivered(1)
	return true
}

// Resume replays desired writes that are newer than the device delivery
// watermark when a device reconnects.
func (s *Service) Resume(deviceID string) []model.DesiredOp {
	dev, ok := s.devices.Get(deviceID)
	if !ok {
		return nil
	}
	start := dev.DeliveredWatermark()
	ops := s.store.ListDesiredSince(deviceID, start)
	for _, op := range ops {
		if !s.Deliver(deviceID, op) {
			break
		}
		s.metrics.addReplayed(1)
	}
	s.store.Publish(model.ChangeEvent{
		DeviceID:   deviceID,
		Type:       model.EventReconnect,
		Version:    s.store.CurrentVersion(deviceID),
		Snapshot:   s.store.Snapshot(deviceID),
		OccurredAt: nowNanos(),
	})
	return ops
}

// Reconcile drains the offline cache and replays the online resume in one
// deduplicated plan when a device comes back.
func (s *Service) Reconcile(deviceID string) []model.DesiredOp {
	dev, ok := s.devices.Get(deviceID)
	if !ok {
		return nil
	}
	cached := dev.DrainCache()
	watermark := dev.DeliveredWatermark()
	online := s.store.ListDesiredSince(deviceID, watermark)
	plan := NewReplayPlan(cached, online)
	for _, op := range plan.Ops() {
		if !s.Deliver(deviceID, op) {
			break
		}
		s.metrics.addReplayed(1)
	}
	s.metrics.addReconciled(1)
	s.store.Publish(model.ChangeEvent{
		DeviceID:   deviceID,
		Type:       model.EventReconnect,
		Version:    s.store.CurrentVersion(deviceID),
		Snapshot:   s.store.Snapshot(deviceID),
		OccurredAt: nowNanos(),
	})
	return plan.Ops()
}

// Metrics returns a copy of the delivery counters.
func (s *Service) Metrics() map[string]int64 {
	return s.metrics.Snapshot()
}
