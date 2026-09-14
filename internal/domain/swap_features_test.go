package domain

import (
	"math"
	"testing"
	"time"
)

func TestSwapBatteryStressProfileFrom(t *testing.T) {
	fleet := []SwapEvent{
		{ReturnedTemperatureC: f64(27), ReturnedDepthOfDischarge: f64(80)},
		{ReturnedTemperatureC: f64(29), ReturnedDepthOfDischarge: f64(70)},
		{ReturnedTemperatureC: f64(31), ReturnedDepthOfDischarge: f64(90)},
	}

	t.Run("positive when rider hotter and deeper", func(t *testing.T) {
		rider := []SwapEvent{{ReturnedTemperatureC: f64(40), ReturnedDepthOfDischarge: f64(95)}}
		if got := SwapBatteryStressProfileFrom(rider, fleet); got <= 0 {
			t.Fatalf("stress = %v, want > 0", got)
		}
	})

	t.Run("zero at fleet average", func(t *testing.T) {
		rider := []SwapEvent{{ReturnedTemperatureC: f64(29), ReturnedDepthOfDischarge: f64(80)}}
		if got := SwapBatteryStressProfileFrom(rider, fleet); math.Abs(got) > 1e-9 {
			t.Fatalf("stress = %v, want 0", got)
		}
	})

	t.Run("zero when fleet has fewer than two points", func(t *testing.T) {
		rider := []SwapEvent{{ReturnedTemperatureC: f64(40), ReturnedDepthOfDischarge: f64(95)}}
		single := []SwapEvent{{ReturnedTemperatureC: f64(27), ReturnedDepthOfDischarge: f64(80)}}
		if got := SwapBatteryStressProfileFrom(rider, single); got != 0 {
			t.Fatalf("stress = %v, want 0", got)
		}
	})

	t.Run("zero when rider has no usable values", func(t *testing.T) {
		rider := []SwapEvent{{SwappedAt: time.Now().UTC()}}
		if got := SwapBatteryStressProfileFrom(rider, fleet); got != 0 {
			t.Fatalf("stress = %v, want 0", got)
		}
	})
}

func TestSwapFeeBurdenFrom(t *testing.T) {
	swaps := []SwapEvent{
		{PaidAmountKes: f64(250.0)},
		{PaidAmountKes: nil}, // nil skipped
		{PaidAmountKes: f64(300.0)},
	}

	total, count := SwapFeeBurdenFrom(swaps)
	if total != 550.0 {
		t.Errorf("SwapFeeBurdenFrom total = %v, want 550.0", total)
	}
	if count != 2 {
		t.Errorf("SwapFeeBurdenFrom count = %v, want 2", count)
	}
}

func TestComputeSwapFeeBurdenRRIAdjustment(t *testing.T) {
	params := DefaultSwapFeeBurdenAdjustmentParams()

	t.Run("zero events returns zero adjustment", func(t *testing.T) {
		got := ComputeSwapFeeBurdenRRIAdjustment(500.0, 0, 100.0, params)
		if got != 0 {
			t.Errorf("got %v, want 0", got)
		}
	})

	t.Run("below ratio threshold returns zero adjustment", func(t *testing.T) {
		got := ComputeSwapFeeBurdenRRIAdjustment(50.0, 1, 100.0, params) // ratio 0.5 <= 1.0
		if got != 0 {
			t.Errorf("got %v, want 0", got)
		}
	})

	t.Run("high ratio reduces RRI", func(t *testing.T) {
		// ratio = 300.0 / 100.0 = 3.0. discount = (3.0 - 1.0) * 5.0 = 10.0 pts RRI reduction
		got := ComputeSwapFeeBurdenRRIAdjustment(300.0, 1, 100.0, params)
		if got != 10.0 {
			t.Errorf("got %v, want 10.0", got)
		}
	})

	t.Run("capped at max discount", func(t *testing.T) {
		// ratio = 1000.0 / 100.0 = 10.0. discount = (10 - 1) * 5 = 45 -> capped at 15.0
		got := ComputeSwapFeeBurdenRRIAdjustment(1000.0, 1, 100.0, params)
		if got != 15.0 {
			t.Errorf("got %v, want 15.0", got)
		}
	})
}
