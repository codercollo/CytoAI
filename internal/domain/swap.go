package domain

import "time"

// SwapEvent mirrors the `swap_events` table (migration 0011).
//
// In the swap-network model the battery is not collateral; it is the physical
// unit this rider happened to be carrying at swap time, so BatteryID can differ
// swap to swap. The telemetry snapshot is taken AT the swap (what the rider
// handed back), so fleet battery-health and rider-stress signals can both be
// derived from this one event.
type SwapEvent struct {
	ID                       int64     `db:"id" json:"id,omitempty"`
	RiderID                  string    `db:"rider_id" json:"rider_id"`
	BatteryID                string    `db:"battery_id" json:"battery_id"`
	StationID                *string   `db:"station_id" json:"station_id,omitempty"`
	SwappedAt                time.Time `db:"swapped_at" json:"swapped_at"`
	ReturnedStateOfCharge    *float64  `db:"returned_state_of_charge" json:"returned_state_of_charge,omitempty"`
	ReturnedTemperatureC     *float64  `db:"returned_temperature_c" json:"returned_temperature_c,omitempty"`
	ReturnedCycleCount       *int      `db:"returned_cycle_count" json:"returned_cycle_count,omitempty"`
	ReturnedDepthOfDischarge *float64  `db:"returned_depth_of_discharge" json:"returned_depth_of_discharge,omitempty"`
	DistanceKmSinceLastSwap  *float64  `db:"distance_km_since_last_swap" json:"distance_km_since_last_swap,omitempty"`
}
