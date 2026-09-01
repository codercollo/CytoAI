package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codercollo/cytoai/internal/domain"
)

func TestRiderList_ReturnsRidersForAuthenticatedPartner(t *testing.T) {
	var gotPartner string
	store := &mockStore{
		listRiders: func(ctx context.Context, partnerID string, limit, offset int) ([]domain.RiderSummary, error) {
			gotPartner = partnerID
			return []domain.RiderSummary{{
				Rider: domain.Rider{ID: "rider-uuid-1", ExternalRef: sptr("rider_123")},
			}}, nil
		},
	}
	auth := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		return "partner-77", nil
	}}
	srv := newTestServer(store, mockScorer{}, auth)

	req := httptest.NewRequest(http.MethodGet, "/v1/riders", nil)
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if gotPartner != "partner-77" {
		t.Errorf("ListRiders called with partnerID %q, want partner-77", gotPartner)
	}
	var body riderListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Riders) != 1 {
		t.Fatalf("riders = %d, want 1", len(body.Riders))
	}
	if body.Riders[0].ExternalRef == nil || *body.Riders[0].ExternalRef != "rider_123" {
		t.Errorf("rider external_ref = %v, want rider_123", body.Riders[0].ExternalRef)
	}
}

func TestRiderList_IgnoresRequestPartnerHint(t *testing.T) {
	var gotPartner string
	store := &mockStore{
		listRiders: func(ctx context.Context, partnerID string, limit, offset int) ([]domain.RiderSummary, error) {
			gotPartner = partnerID
			return nil, nil
		},
	}
	auth := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		return "partner-A", nil
	}}
	srv := newTestServer(store, mockScorer{}, auth)

	req := httptest.NewRequest(http.MethodGet, "/v1/riders?partner_id=partner-B", nil)
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	// The handler must derive the partner from auth, never from the request.
	if gotPartner != "partner-A" {
		t.Errorf("ListRiders called with partnerID %q, want partner-A (auth-derived)", gotPartner)
	}
}

func TestRiderCreate_Valid(t *testing.T) {
	store := &mockStore{
		createRider: func(ctx context.Context, partnerID string, reg domain.RiderRegistration) (domain.RiderSummary, error) {
			return domain.RiderSummary{Rider: domain.Rider{ID: "r1", ExternalRef: reg.ExternalRef}}, nil
		},
	}
	srv := newTestServer(store, mockScorer{}, mockAuth{})

	req := httptest.NewRequest(http.MethodPost, "/v1/riders", strings.NewReader(`{"external_ref":"rider_123"}`))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rr.Code, rr.Body.String())
	}
	var body riderResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ID != "r1" {
		t.Errorf("id = %q, want r1", body.ID)
	}
	if body.ExternalRef == nil || *body.ExternalRef != "rider_123" {
		t.Errorf("external_ref = %v, want rider_123", body.ExternalRef)
	}
	if body.RegistrationStatus != registrationStatusRegistered {
		t.Errorf("registration_status = %q, want registered", body.RegistrationStatus)
	}
	if body.LatestScore != nil {
		t.Errorf("latest_score = %+v, want null (never scored)", body.LatestScore)
	}
}

func TestRiderCreate_InvalidData(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing external_ref", `{}`},
		{"battery without loan", `{"external_ref":"r","battery":{}}`},
		{"loan without battery", `{"external_ref":"r","loan":{"principal_kes":100,"battery_value_kes":50}}`},
		{"loan missing principal", `{"external_ref":"r","battery":{},"loan":{"battery_value_kes":50}}`},
		{"loan missing battery value", `{"external_ref":"r","battery":{},"loan":{"principal_kes":100}}`},
		{"negative principal", `{"external_ref":"r","battery":{},"loan":{"principal_kes":-1,"battery_value_kes":50}}`},
		{"bad started_at", `{"external_ref":"r","battery":{},"loan":{"principal_kes":100,"battery_value_kes":50,"started_at":"not-a-date"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer(&mockStore{}, mockScorer{}, mockAuth{})
			req := httptest.NewRequest(http.MethodPost, "/v1/riders", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer good")
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
			}
			var body struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Error != "invalid_request" {
				t.Errorf("error = %q, want invalid_request", body.Error)
			}
		})
	}
}
