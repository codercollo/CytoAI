package domain

import (
	"testing"

	"github.com/codercollo/cytoai/internal/risk"
)

func TestFromRiskResult(t *testing.T) {
	r := risk.Result{
		BatteryHealthIndex:    80,
		RepaymentRiskIndex:    20,
		CytoScore:             80,
		BatteryModelVersion:   "bhi-v0.1",
		RepaymentModelVersion: "rri-v0.2-proxy",
	}

	s := FromRiskResult(r)

	if s.BatteryHealthIndex != BatteryHealthIndex(80) {
		t.Errorf("BatteryHealthIndex = %v, want 80", s.BatteryHealthIndex)
	}
	if s.RepaymentRiskIndex != RepaymentRiskIndex(20) {
		t.Errorf("RepaymentRiskIndex = %v, want 20", s.RepaymentRiskIndex)
	}
	if s.CytoScore != CytoScore(80) {
		t.Errorf("CytoScore = %v, want 80", s.CytoScore)
	}
	if s.ModelVersion == nil || *s.ModelVersion != "bhi=bhi-v0.1;rri=rri-v0.2-proxy" {
		t.Errorf("ModelVersion = %v, want joined model versions", s.ModelVersion)
	}
	if s.RiderID != nil || s.BatteryID != nil || s.ScoredAt != nil {
		t.Errorf("expected caller-populated fields to be nil, got %+v", s)
	}
}
