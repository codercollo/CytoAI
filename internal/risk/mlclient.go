// Package risk implements CytoScore computation: calling out to the Python
// ML sidecar (ml/app/) for the two model predictions, applying the
// deterministic guardrails in rules.go, and combining them via scoring.go.
package risk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Sentinel errors callers can check with errors.Is.
var (
	// ErrModelUnavailable means the sidecar is up but the requested model
	// artifact hasn't been trained/loaded yet (sidecar returns HTTP 503).
	ErrModelUnavailable = errors.New("ml sidecar: model not loaded")
	// ErrInvalidFeatures means the sidecar rejected the request body
	// (missing/out-of-range fields — sidecar returns HTTP 400).
	ErrInvalidFeatures = errors.New("ml sidecar: invalid feature payload")
)

// BatteryFeatures mirrors ml/training/train_battery_model.py's FEATURE_NAMES
// exactly, in both field set and JSON key spelling — the Flask sidecar
// validates against these five keys and no others (ml/app/schemas/score_schema.py).
type BatteryFeatures struct {
	CycleCount          float64 `json:"cycle_count"`
	AvgDepthOfDischarge float64 `json:"avg_depth_of_discharge"`
	AvgTemperatureC     float64 `json:"avg_temperature_c"`
	AgeDays             float64 `json:"age_days"`
	ChargeRateVariance  float64 `json:"charge_rate_variance"`
}

// RepaymentFeatures mirrors ml/training/train_repayment_model.py's FEATURE_NAMES.
type RepaymentFeatures struct {
	OnTimeRatio float64 `json:"on_time_ratio"`
	AvgDaysLate float64 `json:"avg_days_late"`
	// PaymentCadenceProxy is a usage/income proxy derived from repayment
	// due/paid-date cadence, NOT telemetry-confirmed battery swaps.
	PaymentCadenceProxy     float64 `json:"payment_cadence_proxy"`
	LoanToBatteryValueRatio float64 `json:"loan_to_battery_value_ratio"`
	TenureDays              float64 `json:"tenure_days"`
	// TelemetryCadenceProxy is Feature A's swap/usage regularity signal,
	// derived from telemetry reading_at intervals + distance_km_since_last.
	TelemetryCadenceProxy float64 `json:"telemetry_cadence_proxy"`
}

// BatteryPrediction is the decoded response from POST /predict/battery.
type BatteryPrediction struct {
	BatteryHealthIndex float64 `json:"battery_health_index"`
	ModelVersion       string  `json:"model_version"`
}

// RepaymentPrediction is the decoded response from POST /predict/repayment.
type RepaymentPrediction struct {
	RepaymentRiskIndex float64 `json:"repayment_risk_index"`
	ModelVersion       string  `json:"model_version"`
}

// ReadyStatus is the decoded response from GET /readyz.
type ReadyStatus struct {
	Status               string `json:"status"`
	BatteryModelLoaded   bool   `json:"battery_model_loaded"`
	RepaymentModelLoaded bool   `json:"repayment_model_loaded"`
}

type errorBody struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

// MLClient talks to the Flask ML sidecar over HTTP.
type MLClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewMLClient constructs a client. baseURL is e.g. "http://localhost:5001"
// (no trailing slash required). If httpClient is nil, a client with a 10s
// timeout is used — the sidecar's inference is CPU-bound and fast
// (gradient-boosted trees / logistic regression, not deep learning), so a
// short timeout is intentional: a slow response means something's wrong,
// not that the model needs more time.
func NewMLClient(baseURL string, httpClient *http.Client) *MLClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &MLClient{baseURL: baseURL, httpClient: httpClient}
}

// PredictBattery calls POST /predict/battery.
func (c *MLClient) PredictBattery(ctx context.Context, f BatteryFeatures) (*BatteryPrediction, error) {
	var out BatteryPrediction
	if err := c.postJSON(ctx, "/predict/battery", f, &out); err != nil {
		return nil, fmt.Errorf("predict battery: %w", err)
	}
	return &out, nil
}

// PredictRepayment calls POST /predict/repayment.
func (c *MLClient) PredictRepayment(ctx context.Context, f RepaymentFeatures) (*RepaymentPrediction, error) {
	var out RepaymentPrediction
	if err := c.postJSON(ctx, "/predict/repayment", f, &out); err != nil {
		return nil, fmt.Errorf("predict repayment: %w", err)
	}
	return &out, nil
}

