package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
)

func TestIngestFlow(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	battery1 := seedBattery(t, env.pool, "battery_0001")
	battery2 := seedBattery(t, env.pool, "battery_0002")
	rider := seedRider(t, env.pool, env.partnerID, "rider_0001")
	loan1 := seedLoan(t, env.pool, rider, battery1, "loan_0001")
	loan2 := seedLoan(t, env.pool, rider, battery2, "loan_0002")

	telemetryCSV, err := os.ReadFile("../testdata/sample_telematics.csv")
	if err != nil {
		t.Fatalf("read sample telematics: %v", err)
	}
	repaymentCSV, err := os.ReadFile("../testdata/sample_repayments.csv")
	if err != nil {
		t.Fatalf("read sample repayments: %v", err)
	}

	resp, err := doPost(env.ts.URL+"/v1/telematics", env.key, "text/csv", string(telemetryCSV))
	if err != nil {
		t.Fatalf("POST telematics: %v", err)
	}
	defer resp.Body.Close()
	telemetryBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read telematics response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST telematics status = %d, want 200; body=%s", resp.StatusCode, telemetryBody)
	}
	var telemetryResult struct {
		Inserted int64 `json:"inserted"`
	}
	if err := json.Unmarshal(telemetryBody, &telemetryResult); err != nil {
		t.Fatalf("decode telematics response: %v", err)
	}
	if telemetryResult.Inserted != 10 {
		t.Errorf("telematics inserted = %d, want 10", telemetryResult.Inserted)
	}
	var telemetryRaw map[string]json.RawMessage
	if err := json.Unmarshal(telemetryBody, &telemetryRaw); err != nil {
		t.Fatalf("decode telematics raw response: %v", err)
	}
	if _, ok := telemetryRaw["anomaly_flags"]; ok {
		t.Errorf("telematics ingest response should not include anomaly_flags (detection runs at scoring time)")
	}

	resp2, err := doPost(env.ts.URL+"/v1/repayments", env.key, "text/csv", string(repaymentCSV))
	if err != nil {
		t.Fatalf("POST repayments: %v", err)
	}
	defer resp2.Body.Close()
	repaymentBody, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("read repayments response: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("POST repayments status = %d, want 200; body=%s", resp2.StatusCode, repaymentBody)
	}
	var repaymentResult struct {
		Inserted int64 `json:"inserted"`
	}
	if err := json.Unmarshal(repaymentBody, &repaymentResult); err != nil {
		t.Fatalf("decode repayments response: %v", err)
	}
	if repaymentResult.Inserted != 10 {
		t.Errorf("repayments inserted = %d, want 10", repaymentResult.Inserted)
	}
	var repaymentRaw map[string]json.RawMessage
	if err := json.Unmarshal(repaymentBody, &repaymentRaw); err != nil {
		t.Fatalf("decode repayments raw response: %v", err)
	}
	if _, ok := repaymentRaw["anomaly_flags"]; ok {
		t.Errorf("repayments ingest response should not include anomaly_flags (detection runs at scoring time)")
	}

	var telemetryCount, repaymentCount int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM telemetry_readings WHERE battery_id IN ($1, $2)`, battery1, battery2).Scan(&telemetryCount); err != nil {
		t.Fatalf("count telemetry: %v", err)
	}
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM repayment_events WHERE loan_id IN ($1, $2)`, loan1, loan2).Scan(&repaymentCount); err != nil {
		t.Fatalf("count repayments: %v", err)
	}
	if telemetryCount != 10 {
		t.Errorf("telemetry row count = %d, want 10", telemetryCount)
	}
	if repaymentCount != 10 {
		t.Errorf("repayment row count = %d, want 10", repaymentCount)
	}

	var firstBattery string
	if err := env.pool.QueryRow(ctx, `SELECT battery_id::text FROM telemetry_readings WHERE battery_id = $1 LIMIT 1`, battery1).Scan(&firstBattery); err != nil {
		t.Fatalf("read first telemetry battery_id: %v", err)
	}
	if firstBattery != battery1 {
		t.Errorf("first telemetry battery_id = %q, want %q", firstBattery, battery1)
	}
}
