// Package queries holds hand-written Postgres query functions for the Cyto AI
// API. No sqlc codegen is used.
package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps a pgx pool with the query functions in this package.
type Store struct {
	pool *pgxpool.Pool
}

// New constructs a Store over the given pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// InsertScore inserts a computed score and returns its generated id.
func (s *Store) InsertScore(ctx context.Context, score domain.Score) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO scores
		    (rider_id, battery_id, battery_health_index, repayment_risk_index, cyto_score, scored_at, model_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		nullableString(score.RiderID),
		nullableString(score.BatteryID),
		float64(score.BatteryHealthIndex),
		float64(score.RepaymentRiskIndex),
		float64(score.CytoScore),
		nullableTime(score.ScoredAt),
		nullableString(score.ModelVersion),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("queries: insert score: %w", err)
	}
	return id, nil
}

// LatestScore returns the most recent score for a rider.
func (s *Store) LatestScore(ctx context.Context, riderID string) (domain.Score, error) {
	var (
		out          domain.Score
		bhi, rri     *float64
		cyto         *float64
		rider        *string
		battery      *string
		scoredAt     *time.Time
		modelVersion *string
	)

	err := s.pool.QueryRow(ctx, `
		SELECT id, rider_id::text, battery_id::text, battery_health_index, repayment_risk_index, cyto_score, scored_at, model_version
		FROM scores
		WHERE rider_id = $1
		ORDER BY scored_at DESC, id DESC
		LIMIT 1`, riderID,
	).Scan(&out.ID, &rider, &battery, &bhi, &rri, &cyto, &scoredAt, &modelVersion)
	if err != nil {
		return domain.Score{}, fmt.Errorf("queries: latest score: %w", err)
	}

	out.RiderID = rider
	out.BatteryID = battery
	if bhi != nil {
		out.BatteryHealthIndex = domain.BatteryHealthIndex(*bhi)
	}
	if rri != nil {
		out.RepaymentRiskIndex = domain.RepaymentRiskIndex(*rri)
	}
	if cyto != nil {
		out.CytoScore = domain.CytoScore(*cyto)
	}
	out.ScoredAt = scoredAt
	out.ModelVersion = modelVersion
	return out, nil
}

// PortfolioScores returns a partner's scores, newest first. limit <= 0 means
// no limit; offset is only applied when limit > 0.
func (s *Store) PortfolioScores(ctx context.Context, partnerID string, limit, offset int) ([]domain.Score, error) {
	query := `
		SELECT s.id, s.rider_id::text, s.battery_id::text, s.battery_health_index, s.repayment_risk_index, s.cyto_score, s.scored_at, s.model_version
		FROM scores s
		JOIN riders r ON r.id = s.rider_id
		WHERE r.partner_id = $1
		ORDER BY s.scored_at DESC, s.id DESC`
	args := []any{partnerID}
	if limit > 0 {
		query += ` LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("queries: portfolio scores: %w", err)
	}
	defer rows.Close()

	var out []domain.Score
	for rows.Next() {
		var (
			sc           domain.Score
			bhi, rri     *float64
			cyto         *float64
			rider        *string
			battery      *string
			scoredAt     *time.Time
			modelVersion *string
		)
		if err := rows.Scan(&sc.ID, &rider, &battery, &bhi, &rri, &cyto, &scoredAt, &modelVersion); err != nil {
			return nil, fmt.Errorf("queries: scan portfolio score: %w", err)
		}
		sc.RiderID = rider
		sc.BatteryID = battery
		if bhi != nil {
			sc.BatteryHealthIndex = domain.BatteryHealthIndex(*bhi)
		}
		if rri != nil {
			sc.RepaymentRiskIndex = domain.RepaymentRiskIndex(*rri)
		}
		if cyto != nil {
			sc.CytoScore = domain.CytoScore(*cyto)
		}
		sc.ScoredAt = scoredAt
		sc.ModelVersion = modelVersion
		out = append(out, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queries: iterate portfolio scores: %w", err)
	}
	return out, nil
}
