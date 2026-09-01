package risk

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestScorer_LiveSidecar exercises the real Flask sidecar if one happens to
// be running at localhost:5001 (e.g. `python -m ml.run` in another
// terminal, with both models trained). It self-skips otherwise, so it's
// safe in normal `go test ./...` runs and CI — this is a manual sanity
// check for local development, not a substitute for the mocked tests above.
func TestScorer_LiveSidecar(t *testing.T) {
	const baseURL = "http://127.0.0.1:5001"

	probe := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := probe.Get(baseURL + "/healthz")
	if err != nil {
		t.Skipf("no live sidecar at %s (start with `python -m ml.run`): %v", baseURL, err)
	}
	resp.Body.Close()

	client := NewMLClient(baseURL, nil)
	ready, err := client.Ready(context.Background())
	if err != nil {
		t.Fatalf("sidecar is up but /readyz call failed: %v", err)
	}
	if !ready.BatteryModelLoaded || !ready.RepaymentModelLoaded {
		t.Skipf("sidecar is up but models aren't trained yet: %+v", ready)
	}

	scorer, err := NewScorer(client, DefaultWeights())
	if err != nil {
		t.Fatalf("unexpected error building scorer: %v", err)
	}

	result, err := scorer.Score(context.Background(),
		BatteryFeatures{CycleCount: 120, AvgDepthOfDischarge: 65, AvgTemperatureC: 27.5, AgeDays: 400, ChargeRateVariance: 0.05},
		RepaymentFeatures{OnTimeRatio: 0.82, AvgDaysLate: 1.4, PaymentCadenceProxy: 0.9, LoanToBatteryValueRatio: 1.1, TenureDays: 300},
	)
	if err != nil {
		t.Fatalf("live score call failed: %v", err)
	}

	t.Logf("live sidecar result: %+v", result)
	if result.CytoScore < 0 || result.CytoScore > 100 {
		t.Errorf("CytoScore out of range: %v", result.CytoScore)
	}
}
