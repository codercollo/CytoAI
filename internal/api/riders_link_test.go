package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/jackc/pgx/v5"
)

func TestRiderLink_Success(t *testing.T) {
	var gotPartner, gotRiderID string
	var gotBattery domain.BatteryRegistration
	var gotLoan domain.LoanRegistration

	store := &mockStore{
		linkRider: func(ctx context.Context, partnerID, riderID string, battery domain.BatteryRegistration, loan domain.LoanRegistration) (domain.RiderSummary, error) {
			gotPartner = partnerID
			gotRiderID = riderID
			gotBattery = battery
			gotLoan = loan
			return domain.RiderSummary{
				Rider:   domain.Rider{ID: riderID, ExternalRef: sptr("rider_1")},
				Battery: &domain.Battery{ID: "b1", ExternalRef: battery.ExternalRef},
				Loan: &domain.Loan{
					ID:              "l1",
					RiderID:         riderID,
					BatteryID:       sptr("b1"),
					ExternalRef:     loan.ExternalRef,
					PrincipalKes:    loan.PrincipalKes,
					BatteryValueKes: loan.BatteryValueKes,
				},
			}, nil
		},
	}
	auth := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		return "partner-1", nil
	}}
	srv := newTestServer(store, mockScorer{}, auth)

	body := `{"battery":{"external_ref":"b_ref1","manufacturer":"Roam"},"loan":{"external_ref":"l_ref1","principal_kes":250000,"battery_value_kes":120000}}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/riders/rider_1/link", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if gotPartner != "partner-1" {
		t.Errorf("LinkRider called with partnerID %q, want partner-1", gotPartner)
	}
	if gotRiderID != "rider_1" {
		t.Errorf("LinkRider called with riderID %q, want rider_1", gotRiderID)
	}
	if gotBattery.ExternalRef == nil || *gotBattery.ExternalRef != "b_ref1" {
		t.Errorf("battery ref = %v, want b_ref1", gotBattery.ExternalRef)
	}
	if gotLoan.PrincipalKes == nil || *gotLoan.PrincipalKes != 250000 {
		t.Errorf("loan principal = %v, want 250000", gotLoan.PrincipalKes)
	}

	var resp riderResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RegistrationStatus != registrationStatusLinked {
		t.Errorf("registration_status = %q, want linked", resp.RegistrationStatus)
	}
}

func TestRiderLink_AlreadyLinkedConflict(t *testing.T) {
	store := &mockStore{
		linkRider: func(ctx context.Context, partnerID, riderID string, battery domain.BatteryRegistration, loan domain.LoanRegistration) (domain.RiderSummary, error) {
			return domain.RiderSummary{}, fmt.Errorf("queries: rider already has a loan linked")
		},
	}
	srv := newTestServer(store, mockScorer{}, mockAuth{})

	body := `{"battery":{"external_ref":"b_ref1"},"loan":{"principal_kes":250000,"battery_value_kes":120000}}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/riders/rider_1/link", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "already_linked" {
		t.Errorf("error = %q, want already_linked", resp.Error)
	}
}

func TestRiderLink_NotFound(t *testing.T) {
	store := &mockStore{
		linkRider: func(ctx context.Context, partnerID, riderID string, battery domain.BatteryRegistration, loan domain.LoanRegistration) (domain.RiderSummary, error) {
			return domain.RiderSummary{}, fmt.Errorf("queries: link rider fetch: %w", pgx.ErrNoRows)
		},
	}
	srv := newTestServer(store, mockScorer{}, mockAuth{})

	body := `{"battery":{},"loan":{"principal_kes":250000,"battery_value_kes":120000}}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/riders/unknown/link", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rr.Code, rr.Body.String())
	}
}
