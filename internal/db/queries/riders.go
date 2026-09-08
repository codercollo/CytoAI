package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
)

// riderSummaryColumns is shared by ListRiders and RiderByID. It selects a
// rider plus, via LATERAL joins, its most recent loan/battery relationship
// and its latest score (either may be absent).
const riderSummaryColumns = `
	r.id::text,
	r.external_ref,
	r.onboarded_at,
	b.id::text,
	b.external_ref,
	b.manufacturer,
	b.rated_capacity_wh,
	b.commissioned_at,
	l.id::text,
	l.external_ref,
	l.principal_kes,
	l.battery_value_kes,
	l.term_months,
	l.daily_installment_kes,
	l.started_at,
	l.financing_model,
	s.id,
	s.battery_id::text,
	s.battery_health_index,
	s.repayment_risk_index,
	s.cyto_score,
	s.scored_at,
	s.model_version`

const riderSummaryJoins = `
	FROM riders r
	LEFT JOIN LATERAL (
		SELECT * FROM loans lo WHERE lo.rider_id = r.id
		ORDER BY lo.started_at DESC NULLS LAST
		LIMIT 1
	) l ON true
	LEFT JOIN batteries b ON b.id = l.battery_id
	LEFT JOIN LATERAL (
		SELECT * FROM scores sc WHERE sc.rider_id = r.id
		ORDER BY sc.scored_at DESC NULLS LAST, sc.id DESC
		LIMIT 1
	) s ON true`

