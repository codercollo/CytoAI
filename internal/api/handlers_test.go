package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codercollo/cytoai/internal/risk"
)

func TestScoreCompute_Success(t *testing.T) {
	scorer := mockScorer{score: func(ctx context.Context, b risk.BatteryFeatures, r risk.RepaymentFeatures) (*risk.Result, error) {
		return &risk.Result{
			BatteryHealthIndex:    80,
			RepaymentRiskIndex:    20,
			CytoScore:             80,
			BatteryModelVersion:   "bhi-v0.1",
			RepaymentModelVersion: "rri-v0.2-proxy",
		}, nil
	}}
	srv := newTestServer(validScoreStore(), scorer, mockAuth{})

	req := httptest.NewRequest(http.MethodPost, "/v1/score", strings.NewReader(`{"rider_id":"r1","battery_id":"b1"}`))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var body scoreResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.CytoScore != 80 || body.BatteryHealthIndex != 80 || body.RepaymentRiskIndex != 20 {
		t.Errorf("unexpected score response: %+v", body)
	}
}

func TestScoreCompute_InsufficientData(t *testing.T) {
	scorer := mockScorer{score: func(ctx context.Context, b risk.BatteryFeatures, r risk.RepaymentFeatures) (*risk.Result, error) {
		return nil, risk.ErrInsufficientData
	}}
	srv := newTestServer(validScoreStore(), scorer, mockAuth{})

	req := httptest.NewRequest(http.MethodPost, "/v1/score", strings.NewReader(`{"rider_id":"r1","battery_id":"b1"}`))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (insufficient_data flag); body=%s", rr.Code, rr.Body.String())
	}
	var body insufficientDataResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.InsufficientData {
		t.Errorf("expected insufficient_data=true, got %+v", body)
	}
}

func TestUnauthenticatedRejected(t *testing.T) {
	authn := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		return "", errors.New("invalid key")
	}}
	srv := newTestServer(&mockStore{}, mockScorer{}, authn)

	req := httptest.NewRequest(http.MethodGet, "/v1/portfolio", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rr.Code, rr.Body.String())
	}
}

func TestIngestTelematics_MalformedCSV(t *testing.T) {
	srv := newTestServer(&mockStore{}, mockScorer{}, mockAuth{})

	csv := "battery_id,reading_at,voltage\nb1,2026-01-01,52\nb2,not-a-date,54\n"
	req := httptest.NewRequest(http.MethodPost, "/v1/telematics", strings.NewReader(csv))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "text/csv")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Error   string `json:"error"`
		Details struct {
			RowErrors []struct {
				Row   int    `json:"row"`
				Field string `json:"field"`
				Error string `json:"error"`
			} `json:"row_errors"`
		} `json:"details"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != "invalid_request" {
		t.Errorf("error = %q, want invalid_request", body.Error)
	}
	if len(body.Details.RowErrors) == 0 {
		t.Fatalf("expected per-row errors, got none; body=%s", rr.Body.String())
	}
}
