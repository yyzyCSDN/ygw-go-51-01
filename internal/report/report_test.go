package report

import "testing"

func TestNormalizePropsTrimsAndDropsEmpty(t *testing.T) {
	normalized := NormalizeProps(map[string]string{
		" mode ": " fast ",
		"":       "ignored",
		"keep":   " yes ",
	})
	if normalized["mode"] != "fast" {
		t.Fatalf("key/value not trimmed: %v", normalized)
	}
	if _, ok := normalized[""]; ok {
		t.Fatalf("empty key must be dropped")
	}
	if normalized["keep"] != "yes" {
		t.Fatalf("value not trimmed")
	}
}

func TestValidateDeviceID(t *testing.T) {
	if err := ValidateDeviceID(""); err == nil {
		t.Fatalf("empty device id must be rejected")
	}
	if err := ValidateDeviceID("d1"); err != nil {
		t.Fatalf("valid device id rejected: %v", err)
	}
}

func TestServiceStatsCountsClients(t *testing.T) {
	stats := &ServiceStats{clients: make(map[string]int64)}
	stats.record("gateway-a")
	stats.record("gateway-a")
	stats.record("gateway-b")
	if stats.Total() != 3 {
		t.Fatalf("total reports mismatch")
	}
	if stats.ClientCounts()["gateway-a"] != 2 {
		t.Fatalf("per-client counter mismatch")
	}
}