// CreateRider inserts a rider and, when battery+loan are supplied, the linked
// battery and loan rows in a single transaction. A failure on any later insert
// rolls the whole registration back so it never leaves a partially-created
// rider. Returns the persisted summary (Score is always nil here).
func (s *Store) CreateRider(ctx context.Context, partnerID string, reg domain.RiderRegistration) (domain.RiderSummary, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.RiderSummary{}, fmt.Errorf("queries: begin create rider: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var summary domain.RiderSummary
	if err := tx.QueryRow(ctx, `
		INSERT INTO riders (external_ref, partner_id)
		VALUES ($1, $2)
		RETURNING id::text, external_ref, partner_id::text, onboarded_at`,
		nullableString(reg.ExternalRef),
		partnerID,
	).Scan(&summary.Rider.ID, &summary.Rider.ExternalRef, &summary.Rider.PartnerID, &summary.Rider.OnboardedAt); err != nil {
		return domain.RiderSummary{}, fmt.Errorf("queries: insert rider: %w", err)
	}

	if reg.Battery != nil && reg.Loan != nil {
		var b domain.Battery
		if err := tx.QueryRow(ctx, `
			INSERT INTO batteries (external_ref, manufacturer, rated_capacity_wh, commissioned_at)
			VALUES ($1, $2, $3, $4)
			RETURNING id::text, external_ref, manufacturer, rated_capacity_wh, commissioned_at`,
			nullableString(reg.Battery.ExternalRef),
			nullableString(reg.Battery.Manufacturer),
			nullableFloat(reg.Battery.RatedCapacityWh),
			nullableTime(reg.Battery.CommissionedAt),
		).Scan(&b.ID, &b.ExternalRef, &b.Manufacturer, &b.RatedCapacityWh, &b.CommissionedAt); err != nil {
			return domain.RiderSummary{}, fmt.Errorf("queries: insert battery: %w", err)
		}

		finModel := "leased_fixed"
		if reg.Loan.FinancingModel != nil && *reg.Loan.FinancingModel != "" {
			finModel = *reg.Loan.FinancingModel
		}

		var l domain.Loan
		if err := tx.QueryRow(ctx, `
			INSERT INTO loans (external_ref, rider_id, battery_id, principal_kes, battery_value_kes, term_months, daily_installment_kes, started_at, financing_model)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id::text, external_ref, rider_id::text, battery_id::text, principal_kes, battery_value_kes, term_months, daily_installment_kes, started_at, financing_model`,
			nullableString(reg.Loan.ExternalRef),
			summary.Rider.ID,
			b.ID,
			nullableFloat(reg.Loan.PrincipalKes),
			nullableFloat(reg.Loan.BatteryValueKes),
			nullableInt(reg.Loan.TermMonths),
			nullableFloat(reg.Loan.DailyInstallmentKes),
			nullableTime(reg.Loan.StartedAt),
			finModel,
		).Scan(&l.ID, &l.ExternalRef, &l.RiderID, &l.BatteryID, &l.PrincipalKes, &l.BatteryValueKes, &l.TermMonths, &l.DailyInstallmentKes, &l.StartedAt, &l.FinancingModel); err != nil {
			return domain.RiderSummary{}, fmt.Errorf("queries: insert loan: %w", err)
		}
		summary.Battery = &b
		summary.Loan = &l
	} else if reg.Loan != nil {
		finModel := "swap_network"
		if reg.Loan.FinancingModel != nil && *reg.Loan.FinancingModel != "" {
			finModel = *reg.Loan.FinancingModel
		}

		var l domain.Loan
		if err := tx.QueryRow(ctx, `
			INSERT INTO loans (external_ref, rider_id, battery_id, principal_kes, battery_value_kes, term_months, daily_installment_kes, started_at, financing_model)
			VALUES ($1, $2, NULL, $3, $4, $5, $6, $7, $8)
			RETURNING id::text, external_ref, rider_id::text, battery_id::text, principal_kes, battery_value_kes, term_months, daily_installment_kes, started_at, financing_model`,
			nullableString(reg.Loan.ExternalRef),
			summary.Rider.ID,
			nullableFloat(reg.Loan.PrincipalKes),
			nullableFloat(reg.Loan.BatteryValueKes),
			nullableInt(reg.Loan.TermMonths),
			nullableFloat(reg.Loan.DailyInstallmentKes),
			nullableTime(reg.Loan.StartedAt),
			finModel,
		).Scan(&l.ID, &l.ExternalRef, &l.RiderID, &l.BatteryID, &l.PrincipalKes, &l.BatteryValueKes, &l.TermMonths, &l.DailyInstallmentKes, &l.StartedAt, &l.FinancingModel); err != nil {
			return domain.RiderSummary{}, fmt.Errorf("queries: insert loan: %w", err)
		}
		summary.Loan = &l
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.RiderSummary{}, fmt.Errorf("queries: commit create rider: %w", err)
	}
	return summary, nil
}

// ErrRiderAlreadyLinked is returned by LinkRider when the rider already has a
// loan — preventing silent overwrite through the link endpoint.
var ErrRiderAlreadyLinked = fmt.Errorf("queries: rider already has a loan linked")

// LinkRider attaches a new battery and loan to an existing identity-only rider.
// It verifies partner ownership and that the rider has no existing loan (to
// prevent silent overwrite). On success it returns the updated RiderSummary.
func (s *Store) LinkRider(ctx context.Context, partnerID, riderID string, battery domain.BatteryRegistration, loan domain.LoanRegistration) (domain.RiderSummary, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.RiderSummary{}, fmt.Errorf("queries: begin link rider: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Verify ownership and fetch the rider row inside the transaction so the
	// check and the inserts are serialised on the same connection.
	var summary domain.RiderSummary
	var onboarded *time.Time
	if err := tx.QueryRow(ctx, `
		SELECT id::text, external_ref, onboarded_at
		FROM riders
		WHERE id::text = $1 AND partner_id = $2`,
		riderID, partnerID,
	).Scan(&summary.Rider.ID, &summary.Rider.ExternalRef, &onboarded); err != nil {
		return domain.RiderSummary{}, fmt.Errorf("queries: link rider fetch: %w", err)
	}
	summary.Rider.OnboardedAt = onboarded

	// Guard: reject if a loan already exists for this rider.
	var loanCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM loans WHERE rider_id = $1`, riderID).Scan(&loanCount); err != nil {
		return domain.RiderSummary{}, fmt.Errorf("queries: link rider loan check: %w", err)
	}
	if loanCount > 0 {
		return domain.RiderSummary{}, ErrRiderAlreadyLinked
	}

	// Insert battery.
	var b domain.Battery
	if err := tx.QueryRow(ctx, `
		INSERT INTO batteries (external_ref, manufacturer, rated_capacity_wh, commissioned_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, external_ref, manufacturer, rated_capacity_wh, commissioned_at`,
		nullableString(battery.ExternalRef),
		nullableString(battery.Manufacturer),
		nullableFloat(battery.RatedCapacityWh),
		nullableTime(battery.CommissionedAt),
	).Scan(&b.ID, &b.ExternalRef, &b.Manufacturer, &b.RatedCapacityWh, &b.CommissionedAt); err != nil {
		return domain.RiderSummary{}, fmt.Errorf("queries: link rider insert battery: %w", err)
	}

	// Insert loan linked to the existing rider + new battery.
	var l domain.Loan
	if err := tx.QueryRow(ctx, `
		INSERT INTO loans (external_ref, rider_id, battery_id, principal_kes, battery_value_kes, term_months, daily_installment_kes, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, external_ref, rider_id::text, battery_id::text, principal_kes, battery_value_kes, term_months, daily_installment_kes, started_at`,
		nullableString(loan.ExternalRef),
		summary.Rider.ID,
		b.ID,
		nullableFloat(loan.PrincipalKes),
		nullableFloat(loan.BatteryValueKes),
		nullableInt(loan.TermMonths),
		nullableFloat(loan.DailyInstallmentKes),
		nullableTime(loan.StartedAt),
	).Scan(&l.ID, &l.ExternalRef, &l.RiderID, &l.BatteryID, &l.PrincipalKes, &l.BatteryValueKes, &l.TermMonths, &l.DailyInstallmentKes, &l.StartedAt); err != nil {
		return domain.RiderSummary{}, fmt.Errorf("queries: link rider insert loan: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.RiderSummary{}, fmt.Errorf("queries: commit link rider: %w", err)
	}

	summary.Battery = &b
	summary.Loan = &l
	return summary, nil
}

// ListRiders returns a partner's riders, newest first, each with its primary
// battery/loan relationship and latest score (if any). limit <= 0 means no
// limit; offset is only applied when limit > 0.
func (s *Store) ListRiders(ctx context.Context, partnerID string, limit, offset int) ([]domain.RiderSummary, error) {
	query := `SELECT ` + riderSummaryColumns + riderSummaryJoins + `
		WHERE r.partner_id = $1
		ORDER BY r.onboarded_at DESC NULLS LAST, r.id::text`
	args := []any{partnerID}
	if limit > 0 {
		query += ` LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("queries: list riders: %w", err)
	}
	defer rows.Close()

	out := make([]domain.RiderSummary, 0)
	for rows.Next() {
		sm, err := scanRiderSummary(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("queries: scan rider: %w", err)
		}
		out = append(out, sm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queries: iterate riders: %w", err)
	}
	return out, nil
}

// RiderByID returns one partner-scoped rider summary, or pgx.ErrNoRows when
// the rider does not exist or belongs to another partner. It compares against
// id::text so a malformed UUID returns "not found" rather than a cast error.
func (s *Store) RiderByID(ctx context.Context, partnerID, riderID string) (domain.RiderSummary, error) {
	query := `SELECT ` + riderSummaryColumns + riderSummaryJoins + `
		WHERE r.partner_id = $1 AND r.id::text = $2`
	sm, err := scanRiderSummary(s.pool.QueryRow(ctx, query, partnerID, riderID).Scan)
	if err != nil {
		return domain.RiderSummary{}, fmt.Errorf("queries: rider by id: %w", err)
	}
	return sm, nil
}

func scanRiderSummary(scan func(dest ...any) error) (domain.RiderSummary, error) {
	var (
		out                                             domain.RiderSummary
		onboarded, bCommissioned, loanStarted, scoredAt *time.Time
		bID, bRef, manufacturer                         *string
		loanID, loanRef, scoreBatteryID, modelVersion   *string
		financingModel                                  *string
		rated, principal, batteryValue, daily           *float64
		bhi, rri, cyto                                  *float64
		term                                            *int
		scoreID                                         *int64
	)

	if err := scan(
		&out.Rider.ID,
		&out.Rider.ExternalRef,
		&onboarded,
		&bID,
		&bRef,
		&manufacturer,
		&rated,
		&bCommissioned,
		&loanID,
		&loanRef,
		&principal,
		&batteryValue,
		&term,
		&daily,
		&loanStarted,
		&financingModel,
		&scoreID,
		&scoreBatteryID,
		&bhi,
		&rri,
		&cyto,
		&scoredAt,
		&modelVersion,
	); err != nil {
		return domain.RiderSummary{}, err
	}

	out.Rider.OnboardedAt = onboarded

	if bID != nil {
		out.Battery = &domain.Battery{
			ID:              *bID,
			ExternalRef:     bRef,
			Manufacturer:    manufacturer,
			RatedCapacityWh: rated,
			CommissionedAt:  bCommissioned,
		}
	}

	if loanID != nil {
		out.Loan = &domain.Loan{
			ID:                  *loanID,
			ExternalRef:         loanRef,
			RiderID:             out.Rider.ID,
			BatteryID:           bID,
			PrincipalKes:        principal,
			BatteryValueKes:     batteryValue,
			TermMonths:          term,
			DailyInstallmentKes: daily,
			StartedAt:           loanStarted,
			FinancingModel:      financingModel,
		}
	}

	if scoreID != nil {
		out.Score = &domain.Score{
			ID:                 *scoreID,
			RiderID:            &out.Rider.ID,
			BatteryID:          scoreBatteryID,
			BatteryHealthIndex: domain.BatteryHealthIndex(*bhi),
			RepaymentRiskIndex: domain.RepaymentRiskIndex(*rri),
			CytoScore:          domain.CytoScore(*cyto),
			ScoredAt:           scoredAt,
			ModelVersion:       modelVersion,
		}
	}

	return out, nil
}

// CreateBattery inserts a bare battery row (e.g. for fleet/swap pools).
func (s *Store) CreateBattery(ctx context.Context, reg domain.BatteryRegistration) (domain.Battery, error) {
	var b domain.Battery
	err := s.pool.QueryRow(ctx, `
		INSERT INTO batteries (external_ref, manufacturer, rated_capacity_wh, commissioned_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, external_ref, manufacturer, rated_capacity_wh, commissioned_at`,
		nullableString(reg.ExternalRef),
		nullableString(reg.Manufacturer),
		nullableFloat(reg.RatedCapacityWh),
		nullableTime(reg.CommissionedAt),
	).Scan(&b.ID, &b.ExternalRef, &b.Manufacturer, &b.RatedCapacityWh, &b.CommissionedAt)
	if err != nil {
		return domain.Battery{}, fmt.Errorf("queries: create battery: %w", err)
	}
	return b, nil
}
