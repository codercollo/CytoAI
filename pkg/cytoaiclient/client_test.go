package cytoaiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScoreAndGetScoreParseAnomalyFlags(t *testing.T) {
	scoreBody := `{"rider_id":"r1","battery_id":"b1","battery_health_index":90,"repayment_risk_index":12,"cyto_score":89,"battery_model_version":"bhi-v0.1","repayment_model_version":"rri-v0.3-swap-cadence","anomaly_flags":[{"entity_type":"telemetry_reading","entity_id":"1","rule_or_model":"soc_out_of_bounds","severity":"high","reason":"state_of_charge=140"}]}`
	storedBody := `{"id":1,"rider_id":"r1","battery_id":"b1","battery_health_index":90,"repayment_risk_index":12,"cyto_score":89,"model_version":"bhi=bhi-v0.1;rri=rri-v0.3-swap-cadence","anomaly_flags":[]}`

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/score", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Score method = %s, want POST", r.Method)
		}
		_, _ = w.Write([]byte(scoreBody))
	})
	mux.HandleFunc("/v1/score/r1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(storedBody))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	c := New(ts.URL, "key")
	ctx := context.Background()

	score, err := c.Score(ctx, "r1", "b1")
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if len(score.AnomalyFlags) != 1 {
		t.Fatalf("Score anomaly_flags len = %d, want 1", len(score.AnomalyFlags))
	}
	if score.AnomalyFlags[0].RuleOrModel != "soc_out_of_bounds" {
		t.Errorf("Score anomaly rule = %q, want soc_out_of_bounds", score.AnomalyFlags[0].RuleOrModel)
	}

	stored, err := c.GetScore(ctx, "r1")
	if err != nil {
		t.Fatalf("GetScore: %v", err)
	}
	if stored.AnomalyFlags == nil {
		t.Error("GetScore anomaly_flags should decode as a non-nil slice")
	}
	if stored.ModelVersion == nil || *stored.ModelVersion == "" {
		t.Errorf("GetScore model_version should be non-empty, got %v", stored.ModelVersion)
	}
}
