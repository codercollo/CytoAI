package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/codercollo/cytoai/internal/risk"
	"github.com/go-chi/chi/v5"
)

type scoreRequest struct {
	RiderID   string `json:"rider_id"`
	BatteryID string `json:"battery_id"`
}

type scoreResponse struct {
	RiderID               string             `json:"rider_id"`
	BatteryID             string             `json:"battery_id"`
	BatteryHealthIndex    float64            `json:"battery_health_index"`
	RepaymentRiskIndex    float64            `json:"repayment_risk_index"`
	CytoScore             float64            `json:"cyto_score"`
	BatteryModelVersion   string             `json:"battery_model_version"`
	RepaymentModelVersion string             `json:"repayment_model_version"`
	ScoredAt              string             `json:"scored_at,omitempty"`
	AnomalyFlags          []risk.AnomalyFlag `json:"anomaly_flags"`
}

// storedScoreResponse is the GET /v1/score/{rider_id} response shape. It keeps
// the existing persisted-score fields and adds anomaly_flags.
type storedScoreResponse struct {
	ID                 int64              `json:"id"`
	RiderID            *string            `json:"rider_id,omitempty"`
	BatteryID          *string            `json:"battery_id,omitempty"`
	BatteryHealthIndex float64            `json:"battery_health_index"`
	RepaymentRiskIndex float64            `json:"repayment_risk_index"`
	CytoScore          float64            `json:"cyto_score"`
	ScoredAt           *time.Time         `json:"scored_at,omitempty"`
	ModelVersion       *string            `json:"model_version,omitempty"`
	AnomalyFlags       []risk.AnomalyFlag `json:"anomaly_flags"`
}

type insufficientDataResponse struct {
	InsufficientData bool   `json:"insufficient_data"`
	Reason           string `json:"reason"`
	RiderID          string `json:"rider_id"`
	BatteryID        string `json:"battery_id"`
}

var errNoLoan = errors.New("no loan found for rider/battery")

// insufficientError marks a pair that cannot be scored yet (a skip, not a
// failure). The scorer's risk.ErrInsufficientData is converted into one.
type insufficientError struct{ reason string }

func (e *insufficientError) Error() string { return e.reason }

func isInsufficient(err error) bool {
	var ie *insufficientError
	return errors.As(err, &ie)
}

// detectAnomalies runs the (non-gating) anomaly rules on already-loaded raw
// records. It is best-effort: a sidecar failure returns an empty slice so a
// score is never blocked by anomaly detection (spec.md §9).
func (s *Server) detectAnomalies(ctx context.Context, telemetry []domain.TelemetryReading, events []domain.RepaymentEvent) []risk.AnomalyFlag {
	if s.cfg.AnomalyDetector == nil {
		return []risk.AnomalyFlag{}
	}

	telemetryRecords := make([]risk.TelemetryAnomalyRecord, 0, len(telemetry))
	for _, t := range telemetry {
		telemetryRecords = append(telemetryRecords, risk.TelemetryAnomalyRecord{
			ID:                  t.ID,
			BatteryID:           t.BatteryID,
			ReadingAt:           t.ReadingAt.Format(time.RFC3339),
			StateOfCharge:       t.StateOfCharge,
			Voltage:             t.Voltage,
			TemperatureC:        t.TemperatureC,
			CycleCount:          t.CycleCount,
			DepthOfDischarge:    t.DepthOfDischarge,
			DistanceKmSinceLast: t.DistanceKmSinceLast,
		})
	}

	repaymentRecords := make([]risk.RepaymentAnomalyRecord, 0, len(events))
	for _, e := range events {
		var paidDate *string
		if e.PaidDate != nil {
			v := e.PaidDate.Format(time.RFC3339)
			paidDate = &v
		}
		repaymentRecords = append(repaymentRecords, risk.RepaymentAnomalyRecord{
			ID:            e.ID,
			LoanID:        e.LoanID,
			DueDate:       e.DueDate.Format(time.RFC3339),
			PaidDate:      paidDate,
			AmountDueKes:  e.AmountDueKes,
			AmountPaidKes: e.AmountPaidKes,
			Status:        e.Status,
		})
	}

	var flags []risk.AnomalyFlag
	if len(telemetryRecords) > 0 {
		if tf, err := s.cfg.AnomalyDetector.DetectTelemetryAnomalies(ctx, telemetryRecords); err != nil {
			s.logger.Warn("telemetry anomaly detection failed", "error", err)
		} else {
			flags = append(flags, tf...)
		}
	}
	if len(repaymentRecords) > 0 {
		if rf, err := s.cfg.AnomalyDetector.DetectRepaymentAnomalies(ctx, repaymentRecords); err != nil {
			s.logger.Warn("repayment anomaly detection failed", "error", err)
		} else {
			flags = append(flags, rf...)
		}
	}
	if flags == nil {
		flags = []risk.AnomalyFlag{}
	}
	return flags
}

