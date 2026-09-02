// Package csvutil parses and validates partner CSV exports into domain
// entities. Ingestion handlers use it and reject a whole batch when any row
// is malformed (returned as a list of RowError).
package csvutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
)

// RowError describes a single malformed row so callers can surface a precise,
// per-row error list instead of silently skipping bad data.
type RowError struct {
	Row   int    `json:"row"`
	Field string `json:"field,omitempty"`
	Error string `json:"error"`
}

// ParseTelemetryReadings parses telemetry CSV. Required columns:
// battery_id, reading_at. Unknown columns are ignored.
func ParseTelemetryReadings(r io.Reader) ([]domain.TelemetryReading, []RowError) {
	header, records, err := readAll(r)
	if err != nil {
		return nil, []RowError{{Error: err.Error()}}
	}

	idx := columnIndex(header)
	if missing := missingColumns(idx, "battery_id", "reading_at"); len(missing) > 0 {
		return nil, []RowError{{Error: fmt.Sprintf("missing required column(s): %s", strings.Join(missing, ", "))}}
	}
	batteryCol := idx["battery_id"]
	readingAtCol := idx["reading_at"]

	var out []domain.TelemetryReading
	var rowErrs []RowError

	for rowNum, rec := range records {
		row := rowNum + 2 // 1-based data row, after header
		if batteryCol >= len(rec) || strings.TrimSpace(rec[batteryCol]) == "" {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "battery_id", Error: "required"})
			continue
		}
		readingAt, err := parseTimeFlexible(rec[readingAtCol])
		if err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "reading_at", Error: "must be a date/RFC3339 timestamp"})
			continue
		}

		r := domain.TelemetryReading{BatteryID: rec[batteryCol], ReadingAt: readingAt}

		if r.StateOfCharge, err = parseFloatPtr(get(rec, idx, "state_of_charge")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "state_of_charge", Error: "must be a number"})
			continue
		}
		if r.Voltage, err = parseFloatPtr(get(rec, idx, "voltage")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "voltage", Error: "must be a number"})
			continue
		}
		if r.TemperatureC, err = parseFloatPtr(get(rec, idx, "temperature_c")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "temperature_c", Error: "must be a number"})
			continue
		}
		if r.CycleCount, err = parseIntPtr(get(rec, idx, "cycle_count")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "cycle_count", Error: "must be an integer"})
			continue
		}
		if r.DepthOfDischarge, err = parseFloatPtr(get(rec, idx, "depth_of_discharge")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "depth_of_discharge", Error: "must be a number"})
			continue
		}
		if r.DistanceKmSinceLast, err = parseFloatPtr(get(rec, idx, "distance_km_since_last")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "distance_km_since_last", Error: "must be a number"})
			continue
		}

		out = append(out, r)
	}

	return out, rowErrs
}

func readAll(r io.Reader) ([]string, [][]string, error) {
	records, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("parse csv: %w", err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("empty csv")
	}
	return records[0], records[1:], nil
}

func columnIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[strings.TrimSpace(strings.ToLower(h))] = i
	}
	return m
}

