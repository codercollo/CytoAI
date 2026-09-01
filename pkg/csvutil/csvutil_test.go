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
