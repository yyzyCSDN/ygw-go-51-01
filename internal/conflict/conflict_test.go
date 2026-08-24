package conflict

import (
	"testing"

	"deviceshadow/internal/model"
)

func TestResolveAppliesMatchingVersion(t *testing.T) {
	res := Resolve(0, 5, 5, model.OpReport)
	if !res.ApplyReported {
		t.Fatalf("report matching the reported version must be applied")
	}
	res = Resolve(0, 5, 4, model.OpReport)
	if res.ApplyReported {
		t.Fatalf("stale report must be rejected")
	}
}

func TestMergeReportMergesProperties(t *testing.T) {
	doc := model.NewDocument("d1", 1)
	doc.Reported["temp"] = "20"
	merged := MergeReport(doc, model.ReportOp{
		DeviceID:    "d1",
		Props:       map[string]string{"humidity": "55"},
		BaseVersion: 0,
		Version:     2,
	}, 2)
	if merged.Reported["temp"] != "20" || merged.Reported["humidity"] != "55" {
		t.Fatalf("report merge lost properties: %v", merged.Reported)
	}
	if merged.ReportedVersion != 2 {
		t.Fatalf("reported version not advanced: %d", merged.ReportedVersion)
	}
}

func TestPolicyForSource(t *testing.T) {
	if PolicyForSource("gateway-snapshot").ReplaceSection != true {
		t.Fatalf("gateway snapshot policy must replace the section")
	}
	if PolicyForSource("console").ReplaceSection {
		t.Fatalf("console policy must not replace the section")
	}
}
