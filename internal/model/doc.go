// Package model defines the shadow document, operation and batch types shared
// by every component of the device shadow sync service.
package model

// State describes the lifecycle of one device shadow document.
type State string

const (
	// StateCreated marks a shadow that exists but has not received any report.
	StateCreated State = "created"
	// StateSynced marks a shadow whose reported section matches the device.
	StateSynced State = "synced"
	// StateStale marks a shadow that is waiting for the device to apply or
	// acknowledge the latest desired state.
	StateStale State = "stale"
	// StateDeleted marks a shadow that was explicitly removed.
	StateDeleted State = "deleted"
)

// Document is the in-memory device shadow. Desired holds the platform target
// state and Reported holds the latest state observed from the device itself.
// DesiredVersion and ReportedVersion advance independently; Version is the
// overall watermark used by sync offsets and change notifications.
type Document struct {
	DeviceID string
	Desired  map[string]string
	Reported map[string]string

	DesiredVersion  int64
	ReportedVersion int64
	Version         int64

	State     State
	UpdatedAt int64
}

// NewDocument creates an empty shadow document in the created state.
func NewDocument(deviceID string, now int64) *Document {
	return &Document{
		DeviceID:  deviceID,
		Desired:   map[string]string{},
		Reported:  map[string]string{},
		State:     StateCreated,
		UpdatedAt: now,
	}
}

// Clone returns a deep copy of the document. Callers can safely mutate the
// returned maps without affecting the stored shadow.
func (d *Document) Clone() *Document {
	if d == nil {
		return nil
	}
	out := &Document{
		DeviceID:        d.DeviceID,
		Desired:         CloneMap(d.Desired),
		Reported:        CloneMap(d.Reported),
		DesiredVersion:  d.DesiredVersion,
		ReportedVersion: d.ReportedVersion,
		Version:         d.Version,
		State:           d.State,
		UpdatedAt:       d.UpdatedAt,
	}
	return out
}

// CloneMap returns an independent copy of a string map.
func CloneMap(source map[string]string) map[string]string {
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

// MergeMaps overlays the values of overlay onto a copy of base and returns the
// merged map. Values already present in base are replaced by overlay values.
func MergeMaps(base, overlay map[string]string) map[string]string {
	out := CloneMap(base)
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

// RefreshWatermark recomputes the overall version from the two section
// versions. It must be called after any section version change.
func (d *Document) RefreshWatermark() {
	d.Version = MaxVersion(d.DesiredVersion, d.ReportedVersion)
}

// PropertyCount returns the number of known properties across both sections.
func (d *Document) PropertyCount() int {
	if d == nil {
		return 0
	}
	return len(d.Desired) + len(d.Reported)
}
