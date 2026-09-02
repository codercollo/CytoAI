package queries

import (
	"context"
	"fmt"

	"github.com/codercollo/cytoai/internal/domain"
)

// RiderBatteryPairsForPartner returns every distinct rider/battery pair owned
// by a partner that has at least one loan — i.e. the same universe
// PortfolioScores' JOIN can reach.
func (s *Store) RiderBatteryPairsForPartner(ctx context.Context, partnerID string) ([]domain.RiderBatteryPair, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT l.rider_id::text, l.battery_id::text
		FROM loans l
		JOIN riders r ON r.id = l.rider_id
		WHERE r.partner_id = $1
		  AND l.battery_id IS NOT NULL
		ORDER BY l.rider_id::text, l.battery_id::text`, partnerID)
	if err != nil {
		return nil, fmt.Errorf("queries: rider battery pairs: %w", err)
	}
	defer rows.Close()

	var out []domain.RiderBatteryPair
	for rows.Next() {
		var p domain.RiderBatteryPair
		if err := rows.Scan(&p.RiderID, &p.BatteryID); err != nil {
			return nil, fmt.Errorf("queries: scan rider battery pair: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queries: iterate rider battery pairs: %w", err)
	}
	return out, nil
}

// SwapNetworkRiderIDsForPartner returns every distinct rider whose financing
// model is swap_network (no collateral battery_id), one row per rider. These
// are excluded from RiderBatteryPairsForPartner because they have no battery to
// pair on.
func (s *Store) SwapNetworkRiderIDsForPartner(ctx context.Context, partnerID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT l.rider_id::text
		FROM loans l
		JOIN riders r ON r.id = l.rider_id
		WHERE r.partner_id = $1
		  AND l.financing_model = 'swap_network'
		ORDER BY l.rider_id::text`, partnerID)
	if err != nil {
		return nil, fmt.Errorf("queries: swap network rider ids: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("queries: scan swap rider id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queries: iterate swap rider ids: %w", err)
	}
	return out, nil
}
