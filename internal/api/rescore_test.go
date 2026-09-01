package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/codercollo/cytoai/internal/risk"
)

func TestRescoreAll_ScoresAndSkips(t *testing.T) {
	store := &mockStore{
		riderBatteryPairs: func(ctx context.Context, partnerID string) ([]domain.RiderBatteryPair, error) {
			return []domain.RiderBatteryPair{
				{RiderID: "r1", BatteryID: "b1"},
				{RiderID: "r2", BatteryID: "b2"},
			}, nil
		},
		loanByRiderAndBattery: func(ctx context.Context, riderID, batteryID string) (domain.Loan, error) {
			return domain.Loan{ID: "loan-" + riderID, PrincipalKes: fptr(300000), BatteryValueKes: fptr(150000)}, nil
		},
		telemetryByBattery: func(ctx context.Context, batteryID string) ([]domain.TelemetryReading, error) {
			return []domain.TelemetryReading{{
				BatteryID:        batteryID,
				ReadingAt:        time.Now().UTC(),
				DepthOfDischarge: fptr(60),
				TemperatureC:     fptr(27),
				Voltage:          fptr(52),
				CycleCount:       iptr(5),
			}}, nil
		},
		repaymentByLoan: func(ctx context.Context, loanID string) ([]domain.RepaymentEvent, error) {
			return []domain.RepaymentEvent{{LoanID: loanID, DueDate: time.Now().UTC(), Status: sptr("on_time")}}, nil
		},
		insertScore: func(ctx context.Context, s domain.Score) (int64, error) { return 1, nil },
	}

	calls := 0
	scorer := mockScorer{score: func(ctx context.Context, b risk.BatteryFeatures, r risk.RepaymentFeatures) (*risk.Result, error) {
		calls++
		if calls == 2 {
			return nil, risk.ErrInsufficientData
		}
		return &risk.Result{BatteryHealthIndex: 80, RepaymentRiskIndex: 20, CytoScore: 80}, nil
	}}

	srv := newTestServer(store, scorer, mockAuth{})
	req := httptest.NewRequest(http.MethodPost, "/v1/score/rescore-all", nil)
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var body rescoreResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Scored != 1 {
		t.Errorf("scored = %d, want 1", body.Scored)
	}
	if len(body.Skipped) != 1 {
		t.Fatalf("skipped = %d, want 1; body=%+v", len(body.Skipped), body)
	}
	if body.Skipped[0].RiderID != "r2" || body.Skipped[0].BatteryID != "b2" {
		t.Errorf("skipped pair = %+v, want r2/b2", body.Skipped[0])
	}
	if body.Skipped[0].Reason == "" {
		t.Error("skipped reason should be non-empty")
	}
}

func TestRescoreAll_UsesAuthenticatedPartner(t *testing.T) {
	var gotPartner string
	store := &mockStore{
		riderBatteryPairs: func(ctx context.Context, partnerID string) ([]domain.RiderBatteryPair, error) {
			gotPartner = partnerID
			return nil, nil
		},
	}
	auth := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		return "partner-42", nil
	}}
	srv := newTestServer(store, mockScorer{}, auth)

	req := httptest.NewRequest(http.MethodPost, "/v1/score/rescore-all", nil)
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if gotPartner != "partner-42" {
		t.Errorf("RiderBatteryPairsForPartner received partnerID %q, want partner-42", gotPartner)
	}
}
