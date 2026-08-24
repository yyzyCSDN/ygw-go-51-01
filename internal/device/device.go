package device

import "deviceshadow/internal/model"

// Device is one registered device with its sync watermark. LastDelivered is
// the highest desired version already pushed to the device; LastAck is the
// highest version the device confirmed back.
type Device struct {
	ID            string
	Slot          uint64
	Online        bool
	LastDelivered int64
	LastAck       int64
	RegisteredAt  int64
	Cache         *OfflineCache
}

// NewDevice builds a device handle with an empty offline cache.
func NewDevice(id string, slot uint64, now int64) *Device {
	return &Device{
		ID:           id,
		Slot:         slot,
		Online:       true,
		RegisteredAt: now,
		Cache:        NewOfflineCache(),
	}
}

// MarkDelivered advances the delivered watermark of the device. Watermarks
// only move forward; a stale delivery can never roll the watermark back.
func (d *Device) MarkDelivered(version int64) {
	if version > d.LastDelivered {
		d.LastDelivered = version
	}
}

// Ack records the device confirmation of a version.
func (d *Device) Ack(version int64) {
	if version > d.LastAck {
		d.LastAck = version
	}
}

// DeliveredWatermark returns the highest version pushed to the device.
func (d *Device) DeliveredWatermark() int64 {
	return d.LastDelivered
}

// AckWatermark returns the highest version confirmed by the device.
func (d *Device) AckWatermark() int64 {
	return d.LastAck
}

// CacheDesired stores a desired write for delivery once the device is back
// online.
func (d *Device) CacheDesired(op model.DesiredOp) {
	d.Cache.Add(op)
}

// DrainCache returns the cached desired writes.
func (d *Device) DrainCache() []model.DesiredOp {
	return d.Cache.Drain()
}
