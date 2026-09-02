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
	"github.com/codercollo/cytoai/internal/risk"
)

func fleetSwaps() []domain.SwapEvent {
	return []domain.SwapEvent{
		{ReturnedTemperatureC: fptr(27), ReturnedDepthOfDischarge: fptr(80), ReturnedCycleCount: iptr(100)},
		{ReturnedTemperatureC: fptr(29), ReturnedDepthOfDischarge: fptr(70), ReturnedCycleCount: iptr(101)},
		{ReturnedTemperatureC: fptr(31), ReturnedDepthOfDischarge: fptr(90), ReturnedCycleCount: iptr(102)},
	}
}

func hotRiderSwaps() []domain.SwapEvent {
	return []domain.SwapEvent{
		{ReturnedTemperatureC: fptr(40), ReturnedDepthOfDischarge: fptr(95), ReturnedCycleCount: iptr(103)},
	}
}

func swapLoan() domain.Loan {
	return domain.Loan{
		ID:              "loan-swap",
		PrincipalKes:    fptr(300000),
		BatteryValueKes: fptr(150000),
		FinancingModel:  sptr("swap_network"),
	}
}

func onTimeEvent() []domain.RepaymentEvent {
	return []domain.RepaymentEvent{{
		LoanID:  "loan-swap",
		DueDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:  sptr("on_time"),
	}}
}

func TestScoreCompute_SwapNetwork_WiresBatteryStressProfile(t *testing.T) {
	var captured risk.RepaymentFeatures
	store := &mockStore{
		latestLoanForRider: func(ctx context.Context, riderID string) (domain.Loan, error) {
			return swapLoan(), nil
		},
		repaymentByLoan: func(ctx context.Context, loanID string) ([]domain.RepaymentEvent, error) {
			return onTimeEvent(), nil
		},
		swapEventsByPartner: func(ctx context.Context, partnerID string) ([]domain.SwapEvent, error) {
			return fleetSwaps(), nil
		},
		swapEventsByRider: func(ctx context.Context, riderID string) ([]domain.SwapEvent, error) {
			return hotRiderSwaps(), nil
		},
		insertScore: func(ctx context.Context, s domain.Score) (int64, error) { return 1, nil },
	}
	scorer := mockScorer{score: func(ctx context.Context, b risk.BatteryFeatures, r risk.RepaymentFeatures) (*risk.Result, error) {
		captured = r
		return &risk.Result{BatteryHealthIndex: 80, RepaymentRiskIndex: 20, CytoScore: 80}, nil
	}}

	srv := newTestServer(store, scorer, mockAuth{})
	req := httptest.NewRequest(http.MethodPost, "/v1/score", strings.NewReader(`{"rider_id":"r-swap"}`))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if captured.BatteryStressProfile <= 0 {
		t.Errorf("BatteryStressProfile = %v, want > 0 (hotter/deeper rider vs fleet)", captured.BatteryStressProfile)
	}
}

func TestRescoreAll_CoversSwapNetworkRiders(t *testing.T) {
	store := &mockStore{
		riderBatteryPairs: func(ctx context.Context, partnerID string) ([]domain.RiderBatteryPair, error) {
			return []domain.RiderBatteryPair{{RiderID: "r1", BatteryID: "b1"}}, nil
		},
		swapNetworkRiderIDs: func(ctx context.Context, partnerID string) ([]string, error) {
			return []string{"r2"}, nil
		},
		loanByRiderAndBattery: func(ctx context.Context, riderID, batteryID string) (domain.Loan, error) {
			return domain.Loan{ID: "loan-lease", PrincipalKes: fptr(300000), BatteryValueKes: fptr(150000)}, nil
		},
		latestLoanForRider: func(ctx context.Context, riderID string) (domain.Loan, error) {
			return swapLoan(), nil
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
			return onTimeEvent(), nil
		},
		swapEventsByPartner: func(ctx context.Context, partnerID string) ([]domain.SwapEvent, error) {
			return fleetSwaps(), nil
		},
		swapEventsByRider: func(ctx context.Context, riderID string) ([]domain.SwapEvent, error) {
			return hotRiderSwaps(), nil
		},
		insertScore: func(ctx context.Context, s domain.Score) (int64, error) { return 1, nil },
	}

	calls := 0
	scorer := mockScorer{score: func(ctx context.Context, b risk.BatteryFeatures, r risk.RepaymentFeatures) (*risk.Result, error) {
		calls++
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
	if calls != 2 {
		t.Errorf("scorer calls = %d, want 2 (lease + swap)", calls)
	}
	if body.Scored != 2 {
		t.Errorf("scored = %d, want 2", body.Scored)
	}
	if len(body.Skipped) != 0 {
		t.Errorf("skipped = %d, want 0; body=%+v", len(body.Skipped), body)
	}
}
