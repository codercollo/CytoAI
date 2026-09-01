package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
)

func TestRiderDetail_FactorsComputed(t *testing.T) {
	scoredAt := time.Now().UTC()
	store := &mockStore{
		riderByID: func(ctx context.Context, partnerID, riderID string) (domain.RiderSummary, error) {
			return domain.RiderSummary{
				Rider: domain.Rider{ID: "r1"},
				Score: &domain.Score{ID: 9, BatteryID: sptr("b1"), ScoredAt: &scoredAt},
			}, nil
		},
		loanByRiderAndBattery: func(ctx context.Context, riderID, batteryID string) (domain.Loan, error) {
			return domain.Loan{ID: "l1", PrincipalKes: fptr(300000), BatteryValueKes: fptr(150000)}, nil
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
			return []domain.RepaymentEvent{{
				LoanID:  loanID,
				DueDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				Status:  sptr("on_time"),
			}}, nil
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
	if body.Factors == nil {
		t.Fatalf("factors = nil, want computed factors; body=%s", rr.Body.String())
	}
	if body.Factors.Battery.CycleCount != 5 {
		t.Errorf("battery cycle_count = %v, want 5", body.Factors.Battery.CycleCount)
	}
	if body.Factors.Repayment.OnTimeRatio != 1.0 {
		t.Errorf("repayment on_time_ratio = %v, want 1.0", body.Factors.Repayment.OnTimeRatio)
	}
}
