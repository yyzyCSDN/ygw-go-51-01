package report

import (
	"errors"
	"strings"
)

// ErrNoShadow is returned when a report targets a deleted shadow.
var ErrNoShadow = errors.New("shadow does not exist")

// NormalizeProps trims every key and value, drops empty keys and returns a
// stable property map.
func NormalizeProps(props map[string]string) map[string]string {
	out := make(map[string]string, len(props))
	for key, value := range props {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}

// ValidateDeviceID rejects empty or oversized device identifiers.
func ValidateDeviceID(deviceID string) error {
	if strings.TrimSpace(deviceID) == "" {
		return errors.New("device id is empty")
	}
	if len(deviceID) > 128 {
		return errors.New("device id is too long")
	}
	return nil
}