func missingColumns(idx map[string]int, required ...string) []string {
	var missing []string
	for _, name := range required {
		if _, ok := idx[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing
}

func get(rec []string, idx map[string]int, name string) string {
	if i, ok := idx[name]; ok && i < len(rec) {
		return strings.TrimSpace(rec[i])
	}
	return ""
}

func parseFloatPtr(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseIntPtr(s string) (*int, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseDatePtr(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func parseTimeFlexible(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time %q", s)
}

func validStatus(s string) bool {
	switch s {
	case "on_time", "late", "missed", "partial":
		return true
	default:
		return false
	}
}

// ParseRepaymentEvents parses repayment CSV. Required columns: loan_id,
// due_date. Unknown columns are ignored.
func ParseRepaymentEvents(r io.Reader) ([]domain.RepaymentEvent, []RowError) {
	header, records, err := readAll(r)
	if err != nil {
		return nil, []RowError{{Error: err.Error()}}
	}

	idx := columnIndex(header)
	if missing := missingColumns(idx, "loan_id", "due_date"); len(missing) > 0 {
		return nil, []RowError{{Error: fmt.Sprintf("missing required column(s): %s", strings.Join(missing, ", "))}}
	}
	loanCol := idx["loan_id"]
	dueCol := idx["due_date"]

	var out []domain.RepaymentEvent
	var rowErrs []RowError

	for rowNum, rec := range records {
		row := rowNum + 2
		if loanCol >= len(rec) || strings.TrimSpace(rec[loanCol]) == "" {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "loan_id", Error: "required"})
			continue
		}
		dueDate, err := parseTimeFlexible(rec[dueCol])
		if err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "due_date", Error: "must be a date/RFC3339 timestamp"})
			continue
		}

		e := domain.RepaymentEvent{LoanID: rec[loanCol], DueDate: dueDate}

		if e.PaidDate, err = parseDatePtr(get(rec, idx, "paid_date")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "paid_date", Error: "must be a date"})
			continue
		}
		if e.AmountDueKes, err = parseFloatPtr(get(rec, idx, "amount_due_kes")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "amount_due_kes", Error: "must be a number"})
			continue
		}
		if e.AmountPaidKes, err = parseFloatPtr(get(rec, idx, "amount_paid_kes")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "amount_paid_kes", Error: "must be a number"})
			continue
		}
		if status := get(rec, idx, "status"); status != "" {
			if !validStatus(status) {
				rowErrs = append(rowErrs, RowError{Row: row, Field: "status", Error: "must be one of on_time,late,missed,partial"})
				continue
			}
			e.Status = &status
		}

		out = append(out, e)
	}

	return out, rowErrs
}

// ParseSwapEvents parses swap-event CSV. Required columns: rider_id, battery_id,
// swapped_at. Unknown columns are ignored. rider_id/battery_id are the partner's
// external refs and are resolved to UUIDs by the ingestion handler.
func ParseSwapEvents(r io.Reader) ([]domain.SwapEvent, []RowError) {
	header, records, err := readAll(r)
	if err != nil {
		return nil, []RowError{{Error: err.Error()}}
	}

	idx := columnIndex(header)
	if missing := missingColumns(idx, "rider_id", "battery_id", "swapped_at"); len(missing) > 0 {
		return nil, []RowError{{Error: fmt.Sprintf("missing required column(s): %s", strings.Join(missing, ", "))}}
	}
	riderCol := idx["rider_id"]
	batteryCol := idx["battery_id"]
	swappedCol := idx["swapped_at"]

	var out []domain.SwapEvent
	var rowErrs []RowError

	for rowNum, rec := range records {
		row := rowNum + 2
		if riderCol >= len(rec) || strings.TrimSpace(rec[riderCol]) == "" {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "rider_id", Error: "required"})
			continue
		}
		if batteryCol >= len(rec) || strings.TrimSpace(rec[batteryCol]) == "" {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "battery_id", Error: "required"})
			continue
		}
		swappedAt, err := parseTimeFlexible(rec[swappedCol])
		if err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "swapped_at", Error: "must be a date/RFC3339 timestamp"})
			continue
		}

		e := domain.SwapEvent{RiderID: rec[riderCol], BatteryID: rec[batteryCol], SwappedAt: swappedAt}

		if s := get(rec, idx, "station_id"); s != "" {
			e.StationID = &s
		}
		if e.ReturnedStateOfCharge, err = parseFloatPtr(get(rec, idx, "returned_state_of_charge")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "returned_state_of_charge", Error: "must be a number"})
			continue
		}
		if e.ReturnedTemperatureC, err = parseFloatPtr(get(rec, idx, "returned_temperature_c")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "returned_temperature_c", Error: "must be a number"})
			continue
		}
		if e.ReturnedCycleCount, err = parseIntPtr(get(rec, idx, "returned_cycle_count")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "returned_cycle_count", Error: "must be an integer"})
			continue
		}
		if e.ReturnedDepthOfDischarge, err = parseFloatPtr(get(rec, idx, "returned_depth_of_discharge")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "returned_depth_of_discharge", Error: "must be a number"})
			continue
		}
		if e.DistanceKmSinceLastSwap, err = parseFloatPtr(get(rec, idx, "distance_km_since_last_swap")); err != nil {
			rowErrs = append(rowErrs, RowError{Row: row, Field: "distance_km_since_last_swap", Error: "must be a number"})
			continue
		}

		out = append(out, e)
	}

	return out, rowErrs
}
