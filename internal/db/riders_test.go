package db

import (
	"context"
	"testing"

	"github.com/codercollo/cytoai/internal/db/queries"
	"github.com/codercollo/cytoai/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newPartner(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO partners (name, api_key_hash) VALUES ($1, 'test-hash') RETURNING id::text`, name,
	).Scan(&id); err != nil {
		t.Fatalf("insert partner: %v", err)
	}
	return id
}

func deleteRiderFixture(t *testing.T, pool *pgxpool.Pool, riderID, batteryID, loanID, partnerID string) {
	t.Helper()
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `DELETE FROM scores WHERE rider_id = $1`, riderID)
	if loanID != "" {
		_, _ = pool.Exec(ctx, `DELETE FROM repayment_events WHERE loan_id = $1`, loanID)
	}
	if batteryID != "" {
		_, _ = pool.Exec(ctx, `DELETE FROM telemetry_readings WHERE battery_id = $1`, batteryID)
	}
	if loanID != "" {
		_, _ = pool.Exec(ctx, `DELETE FROM loans WHERE id = $1`, loanID)
	}
	if batteryID != "" {
		_, _ = pool.Exec(ctx, `DELETE FROM batteries WHERE id = $1`, batteryID)
	}
	if riderID != "" {
		_, _ = pool.Exec(ctx, `DELETE FROM riders WHERE id = $1`, riderID)
	}
	if partnerID != "" {
		_, _ = pool.Exec(ctx, `DELETE FROM partners WHERE id = $1`, partnerID)
	}
}

func TestCreateRider_TransactionalWithBatteryLoan(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	q := queries.New(pool)
	partnerID := newPartner(t, pool, "rider-create-partner")

	created, err := q.CreateRider(ctx, partnerID, domain.RiderRegistration{
		ExternalRef: sptr("rider_create_1"),
		Battery: &domain.BatteryRegistration{
			ExternalRef:     sptr("battery_create_1"),
			Manufacturer:    sptr("maker"),
			RatedCapacityWh: fptr(2000),
		},
		Loan: &domain.LoanRegistration{
			ExternalRef:     sptr("loan_create_1"),
			PrincipalKes:    fptr(300000),
			BatteryValueKes: fptr(150000),
			TermMonths:      iptr(12),
		},
	})
	if err != nil {
		t.Fatalf("CreateRider: %v", err)
	}
	if created.Rider.ID == "" || created.Battery == nil || created.Battery.ID == "" || created.Loan == nil || created.Loan.ID == "" {
		t.Fatalf("incomplete created summary: %+v", created)
	}
	if created.Loan.RiderID != created.Rider.ID {
		t.Errorf("loan.rider_id = %q, want %q", created.Loan.RiderID, created.Rider.ID)
	}
	if created.Loan.BatteryID == nil || *created.Loan.BatteryID != created.Battery.ID {
		t.Errorf("loan.battery_id = %v, want %q", created.Loan.BatteryID, created.Battery.ID)
	}
	defer deleteRiderFixture(t, pool, created.Rider.ID, created.Battery.ID, created.Loan.ID, partnerID)
}

// TestCreateRider_RegisteredRiderIsScoreable proves a newly registered,
// linked rider satisfies the existing scoring flow's structural prerequisites:
// the loan is resolvable and carries principal_kes + battery_value_kes.
func TestCreateRider_RegisteredRiderIsScoreable(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	q := queries.New(pool)
	partnerID := newPartner(t, pool, "rider-scoreable-partner")

	created, err := q.CreateRider(ctx, partnerID, domain.RiderRegistration{
		ExternalRef: sptr("rider_scoreable"),
		Battery:     &domain.BatteryRegistration{ExternalRef: sptr("battery_scoreable")},
		Loan: &domain.LoanRegistration{
			ExternalRef:     sptr("loan_scoreable"),
			PrincipalKes:    fptr(300000),
			BatteryValueKes: fptr(150000),
		},
	})
	if err != nil {
		t.Fatalf("CreateRider: %v", err)
	}
	defer deleteRiderFixture(t, pool, created.Rider.ID, created.Battery.ID, created.Loan.ID, partnerID)

	// The scoring flow's first step: LoanByRiderAndBattery.
	loan, err := q.LoanByRiderAndBattery(ctx, created.Rider.ID, created.Battery.ID)
	if err != nil {
		t.Fatalf("LoanByRiderAndBattery: %v", err)
	}
	if loan.PrincipalKes == nil || *loan.PrincipalKes != 300000 {
		t.Errorf("loan principal_kes = %v, want 300000", loan.PrincipalKes)
	}
	if loan.BatteryValueKes == nil || *loan.BatteryValueKes != 150000 {
		t.Errorf("loan battery_value_kes = %v, want 150000", loan.BatteryValueKes)
	}

	// The ingest/scoring resolution chain can also reach the created rows.
	resolvedBattery, err := q.ResolveBatteryID(ctx, partnerID, "battery_scoreable")
	if err != nil || resolvedBattery != created.Battery.ID {
		t.Errorf("ResolveBatteryID = %q, %v; want %q", resolvedBattery, err, created.Battery.ID)
	}
	resolvedLoan, err := q.ResolveLoanID(ctx, partnerID, "loan_scoreable")
	if err != nil || resolvedLoan != created.Loan.ID {
		t.Errorf("ResolveLoanID = %q, %v; want %q", resolvedLoan, err, created.Loan.ID)
	}
}
