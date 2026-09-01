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
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRiderCreate_RelationshipsCreatedCorrectly(t *testing.T) {
	var got domain.RiderRegistration
	var gotPartner string
	store := &mockStore{
		createRider: func(ctx context.Context, partnerID string, reg domain.RiderRegistration) (domain.RiderSummary, error) {
			got = reg
			gotPartner = partnerID
			return domain.RiderSummary{
				Rider:   domain.Rider{ID: "r1", ExternalRef: reg.ExternalRef},
				Battery: &domain.Battery{ID: "b1", ExternalRef: reg.Battery.ExternalRef},
				Loan: &domain.Loan{
					ID:              "l1",
					RiderID:         "r1",
					BatteryID:       sptr("b1"),
					ExternalRef:     reg.Loan.ExternalRef,
					PrincipalKes:    reg.Loan.PrincipalKes,
					BatteryValueKes: reg.Loan.BatteryValueKes,
				},
			}, nil
		},
	}
	auth := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		return "partner-9", nil
	}}
	srv := newTestServer(store, mockScorer{}, auth)

	body := `{"external_ref":"rider_9","battery":{"external_ref":"battery_9","manufacturer":"Roam","rated_capacity_wh":2000},"loan":{"external_ref":"loan_9","principal_kes":300000,"battery_value_kes":150000,"term_months":12,"daily_installment_kes":1000,"started_at":"2026-08-31"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/riders", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rr.Code, rr.Body.String())
	}
	if gotPartner != "partner-9" {
		t.Errorf("CreateRider called with partnerID %q, want partner-9", gotPartner)
	}
	if got.Battery == nil || got.Battery.ExternalRef == nil || *got.Battery.ExternalRef != "battery_9" {
		t.Errorf("battery registration = %+v, want battery_9", got.Battery)
	}
	if got.Loan == nil || got.Loan.PrincipalKes == nil || *got.Loan.PrincipalKes != 300000 {
		t.Errorf("loan registration = %+v, want principal 300000", got.Loan)
	}
	if got.Loan.BatteryValueKes == nil || *got.Loan.BatteryValueKes != 150000 {
		t.Errorf("loan battery_value_kes = %v, want 150000", got.Loan.BatteryValueKes)
	}

	var resp riderResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RegistrationStatus != registrationStatusLinked {
		t.Errorf("registration_status = %q, want linked", resp.RegistrationStatus)
	}
	if resp.Battery == nil || resp.Battery.ID != "b1" {
		t.Errorf("battery = %+v, want id b1", resp.Battery)
	}
	if resp.Loan == nil || resp.Loan.ID != "l1" {
		t.Errorf("loan = %+v, want id l1", resp.Loan)
	}
}

func TestRiderCreate_DuplicateExternalRef(t *testing.T) {
	store := &mockStore{
		createRider: func(ctx context.Context, partnerID string, reg domain.RiderRegistration) (domain.RiderSummary, error) {
			return domain.RiderSummary{}, fmt.Errorf("insert rider: %w", &pgconn.PgError{Code: "23505", ConstraintName: "riders_partner_external_ref_unique"})
		},
	}
	srv := newTestServer(store, mockScorer{}, mockAuth{})

	req := httptest.NewRequest(http.MethodPost, "/v1/riders", strings.NewReader(`{"external_ref":"dup"}`))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "duplicate_external_ref" {
		t.Errorf("error = %q, want duplicate_external_ref", body.Error)
	}
}
