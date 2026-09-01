package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
)

type anomalyFlag struct {
	EntityType  string `json:"entity_type"`
	EntityID    string `json:"entity_id"`
	RuleOrModel string `json:"rule_or_model"`
	Severity    string `json:"severity"`
	Reason      string `json:"reason"`
}

type scoreResp struct {
	RiderID               string        `json:"rider_id"`
	BatteryID             string        `json:"battery_id"`
	BatteryHealthIndex    float64       `json:"battery_health_index"`
	RepaymentRiskIndex    float64       `json:"repayment_risk_index"`
	CytoScore             float64       `json:"cyto_score"`
	BatteryModelVersion   string        `json:"battery_model_version"`
	RepaymentModelVersion string        `json:"repayment_model_version"`
	AnomalyFlags          []anomalyFlag `json:"anomaly_flags"`
}

type storedScore struct {
	ID                 int64         `json:"id"`
	BatteryHealthIndex float64       `json:"battery_health_index"`
	RepaymentRiskIndex float64       `json:"repayment_risk_index"`
	CytoScore          float64       `json:"cyto_score"`
	ModelVersion       *string       `json:"model_version"`
	AnomalyFlags       []anomalyFlag `json:"anomaly_flags"`
}

type insufficientResp struct {
	InsufficientData bool   `json:"insufficient_data"`
	Reason           string `json:"reason"`
}

func requireSidecar(t *testing.T) {
	t.Helper()
	base := os.Getenv("CYTOAI_ML_BASE_URL")
	if base == "" {
		base = "http://localhost:5001"
	}
	resp, err := http.Get(base + "/healthz")
	if err != nil {
		t.Skipf("flask sidecar not running at %s: %v", base, err)
	}
	if resp != nil {
		resp.Body.Close()
	}
}

func TestScoreFlow(t *testing.T) {
	requireSidecar(t)
	env := setup(t)

	rider := seedRider(t, env.pool, env.partnerID, "rider_score")
	battery := seedBattery(t, env.pool, "battery_score")
	loan := seedLoan(t, env.pool, rider, battery, "")
	seedTelemetry(t, env.pool, battery, 10) // 10 cycles >= MinBatteryCycleCount
	seedRepayment(t, env.pool, loan, 30)    // 29-day tenure >= MinRepaymentTenureDays

	body := fmt.Sprintf(`{"rider_id":%q,"battery_id":%q}`, rider, battery)
	resp, err := doPost(env.ts.URL+"/v1/score", env.key, "application/json", body)
	if err != nil {
		t.Fatalf("POST score: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST score status = %d, want 200; body=%s", resp.StatusCode, b)
	}
	var score scoreResp
	if err := json.NewDecoder(resp.Body).Decode(&score); err != nil {
		t.Fatalf("decode score response: %v", err)
	}
	if score.CytoScore < 0 || score.CytoScore > 100 {
		t.Errorf("cyto_score = %v, want 0..100", score.CytoScore)
	}
	if score.BatteryHealthIndex < 0 || score.BatteryHealthIndex > 100 {
		t.Errorf("battery_health_index = %v, want 0..100", score.BatteryHealthIndex)
	}
	if score.RepaymentRiskIndex < 0 || score.RepaymentRiskIndex > 100 {
		t.Errorf("repayment_risk_index = %v, want 0..100", score.RepaymentRiskIndex)
	}
	if score.AnomalyFlags == nil {
		t.Error("POST score anomaly_flags should be a non-nil array")
	}

	// Fetch the persisted score and confirm it matches what was computed.
	getResp, err := doGet(env.ts.URL+"/v1/score/"+rider, env.key)
	if err != nil {
		t.Fatalf("GET score: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(getResp.Body)
		t.Fatalf("GET score status = %d, want 200; body=%s", getResp.StatusCode, b)
	}
	var stored storedScore
	if err := json.NewDecoder(getResp.Body).Decode(&stored); err != nil {
		t.Fatalf("decode stored score: %v", err)
	}
	if stored.CytoScore != score.CytoScore {
		t.Errorf("stored cyto_score = %v, want %v", stored.CytoScore, score.CytoScore)
	}
	if stored.ModelVersion == nil || *stored.ModelVersion == "" {
		t.Errorf("stored model_version should be non-empty, got %v", stored.ModelVersion)
	}
	if stored.AnomalyFlags == nil {
		t.Error("GET score anomaly_flags should be a non-nil array")
	}
}

func TestScoreFlow_InsufficientData(t *testing.T) {
	env := setup(t)

	rider := seedRider(t, env.pool, env.partnerID, "rider_thin")
	battery := seedBattery(t, env.pool, "battery_thin")
	loan := seedLoan(t, env.pool, rider, battery, "")
	seedTelemetry(t, env.pool, battery, 2) // 2 cycles < MinBatteryCycleCount
	seedRepayment(t, env.pool, loan, 1)

	body := fmt.Sprintf(`{"rider_id":%q,"battery_id":%q}`, rider, battery)
	resp, err := doPost(env.ts.URL+"/v1/score", env.key, "application/json", body)
	if err != nil {
		t.Fatalf("POST score: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST score status = %d, want 200; body=%s", resp.StatusCode, b)
	}
	var ins insufficientResp
	if err := json.NewDecoder(resp.Body).Decode(&ins); err != nil {
		t.Fatalf("decode insufficient response: %v", err)
	}
	if !ins.InsufficientData {
		t.Errorf("expected insufficient_data=true, got %+v", ins)
	}
	if ins.Reason == "" {
		t.Error("expected a reason for insufficient data")
	}
}
