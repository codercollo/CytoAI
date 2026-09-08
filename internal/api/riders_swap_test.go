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

func TestRiderCreate_SwapNetwork(t *testing.T) {
	var gotReg domain.RiderRegistration
	var gotPartner string

	store := &mockStore{
		createRider: func(ctx context.Context, partnerID string, reg domain.RiderRegistration) (domain.RiderSummary, error) {
			gotPartner = partnerID
			gotReg = reg
			swapModel := "swap_network"
			return domain.RiderSummary{
				Rider: domain.Rider{ID: "r-swap-1", ExternalRef: reg.ExternalRef},
				Loan: &domain.Loan{
					ID:             "l-swap-1",
					RiderID:        "r-swap-1",
					ExternalRef:    reg.Loan.ExternalRef,
					PrincipalKes:   reg.Loan.PrincipalKes,
					FinancingModel: &swapModel,
				},
			}, nil
		},
	}

	auth := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		return "partner-swap", nil
	}}
	srv := newTestServer(store, mockScorer{}, auth)

	body := `{"external_ref":"rider_n1","loan":{"external_ref":"loan_n1","principal_kes":300000,"financing_model":"swap_network"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/riders", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rr.Code, rr.Body.String())
	}
	if gotPartner != "partner-swap" {
		t.Errorf("partner = %q, want partner-swap", gotPartner)
	}
	if gotReg.Battery != nil {
		t.Errorf("expected no battery for swap_network, got %+v", gotReg.Battery)
	}
	if gotReg.Loan == nil || gotReg.Loan.FinancingModel == nil || *gotReg.Loan.FinancingModel != "swap_network" {
		t.Errorf("loan financing_model = %v, want swap_network", gotReg.Loan)
	}

	var resp riderResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RegistrationStatus != registrationStatusLinked {
		t.Errorf("registration_status = %q, want linked", resp.RegistrationStatus)
	}
	if resp.Loan == nil || resp.Loan.FinancingModel == nil || *resp.Loan.FinancingModel != "swap_network" {
		t.Errorf("loan financing_model = %v, want swap_network", resp.Loan)
	}
}

func TestBatteryCreate_PoolBattery(t *testing.T) {
	var gotBattery domain.BatteryRegistration

	store := &mockStore{
		createBattery: func(ctx context.Context, battery domain.BatteryRegistration) (domain.Battery, error) {
			gotBattery = battery
			return domain.Battery{
				ID:           "b-pool-1",
				ExternalRef:  battery.ExternalRef,
				Manufacturer: battery.Manufacturer,
			}, nil
		},
	}
	auth := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		return "partner-swap", nil
	}}
	srv := newTestServer(store, mockScorer{}, auth)

	body := `{"external_ref":"pool_1","manufacturer":"Spiro"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/batteries", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rr.Code, rr.Body.String())
	}
	if gotBattery.ExternalRef == nil || *gotBattery.ExternalRef != "pool_1" {
		t.Errorf("external_ref = %v, want pool_1", gotBattery.ExternalRef)
	}
}
