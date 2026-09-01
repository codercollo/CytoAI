package domain

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/codercollo/cytoai/internal/risk"
)

// Decision: the feature-mapping helpers live in internal/domain (not
// internal/risk) because they translate persistence entities into risk's
// inference DTOs. The dependency direction is domain -> risk only, so risk
// stays transport/inference-only and there is no import cycle.

// BatteryFeaturesFrom aggregates a battery's telemetry history into the five
// features the ML sidecar expects (risk.BatteryFeatures).
//
// Feature definitions mirror ml/training/train_battery_model.py:
//   - CycleCount:           latest cumulative cycle_count across readings
//   - AvgDepthOfDischarge:  mean of depth_of_discharge
//   - AvgTemperatureC:      mean of temperature_c
//   - AgeDays:              cycle_count (the training script uses a "~1
//     cycle/day" proxy because NASA data has no
//     commissioning date). Battery.CommissionedAt is the
//     domain source of truth for calendar age, but it
//     must be adopted together with a BHI retrain, not
//     unilaterally here.
//   - ChargeRateVariance:   population variance of voltage (matches the .mat
//     path's np.nanvar(volt_series))
//
// NOTE: the current BHI artifact was trained on NASA discharge cycles where
// avg_depth_of_discharge was derived from the voltage range (a 0-1 ratio),
// while the telemetry schema carries an explicit depth_of_discharge
// percentage column. This helper uses the explicit telemetry column (the
// training script's CSV-fallback definition); reconcile the .mat path before
// trusting live BHI scoring.
func BatteryFeaturesFrom(b Battery, readings []TelemetryReading) (risk.BatteryFeatures, error) {
	if len(readings) == 0 {
		return risk.BatteryFeatures{}, fmt.Errorf("battery features: no telemetry readings")
	}

	var (
		cycleCount float64
		dodSum     float64
		dodN       int
		tempSum    float64
		tempN      int
		voltages   []float64
	)

	for _, r := range readings {
		if r.CycleCount != nil {
			if c := float64(*r.CycleCount); c > cycleCount {
				cycleCount = c
			}
		}
		if r.DepthOfDischarge != nil {
			dodSum += *r.DepthOfDischarge
			dodN++
		}
		if r.TemperatureC != nil {
			tempSum += *r.TemperatureC
			tempN++
		}
		if r.Voltage != nil {
			voltages = append(voltages, *r.Voltage)
		}
	}

	if dodN == 0 {
		return risk.BatteryFeatures{}, fmt.Errorf("battery features: no depth_of_discharge readings")
	}
	if tempN == 0 {
		return risk.BatteryFeatures{}, fmt.Errorf("battery features: no temperature_c readings")
	}

	return risk.BatteryFeatures{
		CycleCount:          cycleCount,
		AvgDepthOfDischarge: dodSum / float64(dodN),
		AvgTemperatureC:     tempSum / float64(tempN),
		AgeDays:             cycleCount,
		ChargeRateVariance:  populationVariance(voltages),
	}, nil
}

