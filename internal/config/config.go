// Package config loads the Cyto AI service configuration from a YAML file
// with an environment-variable overlay.
//
// Decision: we use plain YAML (gopkg.in/yaml.v3) + env overlay rather than
// Viper to keep the MVP dependency tree minimal. If remote config, live
// reload, or flag binding is ever needed, swap this loader for Viper.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Defaults applied when a value is absent from both YAML and env.
const (
	DefaultMLBaseURL = "http://localhost:5001"
	DefaultHTTPPort  = 8080
	DefaultLogLevel  = "info"
	DefaultBHIWeight = 0.5
	DefaultRRIWeight = 0.5
	// DefaultCORSAllowedOrigin is the local MVP dashboard origin.
	DefaultCORSAllowedOrigin = "http://localhost:3000"
)

// Config holds the runtime configuration for the Go API service.
type Config struct {
	PostgresDSN string  `yaml:"postgres_dsn"`
	MLBaseURL   string  `yaml:"ml_base_url"`
	HTTPPort    int     `yaml:"http_port"`
	LogLevel    string  `yaml:"log_level"`
	Weights     Weights `yaml:"weights"`
	// CORSAllowedOrigins is the allowlist for browser cross-origin requests.
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins"`
}

// Weights configures how BHI and RRI combine into CytoScore (docs/spec.md §5).
// Keep these in sync with internal/risk.Weights.
type Weights struct {
	BHI float64 `yaml:"bhi"`
	RRI float64 `yaml:"rri"`
}

// Load reads the YAML file at path (when non-empty), overlays environment
// variables, applies defaults, and validates required fields. It returns an
// error — never a zero-value Config — when required fields are missing.
func Load(path string) (Config, error) {
	var c Config

	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("config: read %s: %w", path, err)
		}
		if err := yaml.Unmarshal(b, &c); err != nil {
			return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
		}
	}

	if err := c.applyEnv(); err != nil {
		return Config{}, err
	}
	c.applyDefaults()

	if err := c.validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c *Config) applyEnv() error {
	if v, ok := os.LookupEnv("CYTOAI_POSTGRES_DSN"); ok {
		c.PostgresDSN = v
	}
	if v, ok := os.LookupEnv("CYTOAI_ML_BASE_URL"); ok {
		c.MLBaseURL = v
	}
	if v, ok := os.LookupEnv("CYTOAI_HTTP_PORT"); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("config: CYTOAI_HTTP_PORT must be an integer, got %q", v)
		}
		c.HTTPPort = n
	}
	if v, ok := os.LookupEnv("CYTOAI_LOG_LEVEL"); ok {
		c.LogLevel = v
	}
	if v, ok := os.LookupEnv("CYTOAI_BHI_WEIGHT"); ok {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("config: CYTOAI_BHI_WEIGHT must be a number, got %q", v)
		}
		c.Weights.BHI = f
	}
	if v, ok := os.LookupEnv("CYTOAI_RRI_WEIGHT"); ok {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("config: CYTOAI_RRI_WEIGHT must be a number, got %q", v)
		}
		c.Weights.RRI = f
	}
	if v, ok := os.LookupEnv("CYTOAI_CORS_ALLOWED_ORIGINS"); ok {
		c.CORSAllowedOrigins = splitComma(v)
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.MLBaseURL == "" {
		c.MLBaseURL = DefaultMLBaseURL
	}
	if c.HTTPPort == 0 {
		c.HTTPPort = DefaultHTTPPort
	}
	if c.LogLevel == "" {
		c.LogLevel = DefaultLogLevel
	}
	if len(c.CORSAllowedOrigins) == 0 {
		c.CORSAllowedOrigins = []string{DefaultCORSAllowedOrigin}
	}
	if c.Weights.BHI == 0 && c.Weights.RRI == 0 {
		c.Weights.BHI = DefaultBHIWeight
		c.Weights.RRI = DefaultRRIWeight
	}
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (c Config) validate() error {
	if strings.TrimSpace(c.PostgresDSN) == "" {
		return errors.New("config: postgres_dsn is required (set postgres_dsn in YAML or CYTOAI_POSTGRES_DSN)")
	}
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("config: http_port out of range: %d", c.HTTPPort)
	}
	if c.Weights.BHI < 0 || c.Weights.RRI < 0 {
		return fmt.Errorf("config: weights must be non-negative, got bhi=%.4f rri=%.4f", c.Weights.BHI, c.Weights.RRI)
	}
	const epsilon = 1e-6
	sum := c.Weights.BHI + c.Weights.RRI
	if sum < 1-epsilon || sum > 1+epsilon {
		return fmt.Errorf("config: weights must sum to 1.0, got %.4f (bhi=%.4f rri=%.4f)", sum, c.Weights.BHI, c.Weights.RRI)
	}
	return nil
}
