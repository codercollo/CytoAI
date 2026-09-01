package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/codercollo/cytoai/internal/risk"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	registrationStatusRegistered = "registered"
	registrationStatusLinked     = "linked"
)

type batteryCreateRequest struct {
	ExternalRef     *string  `json:"external_ref,omitempty"`
	Manufacturer    *string  `json:"manufacturer,omitempty"`
	RatedCapacityWh *float64 `json:"rated_capacity_wh,omitempty"`
	CommissionedAt  *string  `json:"commissioned_at,omitempty"`
}

type loanCreateRequest struct {
	ExternalRef         *string  `json:"external_ref,omitempty"`
	PrincipalKes        *float64 `json:"principal_kes,omitempty"`
	BatteryValueKes     *float64 `json:"battery_value_kes,omitempty"`
	TermMonths          *int     `json:"term_months,omitempty"`
	DailyInstallmentKes *float64 `json:"daily_installment_kes,omitempty"`
	StartedAt           *string  `json:"started_at,omitempty"`
}

type riderCreateRequest struct {
	ExternalRef string                `json:"external_ref"`
	Battery     *batteryCreateRequest `json:"battery,omitempty"`
	Loan        *loanCreateRequest    `json:"loan,omitempty"`
}

type batteryInfo struct {
	ID              string     `json:"id"`
	ExternalRef     *string    `json:"external_ref,omitempty"`
	Manufacturer    *string    `json:"manufacturer,omitempty"`
	RatedCapacityWh *float64   `json:"rated_capacity_wh,omitempty"`
	CommissionedAt  *time.Time `json:"commissioned_at,omitempty"`
}

