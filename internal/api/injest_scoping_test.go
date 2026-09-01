package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestIngestTelematics_PassesAuthenticatedPartnerID verifies that the
// partner ID resolved by auth middleware is the one threaded into
// ResolveBatteryID — not left empty, not taken from request data.
func TestIngestTelematics_PassesAuthenticatedPartnerID(t *testing.T) {
	var gotPartnerID string
	store := &mockStore{
		resolveBatteryID: func(ctx context.Context, partnerID, ref string) (string, error) {
			gotPartnerID = partnerID
			return "battery-uuid-1", nil
		},
	}
	auth := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		return "partner-77", nil
	}}
	srv := newTestServer(store, mockScorer{}, auth)

	body := `[{"battery_id":"b1","reading_at":"2026-01-01T00:00:00Z","voltage":52}]`
	req := httptest.NewRequest(http.MethodPost, "/v1/telematics", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if gotPartnerID != "partner-77" {
		t.Errorf("ResolveBatteryID called with partnerID = %q, want %q", gotPartnerID, "partner-77")
	}
}

// TestIngestTelematics_CrossPartnerRefRejected verifies that a battery ref
// belonging to a different partner is rejected identically to an unknown
// ref — the store layer returns "not found" for both cases, and the
// handler must not distinguish between them in its response.
func TestIngestTelematics_CrossPartnerRefRejected(t *testing.T) {
	store := &mockStore{
		resolveBatteryID: func(ctx context.Context, partnerID, ref string) (string, error) {
			// Simulates the real query: a ref that exists but belongs to
			// another partner produces the same not-found error as a
			// nonexistent ref.
			return "", errNotFoundForTest
		},
	}
	srv := newTestServer(store, mockScorer{}, mockAuth{})

	body := `[{"battery_id":"someone-elses-battery","reading_at":"2026-01-01T00:00:00Z","voltage":52}]`
	req := httptest.NewRequest(http.MethodPost, "/v1/telematics", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Error   string `json:"error"`
		Details struct {
			RowErrors []refError `json:"row_errors"`
		} `json:"details"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Details.RowErrors) != 1 {
		t.Fatalf("expected 1 row error, got %d; body=%s", len(resp.Details.RowErrors), rr.Body.String())
	}
	if resp.Details.RowErrors[0].Error != "unknown external ref" {
		t.Errorf("row error = %q, want %q (must not leak that the ref exists under another partner)",
			resp.Details.RowErrors[0].Error, "unknown external ref")
	}
}

// TestIngestRepayments_PassesAuthenticatedPartnerID mirrors the telematics
// test above for the repayment/loan ingestion path.
func TestIngestRepayments_PassesAuthenticatedPartnerID(t *testing.T) {
	var gotPartnerID string
	store := &mockStore{
		resolveLoanID: func(ctx context.Context, partnerID, ref string) (string, error) {
			gotPartnerID = partnerID
			return "loan-uuid-1", nil
		},
	}
	auth := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		return "partner-42", nil
	}}
	srv := newTestServer(store, mockScorer{}, auth)

	body := `[{"loan_id":"l1","due_date":"2026-01-01T00:00:00Z","status":"on_time"}]`
	req := httptest.NewRequest(http.MethodPost, "/v1/repayments", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if gotPartnerID != "partner-42" {
		t.Errorf("ResolveLoanID called with partnerID = %q, want %q", gotPartnerID, "partner-42")
	}
}

var errNotFoundForTest = &notFoundErr{}

type notFoundErr struct{}

func (e *notFoundErr) Error() string { return "not found" }
