// Package domain holds the persistence-facing entity types shared by the
// API, ingestion, and scoring layers. Structs here mirror docs/spec.md §4
// exactly; do not add columns that are not in the SQL schema.
package domain

import "time"

// Battery mirrors the `batteries` table (docs/spec.md §4).
type Battery struct {
	ID              string     `db:"id"`
	ExternalRef     *string    `db:"external_ref"`
	Manufacturer    *string    `db:"manufacturer"`
	RatedCapacityWh *float64   `db:"rated_capacity_wh"`
	CommissionedAt  *time.Time `db:"commissioned_at"`
}

// TelemetryReading mirrors the `telemetry_readings` table (docs/spec.md §4).
type TelemetryReading struct {
	ID                  int64     `db:"id" json:"id,omitempty"`
	BatteryID           string    `db:"battery_id" json:"battery_id"`
	ReadingAt           time.Time `db:"reading_at" json:"reading_at"`
	StateOfCharge       *float64  `db:"state_of_charge" json:"state_of_charge,omitempty"`
	Voltage             *float64  `db:"voltage" json:"voltage,omitempty"`
	TemperatureC        *float64  `db:"temperature_c" json:"temperature_c,omitempty"`
	CycleCount          *int      `db:"cycle_count" json:"cycle_count,omitempty"`
	DepthOfDischarge    *float64  `db:"depth_of_discharge" json:"depth_of_discharge,omitempty"`
	DistanceKmSinceLast *float64  `db:"distance_km_since_last" json:"distance_km_since_last,omitempty"`
}