type loanInfo struct {
	ID                  string     `json:"id"`
	ExternalRef         *string    `json:"external_ref,omitempty"`
	PrincipalKes        *float64   `json:"principal_kes,omitempty"`
	BatteryValueKes     *float64   `json:"battery_value_kes,omitempty"`
	TermMonths          *int       `json:"term_months,omitempty"`
	DailyInstallmentKes *float64   `json:"daily_installment_kes,omitempty"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
}

type scoreInfo struct {
	ID                 int64      `json:"id"`
	BatteryID          *string    `json:"battery_id,omitempty"`
	BatteryHealthIndex float64    `json:"battery_health_index"`
	RepaymentRiskIndex float64    `json:"repayment_risk_index"`
	CytoScore          float64    `json:"cyto_score"`
	ScoredAt           *time.Time `json:"scored_at,omitempty"`
	ModelVersion       *string    `json:"model_version,omitempty"`
}

// factorsResponse exposes the model-input features already computed by the
// scoring pipeline (internal/domain/features.go). They are returned verbatim
// from persisted telemetry/repayment data, never fabricated.
type factorsResponse struct {
	Battery   risk.BatteryFeatures   `json:"battery"`
	Repayment risk.RepaymentFeatures `json:"repayment"`
}

type riderResponse struct {
	ID                 string             `json:"id"`
	ExternalRef        *string            `json:"external_ref,omitempty"`
	OnboardedAt        *time.Time         `json:"onboarded_at,omitempty"`
	RegistrationStatus string             `json:"registration_status"`
	Battery            *batteryInfo       `json:"battery,omitempty"`
	Loan               *loanInfo          `json:"loan,omitempty"`
	LatestScore        *scoreInfo         `json:"latest_score"`
	Factors            *factorsResponse   `json:"factors,omitempty"`
	AnomalyFlags       []risk.AnomalyFlag `json:"anomaly_flags"`
}

type riderListResponse struct {
	Riders []riderResponse `json:"riders"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func toRiderResponse(sm domain.RiderSummary) riderResponse {
	out := riderResponse{
		ID:                 sm.Rider.ID,
		ExternalRef:        sm.Rider.ExternalRef,
		OnboardedAt:        sm.Rider.OnboardedAt,
		RegistrationStatus: registrationStatusRegistered,
		AnomalyFlags:       []risk.AnomalyFlag{},
	}
	if sm.Loan != nil {
		out.RegistrationStatus = registrationStatusLinked
	}
	if sm.Battery != nil {
		out.Battery = &batteryInfo{
			ID:              sm.Battery.ID,
			ExternalRef:     sm.Battery.ExternalRef,
			Manufacturer:    sm.Battery.Manufacturer,
			RatedCapacityWh: sm.Battery.RatedCapacityWh,
			CommissionedAt:  sm.Battery.CommissionedAt,
		}
	}
	if sm.Loan != nil {
		out.Loan = &loanInfo{
			ID:                  sm.Loan.ID,
			ExternalRef:         sm.Loan.ExternalRef,
			PrincipalKes:        sm.Loan.PrincipalKes,
			BatteryValueKes:     sm.Loan.BatteryValueKes,
			TermMonths:          sm.Loan.TermMonths,
			DailyInstallmentKes: sm.Loan.DailyInstallmentKes,
			StartedAt:           sm.Loan.StartedAt,
		}
	}
	if sm.Score != nil {
		out.LatestScore = &scoreInfo{
			ID:                 sm.Score.ID,
			BatteryID:          sm.Score.BatteryID,
			BatteryHealthIndex: float64(sm.Score.BatteryHealthIndex),
			RepaymentRiskIndex: float64(sm.Score.RepaymentRiskIndex),
			CytoScore:          float64(sm.Score.CytoScore),
			ScoredAt:           sm.Score.ScoredAt,
			ModelVersion:       sm.Score.ModelVersion,
		}
	}
	return out
}

func (req riderCreateRequest) toRegistration() (domain.RiderRegistration, error) {
	if strings.TrimSpace(req.ExternalRef) == "" {
		return domain.RiderRegistration{}, fmt.Errorf("external_ref is required")
	}
	ref := strings.TrimSpace(req.ExternalRef)
	reg := domain.RiderRegistration{ExternalRef: &ref}

	if (req.Battery == nil) != (req.Loan == nil) {
		return domain.RiderRegistration{}, fmt.Errorf("battery and loan must be provided together")
	}
	if req.Battery == nil {
		return reg, nil
	}

	b, err := req.Battery.toDomain()
	if err != nil {
		return domain.RiderRegistration{}, err
	}
	l, err := req.Loan.toDomain()
	if err != nil {
		return domain.RiderRegistration{}, err
	}
	reg.Battery = &b
	reg.Loan = &l
	return reg, nil
}

func (b batteryCreateRequest) toDomain() (domain.BatteryRegistration, error) {
	out := domain.BatteryRegistration{
		ExternalRef:     b.ExternalRef,
		Manufacturer:    b.Manufacturer,
		RatedCapacityWh: b.RatedCapacityWh,
	}
	if b.CommissionedAt != nil {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*b.CommissionedAt))
		if err != nil {
			return out, fmt.Errorf("battery.commissioned_at must be an RFC3339 timestamp: %w", err)
		}
		out.CommissionedAt = &t
	}
	if b.RatedCapacityWh != nil && *b.RatedCapacityWh <= 0 {
		return out, fmt.Errorf("battery.rated_capacity_wh must be positive")
	}
	return out, nil
}

func (l loanCreateRequest) toDomain() (domain.LoanRegistration, error) {
	out := domain.LoanRegistration{
		ExternalRef:         l.ExternalRef,
		PrincipalKes:        l.PrincipalKes,
		BatteryValueKes:     l.BatteryValueKes,
		TermMonths:          l.TermMonths,
		DailyInstallmentKes: l.DailyInstallmentKes,
	}
	if l.StartedAt != nil {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*l.StartedAt))
		if err != nil {
			return out, fmt.Errorf("loan.started_at must be a YYYY-MM-DD date: %w", err)
		}
		out.StartedAt = &t
	}
	if l.PrincipalKes == nil || *l.PrincipalKes <= 0 {
		return out, fmt.Errorf("loan.principal_kes is required and must be positive")
	}
	if l.BatteryValueKes == nil || *l.BatteryValueKes <= 0 {
		return out, fmt.Errorf("loan.battery_value_kes is required and must be positive")
	}
	if l.TermMonths != nil && *l.TermMonths <= 0 {
		return out, fmt.Errorf("loan.term_months must be positive")
	}
	if l.DailyInstallmentKes != nil && *l.DailyInstallmentKes < 0 {
		return out, fmt.Errorf("loan.daily_installment_kes must be non-negative")
	}
	return out, nil
}

