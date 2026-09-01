package domain

import (
	"math"
	"testing"
	"time"
)

func f64(v float64) *float64       { return &v }
func intp(v int) *int              { return &v }
func strp(v string) *string        { return &v }
func timep(v time.Time) *time.Time { return &v }

func TestBatteryFeaturesFrom(t *testing.T) {
	readings := []TelemetryReading{
		{CycleCount: intp(10), DepthOfDischarge: f64(50), TemperatureC: f64(20), Voltage: f64(50)},
		{CycleCount: intp(20), DepthOfDischarge: f64(60), TemperatureC: f64(30), Voltage: f64(54)},
		{CycleCount: intp(30), DepthOfDischarge: f64(70), TemperatureC: f64(40), Voltage: f64(58)},
	}

	got, err := BatteryFeaturesFrom(Battery{}, readings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.CycleCount != 30 {
		t.Errorf("CycleCount = %v, want 30", got.CycleCount)
	}
	if got.AvgDepthOfDischarge != 60 {
		t.Errorf("AvgDepthOfDischarge = %v, want 60", got.AvgDepthOfDischarge)
	}
	if got.AvgTemperatureC != 30 {
		t.Errorf("AvgTemperatureC = %v, want 30", got.AvgTemperatureC)
	}
	if got.AgeDays != 30 {
		t.Errorf("AgeDays = %v, want 30 (cycle-count proxy)", got.AgeDays)
	}
	// population variance of [50, 54, 58] = 32/3
	wantVar := 32.0 / 3.0
	if math.Abs(got.ChargeRateVariance-wantVar) > 1e-9 {
		t.Errorf("ChargeRateVariance = %v, want %v", got.ChargeRateVariance, wantVar)
	}
}

func TestBatteryFeaturesFrom_Empty(t *testing.T) {
	if _, err := BatteryFeaturesFrom(Battery{}, nil); err == nil {
		t.Fatal("expected error for empty readings, got nil")
	}
}

func TestRepaymentFeaturesFrom(t *testing.T) {
	d := func(day int) time.Time { return time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC) }

	loan := Loan{PrincipalKes: f64(300_000), BatteryValueKes: f64(150_000)}
	events := []RepaymentEvent{
		// Deliberately unsorted to exercise due-date sorting.
		{DueDate: d(4), PaidDate: timep(d(5)), Status: strp("partial")},
		{DueDate: d(1), PaidDate: timep(d(1)), Status: strp("on_time")},
		{DueDate: d(3), PaidDate: nil, Status: strp("missed")},
		{DueDate: d(2), PaidDate: timep(d(4)), Status: strp("late")},
	}

	got, err := RepaymentFeaturesFrom(loan, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.OnTimeRatio != 0.25 {
		t.Errorf("OnTimeRatio = %v, want 0.25", got.OnTimeRatio)
	}
	if got.AvgDaysLate != 0.75 {
		t.Errorf("AvgDaysLate = %v, want 0.75", got.AvgDaysLate)
	}
	// paid dates: Jan 1, Jan 4, Jan 5 -> gaps [3, 1] -> sample std = sqrt(2)
	wantCadence := math.Sqrt2
	if math.Abs(got.PaymentCadenceProxy-wantCadence) > 1e-9 {
		t.Errorf("PaymentCadenceProxy = %v, want %v", got.PaymentCadenceProxy, wantCadence)
	}
	if got.LoanToBatteryValueRatio != 2.0 {
		t.Errorf("LoanToBatteryValueRatio = %v, want 2.0", got.LoanToBatteryValueRatio)
	}
	if got.TenureDays != 3 {
		t.Errorf("TenureDays = %v, want 3", got.TenureDays)
	}
}

func TestRepaymentFeaturesFrom_MissingLoanValues(t *testing.T) {
	events := []RepaymentEvent{{DueDate: time.Now()}}
	if _, err := RepaymentFeaturesFrom(Loan{}, events); err == nil {
		t.Fatal("expected error for missing principal/battery value, got nil")
	}
}
