package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
)

// TelemetryReadingsByBattery returns a battery's telemetry readings in
// chronological order.
func (s *Store) TelemetryReadingsByBattery(ctx context.Context, batteryID string) ([]domain.TelemetryReading, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, battery_id::text, reading_at, state_of_charge, voltage, temperature_c, cycle_count, depth_of_discharge, distance_km_since_last
		FROM telemetry_readings
		WHERE battery_id = $1
		ORDER BY reading_at ASC`, batteryID)
	if err != nil {
		return nil, fmt.Errorf("queries: telemetry by battery: %w", err)
	}
	defer rows.Close()

	var out []domain.TelemetryReading
	for rows.Next() {
		var (
			r                    domain.TelemetryReading
			soc, volt, temp, dod *float64
			dist                 *float64
			cycles               *int
		)
		if err := rows.Scan(&r.ID, &r.BatteryID, &r.ReadingAt, &soc, &volt, &temp, &cycles, &dod, &dist); err != nil {
			return nil, fmt.Errorf("queries: scan telemetry: %w", err)
		}
		r.StateOfCharge = soc
		r.Voltage = volt
		r.TemperatureC = temp
		r.CycleCount = cycles
		r.DepthOfDischarge = dod
		r.DistanceKmSinceLast = dist
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queries: iterate telemetry: %w", err)
	}
	return out, nil
}

// RepaymentEventsByLoan returns a loan's repayment events in due-date order.
func (s *Store) RepaymentEventsByLoan(ctx context.Context, loanID string) ([]domain.RepaymentEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, loan_id::text, due_date, paid_date, amount_due_kes, amount_paid_kes, status
		FROM repayment_events
		WHERE loan_id = $1
		ORDER BY due_date ASC`, loanID)
	if err != nil {
		return nil, fmt.Errorf("queries: repayment events by loan: %w", err)
	}
	defer rows.Close()

	var out []domain.RepaymentEvent
	for rows.Next() {
		var (
			e               domain.RepaymentEvent
			paid            *time.Time
			dueKes, paidKes *float64
			status          *string
		)
		if err := rows.Scan(&e.ID, &e.LoanID, &e.DueDate, &paid, &dueKes, &paidKes, &status); err != nil {
			return nil, fmt.Errorf("queries: scan repayment event: %w", err)
		}
		e.PaidDate = paid
		e.AmountDueKes = dueKes
		e.AmountPaidKes = paidKes
		e.Status = status
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queries: iterate repayment events: %w", err)
	}
	return out, nil
}

// LoanByRiderAndBattery returns the most recent loan for a rider/battery pair.
func (s *Store) LoanByRiderAndBattery(ctx context.Context, riderID, batteryID string) (domain.Loan, error) {
	var (
		l                       domain.Loan
		batteryOut              *string
		principal, batteryValue *float64
		daily                   *float64
		term                    *int
		started                 *time.Time
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, rider_id::text, battery_id::text, principal_kes, battery_value_kes, term_months, daily_installment_kes, started_at
		FROM loans
		WHERE rider_id = $1 AND battery_id = $2
		ORDER BY started_at DESC NULLS LAST
		LIMIT 1`, riderID, batteryID,
	).Scan(&l.ID, &l.RiderID, &batteryOut, &principal, &batteryValue, &term, &daily, &started)
	if err != nil {
		return domain.Loan{}, fmt.Errorf("queries: loan by rider and battery: %w", err)
	}
	l.BatteryID = batteryOut
	l.PrincipalKes = principal
	l.BatteryValueKes = batteryValue
	l.TermMonths = term
	l.DailyInstallmentKes = daily
	l.StartedAt = started
	return l, nil
}