// anomalyFlagsForRiderBattery loads the raw telemetry + repayment records for a
// rider/battery pair and runs the (best-effort) anomaly rules. It returns an
// empty slice when the pair is missing or any load fails, so callers can always
// render an empty flags array.
func (s *Server) anomalyFlagsForRiderBattery(ctx context.Context, riderID, batteryID string) []risk.AnomalyFlag {
	loan, err := s.cfg.Store.LoanByRiderAndBattery(ctx, riderID, batteryID)
	if err != nil {
		return []risk.AnomalyFlag{}
	}
	telemetry, err := s.cfg.Store.TelemetryReadingsByBattery(ctx, batteryID)
	if err != nil {
		return []risk.AnomalyFlag{}
	}
	events, err := s.cfg.Store.RepaymentEventsByLoan(ctx, loan.ID)
	if err != nil {
		return []risk.AnomalyFlag{}
	}
	return s.detectAnomalies(ctx, telemetry, events)
}

func (s *Server) handleScoreCompute(w http.ResponseWriter, r *http.Request) {
	var req scoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	if req.RiderID == "" || req.BatteryID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "rider_id and battery_id are required", nil)
		return
	}

	ctx := r.Context()
	owned, err := s.cfg.Store.RiderBelongsToPartner(ctx, partnerIDFrom(ctx), req.RiderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to verify rider ownership", nil)
		return
	}
	if !owned {
		writeError(w, http.StatusNotFound, "not_found", "no loan found for rider/battery", nil)
		return
	}

	resp, err := s.scoreRiderBattery(ctx, req.RiderID, req.BatteryID)
	if err != nil {
		switch {
		case errors.Is(err, errNoLoan):
			writeError(w, http.StatusNotFound, "not_found", "no loan found for rider/battery", nil)
		case isInsufficient(err):
			writeInsufficientData(w, req, err)
		default:
			writeError(w, http.StatusInternalServerError, "scoring_failed", err.Error(), nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// scoreRiderBattery runs the full scoring flow for one rider/battery pair:
// load telemetry + repayment events, compute features, call the scorer, and
// persist the result. Ownership is the caller's responsibility.
func (s *Server) scoreRiderBattery(ctx context.Context, riderID, batteryID string) (*scoreResponse, error) {
	loan, err := s.cfg.Store.LoanByRiderAndBattery(ctx, riderID, batteryID)
	if err != nil {
		return nil, errNoLoan
	}
	telemetry, err := s.cfg.Store.TelemetryReadingsByBattery(ctx, batteryID)
	if err != nil {
		return nil, fmt.Errorf("load telemetry: %w", err)
	}
	events, err := s.cfg.Store.RepaymentEventsByLoan(ctx, loan.ID)
	if err != nil {
		return nil, fmt.Errorf("load repayment events: %w", err)
	}

	batteryFeatures, err := domain.BatteryFeaturesFrom(domain.Battery{ID: batteryID}, telemetry)
	if err != nil {
		return nil, &insufficientError{reason: err.Error()}
	}
	repaymentFeatures, err := domain.RepaymentFeaturesFrom(loan, events)
	if err != nil {
		return nil, &insufficientError{reason: err.Error()}
	}
	repaymentFeatures.TelemetryCadenceProxy = domain.TelemetryCadenceProxyFrom(telemetry)

	result, err := s.cfg.Scorer.Score(ctx, batteryFeatures, repaymentFeatures)
	if err != nil {
		if errors.Is(err, risk.ErrInsufficientData) {
			return nil, &insufficientError{reason: err.Error()}
		}
		return nil, fmt.Errorf("scoring: %w", err)
	}

	anomalyFlags := s.detectAnomalies(ctx, telemetry, events)
	result.AnomalyFlags = anomalyFlags

	now := time.Now().UTC()
	score := domain.FromRiskResult(*result)
	score.RiderID = &riderID
	score.BatteryID = &batteryID
	score.ScoredAt = &now
	if _, err := s.cfg.Store.InsertScore(ctx, score); err != nil {
		return nil, fmt.Errorf("persist score: %w", err)
	}

	return &scoreResponse{
		RiderID:               riderID,
		BatteryID:             batteryID,
		BatteryHealthIndex:    float64(result.BatteryHealthIndex),
		RepaymentRiskIndex:    float64(result.RepaymentRiskIndex),
		CytoScore:             float64(result.CytoScore),
		BatteryModelVersion:   result.BatteryModelVersion,
		RepaymentModelVersion: result.RepaymentModelVersion,
		ScoredAt:              now.Format(time.RFC3339),
		AnomalyFlags:          anomalyFlags,
	}, nil
}

func writeInsufficientData(w http.ResponseWriter, req scoreRequest, err error) {
	writeJSON(w, http.StatusOK, insufficientDataResponse{
		InsufficientData: true,
		Reason:           err.Error(),
		RiderID:          req.RiderID,
		BatteryID:        req.BatteryID,
	})
}

// rescoreSkipped reports a rider/battery pair that could not be scored yet.
type rescoreSkipped struct {
	RiderID   string `json:"rider_id"`
	BatteryID string `json:"battery_id"`
	Reason    string `json:"reason"`
}

// rescoreResponse is the summary returned by POST /v1/score/rescore-all.
type rescoreResponse struct {
	Scored  int              `json:"scored"`
	Skipped []rescoreSkipped `json:"skipped"`
}

// handleRescoreAll re-scores every rider/battery pair owned by the
// authenticated partner. Insufficient-data pairs are skipped, not fatal.
func (s *Server) handleRescoreAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	partnerID := partnerIDFrom(ctx)

	pairs, err := s.cfg.Store.RiderBatteryPairsForPartner(ctx, partnerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list rider/battery pairs", nil)
		return
	}

	var skipped []rescoreSkipped
	scored := 0
	for _, p := range pairs {
		if _, err := s.scoreRiderBattery(ctx, p.RiderID, p.BatteryID); err != nil {
			if isInsufficient(err) {
				skipped = append(skipped, rescoreSkipped{RiderID: p.RiderID, BatteryID: p.BatteryID, Reason: err.Error()})
				continue
			}
			writeError(w, http.StatusInternalServerError, "scoring_failed", err.Error(), nil)
			return
		}
		scored++
	}

	writeJSON(w, http.StatusOK, rescoreResponse{Scored: scored, Skipped: skipped})
}

func (s *Server) handleScoreGet(w http.ResponseWriter, r *http.Request) {
	riderID := chi.URLParam(r, "rider_id")
	owned, err := s.cfg.Store.RiderBelongsToPartner(r.Context(), partnerIDFrom(r.Context()), riderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to verify rider ownership", nil)
		return
	}
	if !owned {
		writeError(w, http.StatusNotFound, "not_found", "no score found for rider", nil)
		return
	}
	score, err := s.cfg.Store.LatestScore(r.Context(), riderID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "no score found for rider", nil)
		return
	}

	flags := []risk.AnomalyFlag{}
	if score.BatteryID != nil {
		flags = s.anomalyFlagsForRiderBattery(r.Context(), riderID, *score.BatteryID)
	}

	writeJSON(w, http.StatusOK, storedScoreResponse{
		ID:                 score.ID,
		RiderID:            score.RiderID,
		BatteryID:          score.BatteryID,
		BatteryHealthIndex: float64(score.BatteryHealthIndex),
		RepaymentRiskIndex: float64(score.RepaymentRiskIndex),
		CytoScore:          float64(score.CytoScore),
		ScoredAt:           score.ScoredAt,
		ModelVersion:       score.ModelVersion,
		AnomalyFlags:       flags,
	})
}

func (s *Server) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	partnerID := partnerIDFrom(r.Context())
	limit, offset := pagination(r)
	scores, err := s.cfg.Store.PortfolioScores(r.Context(), partnerID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load portfolio", nil)
		return
	}

	// Attach anomaly flags per rider/battery. This is per-row work (the MVP
	// portfolio is small), and it keeps the response shape identical to
	// GET /v1/score/{rider_id}.
	out := make([]storedScoreResponse, 0, len(scores))
	for _, sc := range scores {
		flags := []risk.AnomalyFlag{}
		if sc.RiderID != nil && sc.BatteryID != nil {
			flags = s.anomalyFlagsForRiderBattery(r.Context(), *sc.RiderID, *sc.BatteryID)
		}
		out = append(out, storedScoreResponse{
			ID:                 sc.ID,
			RiderID:            sc.RiderID,
			BatteryID:          sc.BatteryID,
			BatteryHealthIndex: float64(sc.BatteryHealthIndex),
			RepaymentRiskIndex: float64(sc.RepaymentRiskIndex),
			CytoScore:          float64(sc.CytoScore),
			ScoredAt:           sc.ScoredAt,
			ModelVersion:       sc.ModelVersion,
			AnomalyFlags:       flags,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"scores": out, "limit": limit, "offset": offset})
}

func pagination(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	if limit == 0 {
		limit = 100
	}
	return limit, offset
}
