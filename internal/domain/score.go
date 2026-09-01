package domain

import (
	"fmt"
	"time"

	"github.com/codercollo/cytoai/internal/risk"
)

// BatteryHealthIndex is a battery's predicted health (0-100, higher =
// healthier). It is the persistence-facing view of
// risk.Result.BatteryHealthIndex.
type BatteryHealthIndex float64

// RepaymentRiskIndex is a rider's predicted default risk (0-100, lower =
// safer). It is the persistence-facing view of
// risk.Result.RepaymentRiskIndex.
type RepaymentRiskIndex float64

// CytoScore is the combined weighted score (0-100). It is the
// persistence-facing view of risk.Result.CytoScore.
type CytoScore float64

// Score mirrors the `scores` table (docs/spec.md §4). It is the
// persistence-facing view of the runtime scoring result in internal/risk;
// use FromRiskResult to convert between the two.
type Score struct {
	ID                 int64              `db:"id" json:"id,omitempty"`
	RiderID            *string            `db:"rider_id" json:"rider_id,omitempty"`
	BatteryID          *string            `db:"battery_id" json:"battery_id,omitempty"`
	BatteryHealthIndex BatteryHealthIndex `db:"battery_health_index" json:"battery_health_index"`
	RepaymentRiskIndex RepaymentRiskIndex `db:"repayment_risk_index" json:"repayment_risk_index"`
	CytoScore          CytoScore          `db:"cyto_score" json:"cyto_score"`
	ScoredAt           *time.Time         `db:"scored_at" json:"scored_at,omitempty"`
	ModelVersion       *string            `db:"model_version" json:"model_version,omitempty"`
}

// FromRiskResult converts a runtime risk.Result into the persistence-facing
// Score. The three numeric fields map 1:1 by value. risk.Result carries two
// model versions (battery + repayment) while `scores` stores a single
// model_version column, so they are joined deterministically as
// "bhi=<battery version>;rri=<repayment version>".
//
// RiderID, BatteryID, and ScoredAt are not present on risk.Result; the
// persistence layer must populate them after this conversion.
func FromRiskResult(r risk.Result) Score {
	mv := fmt.Sprintf("bhi=%s;rri=%s", r.BatteryModelVersion, r.RepaymentModelVersion)
	return Score{
		BatteryHealthIndex: BatteryHealthIndex(r.BatteryHealthIndex),
		RepaymentRiskIndex: RepaymentRiskIndex(r.RepaymentRiskIndex),
		CytoScore:          CytoScore(r.CytoScore),
		ModelVersion:       &mv,
	}
}
