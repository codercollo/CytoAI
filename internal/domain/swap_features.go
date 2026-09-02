package domain

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/codercollo/cytoai/internal/risk"
)

// SwapCadenceProxyFrom mirrors ml/app/services/scoring_service.py's
// compute_swap_cadence_proxy: activity regularity from swap_events timestamps
// and distance_km_since_last_swap.
func SwapCadenceProxyFrom(swaps []SwapEvent) float64 {
	sorted := append([]SwapEvent(nil), swaps...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].SwappedAt.Before(sorted[j].SwappedAt) })

	var gaps, distances []float64
	for i := 1; i < len(sorted); i++ {
		gap := float64(wholeDays(sorted[i].SwappedAt.Sub(sorted[i-1].SwappedAt)))
		if gap <= 0 {
			continue
		}
		if sorted[i].DistanceKmSinceLastSwap == nil {
			continue
		}
		gaps = append(gaps, gap)
		distances = append(distances, *sorted[i].DistanceKmSinceLastSwap)
	}
	if len(gaps) < 2 {
		return 0.0
	}
	return 0.5*coefficientOfVariation(gaps) + 0.5*coefficientOfVariation(distances)
}

// SwapBatteryBhiFeaturesFrom aggregates swap_events into fleet-level BHI
// features. Mirrors compute_swap_battery_bhi_features; charge_rate_variance is
// 0.0 because swap_events carry no voltage snapshot.
func SwapBatteryBhiFeaturesFrom(swaps []SwapEvent) (risk.BatteryFeatures, error) {
	var cycles, dods, temps []float64
	var times []time.Time
	for _, s := range swaps {
		times = append(times, s.SwappedAt)
		if s.ReturnedCycleCount != nil {
			cycles = append(cycles, float64(*s.ReturnedCycleCount))
		}
		if s.ReturnedDepthOfDischarge != nil {
			dods = append(dods, *s.ReturnedDepthOfDischarge)
		}
		if s.ReturnedTemperatureC != nil {
			temps = append(temps, *s.ReturnedTemperatureC)
		}
	}
	if len(cycles) == 0 && len(dods) == 0 && len(temps) == 0 {
		return risk.BatteryFeatures{}, fmt.Errorf("swap battery bhi: no usable swap telemetry")
	}

	ageDays := 0.0
	if len(times) >= 2 {
		min, max := times[0], times[0]
		for _, t := range times {
			if t.Before(min) {
				min = t
			}
			if t.After(max) {
				max = t
			}
		}
		ageDays = float64(wholeDays(max.Sub(min)))
	}

	return risk.BatteryFeatures{
		CycleCount:          maxFloat(cycles),
		AvgDepthOfDischarge: meanFloat(dods),
		AvgTemperatureC:     meanFloat(temps),
		AgeDays:             ageDays,
		ChargeRateVariance:  0.0,
	}, nil
}

// SwapBatteryStressProfileFrom mirrors ml/app/services/scoring_service.py's
// compute_battery_stress_profile exactly. It measures how much hotter and more
// deeply-discharged a rider returns batteries vs the fleet mean, as a
// z-score-vs-fleet profile (0.5 temp + 0.5 DoD weighting, floored at 0 so a
// below-average rider gets no penalty). Returns 0.0 when either fleet series
// has fewer than 2 points or the rider has no usable values.
func SwapBatteryStressProfileFrom(riderSwaps, fleetSwaps []SwapEvent) float64 {
	riderTemp := swapTempValues(riderSwaps)
	riderDoD := swapDoDValues(riderSwaps)
	fleetTemp := swapTempValues(fleetSwaps)
	fleetDoD := swapDoDValues(fleetSwaps)

	if len(riderTemp) == 0 && len(riderDoD) == 0 {
		return 0.0
	}

	stress := func(rider, fleet []float64) float64 {
		if len(rider) == 0 || len(fleet) < 2 {
			return 0.0
		}
		sd := sampleStdDev(fleet)
		if sd <= 0 {
			return 0.0
		}
		return math.Max(0.0, (meanFloat(rider)-meanFloat(fleet))/sd)
	}

	raw := 0.5*stress(riderTemp, fleetTemp) + 0.5*stress(riderDoD, fleetDoD)
	return math.Round(raw*1e6) / 1e6
}

func swapTempValues(swaps []SwapEvent) []float64 {
	out := make([]float64, 0, len(swaps))
	for _, s := range swaps {
		if s.ReturnedTemperatureC != nil {
			out = append(out, *s.ReturnedTemperatureC)
		}
	}
	return out
}

func swapDoDValues(swaps []SwapEvent) []float64 {
	out := make([]float64, 0, len(swaps))
	for _, s := range swaps {
		if s.ReturnedDepthOfDischarge != nil {
			out = append(out, *s.ReturnedDepthOfDischarge)
		}
	}
	return out
}

func meanFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func maxFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0.0
	}
	m := vals[0]
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}
