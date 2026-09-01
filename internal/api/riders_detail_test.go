package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/jackc/pgx/v5"
)

func TestRiderDetail_ExistingRiderReturnsIdentity(t *testing.T) {
	store := &mockStore{
		riderByID: func(ctx context.Context, partnerID, riderID string) (domain.RiderSummary, error) {
			return domain.RiderSummary{
				Rider:   domain.Rider{ID: "r1", ExternalRef: sptr("rider_1")},
				Battery: &domain.Battery{ID: "b1", ExternalRef: sptr("battery_1")},
				Loan:    &domain.Loan{ID: "l1", RiderID: "r1", BatteryID: sptr("b1")},
			}, nil
		},
	}
	srv := newTestServer(store, mockScorer{}, mockAuth{})

	req := httptest.NewRequest(http.MethodGet, "/v1/riders/r1", nil)
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var body riderResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ID != "r1" || body.ExternalRef == nil || *body.ExternalRef != "rider_1" {
		t.Errorf("identity = %+v, want r1/rider_1", body)
	}
	if body.RegistrationStatus != registrationStatusLinked {
		t.Errorf("registration_status = %q, want linked", body.RegistrationStatus)
	}
}

func TestRiderDetail_NeverScoredReturnsExplicitNull(t *testing.T) {
	store := &mockStore{
		riderByID: func(ctx context.Context, partnerID, riderID string) (domain.RiderSummary, error) {
			return domain.RiderSummary{Rider: domain.Rider{ID: "r1", ExternalRef: sptr("rider_1")}}, nil
		},
	}
	srv := newTestServer(store, mockScorer{}, mockAuth{})

	req := httptest.NewRequest(http.MethodGet, "/v1/riders/r1", nil)
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var body riderResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.LatestScore != nil {
		t.Errorf("latest_score = %+v, want null", body.LatestScore)
	}
	if !strings.Contains(rr.Body.String(), `"latest_score":null`) {
		t.Errorf("expected explicit `latest_score:null`, got %s", rr.Body.String())
	}
}

func TestRiderDetail_LatestScoreReturned(t *testing.T) {
	scoredAt := time.Now().UTC()
	store := &mockStore{
		riderByID: func(ctx context.Context, partnerID, riderID string) (domain.RiderSummary, error) {
			return domain.RiderSummary{
				Rider: domain.Rider{ID: "r1"},
				Score: &domain.Score{
					ID:                 9,
					BatteryID:          sptr("b1"),
					BatteryHealthIndex: 80,
					RepaymentRiskIndex: 20,
					CytoScore:          80,
					ScoredAt:           &scoredAt,
					ModelVersion:       sptr("bhi=x;rri=y"),
				},
			}, nil
		},
	}
	srv := newTestServer(store, mockScorer{}, mockAuth{})

	req := httptest.NewRequest(http.MethodGet, "/v1/riders/r1", nil)
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var body riderResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.LatestScore == nil {
		t.Fatalf("latest_score = nil, want a score; body=%s", rr.Body.String())
	}
	if body.LatestScore.CytoScore != 80 || body.LatestScore.BatteryHealthIndex != 80 || body.LatestScore.RepaymentRiskIndex != 20 {
		t.Errorf("latest_score = %+v, want cyto=80 bhi=80 rri=20", body.LatestScore)
	}
}

func TestRiderDetail_OtherPartnerRiderInaccessible(t *testing.T) {
	store := &mockStore{
		riderByID: func(ctx context.Context, partnerID, riderID string) (domain.RiderSummary, error) {
			return domain.RiderSummary{}, pgx.ErrNoRows
		},
	}
	srv := newTestServer(store, mockScorer{}, mockAuth{})

	req := httptest.NewRequest(http.MethodGet, "/v1/riders/someone-elses-rider", nil)
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "not_found" {
		t.Errorf("error = %q, want not_found", body.Error)
	}
}
