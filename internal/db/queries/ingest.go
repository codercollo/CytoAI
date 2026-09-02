package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/jackc/pgx/v5"
)

// InsertTelemetryReadings batch-inserts telemetry readings using COPY for
// efficient CSV ingestion. It returns the number of rows inserted.
func (s *Store) InsertTelemetryReadings(ctx context.Context, readings []domain.TelemetryReading) (int64, error) {
	if len(readings) == 0 {
		return 0, nil
	}

	rows := make([][]any, len(readings))
	for i, r := range readings {
		rows[i] = []any{
			r.BatteryID,
			r.ReadingAt,
			nullableFloat(r.StateOfCharge),
			nullableFloat(r.Voltage),
			nullableFloat(r.TemperatureC),
			nullableInt(r.CycleCount),
			nullableFloat(r.DepthOfDischarge),
			nullableFloat(r.DistanceKmSinceLast),
		}
	}

	n, err := s.pool.CopyFrom(ctx,
		pgx.Identifier{"telemetry_readings"},
		[]string{"battery_id", "reading_at", "state_of_charge", "voltage", "temperature_c", "cycle_count", "depth_of_discharge", "distance_km_since_last"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return 0, fmt.Errorf("queries: insert telemetry readings: %w", err)
	}
	return n, nil
}

// InsertRepaymentEvents batch-inserts repayment events using COPY. It returns
// the number of rows inserted.
func (s *Store) InsertRepaymentEvents(ctx context.Context, events []domain.RepaymentEvent) (int64, error) {
	if len(events) == 0 {
		return 0, nil
	}

	rows := make([][]any, len(events))
	for i, e := range events {
		rows[i] = []any{
			e.LoanID,
			e.DueDate,
			nullableTime(e.PaidDate),
			nullableFloat(e.AmountDueKes),
			nullableFloat(e.AmountPaidKes),
			nullableString(e.Status),
		}
	}

	n, err := s.pool.CopyFrom(ctx,
		pgx.Identifier{"repayment_events"},
		[]string{"loan_id", "due_date", "paid_date", "amount_due_kes", "amount_paid_kes", "status"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return 0, fmt.Errorf("queries: insert repayment events: %w", err)
	}
	return n, nil
}

// InsertSwapEvents batch-inserts swap events using COPY. It returns the number
// of rows inserted.
func (s *Store) InsertSwapEvents(ctx context.Context, events []domain.SwapEvent) (int64, error) {
	if len(events) == 0 {
		return 0, nil
	}

	rows := make([][]any, len(events))
	for i, e := range events {
		rows[i] = []any{
			e.RiderID,
			e.BatteryID,
			nullableString(e.StationID),
			e.SwappedAt,
			nullableFloat(e.ReturnedStateOfCharge),
			nullableFloat(e.ReturnedTemperatureC),
			nullableInt(e.ReturnedCycleCount),
			nullableFloat(e.ReturnedDepthOfDischarge),
			nullableFloat(e.DistanceKmSinceLastSwap),
		}
	}

	n, err := s.pool.CopyFrom(ctx,
		pgx.Identifier{"swap_events"},
		[]string{"rider_id", "battery_id", "station_id", "swapped_at", "returned_state_of_charge", "returned_temperature_c", "returned_cycle_count", "returned_depth_of_discharge", "distance_km_since_last_swap"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return 0, fmt.Errorf("queries: insert swap events: %w", err)
	}
	return n, nil
}

func nullableString(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullableFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullableInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullableTime(p *time.Time) any {
	if p == nil {
		return nil
	}
	return *p
}
