package queries

import (
	"context"
	"fmt"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/jackc/pgx/v5"
)

func scanSwapRows(rows pgx.Rows) ([]domain.SwapEvent, error) {
	defer rows.Close()

	var out []domain.SwapEvent
	for rows.Next() {
		var (
			e                    domain.SwapEvent
			stationID            *string
			soc, temp, dod, dist *float64
			cycles               *int
		)
		if err := rows.Scan(&e.ID, &e.RiderID, &e.BatteryID, &stationID, &e.SwappedAt, &soc, &temp, &cycles, &dod, &dist); err != nil {
			return nil, fmt.Errorf("queries: scan swap event: %w", err)
		}
		e.StationID = stationID
		e.ReturnedStateOfCharge = soc
		e.ReturnedTemperatureC = temp
		e.ReturnedCycleCount = cycles
		e.ReturnedDepthOfDischarge = dod
		e.DistanceKmSinceLastSwap = dist
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queries: iterate swap events: %w", err)
	}
	return out, nil
}

// SwapEventsByRider returns a rider's swap events in chronological order.
func (s *Store) SwapEventsByRider(ctx context.Context, riderID string) ([]domain.SwapEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, rider_id::text, battery_id::text, station_id, swapped_at, returned_state_of_charge, returned_temperature_c, returned_cycle_count, returned_depth_of_discharge, distance_km_since_last_swap
		FROM swap_events
		WHERE rider_id = $1
		ORDER BY swapped_at ASC`, riderID)
	if err != nil {
		return nil, fmt.Errorf("queries: swap events by rider: %w", err)
	}
	return scanSwapRows(rows)
}

// SwapEventsByPartner returns the partner's full fleet of swap events in
// chronological order (the fleet this partner's riders draw from).
func (s *Store) SwapEventsByPartner(ctx context.Context, partnerID string) ([]domain.SwapEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT se.id, se.rider_id::text, se.battery_id::text, se.station_id, se.swapped_at, se.returned_state_of_charge, se.returned_temperature_c, se.returned_cycle_count, se.returned_depth_of_discharge, se.distance_km_since_last_swap
		FROM swap_events se
		JOIN riders r ON r.id = se.rider_id
		WHERE r.partner_id = $1
		ORDER BY se.swapped_at ASC`, partnerID)
	if err != nil {
		return nil, fmt.Errorf("queries: swap events by partner: %w", err)
	}
	return scanSwapRows(rows)
}
