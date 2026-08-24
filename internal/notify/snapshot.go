package notify

import (
	"sort"

	"deviceshadow/internal/model"
)

// Change describes the property-level difference between two snapshots.
type Change struct {
	DeviceID string
	Added    []string
	Updated  []string
	Removed  []string
}

// Diff computes which properties changed between the old and new snapshot.
func Diff(deviceID string, oldDoc, newDoc *model.Document) Change {
	change := Change{DeviceID: deviceID}
	if oldDoc == nil || newDoc == nil {
		return change
	}
	all := make(map[string]struct{})
	for key := range oldDoc.Desired {
		all[key] = struct{}{}
	}
	for key := range oldDoc.Reported {
		all[key] = struct{}{}
	}
	for key := range newDoc.Desired {
		all[key] = struct{}{}
	}
	for key := range newDoc.Reported {
		all[key] = struct{}{}
	}
	keys := make([]string, 0, len(all))
	for key := range all {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		oldValue, oldOK := valueOf(oldDoc, key)
		newValue, newOK := valueOf(newDoc, key)
		switch {
		case !oldOK && newOK:
			change.Added = append(change.Added, key)
		case oldOK && !newOK:
			change.Removed = append(change.Removed, key)
		case oldOK && newOK && oldValue != newValue:
			change.Updated = append(change.Updated, key)
		}
	}
	return change
}

func valueOf(doc *model.Document, key string) (string, bool) {
	if value, ok := doc.Desired[key]; ok {
		return value, true
	}
	value, ok := doc.Reported[key]
	return value, ok
}
