// Package cytoaiclient is a minimal Go client for the Cyto AI API.
package cytoaiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// AnomalyFlag is a single fraud/anomaly finding surfaced alongside a score.
type AnomalyFlag struct {
	EntityType  string `json:"entity_type"`
	EntityID    string `json:"entity_id"`
	RuleOrModel string `json:"rule_or_model"`
	Severity    string `json:"severity"`
	Reason      string `json:"reason"`
}

// ScoreResponse is the response from POST /v1/score.
type ScoreResponse struct {
	RiderID               string        `json:"rider_id"`
	BatteryID             string        `json:"battery_id"`
	FinancingModel        string        `json:"financing_model,omitempty"`
	BHIContext            string        `json:"bhi_context,omitempty"`
	BatteryHealthIndex    float64       `json:"battery_health_index"`
	RepaymentRiskIndex    float64       `json:"repayment_risk_index"`
	CytoScore             float64       `json:"cyto_score"`
	BatteryModelVersion   string        `json:"battery_model_version"`
	RepaymentModelVersion string        `json:"repayment_model_version"`
	ScoredAt              string        `json:"scored_at,omitempty"`
	AnomalyFlags          []AnomalyFlag `json:"anomaly_flags"`
}

// Score is the persisted score returned by GET /v1/score/{rider_id}.
type Score struct {
	ID                 int64         `json:"id"`
	RiderID            *string       `json:"rider_id,omitempty"`
	BatteryID          *string       `json:"battery_id,omitempty"`
	FinancingModel     string        `json:"financing_model,omitempty"`
	BHIContext         string        `json:"bhi_context,omitempty"`
	BatteryHealthIndex float64       `json:"battery_health_index"`
	RepaymentRiskIndex float64       `json:"repayment_risk_index"`
	CytoScore          float64       `json:"cyto_score"`
	ScoredAt           *string       `json:"scored_at,omitempty"`
	ModelVersion       *string       `json:"model_version,omitempty"`
	AnomalyFlags       []AnomalyFlag `json:"anomaly_flags"`
}

// Client is a minimal Cyto AI API client.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// New constructs a client. baseURL should not have a trailing slash.
func New(baseURL, apiKey string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, HTTP: http.DefaultClient}
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("cytoaiclient: marshal body: %w", err)
		}
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rdr)
	if err != nil {
		return fmt.Errorf("cytoaiclient: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("cytoaiclient: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cytoaiclient: %s %s: status %d: %s", method, path, resp.StatusCode, string(b))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Score requests an on-demand score for a rider/battery pair.
func (c *Client) Score(ctx context.Context, riderID, batteryID string) (*ScoreResponse, error) {
	var out ScoreResponse
	err := c.do(ctx, http.MethodPost, "/v1/score", map[string]string{
		"rider_id":   riderID,
		"battery_id": batteryID,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetScore fetches the latest persisted score for a rider.
func (c *Client) GetScore(ctx context.Context, riderID string) (*Score, error) {
	var out Score
	if err := c.do(ctx, http.MethodGet, "/v1/score/"+url.PathEscape(riderID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