// Ready calls GET /readyz. It returns the decoded body even on a 503 (the
// sidecar always returns a body describing which model(s) aren't loaded),
// so callers can surface a precise "battery model not trained yet" message
// instead of a bare connection-style error.
func (c *MLClient) Ready(ctx context.Context) (*ReadyStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/readyz", nil)
	if err != nil {
		return nil, fmt.Errorf("build readyz request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call readyz: %w", err)
	}
	defer resp.Body.Close()

	var status ReadyStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode readyz response: %w", err)
	}
	return &status, nil
}

func (c *MLClient) postJSON(ctx context.Context, path string, body, out interface{}) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call sidecar: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		return nil
	case http.StatusServiceUnavailable:
		return fmt.Errorf("%w: %s", ErrModelUnavailable, describeError(respBytes))
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrInvalidFeatures, describeError(respBytes))
	default:
		return fmt.Errorf("unexpected status %d from sidecar: %s", resp.StatusCode, string(respBytes))
	}
}

func describeError(body []byte) string {
	var e errorBody
	if err := json.Unmarshal(body, &e); err != nil || (e.Message == "" && len(e.Details) == 0) {
		return string(body)
	}
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%v", e.Details)
}

// --- Feature B: anomaly detection ------------------------------------------

// TelemetryAnomalyRecord is the raw telemetry record sent to /predict/anomaly.
// The sidecar uses `id` (when present) as the anomaly entity_id.
type TelemetryAnomalyRecord struct {
	ID                  int64    `json:"id,omitempty"`
	BatteryID           string   `json:"battery_id"`
	ReadingAt           string   `json:"reading_at"`
	StateOfCharge       *float64 `json:"state_of_charge,omitempty"`
	Voltage             *float64 `json:"voltage,omitempty"`
	TemperatureC        *float64 `json:"temperature_c,omitempty"`
	CycleCount          *int     `json:"cycle_count,omitempty"`
	DepthOfDischarge    *float64 `json:"depth_of_discharge,omitempty"`
	DistanceKmSinceLast *float64 `json:"distance_km_since_last,omitempty"`
}

// RepaymentAnomalyRecord is the raw repayment record sent to /predict/anomaly.
type RepaymentAnomalyRecord struct {
	ID            int64    `json:"id,omitempty"`
	LoanID        string   `json:"loan_id"`
	DueDate       string   `json:"due_date"`
	PaidDate      *string  `json:"paid_date,omitempty"`
	AmountDueKes  *float64 `json:"amount_due_kes,omitempty"`
	AmountPaidKes *float64 `json:"amount_paid_kes,omitempty"`
	Status        *string  `json:"status,omitempty"`
}

// AnomalyFlag is a single non-gating fraud/anomaly finding from the sidecar.
type AnomalyFlag struct {
	EntityType  string `json:"entity_type"`
	EntityID    string `json:"entity_id"`
	RuleOrModel string `json:"rule_or_model"`
	Severity    string `json:"severity"`
	Reason      string `json:"reason"`
}

type anomalyDetectionResponse struct {
	Flags []AnomalyFlag `json:"flags"`
}

// DetectTelemetryAnomalies calls POST /predict/anomaly for telemetry records.
func (c *MLClient) DetectTelemetryAnomalies(ctx context.Context, records []TelemetryAnomalyRecord) ([]AnomalyFlag, error) {
	return c.detectAnomalies(ctx, map[string]any{"telemetry": records, "repayments": []any{}})
}

// DetectRepaymentAnomalies calls POST /predict/anomaly for repayment records.
func (c *MLClient) DetectRepaymentAnomalies(ctx context.Context, records []RepaymentAnomalyRecord) ([]AnomalyFlag, error) {
	return c.detectAnomalies(ctx, map[string]any{"telemetry": []any{}, "repayments": records})
}

func (c *MLClient) detectAnomalies(ctx context.Context, payload any) ([]AnomalyFlag, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode anomaly request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/predict/anomaly", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build anomaly request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call anomaly sidecar: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read anomaly response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from anomaly sidecar: %s", resp.StatusCode, string(respBytes))
	}

	var out anomalyDetectionResponse
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("decode anomaly response: %w", err)
	}
	if out.Flags == nil {
		out.Flags = []AnomalyFlag{}
	}
	return out.Flags, nil
}
