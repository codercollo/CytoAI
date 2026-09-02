package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestSwapFlow(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	// Battery refs must be unique per run: swap battery resolution is global
	// by external_ref, so a reused ref would resolve to a stale row.
	suffix := time.Now().UnixNano()
	batteryRef1 := fmt.Sprintf("battery_swap_1_%d", suffix)
	batteryRef2 := fmt.Sprintf("battery_swap_2_%d", suffix)

	rider := seedRider(t, env.pool, env.partnerID, "rider_swap")
	battery1 := seedBattery(t, env.pool, batteryRef1)
	battery2 := seedBattery(t, env.pool, batteryRef2)

	csv := fmt.Sprintf(`rider_id,battery_id,station_id,swapped_at,returned_state_of_charge,returned_temperature_c,returned_cycle_count,returned_depth_of_discharge,distance_km_since_last_swap
rider_swap,%s,ST-01,2026-01-01T08:00:00Z,45,27,120,80,42.5
rider_swap,%s,ST-02,2026-01-02T08:00:00Z,50,28,121,78,40.0
`, batteryRef1, batteryRef2)

	resp, err := doPost(env.ts.URL+"/v1/swaps", env.key, "text/csv", csv)
	if err != nil {
		t.Fatalf("POST swaps: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read swaps response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST swaps status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	var result struct {
		Inserted int64 `json:"inserted"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode swaps response: %v", err)
	}
	if result.Inserted != 2 {
		t.Errorf("swaps inserted = %d, want 2", result.Inserted)
	}

	var count int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM swap_events WHERE rider_id = $1 AND battery_id IN ($2, $3)`, rider, battery1, battery2).Scan(&count); err != nil {
		t.Fatalf("count swap events: %v", err)
	}
	if count != 2 {
		t.Errorf("swap event row count = %d, want 2", count)
	}
}

func TestSwapNetworkScoreAndStressFlags(t *testing.T) {
	requireSidecar(t)
	env := setup(t)

	suffix := time.Now().UnixNano()
	repl := map[string]string{
		"rider_n1": fmt.Sprintf("rider_n1_%d", suffix),
		"rider_n2": fmt.Sprintf("rider_n2_%d", suffix),
		"rider_n3": fmt.Sprintf("rider_n3_%d", suffix),
		"rider_n4": fmt.Sprintf("rider_n4_%d", suffix),
		"rider_n5": fmt.Sprintf("rider_n5_%d", suffix),
		"rider_s":  fmt.Sprintf("rider_s_%d", suffix),
		"pool_1":   fmt.Sprintf("pool_1_%d", suffix),
		"pool_2":   fmt.Sprintf("pool_2_%d", suffix),
		"pool_3":   fmt.Sprintf("pool_3_%d", suffix),
		"pool_4":   fmt.Sprintf("pool_4_%d", suffix),
	}

	riders := map[string]string{}
	for _, ref := range []string{"rider_n1", "rider_n2", "rider_n3", "rider_n4", "rider_n5", "rider_s"} {
		riders[ref] = seedRider(t, env.pool, env.partnerID, repl[ref])
	}
	for _, ref := range []string{"pool_1", "pool_2", "pool_3", "pool_4"} {
		seedBattery(t, env.pool, repl[ref])
	}

	stressLoan := seedSwapLoan(t, env.pool, riders["rider_s"], fmt.Sprintf("loan_swap_%d", suffix))
	seedRepayment(t, env.pool, stressLoan, 30)

	csv := substitute(t, "../testdata/sample_swap_events.csv", repl)
	resp, err := doPost(env.ts.URL+"/v1/swaps", env.key, "text/csv", csv)
	if err != nil {
		t.Fatalf("POST swaps: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read swaps response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST swaps status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var inserted struct {
		Inserted int64 `json:"inserted"`
	}
	if err := json.Unmarshal(body, &inserted); err != nil {
		t.Fatalf("decode swaps response: %v", err)
	}
	if inserted.Inserted != 20 {
		t.Errorf("swaps inserted = %d, want 20", inserted.Inserted)
	}

	// Score the stress rider with no battery_id -> swap_network fleet BHI.
	scoreBody := fmt.Sprintf(`{"rider_id":%q}`, riders["rider_s"])
	scoreHTTP, err := doPost(env.ts.URL+"/v1/score", env.key, "application/json", scoreBody)
	if err != nil {
		t.Fatalf("POST score: %v", err)
	}
	defer scoreHTTP.Body.Close()
	if scoreHTTP.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(scoreHTTP.Body)
		t.Fatalf("POST score status = %d, want 200; body=%s", scoreHTTP.StatusCode, b)
	}
	var score scoreResp
	if err := json.NewDecoder(scoreHTTP.Body).Decode(&score); err != nil {
		t.Fatalf("decode score response: %v", err)
	}
	if score.FinancingModel != "swap_network" {
		t.Errorf("financing_model = %q, want swap_network", score.FinancingModel)
	}
	if score.BHIContext != "fleet" {
		t.Errorf("bhi_context = %q, want fleet", score.BHIContext)
	}
	if score.BatteryHealthIndex < 0 || score.BatteryHealthIndex > 100 {
		t.Errorf("battery_health_index = %v, want 0..100", score.BatteryHealthIndex)
	}

	// Operator stress-flags should flag the hot/deep rider.
	flagsHTTP, err := doGet(env.ts.URL+"/v1/operator/stress-flags", env.key)
	if err != nil {
		t.Fatalf("GET stress-flags: %v", err)
	}
	defer flagsHTTP.Body.Close()
	if flagsHTTP.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(flagsHTTP.Body)
		t.Fatalf("GET stress-flags status = %d, want 200; body=%s", flagsHTTP.StatusCode, b)
	}
	var flagsResp struct {
		Flags []anomalyFlag `json:"flags"`
	}
	if err := json.NewDecoder(flagsHTTP.Body).Decode(&flagsResp); err != nil {
		t.Fatalf("decode stress-flags: %v", err)
	}
	found := false
	for _, f := range flagsResp.Flags {
		if f.RuleOrModel == "swap_thermal_stress" || f.RuleOrModel == "swap_deep_discharge_stress" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a swap stress flag, got %+v", flagsResp.Flags)
	}
}
