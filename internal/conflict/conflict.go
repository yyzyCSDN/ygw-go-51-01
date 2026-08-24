// Package conflict implements the version conflict rules of the shadow sync
// service. Desired and reported sections advance independently, so an
// operation is only judged stale against its own section version.
package conflict

import (
	"deviceshadow/internal/model"
)

// Resolution describes how one incoming operation should be applied.
type Resolution struct {
	ApplyDesired  bool
	ApplyReported bool
}

// Resolve compares an incoming operation against the current section versions
// and decides whether it may be applied. opVersion is the version the sender
// claims to be based on; a stale operation is rejected entirely.
func Resolve(currentDesired, currentReported, opVersion int64, kind model.OpKind) Resolution {
	switch kind {
	case model.OpDesired:
		if opVersion < currentDesired {
			return Resolution{}
		}
		return Resolution{
			ApplyDesired: true,
		}
	case model.OpReport:
		if opVersion < currentReported {
			return Resolution{}
		}
		return Resolution{
			ApplyReported: true,
		}
	}
	return Resolution{}
}

// MergeDesired applies a desired write to a clone of the current document.
// Only the desired section is replaced; the reported section is preserved so
// a concurrent on-site report is never clobbered by a stale operator hint.
func MergeDesired(current *model.Document, op model.DesiredOp, now int64) *model.Document {
	out := current.Clone()
	res := Resolve(current.DesiredVersion, current.ReportedVersion, op.Version, model.OpDesired)
	if !res.ApplyDesired {
		out.UpdatedAt = now
		return out
	}
	out.Desired = model.CloneMap(op.Props)
	out.DesiredVersion = model.MaxVersion(out.DesiredVersion, op.Version)
	out.State = model.StateStale
	out.RefreshWatermark()
	out.UpdatedAt = now
	return out
}

// MergeReport applies a device report to a clone of the current document.
// Reported values are merged per property; desired is never touched. A report
// that is stale against the reported section is rejected entirely so a late
// old report cannot roll the shadow back.
func MergeReport(current *model.Document, op model.ReportOp, now int64) *model.Document {
	out := current.Clone()
	// Staleness is judged against the version the device claims to be based
	// on; the allocator-assigned Version only advances the reported section.
	res := Resolve(current.DesiredVersion, current.ReportedVersion, op.BaseVersion, model.OpReport)
	if !res.ApplyReported {
		out.UpdatedAt = now
		return out
	}
	out.Reported = model.MergeMaps(out.Reported, op.Props)
	out.ReportedVersion = model.MaxVersion(out.ReportedVersion, op.Version)
	out.State = model.StateSynced
	out.RefreshWatermark()
	out.UpdatedAt = now
	return out
}
