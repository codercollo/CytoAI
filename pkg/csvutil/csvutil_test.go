package csvutil

import (
	"strings"
	"testing"
)

func TestParseTelemetryReadings_Valid(t *testing.T) {
	csv := `battery_id,reading_at,state_of_charge,voltage,temperature_c,cycle_count,depth_of_discharge,distance_km_since_last
b1,2026-01-01,80,52,27,1,60,12.5
b1,2026-01-02,79,52.5,27.5,2,61,13.0
`
	rows, rowErrs := ParseTelemetryReadings(strings.NewReader(csv))
	if len(rowErrs) != 0 {
		t.Fatalf("expected no row errors, got %+v", rowErrs)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].BatteryID != "b1" {
		t.Errorf("BatteryID = %q, want b1", rows[0].BatteryID)
	}
	if rows[0].CycleCount == nil || *rows[0].CycleCount != 1 {
		t.Errorf("CycleCount = %v, want 1", rows[0].CycleCount)
	}
	if rows[0].StateOfCharge == nil || *rows[0].StateOfCharge != 80 {
		t.Errorf("StateOfCharge = %v, want 80", rows[0].StateOfCharge)
	}
	if rows[1].DistanceKmSinceLast == nil || *rows[1].DistanceKmSinceLast != 13.0 {
		t.Errorf("DistanceKmSinceLast = %v, want 13.0", rows[1].DistanceKmSinceLast)
	}
}

func TestParseTelemetryReadings_MissingColumn(t *testing.T) {
	csv := `battery_id,voltage
b1,52
`
	rows, rowErrs := ParseTelemetryReadings(strings.NewReader(csv))
	if len(rowErrs) == 0 {
		t.Fatal("expected a header-level error for missing reading_at")
	}
	if rows != nil {
		t.Errorf("expected no rows when header is invalid, got %d", len(rows))
	}
	if !strings.Contains(rowErrs[0].Error, "reading_at") {
		t.Errorf("error should mention missing column, got %q", rowErrs[0].Error)
	}
}

func TestParseTelemetryReadings_MalformedRowDoesNotDiscardOthers(t *testing.T) {
	csv := `battery_id,reading_at,voltage,cycle_count
b1,2026-01-01,52,1
b2,not-a-date,54,2
b3,2026-01-03,56,3
`
	rows, rowErrs := ParseTelemetryReadings(strings.NewReader(csv))
	if len(rowErrs) != 1 {
		t.Fatalf("expected exactly 1 row error, got %d: %+v", len(rowErrs), rowErrs)
	}
	if rowErrs[0].Row != 3 {
		t.Errorf("error Row = %d, want 3 (second data row)", rowErrs[0].Row)
	}
	if rowErrs[0].Field != "reading_at" {
		t.Errorf("error Field = %q, want reading_at", rowErrs[0].Field)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 valid rows to survive, got %d", len(rows))
	}
	if rows[0].BatteryID != "b1" || rows[1].BatteryID != "b3" {
		t.Errorf("valid rows not preserved: %+v", rows)
	}
}

func TestParseRepaymentEvents_Valid(t *testing.T) {
	csv := `loan_id,due_date,paid_date,amount_due_kes,amount_paid_kes,status
loan_1,2026-01-01,2026-01-01,460,460,on_time
loan_1,2026-01-02,,460,0,missed
`
	rows, rowErrs := ParseRepaymentEvents(strings.NewReader(csv))
	if len(rowErrs) != 0 {
		t.Fatalf("expected no row errors, got %+v", rowErrs)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].PaidDate == nil {
		t.Error("expected paid_date for on_time row")
	}
	if rows[1].PaidDate != nil {
		t.Error("expected nil paid_date for missed row")
	}
	if rows[1].Status == nil || *rows[1].Status != "missed" {
		t.Errorf("Status = %v, want missed", rows[1].Status)
	}
}

func TestParseSwapEvents_Valid(t *testing.T) {
	csv := `rider_id,battery_id,station_id,swapped_at,returned_state_of_charge,returned_temperature_c,returned_cycle_count,returned_depth_of_discharge,distance_km_since_last_swap
r1,b1,ST-01,2026-01-01T08:00:00Z,45,27,120,80,42.5
r1,b2,,2026-01-02,50,28,121,78,40.0
`
	rows, rowErrs := ParseSwapEvents(strings.NewReader(csv))
	if len(rowErrs) != 0 {
		t.Fatalf("expected no row errors, got %+v", rowErrs)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].RiderID != "r1" || rows[0].BatteryID != "b1" {
		t.Errorf("row0 = %q/%q, want r1/b1", rows[0].RiderID, rows[0].BatteryID)
	}
	if rows[0].StationID == nil || *rows[0].StationID != "ST-01" {
		t.Errorf("row0 StationID = %v, want ST-01", rows[0].StationID)
	}
	if rows[1].StationID != nil {
		t.Errorf("row1 StationID = %v, want nil", rows[1].StationID)
	}
	if rows[0].ReturnedCycleCount == nil || *rows[0].ReturnedCycleCount != 120 {
		t.Errorf("row0 ReturnedCycleCount = %v, want 120", rows[0].ReturnedCycleCount)
	}
	if rows[1].DistanceKmSinceLastSwap == nil || *rows[1].DistanceKmSinceLastSwap != 40.0 {
		t.Errorf("row1 DistanceKmSinceLastSwap = %v, want 40.0", rows[1].DistanceKmSinceLastSwap)
	}
}

func TestParseSwapEvents_MissingColumn(t *testing.T) {
	csv := `rider_id,battery_id
r1,b1
`
	rows, rowErrs := ParseSwapEvents(strings.NewReader(csv))
	if len(rowErrs) == 0 {
		t.Fatal("expected a header-level error for missing swapped_at")
	}
	if rows != nil {
		t.Errorf("expected no rows when header is invalid, got %d", len(rows))
	}
	if !strings.Contains(rowErrs[0].Error, "swapped_at") {
		t.Errorf("error should mention missing column, got %q", rowErrs[0].Error)
	}
}
