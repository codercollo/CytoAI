package integration

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seedBattery(t *testing.T, pool *pgxpool.Pool, ref string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO batteries (external_ref, manufacturer) VALUES ($1, 'demo') RETURNING id::text`, ref).Scan(&id); err != nil {
		t.Fatalf("seed battery %s: %v", ref, err)
	}
	return id
}

func seedRider(t *testing.T, pool *pgxpool.Pool, partnerID, ref string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO riders (external_ref, partner_id) VALUES ($1, $2) RETURNING id::text`, ref, partnerID).Scan(&id); err != nil {
		t.Fatalf("seed rider %s: %v", ref, err)
	}
	return id
}

func seedLoan(t *testing.T, pool *pgxpool.Pool, riderID, batteryID, ref string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO loans (external_ref, rider_id, battery_id, principal_kes, battery_value_kes) VALUES ($1, $2, $3, 300000, 150000) RETURNING id::text`, ref, riderID, batteryID).Scan(&id); err != nil {
		t.Fatalf("seed loan: %v", err)
	}
	return id
}

func seedSwapLoan(t *testing.T, pool *pgxpool.Pool, riderID, ref string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO loans (external_ref, rider_id, battery_id, principal_kes, battery_value_kes, financing_model) VALUES ($1, $2, NULL, 300000, 150000, 'swap_network') RETURNING id::text`, ref, riderID).Scan(&id); err != nil {
		t.Fatalf("seed swap loan: %v", err)
	}
	return id
}

func seedTelemetry(t *testing.T, pool *pgxpool.Pool, batteryID string, n int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		readingAt := time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC)
		if _, err := pool.Exec(ctx, `INSERT INTO telemetry_readings (battery_id, reading_at, state_of_charge, voltage, temperature_c, cycle_count, depth_of_discharge, distance_km_since_last) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			batteryID, readingAt, 80, 52+float64(i%5)*0.5, 27, i+1, 60, 12.5); err != nil {
			t.Fatalf("seed telemetry: %v", err)
		}
	}
}

func seedRepayment(t *testing.T, pool *pgxpool.Pool, loanID string, n int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		due := time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC)
		if _, err := pool.Exec(ctx, `INSERT INTO repayment_events (loan_id, due_date, paid_date, amount_due_kes, amount_paid_kes, status) VALUES ($1,$2,$3,$4,$5,'on_time')`,
			loanID, due, due, 460, 460); err != nil {
			t.Fatalf("seed repayment: %v", err)
		}
	}
}

func substitute(t *testing.T, path string, repl map[string]string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	for old, new := range repl {
		s = strings.ReplaceAll(s, old, new)
	}
	return s
}

func doPost(url, key, contentType, body string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", contentType)
	return http.DefaultClient.Do(req)
}

func doGet(url, key string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	return http.DefaultClient.Do(req)
}