// RepaymentFeaturesFrom aggregates a loan's repayment history into the five
// features the ML sidecar expects (risk.RepaymentFeatures). It mirrors
// ml/training/train_repayment_model.py's build_features, minus the 70/30
// train/test split (a training-only concept).
//
//   - OnTimeRatio:             mean(status == "on_time")
//   - AvgDaysLate:             mean of max(paid_date - due_date, 0), where a
//     missed payment (nil paid_date) contributes 0
//     (matching training's fillna(0))
//   - PaymentCadenceProxy:     sample std (ddof=1) of gaps in days between
//     consecutive paid dates (matching pandas .std())
//   - LoanToBatteryValueRatio: principal_kes / battery_value_kes
//   - TenureDays:              (max due_date - min due_date) in whole days
func RepaymentFeaturesFrom(l Loan, events []RepaymentEvent) (risk.RepaymentFeatures, error) {
	if len(events) == 0 {
		return risk.RepaymentFeatures{}, fmt.Errorf("repayment features: no repayment events")
	}
	if l.PrincipalKes == nil || l.BatteryValueKes == nil || *l.BatteryValueKes <= 0 {
		return risk.RepaymentFeatures{}, fmt.Errorf("repayment features: loan missing principal_kes or battery_value_kes")
	}

	sorted := append([]RepaymentEvent(nil), events...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].DueDate.Before(sorted[j].DueDate) })

	onTime := 0
	var daysLateSum float64
	paidDates := make([]time.Time, 0, len(sorted))
	for _, e := range sorted {
		if e.Status != nil && *e.Status == "on_time" {
			onTime++
		}
		daysLateSum += daysLate(e)
		if e.PaidDate != nil {
			paidDates = append(paidDates, *e.PaidDate)
		}
	}

	// Sort paid dates chronologically, exactly like pandas
	// paid_date.dropna().sort_values() before .diff().
	sort.Slice(paidDates, func(i, j int) bool { return paidDates[i].Before(paidDates[j]) })

	paymentCadenceProxy := 0.0
	if len(paidDates) > 1 {
		gaps := make([]float64, 0, len(paidDates)-1)
		for i := 1; i < len(paidDates); i++ {
			gaps = append(gaps, float64(wholeDays(paidDates[i].Sub(paidDates[i-1]))))
		}
		paymentCadenceProxy = sampleStdDev(gaps)
	}

	tenureDays := float64(wholeDays(sorted[len(sorted)-1].DueDate.Sub(sorted[0].DueDate)))

	return risk.RepaymentFeatures{
		OnTimeRatio:             float64(onTime) / float64(len(sorted)),
		AvgDaysLate:             daysLateSum / float64(len(sorted)),
		PaymentCadenceProxy:     paymentCadenceProxy,
		LoanToBatteryValueRatio: *l.PrincipalKes / *l.BatteryValueKes,
		TenureDays:              tenureDays,
	}, nil
}

// daysLate returns whole days a payment was late, matching Python's
// max((paid - due).days, 0) with missed payments (nil paid_date) treated as 0.
func daysLate(e RepaymentEvent) float64 {
	if e.PaidDate == nil {
		return 0
	}
	d := wholeDays(e.PaidDate.Sub(e.DueDate))
	if d < 0 {
		return 0
	}
	return float64(d)
}

// wholeDays truncates a duration toward zero into whole days, matching
// Python timedelta.days for same-day-resolution dates.
func wholeDays(d time.Duration) int {
	return int(d.Hours() / 24)
}

func populationVariance(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))

	var sum float64
	for _, v := range vals {
		d := v - mean
		sum += d * d
	}
	return sum / float64(len(vals))
}

func sampleStdDev(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))

	var sum float64
	for _, v := range vals {
		d := v - mean
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(vals)-1))
}

// TelemetryCadenceProxyFrom computes Feature A's swap-cadence signal from a
// battery's telemetry. It mirrors ml/app/services/scoring_service.py's
// compute_telemetry_cadence_proxy exactly:
//
//	0.5 * CV(inter-reading gaps in days) + 0.5 * CV(distance_km_since_last)
//
// where CV is sample std (ddof=1) / mean. Higher => more irregular usage.
// Returns 0.0 when fewer than two usable intervals exist (neutral, not risky).
func TelemetryCadenceProxyFrom(readings []TelemetryReading) float64 {
	sorted := append([]TelemetryReading(nil), readings...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ReadingAt.Before(sorted[j].ReadingAt) })

	var gaps, distances []float64
	for i := 1; i < len(sorted); i++ {
		gap := float64(wholeDays(sorted[i].ReadingAt.Sub(sorted[i-1].ReadingAt)))
		if gap <= 0 {
			continue // duplicate/replayed timestamp; cadence undefined
		}
		if sorted[i].DistanceKmSinceLast == nil {
			continue
		}
		gaps = append(gaps, gap)
		distances = append(distances, *sorted[i].DistanceKmSinceLast)
	}
	if len(gaps) < 2 {
		return 0.0
	}
	return 0.5*coefficientOfVariation(gaps) + 0.5*coefficientOfVariation(distances)
}

func coefficientOfVariation(vals []float64) float64 {
	if len(vals) < 2 {
		return 0.0
	}
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))
	if mean <= 0 {
		return 0.0
	}
	return sampleStdDev(vals) / mean
}
