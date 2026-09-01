package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type riderCreateResp struct {
	ID                 string           `json:"id"`
	RegistrationStatus string           `json:"registration_status"`
	Battery            *batteryResp     `json:"battery"`
	Loan               *loanResp        `json:"loan"`
	LatestScore        *json.RawMessage `json:"latest_score"`
}

type batteryResp struct {
	ID string `json:"id"`
}

type loanResp struct {
	ID string `json:"id"`
}

// TestRiderRegistrationFlow exercises the new rider API end-to-end and proves
// the existing scoring flow accepts the registered rider's IDs + loan (it
// should return insufficient_data rather than "no loan" since no telemetry or
// repayment history has been ingested yet).
func TestRiderRegistrationFlow(t *testing.T) {
	env := setup(t)

	body := `{"external_ref":"rider_http","battery":{"external_ref":"battery_http","manufacturer":"demo","rated_capacity_wh":2000},"loan":{"external_ref":"loan_http","principal_kes":300000,"battery_value_kes":150000,"term_months":12}}`
	resp, err := doPost(env.ts.URL+"/v1/riders", env.key, "application/json", body)
	if err != nil {
		t.Fatalf("POST /v1/riders: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /v1/riders status = %d, want 201; body=%s", resp.StatusCode, b)
	}
	var created riderCreateResp
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" || created.Battery == nil || created.Battery.ID == "" || created.Loan == nil || created.Loan.ID == "" {
		t.Fatalf("incomplete create response: %+v", created)
	}
	if created.RegistrationStatus != "linked" {
		t.Errorf("registration_status = %q, want linked", created.RegistrationStatus)
	}
	if created.LatestScore != nil {
		t.Errorf("latest_score = %s, want null", string(*created.LatestScore))
	}

	// Detail returns identity with an explicit no-score state.
	getResp, err := doGet(env.ts.URL+"/v1/riders/"+created.ID, env.key)
	if err != nil {
		t.Fatalf("GET /v1/riders/{id}: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(getResp.Body)
		t.Fatalf("GET rider status = %d, want 200; body=%s", getResp.StatusCode, b)
	}
	var detail struct {
		ID          string           `json:"id"`
		LatestScore *json.RawMessage `json:"latest_score"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.ID != created.ID {
		t.Errorf("detail id = %q, want %q", detail.ID, created.ID)
	}
	if detail.LatestScore != nil {
		t.Errorf("detail latest_score = %s, want null", string(*detail.LatestScore))
	}

	// Scoring compatibility: POST /v1/score with the returned IDs finds the
	// loan, then correctly reports insufficient data (no telemetry yet).
	scoreBody := `{"rider_id":"` + created.ID + `","battery_id":"` + created.Battery.ID + `"}`
	scoreResp, err := doPost(env.ts.URL+"/v1/score", env.key, "application/json", scoreBody)
	if err != nil {
		t.Fatalf("POST /v1/score: %v", err)
	}
	defer scoreResp.Body.Close()
	if scoreResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(scoreResp.Body)
		t.Fatalf("POST /v1/score status = %d, want 200; body=%s", scoreResp.StatusCode, b)
	}
	var ins struct {
		InsufficientData bool   `json:"insufficient_data"`
		Reason           string `json:"reason"`
	}
	if err := json.NewDecoder(scoreResp.Body).Decode(&ins); err != nil {
		t.Fatalf("decode score response: %v", err)
	}
	if !ins.InsufficientData {
		t.Errorf("expected insufficient_data=true for a registered-but-not-yet-scored rider, got %+v", ins)
	}
	if ins.Reason == "" {
		t.Error("expected a reason for insufficient data")
	}
}
