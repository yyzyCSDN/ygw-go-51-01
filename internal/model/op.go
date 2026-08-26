package model

// OpKind distinguishes the two shadow update directions.
type OpKind int

const (
	// OpDesired is a platform to device desired state write.
	OpDesired OpKind = iota
	// OpReport is a device to platform state report.
	OpReport
)

// DesiredOp describes one desired state write for a device. Reported carries
// the operator-observed reported baseline captured when the operation was
// prepared; it is only a conflict hint and never the source of truth.
type DesiredOp struct {
	DeviceID string
	Props    map[string]string
	Version  int64
	Reported map[string]string
	Source   string
}

// ReportOp describes one state report received from a device. BaseVersion is
// the shadow version the device claims to be based on and is used for stale
// detection; Version is the new version assigned by the service allocator.
type ReportOp struct {
	DeviceID    string
	Props       map[string]string
	BaseVersion int64
	Version     int64
	ClientID    string
}
