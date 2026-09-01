package risk

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Weights configures how BHI and RRI combine into CytoScore. Per spec.md
// section 5, this combination is intentionally a simple, documented
// weighted average — not a learned combiner — so a lender or regulator can
// see exactly why a score landed where it did.
type Weights struct {
	BHI float64 // weight on Battery Health Index (higher = healthier)
	RRI float64 // weight on (100 - Repayment Risk Index), i.e. repayment safety
}

// DefaultWeights returns spec.md's default 50/50 split. Configurable per
// partner risk appetite — see spec.md section 5.
func DefaultWeights() Weights {
	return Weights{BHI: 0.5, RRI: 0.5}
}

// Validate checks the weights are non-negative and sum to 1 (within a small
// epsilon for float rounding).
func (w Weights) Validate() error {
	if w.BHI < 0 || w.RRI < 0 {
		return fmt.Errorf("weights must be non-negative, got BHI=%.4f RRI=%.4f", w.BHI, w.RRI)
	}
	const epsilon = 1e-6
	sum := w.BHI + w.RRI
	if sum < 1-epsilon || sum > 1+epsilon {
		return fmt.Errorf("weights must sum to 1.0, got %.4f (BHI=%.4f RRI=%.4f)", sum, w.BHI, w.RRI)
	}
	return nil
}

// Result is the full scoring output for one rider/battery pair.
type Result struct {
	BatteryHealthIndex    float64 `json:"battery_health_index"`
	RepaymentRiskIndex    float64 `json:"repayment_risk_index"`
	CytoScore             float64 `json:"cyto_score"`
	BatteryModelVersion   string  `json:"battery_model_version"`
	RepaymentModelVersion string  `json:"repayment_model_version"`
	// AnomalyFlags are fraud/anomaly findings surfaced alongside the score.
	// They inform confidence/trust only — they never change the CytoScore math.
	AnomalyFlags []AnomalyFlag `json:"anomaly_flags"`
}

// Scorer orchestrates a full CytoScore computation: guardrail checks
// (rules.go), the two ML sidecar calls (mlclient.go, run concurrently since
// they're independent), and the deterministic combination below.
type Scorer struct {
	client  *MLClient
	weights Weights
}

// NewScorer constructs a Scorer. Returns an error if weights are invalid,
// so a misconfigured partner risk-appetite setting fails at startup rather
// than silently producing wrong scores per request.
func NewScorer(client *MLClient, weights Weights) (*Scorer, error) {
	if err := weights.Validate(); err != nil {
		return nil, fmt.Errorf("invalid scorer weights: %w", err)
	}
	return &Scorer{client: client, weights: weights}, nil
}

// Score computes a full CytoScore. Returns ErrInsufficientData (wrapped)
// if either side lacks enough history — callers should surface that as a
// distinct "not enough data yet" state, not a low/bad score, per spec.md's
// bias-monitoring guidance.
func (s *Scorer) Score(ctx context.Context, battery BatteryFeatures, repayment RepaymentFeatures) (*Result, error) {
	if ok, err := SufficientBatteryData(battery); !ok {
		return nil, err
	}
	if ok, err := SufficientRepaymentData(repayment); !ok {
		return nil, err
	}

	var (
		wg              sync.WaitGroup
		batteryResult   *BatteryPrediction
		repaymentResult *RepaymentPrediction
		batteryErr      error
		repaymentErr    error
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		batteryResult, batteryErr = s.client.PredictBattery(ctx, battery)
	}()
	go func() {
		defer wg.Done()
		repaymentResult, repaymentErr = s.client.PredictRepayment(ctx, repayment)
	}()
	wg.Wait()

	if err := errors.Join(batteryErr, repaymentErr); err != nil {
		return nil, fmt.Errorf("scoring: %w", err)
	}

	bhi := ClampScore(batteryResult.BatteryHealthIndex)
	rri := ClampScore(repaymentResult.RepaymentRiskIndex)
	cytoScore := ClampScore(s.weights.RRI*(100-rri) + s.weights.BHI*bhi)

	return &Result{
		BatteryHealthIndex:    bhi,
		RepaymentRiskIndex:    rri,
		CytoScore:             cytoScore,
		BatteryModelVersion:   batteryResult.ModelVersion,
		RepaymentModelVersion: repaymentResult.ModelVersion,
		AnomalyFlags:          []AnomalyFlag{},
	}, nil
}