func (s *Server) handleRiderList(w http.ResponseWriter, r *http.Request) {
	partnerID := partnerIDFrom(r.Context())
	limit, offset := pagination(r)
	riders, err := s.cfg.Store.ListRiders(r.Context(), partnerID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list riders", nil)
		return
	}
	out := make([]riderResponse, 0, len(riders))
	for _, rr := range riders {
		out = append(out, toRiderResponse(rr))
	}
	writeJSON(w, http.StatusOK, riderListResponse{Riders: out, Limit: limit, Offset: offset})
}

func (s *Server) handleRiderCreate(w http.ResponseWriter, r *http.Request) {
	var req riderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	reg, err := req.toRegistration()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	summary, err := s.cfg.Store.CreateRider(r.Context(), partnerIDFrom(r.Context()), reg)
	if err != nil {
		if isDuplicateRiderRef(err) {
			writeError(w, http.StatusConflict, "duplicate_external_ref", "a rider with this external_ref already exists", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create rider", nil)
		return
	}
	writeJSON(w, http.StatusCreated, toRiderResponse(summary))
}

func (s *Server) handleRiderGet(w http.ResponseWriter, r *http.Request) {
	riderID := chi.URLParam(r, "rider_id")
	if strings.TrimSpace(riderID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "rider_id is required", nil)
		return
	}
	summary, err := s.cfg.Store.RiderByID(r.Context(), partnerIDFrom(r.Context()), riderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "rider not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load rider", nil)
		return
	}
	resp := toRiderResponse(summary)
	if factors := s.computeFactors(r.Context(), summary); factors != nil {
		resp.Factors = factors
	}
	if summary.Battery != nil {
		resp.AnomalyFlags = s.anomalyFlagsForRiderBattery(r.Context(), summary.Rider.ID, summary.Battery.ID)
	}
	writeJSON(w, http.StatusOK, resp)
}

// computeFactors recomputes the model-input features for the rider's latest
// score from persisted telemetry/repayment/loan data. It returns nil when
// there is no score or the feature inputs are no longer available, so the
// detail page never fabricates factors for a rider that has none.
func (s *Server) computeFactors(ctx context.Context, sm domain.RiderSummary) *factorsResponse {
	if sm.Score == nil || sm.Score.BatteryID == nil {
		return nil
	}
	batteryID := *sm.Score.BatteryID

	loan, err := s.cfg.Store.LoanByRiderAndBattery(ctx, sm.Rider.ID, batteryID)
	if err != nil {
		return nil
	}
	telemetry, err := s.cfg.Store.TelemetryReadingsByBattery(ctx, batteryID)
	if err != nil {
		return nil
	}
	events, err := s.cfg.Store.RepaymentEventsByLoan(ctx, loan.ID)
	if err != nil {
		return nil
	}

	bf, err := domain.BatteryFeaturesFrom(domain.Battery{ID: batteryID}, telemetry)
	if err != nil {
		return nil
	}
	rf, err := domain.RepaymentFeaturesFrom(loan, events)
	if err != nil {
		return nil
	}
	rf.TelemetryCadenceProxy = domain.TelemetryCadenceProxyFrom(telemetry)
	return &factorsResponse{Battery: bf, Repayment: rf}
}

// isDuplicateRiderRef reports whether err is the unique-violation raised by
// the riders_partner_external_ref_unique index (see migration 0009).
func isDuplicateRiderRef(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "riders_partner_external_ref_unique"
}
