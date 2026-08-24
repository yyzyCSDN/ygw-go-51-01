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
// through their cache instead of an immediate delivery. A delivery whose
// version is not strictly newer than what the device already received is
// skipped: the delivered watermark is the resume point, so replaying an older
// desired state would clobber a newer push on the device and roll its config
// back to a stale value. The op is reported as delivered so the caller can keep
// advancing without re-queuing a version that the device already has.
func (s *Service) Deliver(deviceID string, op model.DesiredOp) bool {
	dev, ok := s.devices.Get(deviceID)
	if !ok {
		return false
	}
	if op.Version > 0 && op.Version <= dev.DeliveredWatermark() {
		// The device already received this version or a newer one; never let a
		// stale desired state overwrite a newer push.
		return true
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
// watermark when a device reconnects. Resuming from the delivered watermark
// (not the acknowledged one) guarantees the resync keeps going from the last
// version actually pushed to the device, so a reconnect never restarts from a
// misaligned version and never replays an older desired state that would
// clobber a newer push. The offline cache is drained and folded into the same
// ascending-version plan so writes buffered while the device was away are not
// lost and the latest desired state is the one applied last.
func (s *Service) Resume(deviceID string) []model.DesiredOp {
	dev, ok := s.devices.Get(deviceID)
	if !ok {
		return nil
	}
	// Resume from the highest version the device already received, not the
	// acknowledged one: NB modules ack unreliably, so the ack watermark trails
	// the delivered one. Starting from the ack would replay the whole
	// unacknowledged backlog, including versions older than what the device
	// already applied, rolling the device back to a stale desired state.
	cached := dev.DrainCache()
	start := dev.DeliveredWatermark()
	online := s.store.ListDesiredSince(deviceID, start)
	plan := NewReplayPlan(cached, online)
	ops := plan.Ops()
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
