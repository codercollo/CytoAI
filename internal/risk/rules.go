package risk

import (
	"errors"
	"fmt"
)

// ErrInsufficientData is returned by Scorer.Score when a rider/battery
// doesn't yet have enough history to score fairly. Per spec.md section 9
// ("Bias monitoring"): thin-file, informal-economy riders are the target
// beneficiary group, so a new rider must never be scored as high-risk
// purely for lacking history — they should get "not enough data yet"
// instead of a bad score.
var ErrInsufficientData = errors.New("insufficient data to score fairly")

// Minimum data thresholds before a score is considered fair to compute.
// These are deliberately conservative for an MVP and are expected to be
// tuned once real partner data volume is understood.
const (
	// MinBatteryCycleCount is the minimum number of charge cycles a
	// battery needs before its degradation trend is meaningful.
	MinBatteryCycleCount = 5.0

	// MinRepaymentTenureDays is the minimum loan age before repayment
	// behaviour is a meaningful signal rather than noise from a handful
	// of early payments.
	MinRepaymentTenureDays = 14.0
)

// ClampScore forces a value into the valid [0, 100] index range. The
// sidecar already clamps on its side (ml/app/models/*.py), but the Go API
// re-clamps defensively — it must never trust a network boundary to
// enforce its own invariants, and it's the layer partners' data ultimately
// flows back out through.
func ClampScore(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 100:
		return 100
	default:
		return v
	}
}

// SufficientBatteryData reports whether a battery has enough telemetry
// history to be scored fairly, and if not, why.
func SufficientBatteryData(f BatteryFeatures) (bool, error) {
	if f.CycleCount < MinBatteryCycleCount {
		return false, fmt.Errorf(
			"%w: battery has %.0f charge cycles, need at least %.0f",
			ErrInsufficientData, f.CycleCount, MinBatteryCycleCount,
		)
	}
	return true, nil
}

// SufficientRepaymentData reports whether a loan has enough repayment
// history to be scored fairly, and if not, why.
func SufficientRepaymentData(f RepaymentFeatures) (bool, error) {
	if f.TenureDays < MinRepaymentTenureDays {
		return false, fmt.Errorf(
			"%w: loan is %.0f days old, need at least %.0f days of history",
			ErrInsufficientData, f.TenureDays, MinRepaymentTenureDays,
		)
	}
	return true, nil
}
