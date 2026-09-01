package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/codercollo/cytoai/pkg/csvutil"
)

func (s *Server) handleIngestTelematics(w http.ResponseWriter, r *http.Request) {
	rows, rowErrs, err := decodeTelematics(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	if len(rowErrs) > 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed rows", map[string]any{"row_errors": rowErrs})
		return
	}

	partnerID := partnerIDFrom(r.Context())
	rows, refErrs := s.resolveBatteryRefs(r.Context(), partnerID, rows)
	if len(refErrs) > 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "unresolved external refs", map[string]any{"row_errors": refErrs})
		return
	}

	n, err := s.cfg.Store.InsertTelemetryReadings(r.Context(), rows)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to insert telemetry", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"inserted": n})
}

func (s *Server) handleIngestRepayments(w http.ResponseWriter, r *http.Request) {
	rows, rowErrs, err := decodeRepayments(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	if len(rowErrs) > 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed rows", map[string]any{"row_errors": rowErrs})
		return
	}

	partnerID := partnerIDFrom(r.Context())
	rows, refErrs := s.resolveLoanRefs(r.Context(), partnerID, rows)
	if len(refErrs) > 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "unresolved external refs", map[string]any{"row_errors": refErrs})
		return
	}

	n, err := s.cfg.Store.InsertRepaymentEvents(r.Context(), rows)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to insert repayment events", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"inserted": n})
}

func decodeTelematics(r *http.Request) ([]domain.TelemetryReading, []csvutil.RowError, error) {
	ct := r.Header.Get("Content-Type")
	if isJSON(ct) {
		var rows []domain.TelemetryReading
		if err := json.NewDecoder(r.Body).Decode(&rows); err != nil {
			return nil, nil, fmt.Errorf("invalid JSON body: %w", err)
		}
		if len(rows) == 0 {
			return nil, nil, fmt.Errorf("empty batch")
		}
		return rows, nil, nil
	}
	if isCSV(ct) {
		rows, rowErrs := csvutil.ParseTelemetryReadings(r.Body)
		return rows, rowErrs, nil
	}
	return nil, nil, fmt.Errorf("unsupported Content-Type %q (use application/json or text/csv)", ct)
}

func decodeRepayments(r *http.Request) ([]domain.RepaymentEvent, []csvutil.RowError, error) {
	ct := r.Header.Get("Content-Type")
	if isJSON(ct) {
		var rows []domain.RepaymentEvent
		if err := json.NewDecoder(r.Body).Decode(&rows); err != nil {
			return nil, nil, fmt.Errorf("invalid JSON body: %w", err)
		}
		if len(rows) == 0 {
			return nil, nil, fmt.Errorf("empty batch")
		}
		return rows, nil, nil
	}
	if isCSV(ct) {
		rows, rowErrs := csvutil.ParseRepaymentEvents(r.Body)
		return rows, rowErrs, nil
	}
	return nil, nil, fmt.Errorf("unsupported Content-Type %q (use application/json or text/csv)", ct)
}

func isJSON(ct string) bool { return strings.Contains(ct, "application/json") }

func isCSV(ct string) bool {
	return strings.Contains(ct, "text/csv") || strings.Contains(ct, "application/csv")
}

type refError struct {
	Field string `json:"field"`
	Value string `json:"value"`
	Error string `json:"error"`
}

func (s *Server) resolveBatteryRefs(ctx context.Context, partnerID string, rows []domain.TelemetryReading) ([]domain.TelemetryReading, []refError) {
	cache := map[string]string{}
	out := append([]domain.TelemetryReading(nil), rows...)
	var errs []refError
	for i := range out {
		ref := out[i].BatteryID
		id, ok := cache[ref]
		if !ok {
			resolved, err := s.cfg.Store.ResolveBatteryID(ctx, partnerID, ref)
			if err != nil {
				errs = append(errs, refError{Field: "battery_id", Value: ref, Error: "unknown external ref"})
				continue
			}
			cache[ref] = resolved
			id = resolved
		}
		out[i].BatteryID = id
	}
	return out, errs
}

func (s *Server) resolveLoanRefs(ctx context.Context, partnerID string, rows []domain.RepaymentEvent) ([]domain.RepaymentEvent, []refError) {
	cache := map[string]string{}
	out := append([]domain.RepaymentEvent(nil), rows...)
	var errs []refError
	for i := range out {
		ref := out[i].LoanID
		id, ok := cache[ref]
		if !ok {
			resolved, err := s.cfg.Store.ResolveLoanID(ctx, partnerID, ref)
			if err != nil {
				errs = append(errs, refError{Field: "loan_id", Value: ref, Error: "unknown external ref"})
				continue
			}
			cache[ref] = resolved
			id = resolved
		}
		out[i].LoanID = id
	}
	return out, errs
}
