package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/codercollo/cytoai/internal/db/queries"
	"github.com/codercollo/cytoai/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func fptr(v float64) *float64     { return &v }
func iptr(v int) *int             { return &v }
func sptr(v string) *string       { return &v }
func tptr(v time.Time) *time.Time { return &v }

// integrationPool connects to a real Postgres for the integration tests.
// It skips gracefully when TEST_DATABASE_URL is unset or the DB is
// unreachable, so normal `go test ./...` runs don't require a database.
func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}
	pool, err := Connect(context.Background(), dsn)
	if err != nil {
		t.Skipf("Postgres unreachable (%v); skipping integration test", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestConnectAndReady(t *testing.T) {
	pool := integrationPool(t)
	if err := Ready(context.Background(), pool); err != nil {
		t.Fatalf("Ready() = %v, want nil", err)
	}
}

// TestRoundTrip inserts one row per table (in FK order) and reads them back,
// exercising the hand-written query functions against a real schema.
// Requires migrations to have been applied first (see scripts/migrate.sh).
func TestRoundTrip(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	q := queries.New(pool)

	var (
		partnerID, riderID, batteryID, loanID string
		telemetryID, repID, scoreID           int64
	)
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM scores WHERE id = $1`, scoreID)
		_, _ = pool.Exec(c, `DELETE FROM repayment_events WHERE id = $1`, repID)
		_, _ = pool.Exec(c, `DELETE FROM telemetry_readings WHERE id = $1`, telemetryID)
		_, _ = pool.Exec(c, `DELETE FROM loans WHERE id = $1`, loanID)
		_, _ = pool.Exec(c, `DELETE FROM batteries WHERE id = $1`, batteryID)
		_, _ = pool.Exec(c, `DELETE FROM riders WHERE id = $1`, riderID)
		_, _ = pool.Exec(c, `DELETE FROM partners WHERE id = $1`, partnerID)
	})

	// partners
	if err := pool.QueryRow(ctx, `INSERT INTO partners (name, api_key_hash) VALUES ($1, $2) RETURNING id`, "test-partner", "test-hash").Scan(&partnerID); err != nil {
		t.Fatalf("insert partner: %v", err)
	}

	// riders
	if err := pool.QueryRow(ctx, `INSERT INTO riders (external_ref, partner_id) VALUES ($1, $2) RETURNING id`, "rider-ext", partnerID).Scan(&riderID); err != nil {
		t.Fatalf("insert rider: %v", err)
	}

	// batteries
	if err := pool.QueryRow(ctx, `INSERT INTO batteries (external_ref, manufacturer, rated_capacity_wh) VALUES ($1, $2, $3) RETURNING id`, "battery-ext", "maker", 2000.0).Scan(&batteryID); err != nil {
		t.Fatalf("insert battery: %v", err)
	}

	// telemetry_readings (via queries.Store COPY insert)
	reading := domain.TelemetryReading{
		BatteryID:           batteryID,
		ReadingAt:           time.Now().UTC().Truncate(time.Microsecond),
		StateOfCharge:       fptr(80),
		Voltage:             fptr(52),
		TemperatureC:        fptr(27),
		CycleCount:          iptr(5),
		DepthOfDischarge:    fptr(60),
		DistanceKmSinceLast: fptr(12.5),
	}
	if _, err := q.InsertTelemetryReadings(ctx, []domain.TelemetryReading{reading}); err != nil {
		t.Fatalf("insert telemetry: %v", err)
	}
	var soc float64
	if err := pool.QueryRow(ctx, `SELECT id, state_of_charge FROM telemetry_readings WHERE battery_id = $1`, batteryID).Scan(&telemetryID, &soc); err != nil {
		t.Fatalf("read telemetry: %v", err)
	}
	if soc != 80 {
		t.Errorf("state_of_charge = %v, want 80", soc)
	}

	// loans
	if err := pool.QueryRow(ctx, `INSERT INTO loans (rider_id, battery_id, principal_kes, battery_value_kes, term_months, daily_installment_kes, started_at) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, riderID, batteryID, 300000.0, 150000.0, 24, 460.0, time.Now().UTC().Truncate(24*time.Hour)).Scan(&loanID); err != nil {
		t.Fatalf("insert loan: %v", err)
	}

	// repayment_events (via queries.Store COPY insert)
	due := time.Now().UTC().Truncate(24 * time.Hour)
	paid := due.AddDate(0, 0, 1)
	event := domain.RepaymentEvent{
		LoanID:        loanID,
		DueDate:       due,
		PaidDate:      &paid,
		AmountDueKes:  fptr(460),
		AmountPaidKes: fptr(460),
		Status:        sptr("on_time"),
	}
	if _, err := q.InsertRepaymentEvents(ctx, []domain.RepaymentEvent{event}); err != nil {
		t.Fatalf("insert repayment event: %v", err)
	}
	var repStatus string
	if err := pool.QueryRow(ctx, `SELECT id, status FROM repayment_events WHERE loan_id = $1`, loanID).Scan(&repID, &repStatus); err != nil {
		t.Fatalf("read repayment event: %v", err)
	}
	if repStatus != "on_time" {
		t.Errorf("status = %q, want on_time", repStatus)
	}

	// scores (via queries.Store)
	score := domain.Score{
		RiderID:            &riderID,
		BatteryID:          &batteryID,
		BatteryHealthIndex: 80,
		RepaymentRiskIndex: 20,
		CytoScore:          80,
		ScoredAt:           tptr(time.Now().UTC().Truncate(time.Microsecond)),
		ModelVersion:       sptr("bhi=bhi-v0.1;rri=rri-v0.2-proxy"),
	}
	var err error
	if scoreID, err = q.InsertScore(ctx, score); err != nil {
		t.Fatalf("insert score: %v", err)
	}

	latest, err := q.LatestScore(ctx, riderID)
	if err != nil {
		t.Fatalf("latest score: %v", err)
	}
	if latest.CytoScore != 80 || latest.BatteryHealthIndex != 80 || latest.RepaymentRiskIndex != 20 {
		t.Errorf("latest score = %+v, want cyto=80 bhi=80 rri=20", latest)
	}

	portfolio, err := q.PortfolioScores(ctx, partnerID, 100, 0)
	if err != nil {
		t.Fatalf("portfolio scores: %v", err)
	}
	found := false
	for _, sc := range portfolio {
		if sc.ID == scoreID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("portfolio did not contain inserted score id %d (got %d rows)", scoreID, len(portfolio))
	}
}
