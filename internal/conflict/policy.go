package conflict

import "deviceshadow/internal/model"

// Policy controls how reports from a particular source are merged.
type Policy struct {
	// ReplaceSection marks that reports from this source are full snapshots;
	// an equal-version report replaces the reported section instead of
	// merging, which prevents stale keys from surviving a snapshot push.
	ReplaceSection bool
}

// DefaultPolicy returns the default conflict policy used by the service.
func DefaultPolicy() Policy {
	return Policy{}
}

// ApplyReport applies op to current using the policy. It is the entry point
// used by the shadow store so the merge rules live in one place.
func (p Policy) ApplyReport(current *model.Document, op model.ReportOp, now int64) *model.Document {
	merged := MergeReport(current, op, now)
	if p.ReplaceSection && op.BaseVersion == current.ReportedVersion && op.BaseVersion > 0 {
		out := current.Clone()
		out.Reported = model.CloneMap(op.Props)
		out.ReportedVersion = model.MaxVersion(out.ReportedVersion, op.Version)
		out.State = model.StateSynced
		out.RefreshWatermark()
		out.UpdatedAt = now
		return out
	}
	return merged
}

// PolicyForSource selects a policy based on the operator source label. Normal
// console and API writes use the default policy.
func PolicyForSource(source string) Policy {
	if source == "gateway-snapshot" {
		return Policy{ReplaceSection: true}
	}
	return DefaultPolicy()
}
