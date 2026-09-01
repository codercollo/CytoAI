package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/codercollo/cytoai/internal/auth"
	"github.com/jackc/pgx/v5"
)

const demoPartnerName = "Demo Lender"

// The demo external refs intentionally match test/testdata/sample_*.csv so the
// sample files can be uploaded with the printed API key.
var (
	demoRiderRef    = "rider_0001"
	demoBatteryRefs = []string{"battery_0001", "battery_0002"}
	demoLoans       = []struct{ ref, batteryRef string }{
		{"loan_0001", "battery_0001"},
		{"loan_0002", "battery_0002"},
	}
)

func main() {
	dsn := os.Getenv("CYTOAI_POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://cyto:cyto@localhost:5433/cytoai?sslmode=disable"
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	// Idempotent: wipe any previous demo data first.
	if err := resetDemo(ctx, conn); err != nil {
		log.Fatal(err)
	}

	// 1. Partner.
	key, err := auth.GenerateAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	hash, err := auth.HashAPIKey(key)
	if err != nil {
		log.Fatal(err)
	}
	var partnerID string
	if err := conn.QueryRow(ctx,
		`INSERT INTO partners (name, api_key_hash) VALUES ($1, $2) RETURNING id`,
		demoPartnerName, hash,
	).Scan(&partnerID); err != nil {
		log.Fatal(err)
	}

	// 2. Rider.
	var riderID string
	if err := conn.QueryRow(ctx,
		`INSERT INTO riders (external_ref, partner_id) VALUES ($1, $2) RETURNING id`,
		demoRiderRef, partnerID,
	).Scan(&riderID); err != nil {
		log.Fatal(err)
	}

	// 3. Batteries.
	batteryIDs := map[string]string{}
	for _, ref := range demoBatteryRefs {
		var id string
		if err := conn.QueryRow(ctx,
			`INSERT INTO batteries (external_ref, manufacturer, rated_capacity_wh) VALUES ($1, 'Demo', 2000) RETURNING id`,
			ref,
		).Scan(&id); err != nil {
			log.Fatal(err)
		}
		batteryIDs[ref] = id
	}

	// 4. Loans linking each battery to the demo rider (this is the chain the
	//    partner-scoped ingest resolution requires).
	for _, loan := range demoLoans {
		if _, err := conn.Exec(ctx,
			`INSERT INTO loans (external_ref, rider_id, battery_id, principal_kes, battery_value_kes) VALUES ($1, $2, $3, 300000, 150000)`,
			loan.ref, riderID, batteryIDs[loan.batteryRef],
		); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Demo data seeded:")
	fmt.Println("  Partner:   ", demoPartnerName, "(", partnerID, ")")
	fmt.Println("  Rider:     ", demoRiderRef)
	fmt.Println("  Batteries: ", demoBatteryRefs)
	fmt.Println("  Loans:     ", "loan_0001, loan_0002")
	fmt.Println()
	fmt.Println("API key (put in the dashboard header, or Authorization: Bearer <key>):")
	fmt.Println(key)
	fmt.Println()
	fmt.Println("Now upload test/testdata/sample_telematics.csv and sample_repayments.csv.")
}

// resetDemo removes any prior demo partner and everything linked to it, so
// re-running this seed is safe and does not create duplicate external_refs.
func resetDemo(ctx context.Context, conn *pgx.Conn) error {
	stmts := []string{
		`DELETE FROM repayment_events WHERE loan_id IN (SELECT id FROM loans WHERE external_ref IN ('loan_0001','loan_0002'))`,
		`DELETE FROM telemetry_readings WHERE battery_id IN (SELECT id FROM batteries WHERE external_ref IN ('battery_0001','battery_0002'))`,
		`DELETE FROM scores WHERE rider_id IN (SELECT id FROM riders WHERE external_ref = 'rider_0001') OR battery_id IN (SELECT id FROM batteries WHERE external_ref IN ('battery_0001','battery_0002'))`,
		`DELETE FROM loans WHERE external_ref IN ('loan_0001','loan_0002')`,
		`DELETE FROM riders WHERE external_ref = 'rider_0001'`,
		`DELETE FROM batteries WHERE external_ref IN ('battery_0001','battery_0002')`,
		`DELETE FROM partners WHERE name = $1`,
	}
	for _, stmt := range stmts[:len(stmts)-1] {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	if _, err := conn.Exec(ctx, stmts[len(stmts)-1], demoPartnerName); err != nil {
		return err
	}
	return nil
}
