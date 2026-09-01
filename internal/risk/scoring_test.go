package risk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWeights_Validate(t *testing.T) {
	cases := []struct {
		name    string
		w       Weights
		wantErr bool
	}{
		{"default 50/50", DefaultWeights(), false},
		{"valid 70/30", Weights{BHI: 0.7, RRI: 0.3}, false},
		{"doesn't sum to 1", Weights{BHI: 0.6, RRI: 0.6}, true},
		{"negative weight", Weights{BHI: -0.1, RRI: 1.1}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.w.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewScorer_RejectsInvalidWeights(t *testing.T) {
	client := NewMLClient("http://unused", nil)
	_, err := NewScorer(client, Weights{BHI: 0.9, RRI: 0.9})
	if err == nil {
		t.Fatal("expected NewScorer to reject weights that don't sum to 1")
	}
}

func TestClampScore(t *testing.T) {
	cases := map[float64]float64{-10: 0, 0: 0, 50: 50, 100: 100, 150: 100}
	for in, want := range cases {
		if got := ClampScore(in); got != want {
			t.Errorf("ClampScore(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestSufficientBatteryData(t *testing.T) {
	ok, err := SufficientBatteryData(BatteryFeatures{CycleCount: 2})
	if ok || !errors.Is(err, ErrInsufficientData) {
		t.Errorf("expected insufficient data for 2 cycles, got ok=%v err=%v", ok, err)
	}

	ok, err = SufficientBatteryData(BatteryFeatures{CycleCount: 50})
	if !ok || err != nil {
		t.Errorf("expected sufficient data for 50 cycles, got ok=%v err=%v", ok, err)
	}
}

func TestSufficientRepaymentData(t *testing.T) {
	ok, err := SufficientRepaymentData(RepaymentFeatures{TenureDays: 3})
	if ok || !errors.Is(err, ErrInsufficientData) {
		t.Errorf("expected insufficient data for 3-day-old loan, got ok=%v err=%v", ok, err)
	}
}

func TestScorer_Score_CombinesCorrectly(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/predict/battery", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(BatteryPrediction{BatteryHealthIndex: 80, ModelVersion: "bhi-v0.1"})
	})
	mux.HandleFunc("/predict/repayment", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(RepaymentPrediction{RepaymentRiskIndex: 20, ModelVersion: "rri-v0.2-proxy"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewMLClient(srv.URL, nil)
	scorer, err := NewScorer(client, DefaultWeights())
	if err != nil {
		t.Fatalf("unexpected error building scorer: %v", err)
	}

	result, err := scorer.Score(context.Background(),
		BatteryFeatures{CycleCount: 100, AvgDepthOfDischarge: 60, AvgTemperatureC: 27, AgeDays: 300, ChargeRateVariance: 0.03},
		RepaymentFeatures{OnTimeRatio: 0.9, AvgDaysLate: 0.5, PaymentCadenceProxy: 1, LoanToBatteryValueRatio: 1, TenureDays: 200},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CytoScore = 0.5*(100-20) + 0.5*80 = 40 + 40 = 80
	if result.CytoScore != 80 {
		t.Errorf("expected CytoScore 80, got %v", result.CytoScore)
	}
	if result.BatteryHealthIndex != 80 || result.RepaymentRiskIndex != 20 {
		t.Errorf("unexpected component scores: %+v", result)
	}
}

func TestScorer_Score_InsufficientBatteryData(t *testing.T) {
	client := NewMLClient("http://unused", nil)
	scorer, _ := NewScorer(client, DefaultWeights())

	_, err := scorer.Score(context.Background(),
		BatteryFeatures{CycleCount: 1},
		RepaymentFeatures{TenureDays: 200},
	)
	if !errors.Is(err, ErrInsufficientData) {
		t.Fatalf("expected ErrInsufficientData, got %v", err)
	}
}

func TestScorer_Score_PropagatesSidecarError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/predict/battery", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "model_unavailable", "message": "not trained"})
	})
	mux.HandleFunc("/predict/repayment", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(RepaymentPrediction{RepaymentRiskIndex: 20, ModelVersion: "rri-v0.2-proxy"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewMLClient(srv.URL, nil)
	scorer, _ := NewScorer(client, DefaultWeights())

	_, err := scorer.Score(context.Background(),
		BatteryFeatures{CycleCount: 100, AvgDepthOfDischarge: 60, AvgTemperatureC: 27, AgeDays: 300, ChargeRateVariance: 0.03},
		RepaymentFeatures{OnTimeRatio: 0.9, AvgDaysLate: 0.5, PaymentCadenceProxy: 1, LoanToBatteryValueRatio: 1, TenureDays: 200},
	)
	if !errors.Is(err, ErrModelUnavailable) {
		t.Fatalf("expected ErrModelUnavailable to propagate, got %v", err)
	}
}
