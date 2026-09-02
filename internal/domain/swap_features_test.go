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
