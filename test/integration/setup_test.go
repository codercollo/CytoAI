package integration

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/codercollo/cytoai/internal/api"
	"github.com/codercollo/cytoai/internal/auth"
	"github.com/codercollo/cytoai/internal/db"
	"github.com/codercollo/cytoai/internal/db/queries"
	"github.com/codercollo/cytoai/internal/risk"
	"github.com/jackc/pgx/v5/pgxpool"
)

type testEnv struct {
	ts        *httptest.Server
	store     *queries.Store
	pool      *pgxpool.Pool
	partnerID string
	key       string
}

// setup boots a real api.Server over a real Postgres pool and seeds one
// partner + API key. It skips when TEST_DATABASE_URL is unset or unreachable.
func setup(t *testing.T) *testEnv {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable (%v); skipping integration test", err)
	}
	t.Cleanup(pool.Close)

	store := queries.New(pool)

	key, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}
	hash, err := auth.HashAPIKey(key)
	if err != nil {
		t.Fatalf("hash api key: %v", err)
	}
	var partnerID string
	if err := pool.QueryRow(ctx, `INSERT INTO partners (name, api_key_hash) VALUES ($1, $2) RETURNING id::text`, "integration-test", hash).Scan(&partnerID); err != nil {
		t.Fatalf("seed partner: %v", err)
	}

	mlBase := os.Getenv("CYTOAI_ML_BASE_URL")
	if mlBase == "" {
		mlBase = "http://localhost:5001"
	}
	mlClient := risk.NewMLClient(mlBase, nil)
	scorer, err := risk.NewScorer(mlClient, risk.DefaultWeights())
	if err != nil {
		t.Fatalf("construct scorer: %v", err)
	}

	server := api.NewServer(api.Config{
		Store:           store,
		Scorer:          scorer,
		Auth:            auth.New(store),
		PingDB:          func(ctx context.Context) error { return db.Ready(ctx, pool) },
		MLReady:         mlClient.Ready,
		AnomalyDetector: mlClient,
	})

	ts := httptest.NewServer(server.Routes())
	t.Cleanup(ts.Close)

	return &testEnv{ts: ts, store: store, pool: pool, partnerID: partnerID, key: key}
}
