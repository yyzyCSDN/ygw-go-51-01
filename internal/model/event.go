package model

// EventType classifies shadow change notifications.
type EventType int

const (
	// EventDesired fires after a desired state write.
	EventDesired EventType = iota
	// EventReport fires after a device report is merged.
	EventReport
	// EventDelete fires after a shadow is deleted.
	EventDelete
	// EventReconnect fires after a device reconnects and resynchronises.
	EventReconnect
)

// ChangeEvent is emitted after every committed shadow change. Snapshot is a
// deep copy taken at commit time so consumers never observe a half update.
type ChangeEvent struct {
	DeviceID   string
	Type       EventType
	Version    int64
	Snapshot   *Document
	OccurredAt int64
}
