package config

import "testing"

const examplePath = "../../configs/config.example.yaml"

func TestLoadExample(t *testing.T) {
	c, err := Load(examplePath)
	if err != nil {
		t.Fatalf("Load(example) error: %v", err)
	}

	if c.PostgresDSN == "" {
		t.Error("PostgresDSN should be non-empty in example config")
	}
	if c.MLBaseURL != "http://localhost:5001" {
		t.Errorf("MLBaseURL = %q, want http://localhost:5001", c.MLBaseURL)
	}
	if c.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080", c.HTTPPort)
	}
	if c.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", c.LogLevel)
	}
	if c.Weights.BHI != 0.5 || c.Weights.RRI != 0.5 {
		t.Errorf("Weights = %+v, want bhi=0.5 rri=0.5", c.Weights)
	}
	if len(c.CORSAllowedOrigins) != 1 || c.CORSAllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("CORSAllowedOrigins = %v, want [http://localhost:3000]", c.CORSAllowedOrigins)
	}
}

func TestLoad_MissingPostgresDSN(t *testing.T) {
	// Empty path + no env DSN must fail loudly, not return a zero Config.
	t.Setenv("CYTOAI_POSTGRES_DSN", "")
	if _, err := Load(""); err == nil {
		t.Fatal("expected error for missing postgres_dsn, got nil")
	}
}

func TestLoad_EnvOverlay(t *testing.T) {
	t.Setenv("CYTOAI_POSTGRES_DSN", "postgres://env:env@example:5432/db")
	t.Setenv("CYTOAI_ML_BASE_URL", "http://127.0.0.1:9999")
	t.Setenv("CYTOAI_HTTP_PORT", "9090")
	t.Setenv("CYTOAI_LOG_LEVEL", "debug")
	t.Setenv("CYTOAI_BHI_WEIGHT", "0.7")
	t.Setenv("CYTOAI_RRI_WEIGHT", "0.3")

	c, err := Load(examplePath)
	if err != nil {
		t.Fatalf("Load(example) with env overlay error: %v", err)
	}

	if c.PostgresDSN != "postgres://env:env@example:5432/db" {
		t.Errorf("PostgresDSN = %q, want env value", c.PostgresDSN)
	}
	if c.MLBaseURL != "http://127.0.0.1:9999" {
		t.Errorf("MLBaseURL = %q, want http://127.0.0.1:9999", c.MLBaseURL)
	}
	if c.HTTPPort != 9090 {
		t.Errorf("HTTPPort = %d, want 9090", c.HTTPPort)
	}
	if c.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", c.LogLevel)
	}
	if c.Weights.BHI != 0.7 || c.Weights.RRI != 0.3 {
		t.Errorf("Weights = %+v, want bhi=0.7 rri=0.3", c.Weights)
	}
}

func TestLoad_InvalidWeights(t *testing.T) {
	t.Setenv("CYTOAI_POSTGRES_DSN", "postgres://x")
	t.Setenv("CYTOAI_BHI_WEIGHT", "0.6")
	t.Setenv("CYTOAI_RRI_WEIGHT", "0.6")

	if _, err := Load(""); err == nil {
		t.Fatal("expected error for weights that do not sum to 1, got nil")
	}
}
